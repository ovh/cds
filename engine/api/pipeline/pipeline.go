package pipeline

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/sdk"
)

type structarg struct {
	loadstages     bool
	loadparameters bool
}

// UpdatePipelineLastModified Update last_modified date on pipeline
func UpdatePipelineLastModified(db gorp.SqlExecutor, proj *sdk.Project, p *sdk.Pipeline, u *sdk.User) error {
	query := "UPDATE pipeline SET last_modified = current_timestamp WHERE id = $1 RETURNING last_modified"
	var lastModified time.Time
	err := db.QueryRow(query, p.ID).Scan(&lastModified)
	if err == nil {
		p.LastModified = lastModified.Unix()
	}

	t := time.Now()

	if u != nil {
		cache.SetWithTTL(cache.Key("lastModified", proj.Key, "pipeline", p.Name), sdk.LastModification{
			Name:         p.Name,
			Username:     u.Username,
			LastModified: t.Unix(),
		}, 0)

		updates := sdk.LastModification{
			Key:          proj.Key,
			Name:         p.Name,
			LastModified: lastModified.Unix(),
			Username:     u.Username,
			Type:         sdk.PipelineLastModificationType,
		}
		b, errP := json.Marshal(updates)
		if errP == nil {
			cache.Publish("lastUpdates", string(b))
		}
	}

	return err
}

// LoadPipeline loads a pipeline from database
func LoadPipeline(db gorp.SqlExecutor, projectKey, name string, deep bool) (*sdk.Pipeline, error) {
	var p sdk.Pipeline

	var lastModified time.Time
	query := `SELECT pipeline.id, pipeline.name, pipeline.project_id, pipeline.type, pipeline.last_modified FROM pipeline
	 		JOIN project on pipeline.project_id = project.id
	 		WHERE pipeline.name = $1 AND project.projectKey = $2`

	err := db.QueryRow(query, name, projectKey).Scan(&p.ID, &p.Name, &p.ProjectID, &p.Type, &lastModified)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrPipelineNotFound
		}
		return nil, err
	}
	p.LastModified = lastModified.Unix()
	p.ProjectKey = projectKey

	if deep {
		if err := loadPipelineDependencies(db, &p); err != nil {
			return nil, err
		}
	}

	return &p, nil
}

// LoadPipelineByID loads a pipeline from database
func LoadPipelineByID(db gorp.SqlExecutor, pipelineID int64, deep bool) (*sdk.Pipeline, error) {
	var p sdk.Pipeline
	query := `SELECT pipeline.name, pipeline.type, project.projectKey FROM pipeline
	JOIN project on pipeline.project_id = project.id
	WHERE pipeline.id = $1`

	err := db.QueryRow(query, pipelineID).Scan(&p.Name, &p.Type, &p.ProjectKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrPipelineNotFound
		}
		return nil, err
	}
	p.ID = pipelineID

	if deep {
		if err := loadPipelineDependencies(db, &p); err != nil {
			return nil, err
		}
	}

	return &p, nil
}

func loadPipelineDependencies(db gorp.SqlExecutor, p *sdk.Pipeline) error {
	if err := LoadPipelineStage(db, p); err != nil {
		return err
	}
	if err := LoadGroupByPipeline(db, p); err != nil {
		return err
	}

	parameters, err := GetAllParametersInPipeline(db, p.ID)
	if err != nil {
		return err
	}
	p.Parameter = parameters
	return nil
}

