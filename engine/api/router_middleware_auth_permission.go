package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/engine/featureflipping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

// PermCheckFunc defines func call to check permission
type PermCheckFunc func(ctx context.Context, w http.ResponseWriter, key string, perm int, routeVars map[string]string) error

func permissionFunc(api *API) map[string]PermCheckFunc {
	return map[string]PermCheckFunc{
		"permProjectKey":           api.checkProjectPermissions,
		"permWorkflowName":         api.checkWorkflowPermissions,
		"permWorkflowNameAdvanced": api.checkWorkflowAdvancedPermissions,
		"permGroupName":            api.checkGroupPermissions,
		"permModelName":            api.checkWorkerModelPermissions,
		"permActionName":           api.checkActionPermissions,
		"permActionBuiltinName":    api.checkActionBuiltinPermissions,
		"permTemplateSlug":         api.checkTemplateSlugPermissions,
		"permUsernamePublic":       api.checkUserPublicPermissions,
		"permUsername":             api.checkUserPermissions,
		"permConsumerID":           api.checkConsumerPermissions,
		"permSessionID":            api.checkSessionPermissions,
		"permJobID":                api.checkJobIDPermissions,
	}
}

func (api *API) checkPermission(ctx context.Context, w http.ResponseWriter, routeVar map[string]string, permission int) error {
	for key, value := range routeVar {
		if permFunc, ok := permissionFunc(api)[key]; ok {
			if err := permFunc(ctx, w, value, permission, routeVar); err != nil {
				return err
			}
		}
	}
	return nil
}

func (api *API) checkJobIDPermissions(ctx context.Context, w http.ResponseWriter, jobID string, perm int, routeVars map[string]string) error {
	ctx, end := telemetry.Span(ctx, "api.checkJobIDPermissions")
	defer end()

	id, err := strconv.ParseInt(jobID, 10, 64)
	if err != nil {
		log.ErrorWithStackTrace(ctx, sdk.WrapError(err, "unable to parse job id: %s", jobID))
		return sdk.WithStack(sdk.ErrForbidden)
	}

	runNodeJob, err := workflow.LoadNodeJobRun(ctx, api.mustDB(), api.Cache, id)
	if err != nil {
		log.ErrorWithStackTrace(ctx, sdk.WrapError(err, "unable to get job with id: %d", id))
		return sdk.WithStack(sdk.ErrForbidden)
	}

	consumer := getAPIConsumer(ctx)

	// If the expected permission if >= RX and the consumer is a worker
	// We check that the worker has took this job
	if consumer.Worker != nil {
		wk := consumer.Worker
		if wk.JobRunID != nil && runNodeJob.ID == *wk.JobRunID && perm <= sdk.PermissionReadExecute {
			return nil
		}
		return sdk.WrapError(sdk.ErrForbidden, "not authorized for job %s", jobID)
	}

	// Else we check the exec groups
	if runNodeJob.ExecGroups.HasOneOf(getAPIConsumer(ctx).GetGroupIDs()...) {
		return nil
	}
	if perm == sdk.PermissionRead {
		if isHatcheryShared(ctx) || isMaintainer(ctx) {
			return nil
		}
	} else {
		if isAdmin(ctx) {
			trackSudo(ctx, w)
			return nil
		}
	}

	return sdk.WrapError(sdk.ErrForbidden, "not authorized for job %s", jobID)
}

