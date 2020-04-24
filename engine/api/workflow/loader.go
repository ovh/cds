package workflow

import (
	"context"
	"fmt"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk/log"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
)

type LoadAllWorkflowsOptions struct {
	Filter struct {
		ProjectKey   string
		WorkflowName string
		VCSServer    string
		Repository   string
		GroupIDs     gorpmapping.IDs
	}
	Options struct {
		WithApplications       bool
		WithPipelines          bool
		WithEnvironments       bool
		WithIntegrations       bool
		WithIcon               bool
		WithAsCodeUpdateEvents bool
		WithTemplate           bool
	}
	Offset    int
	Limit     int
	Ascending bool
}

func (opt LoadAllWorkflowsOptions) Query() gorpmapping.Query {
	var queryString = `
	WITH 
    workflow_root_application_id AS (
        SELECT 
            id as "workflow_id", 
            project_id,
            name as "workflow_name",
            (workflow_data -> 'node' -> 'context' ->> 'application_id')::BIGINT as "root_application_id"
        FROM workflow
    ),
    project_permission AS (
        SELECT 
            project_id,
            ARRAY_AGG(group_id) as "groups"
        FROM project_group
        GROUP BY project_id
	),
	selected_workflow AS (
		SELECT 
			project.id, 
			workflow_root_application_id.workflow_id, 
			project.projectkey, 
			workflow_name, 
			application.id, 
			application.name, 
			application.vcs_server, 
			application.repo_fullname, 
			project_permission.groups
		FROM workflow_root_application_id
		LEFT OUTER JOIN application ON application.id = root_application_id
		JOIN project ON project.id = workflow_root_application_id.project_id
		JOIN project_permission ON project_permission.project_id = project.id	
	)
	SELECT * 
	FROM workflow 
	JOIN selected_workflow ON selected_workflow.workflow_id = workflow.id
	`

	var filters []string
	var args []interface{}
	if opt.Filter.ProjectKey != "" {
		filters = append(filters, "selected_workflow.projectkey = $%d")
		args = append(args, opt.Filter.ProjectKey)
	}
	if opt.Filter.WorkflowName != "" {
		filters = append(filters, "selected_workflow.workflow_name = $%d")
		args = append(args, opt.Filter.WorkflowName)
	}
	if opt.Filter.VCSServer != "" {
		filters = append(filters, "selected_workflow.vcs_server = $%d")
		args = append(args, opt.Filter.VCSServer)
	}
	if opt.Filter.Repository != "" {
		filters = append(filters, "selected_workflow.repo_fullname = $%d")
		args = append(args, opt.Filter.Repository)
	}
	if len(opt.Filter.GroupIDs) != 0 {
		filters = append(filters, "selected_workflow.groups && $%d")
		args = append(args, opt.Filter.GroupIDs)
	}

	for i, f := range filters {
		if i == 0 {
			queryString += " WHERE "
		} else {
			queryString += " AND "
		}
		queryString += fmt.Sprintf(f, i)
	}

	var order = " ORDER BY selected_workflow.projectkey, selected_workflow.workflow_name "
	if opt.Ascending {
		order += "ASC"
	} else {
		order += "DESC"
	}
	queryString += order

	if opt.Offset != 0 {
		queryString += fmt.Sprintf(" OFFSET %d", opt.Offset)
	}

	if opt.Limit != 0 {
		queryString += fmt.Sprintf(" LIMIT %d", opt.Limit)
	}

	q := gorpmapping.NewQuery(queryString).Args(args...)

	log.Debug("workflow.LoadAllWorkflowsOptions.Query> %v", q)

	return q
}

func (opt LoadAllWorkflowsOptions) Loaders() []gorpmapping.GetOptionFunc {
	return nil
}

func LoadAllWorkflows(ctx context.Context, db gorp.SqlExecutor, opts LoadAllWorkflowsOptions) ([]sdk.Workflow, error) {
	var workflows []sdk.Workflow
	if err := gorpmapping.GetAll(ctx, db, opts.Query(), &workflows, opts.Loaders()...); err != nil {
		return nil, err
	}
	return workflows, nil
}
