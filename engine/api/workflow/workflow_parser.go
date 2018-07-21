package workflow

import (
	"context"
	"sync"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/tracing"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

// ImportOptions is option to parse a workflow
type ImportOptions struct {
	DryRun       bool
	Force        bool
	WorkflowName string
}

// Parse parse an exportentities.workflow and return the parsed workflow
func Parse(proj *sdk.Project, ew *exportentities.Workflow) (*sdk.Workflow, error) {
	log.Info("Parse>> Parse workflow %s in project %s", ew.Name, proj.Key)
	log.Debug("Parse>> Workflow: %+v", ew)

	//Check valid application name
	rx := sdk.NamePatternRegex
	if !rx.MatchString(ew.Name) {
		return nil, sdk.WrapError(sdk.ErrInvalidApplicationPattern, "Parse>> Workflow name %s do not respect pattern %s", ew.Name, sdk.NamePattern)
	}

	//Inherit permissions from project
	if len(ew.Permissions) == 0 {
		ew.Permissions = make(map[string]int)
		for _, p := range proj.ProjectGroups {
			ew.Permissions[p.Group.Name] = p.Permission
		}
	}

	//Parse workflow
	w, errW := ew.GetWorkflow()
	if errW != nil {
		return nil, sdk.NewError(sdk.ErrWrongRequest, errW)
	}
	w.ProjectID = proj.ID
	w.ProjectKey = proj.Key

	return w, nil
}

// ParseAndImport parse an exportentities.workflow and insert or update the workflow in database
func ParseAndImport(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, ew *exportentities.Workflow, u *sdk.User, opts ImportOptions) (*sdk.Workflow, []sdk.Message, error) {
	ctx, end := tracing.Span(ctx, "workflow.ParseAndImport")
	defer end()

	log.Info("ParseAndImport>> Import workflow %s in project %s (force=%v)", ew.Name, proj.Key, opts.Force)
	log.Debug("ParseAndImport>> Workflow: %+v", ew)
	//Parse workflow
	w, errW := Parse(proj, ew)
	if errW != nil {
		return nil, nil, errW
	}

	if opts.WorkflowName != "" && w.Name != opts.WorkflowName {
		return nil, nil, sdk.ErrWorkflowNameImport
	}

	//Import
	done := new(sync.WaitGroup)
	done.Add(1)
	msgChan := make(chan sdk.Message)
	msgList := []sdk.Message{}
	go func(array *[]sdk.Message) {
		defer done.Done()
		for {
			m, more := <-msgChan
			if !more {
				return
			}
			*array = append(*array, m)
		}
	}(&msgList)

	globalError := Import(ctx, db, store, proj, w, u, opts.Force, msgChan, opts.DryRun)
	close(msgChan)
	done.Wait()

	return w, msgList, globalError
}