func (api *API) checkProjectPermissions(ctx context.Context, w http.ResponseWriter, projectKey string, requiredPerm int, routeVars map[string]string) error {
	ctx, end := telemetry.Span(ctx, "api.checkProjectPermissions")
	defer end()

	if supportMFA(ctx) && !isMFA(ctx) {
		_, requireMFA := featureflipping.IsEnabled(ctx, gorpmapping.Mapper, api.mustDB(), sdk.FeatureMFARequired, map[string]string{
			"project_key": projectKey,
		})
		if requireMFA {
			return sdk.WithStack(sdk.ErrMFARequired)
		}
	}

	proj, err := project.Load(ctx, api.mustDB(), projectKey)
	if err != nil {
		return err
	}

	consumer := getAPIConsumer(ctx)

	// A worker can only read/exec access the project of the its job.
	if consumer.Worker != nil {
		jobRunID := consumer.Worker.JobRunID
		if jobRunID != nil {
			nodeJobRun, err := workflow.LoadNodeJobRun(ctx, api.mustDB(), api.Cache, *jobRunID)
			if err != nil {
				return sdk.WrapError(sdk.ErrForbidden, "can't load node job run with id %q", *jobRunID)
			}

			if nodeJobRun.ProjectID == proj.ID && requiredPerm <= sdk.PermissionReadExecute {
				return nil
			}
		}

		return sdk.WrapError(sdk.ErrForbidden, "worker %q(%s) not authorized for project %q", consumer.Worker.Name, consumer.Worker.ID, projectKey)
	}

	perms, err := permission.LoadProjectMaxLevelPermission(ctx, api.mustDB(), []string{projectKey}, getAPIConsumer(ctx).GetGroupIDs())
	if err != nil {
		return sdk.WrapError(err, "cannot get max project permissions for %s", projectKey)
	}

	callerPermission := perms.Level(projectKey)
	// If the caller based on its group doesn't have enough permission level
	if callerPermission < requiredPerm {
		log.Debug(ctx, "checkProjectPermissions> callerPermission=%d ", callerPermission)
		// If it's about READ: we have to check if the user is a maintainer or an admin
		if requiredPerm == sdk.PermissionRead {
			if !isMaintainer(ctx) {
				// The caller doesn't enough permission level from its groups and is neither a maintainer nor an admin
				log.Debug(ctx, "checkProjectPermissions> %s(%s) is not authorized to %s", getAPIConsumer(ctx).Name, getAPIConsumer(ctx).ID, projectKey)
				return sdk.WrapError(sdk.ErrNoProject, "not authorized for project %s", projectKey)
			}
			if isMaintainer(ctx) {
				log.Debug(ctx, "checkProjectPermissions> %s(%s) access granted to %s because is maintainer", getAPIConsumer(ctx).Name, getAPIConsumer(ctx).ID, projectKey)
				telemetry.Current(ctx, telemetry.Tag(telemetry.TagPermission, "is_maintainer"))
			}
			return nil
		}

		// If it's about Execute of Write: we have to check if the user is an admin
		if !isAdmin(ctx) {
			// The caller doesn't enough permission level from its groups and is not an admin
			log.Debug(ctx, "checkProjectPermissions> %s(%s) is not authorized to %s", getAPIConsumer(ctx).Name, getAPIConsumer(ctx).ID, projectKey)
			return sdk.WrapError(sdk.ErrForbidden, "not authorized for project %s", projectKey)
		}
		log.Debug(ctx, "checkProjectPermissions> %s(%s) access granted to %s because is admin", getAPIConsumer(ctx).Name, getAPIConsumer(ctx).ID, projectKey)
		telemetry.Current(ctx, telemetry.Tag(telemetry.TagPermission, "is_admin"))
		trackSudo(ctx, w)
		return nil
	}
	log.Debug(ctx, "checkProjectPermissions> %s(%s) access granted to %s because has permission (max permission = %d)", getAPIConsumer(ctx).Name, getAPIConsumer(ctx).ID, projectKey, callerPermission)
	telemetry.Current(ctx, telemetry.Tag(telemetry.TagPermission, "is_granted"))
	return nil
}

func (api *API) checkWorkflowPermissions(ctx context.Context, w http.ResponseWriter, workflowName string, perm int, routeVars map[string]string) error {
	return api.checkWorkflowPermissionsWithOpts(CheckWorkflowPermissionsOpts{})(ctx, w, workflowName, perm, routeVars)
}

// Same as checkWorkflowPermissions but also allows GET for workers on same project's workflows.
// This is needed as artifact download is allowed from a workflow to another in the same project.
func (api *API) checkWorkflowAdvancedPermissions(ctx context.Context, w http.ResponseWriter, workflowName string, perm int, routeVars map[string]string) error {
	return api.checkWorkflowPermissionsWithOpts(
		CheckWorkflowPermissionsOpts{
			AllowGETForWorkerOnSameProject: true,
			AllowHooks:                     true,
		})(ctx, w, workflowName, perm, routeVars)
}

type CheckWorkflowPermissionsOpts struct {
	AllowGETForWorkerOnSameProject bool
	AllowHooks                     bool
}

