package workflow

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// LoadByPipelineName loads a workflow for a given project key and pipeline name
func LoadByPipelineName(ctx context.Context, db gorp.SqlExecutor, projectKey string, pipName string) ([]sdk.WorkflowName, error) {
	query := `SELECT distinct workflow.*, project.projectkey as "project_key", project.id as "project_id"
		from workflow
		join project on project.id = workflow.project_id
		join w_node on w_node.workflow_id = workflow.id
    	join w_node_context on w_node_context.node_id = w_node.id
		join pipeline on pipeline.id = w_node_context.pipeline_id
		where project.projectkey = $1 and pipeline.name = $2
		and workflow.to_delete = false
		order by workflow.name asc`
	var result []sdk.WorkflowName // This struct is not registered as a gorpmapping entity so we can't use gorpmapping.Query
	_, err := db.Select(&result, query, projectKey, pipName)
	return result, sdk.WithStack(err)
}

// LoadByApplicationName loads a workflow for a given project key and application name
func LoadByApplicationName(ctx context.Context, db gorp.SqlExecutor, projectKey string, appName string) ([]sdk.WorkflowName, error) {
	query := `SELECT distinct workflow.*, project.projectkey as "project_key", project.id as "project_id"
	from workflow
	join project on project.id = workflow.project_id
	join w_node on w_node.workflow_id = workflow.id
	join w_node_context on w_node_context.node_id = w_node.id
	join application on w_node_context.application_id = application.id
	where project.projectkey = $1 and application.name = $2
	and workflow.to_delete = false
	order by workflow.name asc`
	var result []sdk.WorkflowName // This struct is not registered as a gorpmapping entity so we can't use gorpmapping.Query
	_, err := db.Select(&result, query, projectKey, appName)
	return result, sdk.WithStack(err)
}

// LoadByEnvName loads a workflow for a given project key and environment name (ie. checking permissions)
func LoadByEnvName(ctx context.Context, db gorp.SqlExecutor, projectKey string, envName string) ([]sdk.WorkflowName, error) {
	query := `SELECT distinct workflow.*, project.projectkey as "project_key", project.id as "project_id"
	from workflow
	join project on project.id = workflow.project_id
	join w_node on w_node.workflow_id = workflow.id
	join w_node_context on w_node_context.node_id = w_node.id
	join environment on w_node_context.environment_id = environment.id
	where project.projectkey = $1 and environment.name = $2
	and workflow.to_delete = false
	order by workflow.name asc`
	var result []sdk.WorkflowName // This struct is not registered as a gorpmapping entity so we can't use gorpmapping.Query
	_, err := db.Select(&result, query, projectKey, envName)
	return result, sdk.WithStack(err)
}

// LoadByWorkflowTemplateID load all workflow names linked to a workflow template
func LoadByWorkflowTemplateID(ctx context.Context, db gorp.SqlExecutor, templateID int64) ([]sdk.WorkflowName, error) {
	query := `SELECT distinct workflow.*, project.projectkey as "project_key", project.id as "project_id"
	FROM workflow
	JOIN workflow_template_instance ON workflow_template_instance.workflow_id = workflow.id
	JOIN project on project.id = workflow.project_id
	WHERE workflow_template_instance.workflow_template_id = $1 AND workflow.to_delete = false`
	var result []sdk.WorkflowName // This struct is not registered as a gorpmapping entity so we can't use gorpmapping.Query
	_, err := db.Select(&result, query, templateID)
	return result, sdk.WithStack(err)
}
