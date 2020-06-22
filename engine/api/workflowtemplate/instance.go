package workflowtemplate

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

type TemplateRequestModifierFunc func(wt sdk.WorkflowTemplate, req *sdk.WorkflowTemplateRequest) error

var TemplateRequestModifiers = struct {
	Detached                   TemplateRequestModifierFunc
	DefaultKeys                func(proj sdk.Project) TemplateRequestModifierFunc
	DefaultNameAndRepositories func(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj sdk.Project, repoURL string) TemplateRequestModifierFunc
}{
	Detached:                   requestModifyDetached,
	DefaultKeys:                requestModifyDefaultKeysfunc,
	DefaultNameAndRepositories: requestModifyDefaultNameAndRepositories,
}

func requestModifyDetached(wt sdk.WorkflowTemplate, req *sdk.WorkflowTemplateRequest) error {
	req.Detached = true
	return nil
}

func requestModifyDefaultKeysfunc(proj sdk.Project) TemplateRequestModifierFunc {
	return func(wt sdk.WorkflowTemplate, req *sdk.WorkflowTemplateRequest) error {
		defaultSSHKey := sdk.GenerateProjectDefaultKeyName(proj.Key, sdk.KeyTypeSSH)
		defaultPGPKey := sdk.GenerateProjectDefaultKeyName(proj.Key, sdk.KeyTypePGP)
		var defaultSSHKeyFound, defaultPGPKeyFound bool
		for _, p := range proj.Keys {
			if p.Type == sdk.KeyTypeSSH && p.Name == defaultSSHKey {
				defaultSSHKeyFound = true
			}
			if p.Type == sdk.KeyTypePGP && p.Name == defaultPGPKey {
				defaultPGPKeyFound = true
			}
			if defaultSSHKeyFound && defaultPGPKeyFound {
				break
			}
		}
		if req.Parameters == nil {
			req.Parameters = make(map[string]string)
		}
		for _, p := range wt.Parameters {
			if _, ok := req.Parameters[p.Key]; ok || !p.Required {
				continue
			}
			if p.Type == sdk.ParameterTypeSSHKey && defaultSSHKeyFound {
				req.Parameters[p.Key] = defaultSSHKey
			}
			if p.Type == sdk.ParameterTypePGPKey && defaultPGPKeyFound {
				req.Parameters[p.Key] = defaultPGPKey
			}
		}
		return nil
	}
}

func requestModifyDefaultNameAndRepositories(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj sdk.Project, repoURL string) TemplateRequestModifierFunc {
	return func(wt sdk.WorkflowTemplate, req *sdk.WorkflowTemplateRequest) error {
		var repoPath string
	loopVCSServer:
		for _, vcs := range proj.VCSServers {
			repos, err := repositoriesmanager.GetReposForProjectVCSServer(ctx, db, store, proj, vcs.Name, repositoriesmanager.Options{})
			if err != nil {
				return err
			}
			for _, r := range repos {
				path := fmt.Sprintf("%s/%s", vcs.Name, r.Fullname)
				if repoURL == r.HTTPCloneURL || repoURL == r.SSHCloneURL {
					repoPath = path
					break loopVCSServer
				}
			}
		}

		if repoPath == "" {
			return nil
		}

		splittedPath := strings.Split(repoPath, "/")
		repoName := splittedPath[len(splittedPath)-1]
		if req.WorkflowName == "" {
			req.WorkflowName = repoName
		}
		if req.Parameters == nil {
			req.Parameters = make(map[string]string)
		}
		for _, p := range wt.Parameters {
			if _, ok := req.Parameters[p.Key]; ok || !p.Required {
				continue
			}
			if p.Type == sdk.ParameterTypeRepository {
				req.Parameters[p.Key] = repoPath
			}
		}

		return nil
	}
}