func (api *API) checkWorkflowPermissionsWithOpts(opts CheckWorkflowPermissionsOpts) PermCheckFunc {
	return func(ctx context.Context, w http.ResponseWriter, workflowName string, perm int, routeVars map[string]string) error {
		ctx, end := telemetry.Span(ctx, "api.checkWorkflowPermissions")
		defer end()

		projectKey, has := routeVars["permProjectKey"]
		if projectKey == "" {
			projectKey, has = routeVars["key"]
		}
		if !has {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		if supportMFA(ctx) && !isMFA(ctx) {
			_, requireMFA := featureflipping.IsEnabled(ctx, gorpmapping.Mapper, api.mustDB(), sdk.FeatureMFARequired, map[string]string{
				"project_key": projectKey,
			})
			if requireMFA {
				return sdk.WithStack(sdk.ErrMFARequired)
			}
		}

		if workflowName == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "invalid given workflow name")
		}

		exists, err := workflow.Exists(ctx, api.mustDB(), projectKey, workflowName)
		if err != nil {
			return err
		}
		if !exists {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		consumer := getAPIConsumer(ctx)

		// A worker can only read/exec access the workflow of the its job.
		if consumer.Worker != nil {
			jobRunID := consumer.Worker.JobRunID
			if jobRunID != nil {
				nodeJobRun, err := workflow.LoadNodeJobRun(ctx, api.mustDB(), api.Cache, *jobRunID)
				if err != nil {
					return sdk.WrapError(sdk.ErrForbidden, "can't load node job run with id %q", *jobRunID)
				}

				nodeRun, err := workflow.LoadNodeRunByID(ctx, api.mustDB(), nodeJobRun.WorkflowNodeRunID, workflow.LoadRunOptions{})
				if err != nil {
					return sdk.WrapError(sdk.ErrForbidden, "can't load node run with id %q", nodeJobRun.WorkflowNodeRunID)
				}

				daoTarget := workflow.LoadOptions{Minimal: true}.GetWorkflowDAO()
				daoTarget.Filters.ProjectKey = projectKey
				daoTarget.Filters.WorkflowName = workflowName
				targetWf, err := daoTarget.Load(ctx, api.mustDB())
				if err != nil {
					return err
				}

				if nodeRun.WorkflowID == targetWf.ID && perm <= sdk.PermissionReadExecute {
					return nil
				}

				daoSrc := workflow.LoadOptions{Minimal: true}.GetWorkflowDAO()
				daoSrc.Filters.WorkflowIDs = []int64{nodeRun.WorkflowID}
				workerSrcWf, err := daoSrc.Load(ctx, api.mustDB())
				if err != nil {
					return sdk.WrapError(err, "can't load worker source workflow with id %d on project %q", nodeRun.WorkflowID, projectKey)
				}

				if projectKey == workerSrcWf.ProjectKey && perm == sdk.PermissionRead && opts.AllowGETForWorkerOnSameProject {
					return nil
				}
			}

			return sdk.WrapError(sdk.ErrForbidden, "worker %q(%s) not authorized for workflow %s/%s", consumer.Worker.Name, consumer.Worker.ID, projectKey, workflowName)
		}

		perms, err := permission.LoadWorkflowMaxLevelPermission(ctx, api.mustDB(), projectKey, []string{workflowName}, getAPIConsumer(ctx).GetGroupIDs())
		if err != nil {
			return sdk.NewError(sdk.ErrForbidden, err)
		}

		maxLevelPermission := perms.Level(workflowName)

		if maxLevelPermission < perm { // If the caller based on its group doesn have enough permission level
			// If it's about READ: we have to check if the user is a maintainer or an admin
			if perm < sdk.PermissionReadExecute {
				if isMaintainer(ctx) || (isHooks(ctx) && opts.AllowHooks) {
					if isHooks(ctx) {
						log.Debug(ctx, "checkWorkflowPermissions> %s access granted to %s/%s because is hooks service", getAPIConsumer(ctx).ID, projectKey, workflowName)
						telemetry.Current(ctx, telemetry.Tag(telemetry.TagPermission, "is_hooks"))
					} else if isMaintainer(ctx) {
						log.Debug(ctx, "checkWorkflowPermissions> %s access granted to %s/%s because is maintainer", getAPIConsumer(ctx).ID, projectKey, workflowName)
						telemetry.Current(ctx, telemetry.Tag(telemetry.TagPermission, "is_maintainer"))
					}
					return nil
				}
				// The caller doesn't enough permission level from its groups and is neither a maintainer nor an admin
				log.Debug(ctx, "checkWorkflowPermissions> %s is not authorized to %s/%s", getAPIConsumer(ctx).ID, projectKey, workflowName)
				return sdk.WrapError(sdk.ErrForbidden, "not authorized for workflow %s/%s", projectKey, workflowName)
			}

			// If it's about Execute of Write: we have to check if the user is an admin or if it hooks service
			if isAdmin(ctx) || (isHooks(ctx) && opts.AllowHooks) {
				if isHooks(ctx) {
					log.Debug(ctx, "checkWorkflowPermissions> %s access granted to %s/%s because is hooks service", getAPIConsumer(ctx).ID, projectKey, workflowName)
					telemetry.Current(ctx, telemetry.Tag(telemetry.TagPermission, "is_hooks"))
				} else if isAdmin(ctx) {
					log.Debug(ctx, "checkWorkflowPermissions> %s access granted to %s/%s because is admin", getAPIConsumer(ctx).ID, projectKey, workflowName)
					telemetry.Current(ctx, telemetry.Tag(telemetry.TagPermission, "is_admin"))
					trackSudo(ctx, w)
				}
				return nil
			}

			// The caller doesn't enough permission level from its groups and is not an admin
			log.Debug(ctx, "checkWorkflowPermissions> %s is not authorized to %s/%s", getAPIConsumer(ctx).ID, projectKey, workflowName)
			return sdk.WrapError(sdk.ErrForbidden, "not authorized for workflow %s/%s", projectKey, workflowName)
		}
		log.Debug(ctx, "checkWorkflowPermissions> %s access granted to %s/%s because has permission (max permission = %d)", getAPIConsumer(ctx).ID, projectKey, workflowName, maxLevelPermission)
		telemetry.Current(ctx, telemetry.Tag(telemetry.TagPermission, "is_granted"))
		return nil
	}
}