// DeletePipeline remove given pipeline and all history from database
func DeletePipeline(db gorp.SqlExecutor, pipelineID int64, userID int64) error {

	if err := DeleteAllStage(db, pipelineID, userID); err != nil {
		return err
	}

	// Delete pipeline groups
	query := `DELETE FROM pipeline_group WHERE pipeline_id = $1`
	if _, err := db.Exec(query, pipelineID); err != nil {
		return err
	}

	if err := DeleteAllParameterFromPipeline(db, pipelineID); err != nil {
		return err
	}

	// Delete triggers
	if err := trigger.DeletePipelineTriggers(db, pipelineID); err != nil {
		return err
	}

	// Delete test results
	if err := DeletePipelineTestResults(db, pipelineID); err != nil {
		return err
	}

	var pipelineBuildIDs []int64
	query = `SELECT id FROM pipeline_build where pipeline_id = $1`
	rows, err := db.Query(query, pipelineID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var pbID int64
		err = rows.Scan(&pbID)
		if err != nil {
			return err
		}
		pipelineBuildIDs = append(pipelineBuildIDs, pbID)
	}
	for _, id := range pipelineBuildIDs {
		err = DeletePipelineBuildByID(db, id)
		if err != nil {
			return err
		}
	}

	// Delete artifacts left
	query = `DELETE FROM artifact WHERE pipeline_id = $1`
	_, err = db.Exec(query, pipelineID)
	if err != nil {
		return err
	}

	// Delete application_pipeline_notif
	query = `
		DELETE FROM application_pipeline_notif WHERE application_pipeline_id IN (
			SELECT id FROM application_pipeline WHERE pipeline_id = $1
		)`
	if _, err := db.Exec(query, pipelineID); err != nil {
		return err
	}

	// Delete pipeline
	query = `DELETE FROM pipeline WHERE id = $1`
	if _, err := db.Exec(query, pipelineID); err != nil {
		return err
	}

	return nil
}

// LoadPipelines loads all pipelines in a project
func LoadPipelines(db gorp.SqlExecutor, projectID int64, loadDependencies bool, user *sdk.User) ([]sdk.Pipeline, error) {
	var pip []sdk.Pipeline

	var rows *sql.Rows
	var errquery error

	if user == nil || user.Admin {
		query := `SELECT id, name, project_id, type, last_modified
			  FROM pipeline
			  WHERE project_id = $1
			  ORDER BY pipeline.name`
		rows, errquery = db.Query(query, projectID)
	} else {
		query := `SELECT distinct(pipeline.id), pipeline.name, pipeline.project_id, pipeline.type, last_modified
			  FROM pipeline
			  JOIN pipeline_group ON pipeline.id = pipeline_group.pipeline_id
			  JOIN group_user ON pipeline_group.group_id = group_user.group_id
			  WHERE group_user.user_id = $1
			  AND pipeline.project_id = $2
			  ORDER by pipeline.name`
		rows, errquery = db.Query(query, user.ID, projectID)
	}

	if errquery != nil {
		return nil, errquery
	}
	defer rows.Close()

	for rows.Next() {
		var p sdk.Pipeline
		var lastModified time.Time

		// scan pipeline id
		if err := rows.Scan(&p.ID, &p.Name, &p.ProjectID, &p.Type, &lastModified); err != nil {
			return nil, err
		}
		p.LastModified = lastModified.Unix()

		if loadDependencies {
			// load pipeline stages
			if err := LoadPipelineStage(db, &p); err != nil {
				return nil, err
			}

			params, err := GetAllParametersInPipeline(db, p.ID)
			if err != nil {
				return nil, err
			}
			p.Parameter = params
		}

		pip = append(pip, p)
	}

	return pip, nil
}

// LoadPipelineByGroup loads all pipelines where group has access
func LoadPipelineByGroup(db gorp.SqlExecutor, group *sdk.Group) error {
	query := `SELECT project.projectKey, pipeline.id, pipeline.name,pipeline_group.role FROM pipeline
	 		  JOIN pipeline_group ON pipeline_group.pipeline_id = pipeline.id
	 		  JOIN project ON pipeline.project_id = project.id
	 		  WHERE pipeline_group.group_id = $1 ORDER BY pipeline.name ASC`
	rows, err := db.Query(query, group.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var pipeline sdk.Pipeline
		var perm int
		err = rows.Scan(&pipeline.ProjectKey, &pipeline.ID, &pipeline.Name, &perm)
		if err != nil {
			return err
		}
		group.PipelineGroups = append(group.PipelineGroups, sdk.PipelineGroup{
			Pipeline:   pipeline,
			Permission: perm,
		})
	}
	return nil
}

func updateParamInList(params []sdk.Parameter, paramAction sdk.Parameter) (bool, []sdk.Parameter) {
	for i := range params {
		p := &params[i]
		if p.Name == paramAction.Name {
			p.Type = paramAction.Type
			return true, params
		}
	}
	return false, params
}