// CheckAndExecuteTemplate will execute the workflow template if given workflow components contains a template instance.
// When detached is set this will not create/update any template instance in database (this is useful for workflow ascode branches).
func CheckAndExecuteTemplate(ctx context.Context, db *gorp.DbMap, consumer sdk.AuthConsumer, p sdk.Project,
	data *exportentities.WorkflowComponents, mods ...TemplateRequestModifierFunc) ([]sdk.Message, *sdk.WorkflowTemplateInstance, error) {
	var allMsgs []sdk.Message

	if data.Template.From == "" {
		return allMsgs, nil, nil
	}

	groupName, templateSlug, templateVersion, err := data.Template.ParseFrom()
	if err != nil {
		return allMsgs, nil, err
	}

	// check that group exists
	grp, err := group.LoadByName(ctx, db, groupName)
	if err != nil {
		return allMsgs, nil, err
	}

	var groupPermissionValid bool
	if consumer.Admin() || consumer.Maintainer() {
		groupPermissionValid = true
	} else if grp.ID == group.SharedInfraGroup.ID {
		groupPermissionValid = true
	} else {
		groupIDs := consumer.GetGroupIDs()
		for i := range groupIDs {
			if groupIDs[i] == grp.ID {
				groupPermissionValid = true
				break
			}
		}
	}
	if !groupPermissionValid {
		return allMsgs, nil, sdk.NewErrorFrom(sdk.ErrWrongRequest, "could not find given workflow template")
	}

	wt, err := LoadBySlugAndGroupID(ctx, db, templateSlug, grp.ID, LoadOptions.Default)
	if err != nil {
		return allMsgs, nil, sdk.NewErrorFrom(err, "could not find a template with slug %s in group %s", templateSlug, grp.Name)
	}
	if templateVersion > 0 {
		wta, err := LoadAuditByTemplateIDAndVersion(ctx, db, wt.ID, templateVersion)
		if err != nil {
			return allMsgs, nil, err
		}
		wt = &wta.DataAfter
	}

	allMsgs = append(allMsgs, sdk.NewMessage(sdk.MsgWorkflowGeneratedFromTemplateVersion, wt.PathWithVersion()))

	req := sdk.WorkflowTemplateRequest{
		ProjectKey:   p.Key,
		WorkflowName: data.Template.Name,
		Parameters:   data.Template.Parameters,
	}
	for i := range mods {
		if err := mods[i](*wt, &req); err != nil {
			return allMsgs, nil, err
		}
	}

	if err := wt.CheckParams(req); err != nil {
		return allMsgs, nil, err
	}

	var result exportentities.WorkflowComponents

	if req.Detached {
		wti := &sdk.WorkflowTemplateInstance{
			ID:                      time.Now().Unix(), // if is a detached apply set an id based on time
			ProjectID:               p.ID,
			WorkflowTemplateID:      wt.ID,
			WorkflowTemplateVersion: wt.Version,
			Request:                 req,
		}

		// execute template with request
		result, err = Execute(*wt, *wti)
		if err != nil {
			return allMsgs, nil, err
		}

		// do not return an instance if detached
		*data = result
		return allMsgs, wti, nil
	}

	tx, err := db.Begin()
	if err != nil {
		return allMsgs, nil, sdk.WrapError(err, "cannot start transaction")
	}
	defer tx.Rollback() // nolint

	// try to get a instance not assign to a workflow but with the same slug
	wti, err := LoadInstanceByTemplateIDAndProjectIDAndRequestWorkflowName(ctx, tx, wt.ID, p.ID, req.WorkflowName)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return allMsgs, nil, err
	}

	// if a previous instance exist for the same workflow update it, else create a new one
	var old *sdk.WorkflowTemplateInstance
	if wti != nil {
		clone := sdk.WorkflowTemplateInstance(*wti)
		old = &clone
		wti.WorkflowTemplateVersion = wt.Version
		wti.Request = req
		if err := UpdateInstance(tx, wti); err != nil {
			return allMsgs, nil, err
		}
	} else {
		wti = &sdk.WorkflowTemplateInstance{
			ProjectID:               p.ID,
			WorkflowTemplateID:      wt.ID,
			WorkflowTemplateVersion: wt.Version,
			Request:                 req,
		}
		// only store the new instance if request is not for a detached workflow
		if err := InsertInstance(tx, wti); err != nil {
			return allMsgs, nil, err
		}
	}

	// execute template with request
	result, err = Execute(*wt, *wti)
	if err != nil {
		return allMsgs, nil, err
	}

	// parse the generated workflow to find its name an update it in instance if not detached
	// also set the template path in generated workflow if not detached
	wti.WorkflowName = result.Workflow.GetName()
	if err := UpdateInstance(tx, wti); err != nil {
		return allMsgs, nil, err
	}

	if old != nil {
		if err := CreateAuditInstanceUpdate(tx, *old, *wti, consumer); err != nil {
			return allMsgs, nil, err
		}
	} else if !req.Detached {
		if err := CreateAuditInstanceAdd(tx, *wti, consumer); err != nil {
			return allMsgs, nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return allMsgs, nil, sdk.WithStack(err)
	}

	// if the template was successfully executed we want to return only the a file with template instance data
	*data = result
	return allMsgs, wti, nil
}

// UpdateTemplateInstanceWithWorkflow will perform some action after a successful workflow push, if it was generated
// from a template we want to set the workflow id on generated template instance.
func UpdateTemplateInstanceWithWorkflow(ctx context.Context, db gorp.SqlExecutor, w sdk.Workflow,
	u sdk.Identifiable, wti *sdk.WorkflowTemplateInstance) error {
	if wti == nil || wti.Request.Detached {
		return nil
	}

	// remove existing relations between workflow and template
	if err := DeleteInstanceNotIDAndWorkflowID(db, wti.ID, w.ID); err != nil {
		return err
	}

	old := sdk.WorkflowTemplateInstance(*wti)

	// set the workflow id on target instance
	log.Debug("SetTemplateData> setting workflow ID=%d on template instance %d", w.ID, wti.ID)
	wti.WorkflowID = &w.ID
	if err := UpdateInstance(db, wti); err != nil {
		return err
	}

	if err := CreateAuditInstanceUpdate(db, old, *wti, u); err != nil {
		return err
	}

	return nil
}