func (api *API) checkGroupPermissions(ctx context.Context, w http.ResponseWriter, groupName string, permissionValue int, routeVars map[string]string) error {
	if groupName == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid given group name")
	}

	// check that group exists
	g, err := group.LoadByName(ctx, api.mustDB(), groupName, group.LoadOptions.WithMembers)
	if err != nil {
		return sdk.WrapError(err, "cannot get group for name %s", groupName)
	}

	log.Debug(ctx, "api.checkGroupPermissions> group %d has members %v", g.ID, g.Members)

	if isWorker(ctx) {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	if permissionValue > sdk.PermissionRead {
		// Hatcheries started for "shared.infra" group are granted for group "shared.infra"
		if isHatcheryShared(ctx) {
			return nil
		}
		// Only group administror or CDS administrator can update a group or its dependencies
		if !isGroupAdmin(ctx, g) {
			if isAdmin(ctx) {
				trackSudo(ctx, w)
			} else {
				return sdk.WithStack(sdk.ErrForbidden)
			}
		}
	} else {
		if !isGroupMember(ctx, g) && !isMaintainer(ctx) { // Only group member or CDS maintainer can get a group or its dependencies
			return sdk.WithStack(sdk.ErrForbidden)
		}
	}

	return nil
}

func (api *API) checkWorkerModelPermissions(ctx context.Context, w http.ResponseWriter, modelName string, perm int, routeVars map[string]string) error {
	if modelName == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid worker model name")
	}

	g, err := group.LoadByName(ctx, api.mustDB(), routeVars["permGroupName"])
	if err != nil {
		return err
	}

	if _, err := workermodel.LoadByNameAndGroupID(ctx, api.mustDB(), modelName, g.ID, workermodel.LoadOptions.Default); err != nil {
		return err
	}

	return nil
}

func (api *API) checkActionPermissions(ctx context.Context, w http.ResponseWriter, actionName string, perm int, routeVars map[string]string) error {
	if actionName == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid action name")
	}

	g, err := group.LoadByName(ctx, api.mustDB(), routeVars["permGroupName"])
	if err != nil {
		return err
	}

	a, err := action.LoadTypeDefaultByNameAndGroupID(ctx, api.mustDB(), actionName, g.ID)
	if err != nil {
		return err
	}
	if a == nil {
		return sdk.WithStack(sdk.ErrNotFound)
	}

	return nil
}