// LoadGroupByPipeline load group permission on one pipeline
func LoadGroupByPipeline(db gorp.SqlExecutor, pipeline *sdk.Pipeline) error {
	query := `SELECT "group".id,"group".name,pipeline_group.role FROM "group"
	 		  JOIN pipeline_group ON pipeline_group.group_id = "group".id
	 		  WHERE pipeline_group.pipeline_id = $1 ORDER BY "group".name ASC`

	rows, errq := db.Query(query, pipeline.ID)
	if errq != nil {
		return errq
	}
	defer rows.Close()

	for rows.Next() {
		var group sdk.Group
		var perm int
		if err := rows.Scan(&group.ID, &group.Name, &perm); err != nil {
			return err
		}
		pipeline.GroupPermission = append(pipeline.GroupPermission, sdk.GroupPermission{
			Group:      group,
			Permission: perm,
		})
	}
	return nil
}

// UpdatePipeline update the pipeline
func UpdatePipeline(db gorp.SqlExecutor, p *sdk.Pipeline) error {
	//Update pipeline
	query := `UPDATE pipeline SET name=$1, type=$2 WHERE id=$3`
	_, err := db.Exec(query, p.Name, string(p.Type), p.ID)
	return err
}

// InsertPipeline inserts pipeline informations in database
func InsertPipeline(db gorp.SqlExecutor, proj *sdk.Project, p *sdk.Pipeline, u *sdk.User) error {
	query := `INSERT INTO pipeline (name, project_id, type, last_modified) VALUES ($1,$2,$3, current_timestamp) RETURNING id`

	if p.Name == "" {
		return sdk.ErrInvalidName
	}

	if p.Type != sdk.BuildPipeline && p.Type != sdk.DeploymentPipeline && p.Type != sdk.TestingPipeline {
		return sdk.WrapError(sdk.ErrInvalidType, "InsertPipeline>")
	}

	if p.ProjectID == 0 {
		return sdk.WrapError(sdk.ErrInvalidProject, "InsertPipeline>")
	}

	if err := db.QueryRow(query, p.Name, p.ProjectID, string(p.Type)).Scan(&p.ID); err != nil {
		return err
	}

	for i := range p.Parameter {
		if err := InsertParameterInPipeline(db, p.ID, &p.Parameter[i]); err != nil {
			return sdk.WrapError(err, "InsertPipeline>")
		}
	}

	return UpdatePipelineLastModified(db, proj, p, u)
}

// ExistPipeline Check if the given pipeline exist in database
func ExistPipeline(db gorp.SqlExecutor, projectID int64, name string) (bool, error) {
	query := `SELECT COUNT(id) FROM pipeline WHERE pipeline.project_id = $1 AND pipeline.name= $2`

	var nb int64
	err := db.QueryRow(query, projectID, name).Scan(&nb)
	if err != nil {
		return false, err
	}
	if nb != 0 {
		return true, nil
	}
	return false, nil
}

// AttachPipelinesWarnings add warnings about optional steps for several PipelineBuild
func AttachPipelinesWarnings(pbs *[]sdk.PipelineBuild) {
	for iPb := range *pbs {
		pb := &(*pbs)[iPb]
		attachPipelineWarnings(pb)
	}
}

// attachPipelineWarnings add warnings about optional steps for one PipelineBuild
func attachPipelineWarnings(pb *sdk.PipelineBuild) {
	if pb.Status == sdk.StatusSuccess {
		for iS := range pb.Stages {
			stage := &pb.Stages[iS]
			if stage.Enabled {
				for iB := range stage.PipelineBuildJobs {
					build := &stage.PipelineBuildJobs[iB]
					job := &stage.Jobs[iB]
					for iSt := range build.Job.StepStatus {
						step := &build.Job.StepStatus[iSt]
						if build.Job.Action.Actions[iSt].Enabled && build.Job.Action.Actions[iSt].Optional && step.Status == sdk.StatusFail.String() {
							w := sdk.PipelineBuildWarning{Type: sdk.OptionalStepFailed, Action: build.Job.Action.Actions[iSt]}
							pb.Warnings = append(pb.Warnings, w)
							stage.Warnings = append(stage.Warnings, w)
							build.Warnings = append(build.Warnings, w)
							job.Warnings = append(job.Warnings, w)
						}
					}
				}
			}
		}
	}
}
