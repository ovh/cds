package action

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// Usage for action.
type Usage struct {
	Pipelines []UsagePipeline `json:"pipelines"`
	Actions   []UsageAction   `json:"actions"`
}

// UsagePipeline represent a pipeline using an action.
type UsagePipeline struct {
	ProjectID    string `json:"project_id"`
	ProjectKey   string `json:"project_key"`
	ProjectName  string `json:"project_name"`
	PipelineID   int64  `json:"pipeline_id"`
	PipelineName string `json:"pipeline_name"`
	StageID      int64  `json:"stage_id"`
	StageName    string `json:"stage_Name"`
	JobID        int64  `json:"job_id"`
	JobName      string `json:"job_name"`
	ActionID     int64  `json:"action_id"`
	ActionName   string `json:"action_name"`
	Warning      bool   `json:"warning"`
}

// GetPipelineUsages returns the list of pipelines using an action
func GetPipelineUsages(db gorp.SqlExecutor, sharedInfraGroupID, actionID int64) ([]UsagePipeline, error) {
	rows, err := db.Query(`
    SELECT
      project.id, project.projectKey, project.name,
      pipeline.id, pipeline.name,
      pipeline_stage.id, pipeline_stage.name,
      parent.id, parent.name,
      action.id, action.name,
      CAST((CASE WHEN project_group.role IS NOT NULL OR action.group_id = $1 OR action.group_id IS NULL THEN 0 ELSE 1 END) AS BIT)
		FROM action
    INNER JOIN action_edge ON action_edge.child_id = action.id
    LEFT JOIN action as parent ON parent.id = action_edge.parent_id
		INNER JOIN pipeline_action ON pipeline_action.action_id = parent.id
		LEFT JOIN pipeline_stage ON pipeline_stage.id = pipeline_action.pipeline_stage_id
		LEFT JOIN pipeline ON pipeline.id = pipeline_stage.pipeline_id
    LEFT JOIN project ON pipeline.project_id = project.id
    LEFT JOIN project_group ON project_group.project_id = project.id AND project_group.group_id = action.group_id
		WHERE action.id = $2
		ORDER BY project.projectKey, pipeline.name, action.name;
	`, sharedInfraGroupID, actionID)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot load pipeline usages for action with id %d", actionID)
	}
	defer rows.Close()

	us := []UsagePipeline{}
	for rows.Next() {
		var u UsagePipeline
		if err := rows.Scan(
			&u.ProjectID, &u.ProjectKey, &u.ProjectName,
			&u.PipelineID, &u.PipelineName,
			&u.StageID, &u.StageName,
			&u.ActionID, &u.ActionName,
			&u.JobID, &u.JobName,
			&u.Warning,
		); err != nil {
			return nil, sdk.WrapError(err, "cannot scan sql rows")
		}
		us = append(us, u)
	}

	return us, nil
}

// UsageAction represent a action using an action.
type UsageAction struct {
	GroupID          int64  `json:"group_id"`
	GroupName        string `json:"group_name"`
	ParentActionID   int64  `json:"parent_action_id"`
	ParentActionName string `json:"parent_action_name"`
	ActionID         int64  `json:"action_id"`
	ActionName       string `json:"action_name"`
	Warning          bool   `json:"warning"`
}

// GetActionUsages returns the list of actions using an action
func GetActionUsages(db gorp.SqlExecutor, sharedInfraGroupID, actionID int64) ([]UsageAction, error) {
	rows, err := db.Query(`
    SELECT
			"group".id, "group".name,
			parent.id, parent.name,
      action.id, action.name,
      CAST((CASE WHEN action.group_id = parent.group_id OR action.group_id = $1 OR action.group_id IS NULL THEN 0 ELSE 1 END) AS BIT)
		FROM action
		INNER JOIN action_edge ON action_edge.child_id = action.id
		LEFT JOIN action as parent ON parent.id = action_edge.parent_id
		LEFT JOIN "group" ON "group".id = parent.group_id
		WHERE action.id = $2 AND parent.group_id IS NOT NULL
		ORDER BY parent.name, action.name;
	`, sharedInfraGroupID, actionID)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot load pipeline usages for action with id %d", actionID)
	}
	defer rows.Close()

	us := []UsageAction{}
	for rows.Next() {
		var u UsageAction
		if err := rows.Scan(
			&u.GroupID, &u.GroupName,
			&u.ParentActionID, &u.ParentActionName,
			&u.ActionID, &u.ActionName,
			&u.Warning,
		); err != nil {
			return nil, sdk.WrapError(err, "cannot scan sql rows")
		}
		us = append(us, u)
	}

	return us, nil
}

// Used checks if action is used in another action or in a pipeline.
func Used(db gorp.SqlExecutor, actionID int64) (bool, error) {
	var count int

	if err := db.QueryRow(`SELECT COUNT(id) FROM pipeline_action WHERE action_id = $1`, actionID).Scan(&count); err != nil {
		return false, sdk.WithStack(err)
	}
	if count > 0 {
		return true, nil
	}

	if err := db.QueryRow(`SELECT COUNT(id) FROM action_edge WHERE child_id = $1`, actionID).Scan(&count); err != nil {
		return false, sdk.WithStack(err)
	}
	return count > 0, nil
}