func (api *API) checkActionBuiltinPermissions(ctx context.Context, w http.ResponseWriter, actionName string, perm int, routeVars map[string]string) error {
	if actionName == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid given action name")
	}

	a, err := action.LoadByTypesAndName(ctx, api.mustDB(), []string{sdk.BuiltinAction, sdk.PluginAction}, actionName)
	if err != nil {
		return err
	}
	if a == nil {
		return sdk.WithStack(sdk.ErrNoAction)
	}

	return nil
}

func (api *API) checkTemplateSlugPermissions(ctx context.Context, w http.ResponseWriter, templateSlug string, permissionValue int, routeVars map[string]string) error {
	if templateSlug == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid workflow template slug")
	}

	g, err := group.LoadByName(ctx, api.mustDB(), routeVars["permGroupName"])
	if err != nil {
		return err
	}

	wt, err := workflowtemplate.LoadBySlugAndGroupID(ctx, api.mustDB(), templateSlug, g.ID)
	if err != nil {
		return err
	}
	if wt == nil {
		return sdk.WithStack(sdk.ErrNotFound)
	}

	return nil
}

// checkUserPublicPermissions give user R to everyone, RW to itself and RW to admin.
func (api *API) checkUserPublicPermissions(ctx context.Context, w http.ResponseWriter, username string, permissionValue int, routeVars map[string]string) error {
	if username == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid username")
	}

	consumer := getAPIConsumer(ctx)

	var u *sdk.AuthentifiedUser
	var err error

	// Load user from database, returns an error if not exists
	if username == "me" {
		u, err = user.LoadByID(ctx, api.mustDB(), consumer.AuthentifiedUserID)
	} else {
		u, err = user.LoadByUsername(ctx, api.mustDB(), username)
	}
	if err != nil {
		return sdk.NewErrorWithStack(err, sdk.WrapError(sdk.ErrForbidden, "not authorized for user %s", username))
	}

	// Valid if the current consumer match given username
	if consumer.AuthentifiedUserID == u.ID {
		log.Debug(ctx, "checkUserPermissions> %s read/write access granted to %s because itself", getAPIConsumer(ctx).ID, u.ID)
		return nil
	}

	// Everyone can read public user data
	if permissionValue == sdk.PermissionRead {
		log.Debug(ctx, "checkUserPermissions> %s read access granted to %s on public user data", getAPIConsumer(ctx).ID, u.ID)
		return nil
	}

	// If the current user is an admin
	if isAdmin(ctx) {
		log.Debug(ctx, "checkUserPermissions> %s read/write access granted to %s because is admin", getAPIConsumer(ctx).ID, u.ID)
		trackSudo(ctx, w)
		return nil
	}

	log.Debug(ctx, "checkUserPermissions> %s is not authorized to %s", getAPIConsumer(ctx).ID, u.ID)
	return sdk.WrapError(sdk.ErrForbidden, "not authorized for user %s", username)
}

// checkUserPermissions give user RW to itself, R to maintainer and RW to admin.
func (api *API) checkUserPermissions(ctx context.Context, w http.ResponseWriter, username string, permissionValue int, routeVars map[string]string) error {
	if username == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid username")
	}

	consumer := getAPIConsumer(ctx)

	var u *sdk.AuthentifiedUser
	var err error

	// Load user from database, returns an error if not exists
	if username == "me" {
		u, err = user.LoadByID(ctx, api.mustDB(), consumer.AuthentifiedUserID)
	} else {
		u, err = user.LoadByUsername(ctx, api.mustDB(), username)
	}
	if err != nil {
		return sdk.NewErrorWithStack(err, sdk.WrapError(sdk.ErrForbidden, "not authorized for user %s", username))
	}

	// Valid if the current consumer match given username
	if consumer.AuthentifiedUserID == u.ID {
		log.Debug(ctx, "checkUserPermissions> %s read/write access granted to %s because itself", getAPIConsumer(ctx).ID, u.ID)
		return nil
	}

	// If the current user is a maintainer and we want to read a user
	if permissionValue == sdk.PermissionRead && isMaintainer(ctx) {
		log.Debug(ctx, "checkUserPermissions> %s read access granted to %s because is maintainer", getAPIConsumer(ctx).ID, u.ID)
		return nil
	}

	// If the current user is an admin, gives RW on the user
	if isAdmin(ctx) {
		log.Debug(ctx, "checkUserPermissions> %s read/write access granted to %s because is admin", getAPIConsumer(ctx).ID, u.ID)
		trackSudo(ctx, w)
		return nil
	}

	log.Debug(ctx, "checkUserPermissions> %s is not authorized to %s", getAPIConsumer(ctx).ID, u.ID)
	return sdk.WrapError(sdk.ErrForbidden, "not authorized for user %s", username)
}

func (api *API) checkConsumerPermissions(ctx context.Context, w http.ResponseWriter, consumerID string, permissionValue int, routeVars map[string]string) error {
	if consumerID == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given consumer id")
	}

	authConsumer := getAPIConsumer(ctx)
	consumer, err := authentication.LoadConsumerByID(ctx, api.mustDB(), consumerID)
	if err != nil {
		return sdk.NewErrorWithStack(err, sdk.WrapError(sdk.ErrForbidden, "not authorized for consumer %s", consumerID))
	}

	// If current consumer's authentified user match given one
	if consumer.AuthentifiedUserID == authConsumer.AuthentifiedUserID {
		log.Debug(ctx, "checkConsumerPermissions> %s access granted to %s because is owner", authConsumer.ID, consumer.ID)
		return nil
	}

	// If the current user is a maintainer and we want to read a consumer
	if permissionValue == sdk.PermissionRead && isMaintainer(ctx) {
		log.Debug(ctx, "checkConsumerPermissions> %s read access granted to %s because is maintainer", authConsumer.ID, consumer.ID)
		return nil
	}

	// If the current user is an admin, gives RW on the consumer
	if isAdmin(ctx) {
		log.Debug(ctx, "checkConsumerPermissions> %s read/write access granted to %s because is admin", authConsumer.ID, consumer.ID)
		trackSudo(ctx, w)
		return nil
	}

	log.Debug(ctx, "checkConsumerPermissions> %s is not authorized to %s", authConsumer.ID, consumer.ID)
	return sdk.WrapError(sdk.ErrForbidden, "not authorized for consumer %s", consumerID)
}

func (api *API) checkSessionPermissions(ctx context.Context, w http.ResponseWriter, sessionID string, permissionValue int, routeVars map[string]string) error {
	if sessionID == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given session id")
	}

	authConsumer := getAPIConsumer(ctx)
	session, err := authentication.LoadSessionByID(ctx, api.mustDB(), sessionID)
	if err != nil {
		return sdk.NewErrorWithStack(err, sdk.WrapError(sdk.ErrForbidden, "not authorized for session %s", sessionID))
	}
	consumer, err := authentication.LoadConsumerByID(ctx, api.mustDB(), session.ConsumerID)
	if err != nil {
		return sdk.NewErrorWithStack(err, sdk.WrapError(sdk.ErrForbidden, "not authorized for session %s", sessionID))
	}

	// If current consumer's authentified user match session's consumer
	if consumer.AuthentifiedUserID == authConsumer.AuthentifiedUserID {
		log.Debug(ctx, "checkSessionPermissions> %s access granted to %s because is owner", authConsumer.ID, session.ID)
		return nil
	}

	// If the current user is a maintainer and we want to read a session
	if permissionValue == sdk.PermissionRead && isMaintainer(ctx) {
		log.Debug(ctx, "checkSessionPermissions> %s read access granted to %s because is maintainer", authConsumer.ID, session.ID)
		return nil
	}

	// If the current user is an admin, gives RW on the session
	if isAdmin(ctx) {
		log.Debug(ctx, "checkSessionPermissions> %s read/write access granted to %s because is admin", authConsumer.ID, session.ID)
		trackSudo(ctx, w)
		return nil
	}

	log.Debug(ctx, "checkSessionPermissions> %s is not authorized to %s", authConsumer.ID, session.ID)
	return sdk.WrapError(sdk.ErrForbidden, "not authorized for session %s", sessionID)
}
