package pipeline

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/artifact"
	"github.com/ovh/cds/engine/api/build"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/stats"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// LoadPipelineBuildRequest Load pipeline build activities.
// Use also in api/project/project.go to load the last 5 builds by applications
const LoadPipelineBuildRequest = `
SELECT  pb.pipeline_id, pb.application_id, pb.environment_id, pb.id, project.id as project_id,
	environment.name as envName, application.name as appName, pipeline.name as pipName, project.projectkey,
	pipeline.type,
	pb.build_number, pb.version, pb.status,
	pb.start, pb.done,
	pb.manual_trigger, pb.scheduled_trigger, pb.triggered_by, pb.parent_pipeline_build_id, pb.vcs_changes_branch, pb.vcs_changes_hash, pb.vcs_changes_author,
	"user".username, pipTriggerFrom.name as pipTriggerFrom, pbTriggerFrom.version as versionTriggerFrom
FROM pipeline_build pb
JOIN environment ON environment.id = pb.environment_id
JOIN application ON application.id = pb.application_id
JOIN pipeline ON pipeline.id = pb.pipeline_id
JOIN project ON project.id = pipeline.project_id
LEFT JOIN "user" ON "user".id = pb.triggered_by
LEFT JOIN pipeline_build as pbTriggerFrom ON pbTriggerFrom.id = pb.parent_pipeline_build_id
LEFT JOIN pipeline as pipTriggerFrom ON pipTriggerFrom.id = pbTriggerFrom.pipeline_id
%s
WHERE %s
ORDER BY start DESC
%s
`

// LoadPipelineBuildStage Load pipeline build stage + action builds
const LoadPipelineBuildStage = `
SELECT pipeline_action_R.start, pipeline_action_R.done, pipeline_action_R.id, pipeline_stage.name, pipeline_stage.build_order
FROM pipeline_stage
JOIN pipeline on pipeline.id = pipeline_stage.pipeline_id
JOIN pipeline_build on pipeline_build.pipeline_id = pipeline.id
LEFT OUTER JOIN (
    SELECT pipeline_action.id, pipeline_action.pipeline_stage_id, action_build.start, action_build.done
    FROM pipeline_action
    JOIN action_build ON action_build.pipeline_action_id = pipeline_action.id
    WHERE pipeline_build_id = $1
) AS pipeline_action_R ON pipeline_action_R.pipeline_stage_id = pipeline_stage.id
WHERE pipeline_build.id = $1
ORDER BY pipeline_stage.build_order ASC, pipeline_action_R.id;
`

// LoadPipelineBuildWithActions loads pipelines builds with its stages and action builds
const LoadPipelineBuildWithActions = `
SELECT
project.id as projectID, project.projectkey,
application.id as appID, application.name,
environment.id as envID, environment.name,
pb.id as pbID, pb.status, pb.version,
pb.build_number, pb.args, pb.manual_trigger, pb.scheduled_trigger,
pb.triggered_by, pb.parent_pipeline_build_id,
pb.vcs_changes_branch, pb.vcs_changes_hash, pb.vcs_changes_author,
pipeline.id as pipID, pipeline.name, pipeline.type,
pipeline_stage.id as stageID, pipeline_stage.name,
action.name, action_build.id, action_build.status
FROM pipeline_build as pb
JOIN application ON application.id = pb.application_id
JOIN environment ON environment.id = pb.environment_id
JOIN pipeline ON pipeline.id = pb.pipeline_id
JOIN project ON pipeline.project_id = project.id
JOIN pipeline_stage ON pipeline_stage.pipeline_id = pipeline.id
JOIN pipeline_action ON pipeline_action.pipeline_stage_id = pipeline_stage.id
JOIN action ON action.id = pipeline_action.action_id
LEFT JOIN action_build ON action_build.pipeline_build_id = pb.id AND action_build.pipeline_action_id = pipeline_action.id
WHERE %s
ORDER BY project.projectkey, application.name, pb.id, pipeline_stage.build_order
`

// WithStages set boolean to load stages
func WithStages() FuncArg {
	return func(args *structarg) {
		args.loadstages = true
	}
}

// WithParameters set boolean to load parameters
func WithParameters() FuncArg {
	return func(args *structarg) {
		args.loadparameters = true
	}
}

// LoadPipelineBuildByHash look for a pipeline build triggered by a change with given hash
func LoadPipelineBuildByHash(db *sql.DB, hash string) ([]sdk.PipelineBuild, error) {
	var pbs []sdk.PipelineBuild
	query := fmt.Sprintf(LoadPipelineBuildWithActions, "pb.vcs_changes_hash = $1")

	rows, err := db.Query(query, hash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	pbs, err = scanPbWithStagesAndActions(rows)
	if err != nil {
		return nil, err
	}

	return pbs, nil
}

// GetBranches from pipeline build and pipeline history for the given application
func GetBranches(db *sql.DB, app *sdk.Application) ([]sdk.VCSBranch, error) {
	branches := []sdk.VCSBranch{}
	query := `
		SELECT vcs_changes_branch
		FROM
			(
				SELECT DISTINCT vcs_changes_branch
				FROM pipeline_build
				WHERE application_id = $1

				UNION

				SELECT DISTINCT vcs_changes_branch
				FROM pipeline_history
				WHERE application_id = $1
			) as tmp
		ORDER BY vcs_changes_branch DESC

	`
	rows, err := db.Query(query, app.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var b sql.NullString
		err := rows.Scan(&b)
		if err != nil {
			return nil, err
		}
		if b.Valid {
			branches = append(branches, sdk.VCSBranch{DisplayID: b.String})
		}

	}
	return branches, nil
}

// LoadUserRecentPipelineBuild retrieves all user accessible pipeline build finished less than a minute ago
func LoadUserRecentPipelineBuild(db *sql.DB, userID int64) ([]sdk.PipelineBuild, error) {
	var pbs []sdk.PipelineBuild

	subquery := fmt.Sprintf(LoadPipelineBuildRequest,
		"JOIN pipeline_group ON pipeline_group.pipeline_id = pipeline.id JOIN \"group\" ON \"group\".id = pipeline_group.group_id JOIN group_user ON group_user.group_id = \"group\".id",
		"pb.status != $1 AND group_user.user_id = $2 AND pb.done > NOW() - INTERVAL '1 minutes'",
		"LIMIT 50")
	query := `WITH load_pb AS (%s)
	  	  SELECT *
		  FROM (
		  	SELECT
				*
			FROM load_pb
		  ) temp
		  ORDER BY temp.projectkey, temp.appName, temp.id`
	query = fmt.Sprintf(query, subquery)
	rows, err := db.Query(query, sdk.StatusBuilding.String(), userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p sdk.PipelineBuild
		err := scanPbShort(&p, rows)
		if err != nil {
			return nil, err
		}
		pbs = append(pbs, p)
	}
	return pbs, nil
}

// LoadUserBuildingPipelines retrieves all building pipelines user has access to
func LoadUserBuildingPipelines(db *sql.DB, userID int64) ([]sdk.PipelineBuild, error) {

	subquery := fmt.Sprintf(LoadPipelineBuildRequest,
		"JOIN pipeline_group ON pipeline_group.pipeline_id = pipeline.id JOIN \"group\" ON \"group\".id = pipeline_group.group_id JOIN group_user ON group_user.group_id = \"group\".id",
		"pb.status = $1 AND group_user.user_id = $2",
		"LIMIT 100")
	query := `WITH load_pb AS (%s)
	  	  SELECT *
		  FROM (
		  	SELECT
				*
			FROM load_pb
		  ) temp
		  ORDER BY temp.projectkey, temp.appName, temp.id`
	query = fmt.Sprintf(query, subquery)
	rows, err := db.Query(query, sdk.StatusBuilding.String(), userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pip []sdk.PipelineBuild
	for rows.Next() {
		var p sdk.PipelineBuild
		err := scanPbShort(&p, rows)
		if err != nil {
			return nil, err
		}
		pip = append(pip, p)
	}
	return pip, nil
}

// LoadRecentPipelineBuild retrieves pipelines in database having a build running or finished
// less than a minute ago
func LoadRecentPipelineBuild(db *sql.DB, args ...FuncArg) ([]sdk.PipelineBuild, error) {
	var pbs []sdk.PipelineBuild
	query := fmt.Sprintf(LoadPipelineBuildWithActions, "pb.status = $1 OR (pb.status != $1 AND pb.done > NOW() - INTERVAL '1 minutes')")

	rows, err := db.Query(query, string(sdk.StatusBuilding))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	pbs, err = scanPbWithStagesAndActions(rows)
	return pbs, err
}

func scanPbWithStagesAndActions(rows *sql.Rows) ([]sdk.PipelineBuild, error) {
	var pb sdk.PipelineBuild
	var pbs []sdk.PipelineBuild

	var pbI, stageI, abI int
	var pbStatus, pbArgs, pipType, stageName string
	var stageID int64
	var manual sql.NullBool
	var trigBy, parentID, abID sql.NullInt64
	var branch, hash, author, actionName, abStatus sql.NullString
	for rows.Next() {
		err := rows.Scan(&pb.Pipeline.ProjectID, &pb.Pipeline.ProjectKey,
			&pb.Application.ID, &pb.Application.Name,
			&pb.Environment.ID, &pb.Environment.Name,
			&pb.ID, &pbStatus, &pb.Version,
			&pb.BuildNumber, &pbArgs, &manual,
			&trigBy, &parentID,
			&branch, &hash, &author,
			&pb.Pipeline.ID, &pb.Pipeline.Name, &pipType,
			&stageID, &stageName,
			&actionName, &abID, &abStatus,
		)
		if err != nil {
			return nil, err
		}
		pb.Pipeline.Type = sdk.PipelineTypeFromString(pipType)
		pb.Status = sdk.StatusFromString(pbStatus)

		// manual trigger
		if manual.Valid && branch.Valid {
			pb.Trigger.ManualTrigger = manual.Bool
			pb.Trigger.VCSChangesBranch = branch.String
		}
		// moar info on automatic trigger
		if manual.Valid && trigBy.Valid && parentID.Valid && branch.Valid && hash.Valid && author.Valid {
			pb.Trigger.TriggeredBy = &sdk.User{ID: trigBy.Int64}
			pb.Trigger.ParentPipelineBuild = &sdk.PipelineBuild{ID: parentID.Int64}
			pb.Trigger.VCSChangesHash = hash.String
			pb.Trigger.VCSChangesAuthor = author.String
		}

		// If there is no pb in pbs, we obviously got the first
		if len(pbs) == 0 {
			err = json.Unmarshal([]byte(pbArgs), &pb.Parameters)
			if err != nil {
				return nil, err
			}
			pbs = append(pbs, pb)
		}
		// If pbID differs from the last one, append a new pb in pbs
		// reset stage and action index
		if pb.ID != pbs[pbI].ID {
			err = json.Unmarshal([]byte(pbArgs), &pb.Parameters)
			if err != nil {
				return nil, err
			}
			pbs = append(pbs, pb)
			pbI++
			stageI = 0
			abI = 0
		}
		// If there is no stages in pb, appends a new one
		if len(pbs[pbI].Stages) == 0 {
			s := sdk.Stage{ID: stageID, Name: stageName}
			pbs[pbI].Stages = append(pbs[pbI].Stages, s)
			stageI = 0
			abI = 0
		}
		// if stageID differs from the last one, append a new stage in pb
		if pbs[pbI].Stages[stageI].ID != stageID {
			s := sdk.Stage{ID: stageID, Name: stageName}
			pbs[pbI].Stages = append(pbs[pbI].Stages, s)
			stageI++
			abI = 0
		}
		// if there is no action build in stage, add a new one
		if len(pbs[pbI].Stages[stageI].ActionBuilds) == 0 && abID.Valid {
			ab := sdk.ActionBuild{ID: abID.Int64, ActionName: actionName.String, Status: sdk.StatusFromString(abStatus.String)}
			pbs[pbI].Stages[stageI].ActionBuilds = append(pbs[pbI].Stages[stageI].ActionBuilds, ab)
			abI = 0
		}
		// if abID differs from the last one, append a new ActionBuild
		if abID.Valid && pbs[pbI].Stages[stageI].ActionBuilds[abI].ID != abID.Int64 {
			ab := sdk.ActionBuild{ID: abID.Int64, ActionName: actionName.String, Status: sdk.StatusFromString(abStatus.String)}
			pbs[pbI].Stages[stageI].ActionBuilds = append(pbs[pbI].Stages[stageI].ActionBuilds, ab)
			abI++
		}
	}
	return pbs, nil
}

// InsertBuildVariable adds a variable exported in user scripts and forwarded by building worker
func InsertBuildVariable(db *sql.DB, pbID int64, v sdk.Variable) error {

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Load args from pipeline build and lock it
	query := `SELECT args FROM pipeline_build WHERE id = $1 FOR UPDATE`
	var argsJSON string
	err = tx.QueryRow(query, pbID).Scan(&argsJSON)
	if err != nil {
		return err
	}

	// Load parameters
	var params []sdk.Parameter
	if err := json.Unmarshal([]byte(argsJSON), &params); err != nil {
		return err
	}

	// Add build variable
	params = append(params, sdk.Parameter{
		Name:  "cds.build." + v.Name,
		Type:  sdk.StringParameter,
		Value: v.Value,
	})

	// Update pb in database
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}

	query = `UPDATE pipeline_build SET args = $1 WHERE id = $2`
	_, err = tx.Exec(query, string(data), pbID)
	if err != nil {
		return err
	}

	// now load all related action build
	query = `SELECT id, args FROM action_build WHERE pipeline_build_id = $1 FOR UPDATE`
	rows, err := tx.Query(query, pbID)
	if err != nil {
		return err
	}
	defer rows.Close()
	var abs []sdk.ActionBuild
	for rows.Next() {
		var ab sdk.ActionBuild
		err = rows.Scan(&ab.ID, &argsJSON)
		if err != nil {
			return err
		}
		err = json.Unmarshal([]byte(argsJSON), &ab.Args)
		if err != nil {
			return err
		}
		abs = append(abs, ab)
	}
	rows.Close()

	query = `UPDATE action_build SET args = $1 WHERE id = $2`
	for _, ab := range abs {
		ab.Args = append(ab.Args, sdk.Parameter{
			Name:  "cds.build." + v.Name,
			Type:  sdk.StringParameter,
			Value: v.Value,
		})

		data, err := json.Marshal(ab.Args)
		if err != nil {
			return err
		}

		_, err = tx.Exec(query, string(data), ab.ID)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

// LoadBuildingPipelines retrieves pipelines in database having a build running
func LoadBuildingPipelines(db *sql.DB, args ...FuncArg) ([]sdk.PipelineBuild, error) {
	query := `
SELECT DISTINCT ON (project.projectkey, application.name, pb.application_id, pb.pipeline_id, pb.environment_id, pb.vcs_changes_branch)
	pb.pipeline_id, pb.application_id, pb.environment_id, pb.id, project.id as project_id,
	environment.name as envName, application.name as appName, pipeline.name as pipName, project.projectkey,
	pipeline.type,
	pb.build_number, pb.version, pb.status, pb.args,
	pb.start, pb.done,
	pb.manual_trigger, pb.scheduled_trigger, pb.triggered_by, pb.parent_pipeline_build_id, pb.vcs_changes_branch, pb.vcs_changes_hash, pb.vcs_changes_author,
	"user".username, pipTriggerFrom.name as pipTriggerFrom, pbTriggerFrom.version as versionTriggerFrom
FROM pipeline_build pb
JOIN environment ON environment.id = pb.environment_id
JOIN application ON application.id = pb.application_id
JOIN pipeline ON pipeline.id = pb.pipeline_id
JOIN project ON project.id = pipeline.project_id
LEFT JOIN "user" ON "user".id = pb.triggered_by
LEFT JOIN pipeline_build as pbTriggerFrom ON pbTriggerFrom.id = pb.parent_pipeline_build_id
LEFT JOIN pipeline as pipTriggerFrom ON pipTriggerFrom.id = pbTriggerFrom.pipeline_id
WHERE pb.status = $1
ORDER BY project.projectkey, application.name, pb.application_id, pb.pipeline_id, pb.environment_id, pb.vcs_changes_branch, pb.id
LIMIT 1000`
	rows, err := db.Query(query, sdk.StatusBuilding.String())
	if err != nil {
		log.Warning("LoadBuildingPipelines>Cannot load buliding pipelines: %s", err)
		return nil, err
	}
	defer rows.Close()
	var pip []sdk.PipelineBuild
	for rows.Next() {
		p := sdk.PipelineBuild{}

		var status, typePipeline, argsJSON string
		var manual, scheduled sql.NullBool
		var trigBy, pPbID, version sql.NullInt64
		var branch, hash, author, fromUser, fromPipeline sql.NullString

		err := rows.Scan(&p.Pipeline.ID, &p.Application.ID, &p.Environment.ID, &p.ID, &p.Pipeline.ProjectID,
			&p.Environment.Name, &p.Application.Name, &p.Pipeline.Name, &p.Pipeline.ProjectKey,
			&typePipeline,
			&p.BuildNumber, &p.Version, &status, &argsJSON,
			&p.Start, &p.Done,
			&manual, &scheduled, &trigBy, &pPbID, &branch, &hash, &author,
			&fromUser, &fromPipeline, &version)
		if err != nil {
			log.Warning("LoadBuildingPipelines> Error while loading build information: %s", err)
			return nil, err
		}
		p.Status = sdk.StatusFromString(status)
		p.Pipeline.Type = sdk.PipelineTypeFromString(typePipeline)
		p.Application.ProjectKey = p.Pipeline.ProjectKey
		loadPbTrigger(&p, manual, scheduled, pPbID, branch, hash, author, fromUser, fromPipeline, version)

		if trigBy.Valid && p.Trigger.TriggeredBy != nil {
			p.Trigger.TriggeredBy.ID = trigBy.Int64
		}

		// Load parameters
		if err := json.Unmarshal([]byte(argsJSON), &p.Parameters); err != nil {
			log.Warning("Cannot unmarshal args : %s", err)
			return nil, err
		}

		// load pipeline actions
		if err := LoadPipelineStage(db, &p.Pipeline, args...); err != nil {
			log.Warning("Cannot load pipeline stages : %s", err)
			return nil, err
		}

		pip = append(pip, p)
	}

	return pip, nil
}

// UpdatePipelineBuildStatus Update status of pipeline_build
func UpdatePipelineBuildStatus(db database.QueryExecuter, pb sdk.PipelineBuild, status sdk.Status) error {
	query := `UPDATE pipeline_build SET status = $1, done = $3 WHERE id = $2`

	_, err := db.Exec(query, status.String(), pb.ID, time.Now())
	if err != nil {
		return err
	}

	pb.Status = status

	//Send notification
	//Load previous pipeline (some app, pip, env and branch)
	//Load branch
	branch := ""
	params := pb.Parameters
	for _, param := range params {
		if param.Name == ".git.branch" {
			branch = param.Value
			break
		}
	}
	//Get the history
	var previous *sdk.PipelineBuild
	history, err := LoadPipelineBuildHistoryByApplicationAndPipeline(db, pb.Application.ID, pb.Pipeline.ID, pb.Environment.ID, 2, "", branch)
	if err != nil {
		log.Critical("UpdatePipelineBuildStatus> error while loading previous pipeline build")
	}
	//Be sure to get the previous one
	if len(history) == 2 {
		for i := range history {
			if previous == nil || previous.BuildNumber > history[i].BuildNumber {
				previous = &history[i]
			}
		}
	}

	k := cache.Key("application", pb.Application.ProjectKey, "*")
	cache.DeleteAll(k)

	notification.SendPipeline(db, &pb, sdk.UpdateNotifEvent, status, previous)

	return nil
}

// GetAllLastBuildByApplicationAndVersion Get the last build for current application/branch/version
func GetAllLastBuildByApplicationAndVersion(db database.Querier, applicationID int64, branchName string, version int) ([]sdk.PipelineBuild, error) {
	pb := []sdk.PipelineBuild{}

	query := `
		WITH load_pb AS (%s), load_history AS (%s)
		SELECT 	distinct on(pipeline_id, environment_id) pipeline_id, application_id, environment_id, 0, project_id,
			envName, appName, pipName, projectkey,
			type,
			build_number, version, status,
			start, done,
			manual_trigger, scheduled_trigger, triggered_by, parent_pipeline_build_id, vcs_changes_branch, vcs_changes_hash, vcs_changes_author,
			username, pipTriggerFrom, versionTriggerFrom
		FROM (
			(SELECT
				distinct on (pipeline_id, environment_id) pipeline_id, environment_id, application_id, project_id,
				envName, appName, pipName, projectkey,
				type,
				build_number, version, status,
				start, done,
				manual_trigger, scheduled_trigger, triggered_by, parent_pipeline_build_id, vcs_changes_branch, vcs_changes_hash, vcs_changes_author,
				username, pipTriggerFrom, versionTriggerFrom
			FROM load_pb
			ORDER BY pipeline_id, environment_id, build_number DESC)

			UNION

			(SELECT
				distinct on (pipeline_id, environment_id) pipeline_id, environment_id, application_id, project_id,
				envName, appName, pipName, projectkey,
				type,
				build_number, version, status,
				start, done,
				manual_trigger, scheduled_trigger, triggered_by, parent_pipeline_build_id, vcs_changes_branch, vcs_changes_hash, vcs_changes_author,
				username, pipTriggerFrom, versionTriggerFrom
			FROM load_history
			ORDER BY pipeline_id, environment_id, build_number DESC)
		) as pb
		ORDER BY pipeline_id, environment_id, build_number DESC
		LIMIT 100
	`

	query = fmt.Sprintf(query,
		fmt.Sprintf(LoadPipelineBuildRequest,
			"",
			"pb.application_id = $1 AND pb.vcs_changes_branch = $2 AND pb.version = $3",
			"LIMIT 100"),
		fmt.Sprintf(LoadPipelineHistoryRequest,
			"",
			"ph.application_id = $1  AND ph.vcs_changes_branch = $2 AND ph.version = $3",
			"LIMIT 100"))
	rows, err := db.Query(query, applicationID, branchName, version)

	if err != nil && err != sql.ErrNoRows {
		return pb, err
	}

	defer rows.Close()
	for rows.Next() {
		p := sdk.PipelineBuild{}
		err = scanPbShort(&p, rows)

		pb = append(pb, p)
	}
	return pb, nil
}

// GetAllLastBuildByApplication Get the last build result for all pipelines in the given application
func GetAllLastBuildByApplication(db database.Querier, applicationID int64, branchName string) ([]sdk.PipelineBuild, error) {
	pb := []sdk.PipelineBuild{}

	query := `
		WITH load_pb AS (%s), load_history AS (%s)
		SELECT 	distinct on(pipeline_id, environment_id) pipeline_id, application_id, environment_id, 0, project_id,
			envName, appName, pipName, projectkey,
			type,
			build_number, version, status,
			start, done,
			manual_trigger, scheduled_trigger, triggered_by, parent_pipeline_build_id, vcs_changes_branch, vcs_changes_hash, vcs_changes_author,
			username, pipTriggerFrom, versionTriggerFrom
		FROM (
			(SELECT
				distinct on (pipeline_id, environment_id) pipeline_id, environment_id, application_id, project_id,
				envName, appName, pipName, projectkey,
				type,
				build_number, version, status,
				start, done,
				manual_trigger, scheduled_trigger, triggered_by, parent_pipeline_build_id, vcs_changes_branch, vcs_changes_hash, vcs_changes_author,
				username, pipTriggerFrom, versionTriggerFrom
			FROM load_pb
			ORDER BY pipeline_id, environment_id, build_number DESC)

			UNION

			(SELECT
				distinct on (pipeline_id, environment_id) pipeline_id, environment_id, application_id, project_id,
				envName, appName, pipName, projectkey,
				type,
				build_number, version, status,
				start, done,
				manual_trigger, scheduled_trigger, triggered_by, parent_pipeline_build_id, vcs_changes_branch, vcs_changes_hash, vcs_changes_author,
				username, pipTriggerFrom, versionTriggerFrom
			FROM load_history
			ORDER BY pipeline_id, environment_id, build_number DESC)
		) as pb
		ORDER BY pipeline_id, environment_id, build_number DESC
		LIMIT 100
	`

	var rows *sql.Rows
	var err error
	if branchName == "" {
		query = fmt.Sprintf(query,
			fmt.Sprintf(LoadPipelineBuildRequest,
				"",
				"pb.application_id = $1",
				"LIMIT 100"),
			fmt.Sprintf(LoadPipelineHistoryRequest,
				"",
				"ph.application_id = $1",
				"LIMIT 100"))
		rows, err = db.Query(query, applicationID)
	} else {
		query = fmt.Sprintf(query,
			fmt.Sprintf(LoadPipelineBuildRequest,
				"",
				"pb.application_id = $1 AND pb.vcs_changes_branch = $2",
				"LIMIT 100"),
			fmt.Sprintf(LoadPipelineHistoryRequest,
				"",
				"ph.application_id = $1  AND ph.vcs_changes_branch = $2",
				"LIMIT 100"))
		rows, err = db.Query(query, applicationID, branchName)
	}

	if err != nil && err != sql.ErrNoRows {
		return pb, err
	}
	if err != nil && err == sql.ErrNoRows {
		return pb, nil
	}
	defer rows.Close()
	for rows.Next() {
		p := sdk.PipelineBuild{}
		err = scanPbShort(&p, rows)

		pb = append(pb, p)
	}
	return pb, nil
}

func scanPbShort(p *sdk.PipelineBuild, rows database.Scanner) error {
	var status, typePipeline string
	var manual, scheduled sql.NullBool
	var trigBy, pPbID, version sql.NullInt64
	var branch, hash, author, fromUser, fromPipeline sql.NullString

	err := rows.Scan(&p.Pipeline.ID, &p.Application.ID, &p.Environment.ID, &p.ID, &p.Pipeline.ProjectID,
		&p.Environment.Name, &p.Application.Name, &p.Pipeline.Name, &p.Pipeline.ProjectKey,
		&typePipeline,
		&p.BuildNumber, &p.Version, &status,
		&p.Start, &p.Done,
		&manual, &scheduled, &trigBy, &pPbID, &branch, &hash, &author,
		&fromUser, &fromPipeline, &version)
	if err != nil {
		log.Warning("scanPbShort> Error while loading build information: %s", err)
		return err
	}
	p.Status = sdk.StatusFromString(status)
	p.Pipeline.Type = sdk.PipelineTypeFromString(typePipeline)
	p.Application.ProjectKey = p.Pipeline.ProjectKey
	loadPbTrigger(p, manual, scheduled, pPbID, branch, hash, author, fromUser, fromPipeline, version)

	return nil
}

func loadPbTrigger(pb *sdk.PipelineBuild, manual, scheduled sql.NullBool, parentID sql.NullInt64, branch, hash, author, fromUser, fromPipeline sql.NullString, version sql.NullInt64) {
	if manual.Valid {
		pb.Trigger.ManualTrigger = manual.Bool
	}

	if scheduled.Valid {
		pb.Trigger.ScheduledTrigger = scheduled.Bool
	}

	if fromUser.Valid {
		pb.Trigger.TriggeredBy = &sdk.User{
			Username: fromUser.String,
		}
	}

	if fromPipeline.Valid && version.Valid && parentID.Valid {
		pb.Trigger.ParentPipelineBuild = &sdk.PipelineBuild{
			ID: parentID.Int64,
			Pipeline: sdk.Pipeline{
				Name: fromPipeline.String,
			},
			Version: version.Int64,
		}
	}

	if branch.Valid {
		pb.Trigger.VCSChangesBranch = branch.String
	}
	if hash.Valid {
		pb.Trigger.VCSChangesHash = hash.String
	}
	if author.Valid {
		pb.Trigger.VCSChangesAuthor = author.String
	}
}

// GetProbableLastBuildNumber returns the last build number at the time of query.
// Should be used only for non-sensitive query
func GetProbableLastBuildNumber(db *sql.DB, pipID, appID, envID int64) (int64, bool, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, false, err
	}
	defer tx.Rollback()

	i, o, err := GetLastBuildNumber(tx, pipID, appID, envID)
	if err != nil {
		return 0, false, err
	}

	err = tx.Commit()
	if err != nil {
		return 0, false, err
	}

	return i, o, nil
}

// GetLastBuildNumber Get the last build number for the given pipeline
func GetLastBuildNumber(tx database.QueryExecuter, pipelineID int64, applicationID int64, environmentID int64) (int64, bool, error) {
	var lastBuildingBuildNumber int64
	var lastFinishedBuildNumber int64

	// JIRA CD-1164: When starting a lot of pipeline in a short time,
	// there is a race condition when fetching the last build number used.
	// The solution implemented here is to lock the actual last build.
	// We then try to select build number twice until we got the same value locked
	// This is why GetLastBuildNumber now requires a transaction.
	var err error
	var actualValue, candidate int64
	for {
		query := `SELECT build_number FROM pipeline_build WHERE pipeline_id = $1 AND application_id = $2 AND environment_id = $3 ORDER BY build_number DESC LIMIT 1 FOR UPDATE`
		err = tx.QueryRow(query, pipelineID, applicationID, environmentID).Scan(&lastBuildingBuildNumber)
		if err != nil && err != sql.ErrNoRows {
			return 0, false, err
		}

		queryHistory := `SELECT build_number FROM pipeline_history WHERE pipeline_id = $1 AND application_id = $2 AND environment_id = $3 ORDER BY build_number DESC LIMIT 1`
		err = tx.QueryRow(queryHistory, pipelineID, applicationID, environmentID).Scan(&lastFinishedBuildNumber)
		if err != nil && err != sql.ErrNoRows {
			return 0, false, err
		}

		if lastFinishedBuildNumber >= lastBuildingBuildNumber {
			candidate = lastFinishedBuildNumber
			return lastFinishedBuildNumber, true, nil
		}

		candidate = lastBuildingBuildNumber

		query = `SELECT build_number FROM pipeline_build WHERE pipeline_id = $1 AND application_id = $2 AND environment_id = $3 ORDER BY build_number DESC LIMIT 1 FOR UPDATE`
		err = tx.QueryRow(query, pipelineID, applicationID, environmentID).Scan(&actualValue)
		if err != nil && err != sql.ErrNoRows {
			return 0, false, err
		}

		if candidate == actualValue {
			return candidate, false, nil
		}
	}
}

// InsertPipelineBuild insert build informations in database so Scheduler can pick it up
func InsertPipelineBuild(tx database.QueryExecuter, project *sdk.Project, p *sdk.Pipeline, applicationData *sdk.Application, applicationPipelineArgs []sdk.Parameter, params []sdk.Parameter, env *sdk.Environment, version int64, trigger sdk.PipelineBuildTrigger) (sdk.PipelineBuild, error) {
	var buildNumber int64
	var pb sdk.PipelineBuild
	var client sdk.RepositoriesManagerClient

	// Load last finished build
	buildNumber, _, err := GetLastBuildNumber(tx, p.ID, applicationData.ID, env.ID)
	if err != nil && err != sql.ErrNoRows {
		return pb, err
	}
	pb.BuildNumber = buildNumber + 1

	pb.Trigger = trigger

	// Reset version number when:
	// - provided version is invalid
	// - there is no parent
	// - the parent is not in the child application AND pipeline type is sdk.BuildPipeline
	pb.Version = version
	if pb.Version <= 0 ||
		trigger.ParentPipelineBuild == nil ||
		(applicationData.ID != trigger.ParentPipelineBuild.Application.ID && p.Type == sdk.BuildPipeline) {
		log.Debug("InsertPipelineBuild: Set version to buildnumber (provided: %d), has parent (%t), appID (%d)", version, trigger.ParentPipelineBuild != nil, applicationData.ID)
		pb.Version = pb.BuildNumber
	}

	params = append(params, sdk.Parameter{
		Name:  "cds.pipeline",
		Value: p.Name,
		Type:  sdk.StringParameter,
	})
	params = append(params, sdk.Parameter{
		Name:  "cds.project",
		Value: p.ProjectKey,
		Type:  sdk.StringParameter,
	})
	params = append(params, sdk.Parameter{
		Name:  "cds.application",
		Value: applicationData.Name,
		Type:  sdk.StringParameter,
	})
	params = append(params, sdk.Parameter{
		Name:  "cds.environment",
		Value: env.Name,
		Type:  sdk.StringParameter,
	})
	params = append(params, sdk.Parameter{
		Name:  "cds.buildNumber",
		Value: strconv.FormatInt(pb.BuildNumber, 10),
		Type:  sdk.StringParameter,
	})
	params = append(params, sdk.Parameter{
		Name:  "cds.version",
		Value: strconv.FormatInt(pb.Version, 10),
		Type:  sdk.StringParameter,
	})
	if pb.Trigger.TriggeredBy != nil {
		//Load user information to store them as args
		params = append(params, sdk.Parameter{
			Name:  "cds.triggered_by.username",
			Value: pb.Trigger.TriggeredBy.Username,
			Type:  sdk.StringParameter,
		})
		params = append(params, sdk.Parameter{
			Name:  "cds.triggered_by.fullname",
			Value: pb.Trigger.TriggeredBy.Fullname,
			Type:  sdk.StringParameter,
		})
		params = append(params, sdk.Parameter{
			Name:  "cds.triggered_by.email",
			Value: pb.Trigger.TriggeredBy.Email,
			Type:  sdk.StringParameter,
		})
	}

	if pb.Trigger.VCSChangesBranch != "" {
		// child inherit git.branch from parent
		params = append(params, sdk.Parameter{
			Name:  "git.branch",
			Value: pb.Trigger.VCSChangesBranch,
			Type:  sdk.StringParameter,
		})
		// child inherit git.hash from parent
		params = append(params, sdk.Parameter{
			Name:  "git.hash",
			Value: pb.Trigger.VCSChangesHash,
			Type:  sdk.StringParameter,
		})
	} else {
		//We consider default branch is master
		defautlBranch := "master"
		lastGitHash := map[string]string{}
		if applicationData.RepositoriesManager != nil && applicationData.RepositoryFullname != "" {
			client, _ = repositoriesmanager.AuthorizedClient(tx, project.Key, applicationData.RepositoriesManager.Name)
			if client != nil {
				branches, _ := client.Branches(applicationData.RepositoryFullname)
				for _, b := range branches {
					//If application is linked to a repository manager, we try to found de default branch
					if b.Default {
						defautlBranch = b.DisplayID
					}
					//And we store LatestCommit for each branches
					lastGitHash[b.DisplayID] = b.LatestCommit
				}
			}
		}

		// If branch is not provided from parent
		// then maybe it was directly set by pipeline parameters
		// if not, then it's master
		found := false
		hashFound := false
		for _, p := range params {
			if p.Name == "git.branch" && p.Value != "" {
				found = true
				pb.Trigger.VCSChangesBranch = p.Value
			}
			if p.Name == "git.hash" && p.Value != "" {
				hashFound = true
				pb.Trigger.VCSChangesHash = p.Value
			}
		}

		if !found {
			//If git.branch was not found is pipeline parameters, we set de previously found defaultBranch
			params = append(params, sdk.Parameter{
				Name:  "git.branch",
				Value: defautlBranch,
				Type:  sdk.StringParameter,
			})
			pb.Trigger.VCSChangesBranch = defautlBranch

			//And we try to put the lastestCommit for this branch
			if lastGitHash[defautlBranch] != "" {
				params = append(params, sdk.Parameter{
					Name:  "git.hash",
					Value: lastGitHash[defautlBranch],
					Type:  sdk.StringParameter,
				})
				pb.Trigger.VCSChangesHash = lastGitHash[defautlBranch]
			}
		} else {
			//If git.branch was found but git.hash wasn't found in pipeline parameters
			//we try to found the LatestCommit
			if !hashFound && lastGitHash[pb.Trigger.VCSChangesBranch] != "" {
				params = append(params, sdk.Parameter{
					Name:  "git.hash",
					Value: lastGitHash[pb.Trigger.VCSChangesBranch],
					Type:  sdk.StringParameter,
				})
				pb.Trigger.VCSChangesHash = lastGitHash[pb.Trigger.VCSChangesBranch]
			}
		}
	}

	// Process Pipeline Argument
	mapVar, err := ProcessPipelineBuildVariables(p.Parameter, applicationPipelineArgs, params)
	if err != nil {
		log.Warning("InsertPipelineBuild> Cannot process args: %s\n", err)
		return pb, err
	}

	// sdk.Build should have sdk.Variable instead of []string
	var argsFinal []sdk.Parameter
	for _, v := range mapVar {
		argsFinal = append(argsFinal, v)
	}

	argsJSON, err := json.Marshal(argsFinal)
	if err != nil {
		log.Warning("InsertPipelineBuild> Cannot marshal build parameters: %s\n", err)
		return pb, err
	}

	err = insertPipelineBuild(tx, string(argsJSON), applicationData.ID, p.ID, &pb, env.ID)
	if err != nil {
		log.Warning("InsertPipelineBuild> Cannot insert pipeline build: %s\n", err)
		return pb, err
	}

	pb.Status = sdk.StatusBuilding
	pb.Pipeline = *p
	pb.Parameters = params
	pb.Application = *applicationData
	pb.Environment = *env

	// Update stats
	stats.PipelineEvent(tx, p.Type, project.ID, applicationData.ID)

	//Send notification
	//Load previous pipeline (some app, pip, env and branch)
	//Load branch
	branch := ""
	for _, param := range pb.Parameters {
		if param.Name == ".git.branch" {
			branch = param.Value
			break
		}
	}
	//Get the history
	var previous *sdk.PipelineBuild
	history, err := LoadPipelineBuildHistoryByApplicationAndPipeline(tx, pb.Application.ID, pb.Pipeline.ID, pb.Environment.ID, 2, "", branch)
	if err != nil {
		log.Critical("InsertPipelineBuild> error while loading previous pipeline build")
	}
	//Be sure to get the previous one
	if len(history) == 2 {
		for i := range history {
			if previous == nil || previous.BuildNumber > history[i].BuildNumber {
				previous = &history[i]
			}
		}
	}

	notification.SendPipeline(tx, &pb, sdk.CreateNotifEvent, sdk.StatusBuilding, previous)

	return pb, nil
}

func insertPipelineBuild(db database.QueryExecuter, args string, applicationID, pipelineID int64, pb *sdk.PipelineBuild, envID int64) error {
	query := `INSERT INTO pipeline_build (pipeline_id, build_number, version, status, args, start, application_id,environment_id, done, manual_trigger, triggered_by, parent_pipeline_build_id, vcs_changes_branch, vcs_changes_hash, vcs_changes_author, scheduled_trigger)
						VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16) RETURNING id`

	var triggeredBy, parentPipelineID int64
	if pb.Trigger.TriggeredBy != nil {
		triggeredBy = pb.Trigger.TriggeredBy.ID
	}
	if pb.Trigger.ParentPipelineBuild != nil {
		parentPipelineID = pb.Trigger.ParentPipelineBuild.ID
	}

	statement := db.QueryRow(
		query, pipelineID, pb.BuildNumber, pb.Version, sdk.StatusBuilding.String(),
		args, time.Now(), applicationID, envID, time.Now(), pb.Trigger.ManualTrigger,
		sql.NullInt64{Int64: triggeredBy, Valid: triggeredBy != 0},
		sql.NullInt64{Int64: parentPipelineID, Valid: parentPipelineID != 0},
		pb.Trigger.VCSChangesBranch, pb.Trigger.VCSChangesHash, pb.Trigger.VCSChangesAuthor, pb.Trigger.ScheduledTrigger)
	err := statement.Scan(&pb.ID)
	if err != nil {
		return fmt.Errorf("App:%d,Pip:%d,Env:%d> %s", applicationID, pipelineID, envID, err)
	}

	return nil
}

// LoadPipelineBuildHistoryByApplication Load application history
// DEPRECATED! See: project.LoadBuildActivity
func LoadPipelineBuildHistoryByApplication(db database.Querier, applicationID int64, limit int) ([]sdk.PipelineBuild, error) {
	pbs := []sdk.PipelineBuild{}

	// Load history from pipeline build
	query := fmt.Sprintf(LoadPipelineBuildRequest, "", "pb.application_id = $1", "LIMIT $2")
	rows, err := db.Query(query, applicationID, limit)
	if err != nil {
		return nil, fmt.Errorf("LoadPipelineBuildHistoryByApplication> Cannot load pipeline build: %s", err)
	}

	for rows.Next() {
		var pb sdk.PipelineBuild
		err = scanPbShort(&pb, rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		pbs = append(pbs, pb)
	}
	rows.Close()

	if len(pbs) < limit {
		query := `SELECT pipeline_history.data,
			                 pipeline.name,
			                 environment.name
			          FROM pipeline_history
			          JOIN pipeline on pipeline.id = pipeline_history.pipeline_id
			          JOIN environment on environment.id = pipeline_history.environment_id
			          WHERE pipeline_history.application_id = $1
			          ORDER BY pipeline_history.done DESC
			          LIMIT $2`
		rows, err := db.Query(query, applicationID, limit-len(pbs))
		if err != nil {
			return nil, fmt.Errorf("LoadPipelineBuildHistoryByApplication> Cannot load pipeline history: %s", err)
		}

		for rows.Next() {
			var sData, pipName, envName string
			err = rows.Scan(&sData, &pipName, &envName)
			if err != nil {
				rows.Close()
				return nil, fmt.Errorf("LoadPipelineBuildHistoryByApplication> Cannot read db result when loading pipeline history: %s", err)
			}

			var pb sdk.PipelineBuild
			err = json.Unmarshal([]byte(sData), &pb)

			if err != nil {
				rows.Close()
				return nil, fmt.Errorf("LoadPipelineBuildHistoryByApplication> Cannot unmarshal history: %s", err)
			}
			pb.Pipeline.Name = pipName
			pb.Environment.Name = envName
			//Emptying stages, parameters
			pb.Stages = []sdk.Stage{}
			pbs = append(pbs, pb)
		}
		rows.Close()
	}

	return pbs, nil
}

// LoadPipelineBuildHistoryByApplicationAndPipeline Load pipeline history
func LoadPipelineBuildHistoryByApplicationAndPipeline(db database.Querier, applicationID, pipelineID, environmentID int64, limit int, status, branchName string, args ...FuncArg) ([]sdk.PipelineBuild, error) {
	pbs := []sdk.PipelineBuild{}
	var query string
	var rows *sql.Rows
	var err error

	c := structarg{}
	for _, f := range args {
		f(&c)
	}

	// Load history from pipeline build
	if status == "" && branchName == "" {
		query = fmt.Sprintf(LoadPipelineBuildRequest, "", "pb.application_id= $1 AND pb.pipeline_id = $2 AND pb.environment_id = $3", "LIMIT $4")
		rows, err = db.Query(query, applicationID, pipelineID, environmentID, limit)
	} else if status != "" && branchName == "" {
		query = fmt.Sprintf(LoadPipelineBuildRequest, "", "pb.application_id= $1 AND pb.pipeline_id = $2 AND pb.environment_id = $3 AND pb.status = $5", "LIMIT $4")
		rows, err = db.Query(query, applicationID, pipelineID, environmentID, limit, status)
	} else if status == "" && branchName != "" {
		query = fmt.Sprintf(LoadPipelineBuildRequest, "", "pb.application_id= $1 AND pb.pipeline_id = $2 AND pb.environment_id = $3 AND pb.vcs_changes_branch = $5", "LIMIT $4")
		rows, err = db.Query(query, applicationID, pipelineID, environmentID, limit, branchName)
	} else {
		query = fmt.Sprintf(LoadPipelineBuildRequest, "", "pb.application_id= $1 AND pb.pipeline_id = $2 AND pb.environment_id = $3 AND pb.status = $5 AND pb.vcs_changes_branch = $6", "LIMIT $4")
		rows, err = db.Query(query, applicationID, pipelineID, environmentID, limit, status, branchName)
	}

	if err != nil {
		return nil, fmt.Errorf("LoadPipelineBuildHistoryByApplicationAndPipeline> Cannot load pipeline build: %s", err)
	}

	for rows.Next() {
		var pb sdk.PipelineBuild
		err = scanPbShort(&pb, rows)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("LoadPipelineBuildHistoryByApplicationAndPipeline> Cannot read db result when loading pipeline build: %s", err)
		}

		if c.loadstages {
			err := loadStageAndActionBuilds(db, &pb)
			if err != nil {
				return nil, fmt.Errorf("LoadPipelineBuildHistoryByApplicationAndPipeline> Cannot load stages from pipeline build %d: %s", pb.ID, err)
			}
		}

		if c.loadparameters {
			queryParams := `SELECT args FROM pipeline_build WHERE id = $1`
			var params string
			err = db.QueryRow(queryParams, pb.ID).Scan(&params)
			if err != nil {
				return nil, fmt.Errorf("LoadPipelineBuildHistoryByApplicationAndPipeline> Cannot load pipeline build parameters: %s", err)
			}
			err = json.Unmarshal([]byte(params), &pb.Parameters)
			if err != nil {
				return nil, fmt.Errorf("LoadPipelineBuildHistoryByApplicationAndPipeline> Cannot unmarshal pipeline build parameters: %s", err)
			}
		}

		pbs = append(pbs, pb)
	}
	rows.Close()

	if len(pbs) < limit {

		if c.loadstages || c.loadparameters {
			// Load all pipeline history
			phs, err := SelectBuildsInHistory(db, pipelineID, applicationID, environmentID, limit, status)
			if err != nil {
				return nil, fmt.Errorf("LoadPipelineBuildHistoryByApplication> Cannot load pipeline history with stages: %s", err)
			}
			pbs = append(pbs, phs...)

		} else {
			if status == "" && branchName == "" {
				query = fmt.Sprintf(LoadPipelineHistoryRequest, "", "ph.application_id= $1 AND ph.pipeline_id = $2 AND ph.environment_id = $3", "LIMIT $4")
				rows, err = db.Query(query, applicationID, pipelineID, environmentID, limit)
			} else if status != "" && branchName == "" {
				query = fmt.Sprintf(LoadPipelineHistoryRequest, "", "ph.application_id= $1 AND ph.pipeline_id = $2 AND ph.environment_id = $3 AND ph.status = $5", "LIMIT $4")
				rows, err = db.Query(query, applicationID, pipelineID, environmentID, limit, status)
			} else if status == "" && branchName != "" {
				query = fmt.Sprintf(LoadPipelineHistoryRequest, "", "ph.application_id= $1 AND ph.pipeline_id = $2 AND ph.environment_id = $3 ph.vcs_changes_branch = $5", "LIMIT $4")
				rows, err = db.Query(query, applicationID, pipelineID, environmentID, limit, branchName)
			} else {
				query = fmt.Sprintf(LoadPipelineHistoryRequest, "", "ph.application_id= $1 AND ph.pipeline_id = $2 AND ph.environment_id = $3 AND ph.status = $5 AND ph.vcs_changes_branch = $6", "LIMIT $4")
				rows, err = db.Query(query, applicationID, pipelineID, environmentID, limit, status, branchName)
			}
			if err != nil {
				return nil, fmt.Errorf("LoadPipelineBuildHistoryByApplication> Cannot load pipeline history: %s", err)
			}

			for rows.Next() {
				var pb sdk.PipelineBuild
				err = scanPbShort(&pb, rows)
				if err != nil {
					rows.Close()
					return nil, fmt.Errorf("LoadPipelineBuildHistoryByApplication> Cannot read db result when loading pipeline build: %s", err)
				}
				pbs = append(pbs, pb)
			}
			rows.Close()
		}
	}

	return pbs, nil
}

// LoadPipelineHistoryBuild retrieves informations about a specific build in pipeline_history
func LoadPipelineHistoryBuild(db database.Querier, pipelineID int64, applicationID int64, buildNumber int64, environmentID int64) (sdk.PipelineBuild, error) {
	var pb sdk.PipelineBuild

	query := fmt.Sprintf(LoadPipelineHistoryRequest, "", "ph.pipeline_id = $1 AND ph.build_number = $2 AND ph.application_id= $3 AND ph.environment_id = $4", "LIMIT 1")
	rows, err := db.Query(query, pipelineID, buildNumber, applicationID, environmentID)
	if err != nil {
		return pb, err
	}

	defer rows.Close()
	for rows.Next() {
		err = scanPbShort(&pb, rows)
		return pb, err
	}
	return pb, sdk.ErrNoPipelineBuild
}

// LoadPipelineBuild retrieves informations about a specific build
func LoadPipelineBuild(db database.Querier, pipelineID int64, applicationID int64, buildNumber int64, environmentID int64, args ...FuncArg) (sdk.PipelineBuild, error) {
	var pb sdk.PipelineBuild

	query := fmt.Sprintf(LoadPipelineBuildRequest, "", "pb.pipeline_id = $1 AND pb.build_number = $2 AND pb.application_id = $3 AND pb.environment_id = $4", "")

	rows, err := db.Query(query, pipelineID, buildNumber, applicationID, environmentID)
	if err != nil {
		return pb, err
	}
	defer rows.Close()
	for rows.Next() {
		err = scanPbShort(&pb, rows)
		if err != nil {
			return pb, err
		}

		c := structarg{}
		for _, f := range args {
			f(&c)
		}

		if c.loadparameters {
			queryParams := `SELECT args FROM pipeline_build WHERE id = $1`
			var params string
			err = db.QueryRow(queryParams, pb.ID).Scan(&params)
			if err != nil {
				return pb, fmt.Errorf("LoadPipelineBuild> Cannot load pipeline build parameters: %s", err)
			}
			err = json.Unmarshal([]byte(params), &pb.Parameters)
			if err != nil {
				return pb, fmt.Errorf("LoadPipelineBuild> Cannot unmarshal pipeline build parameters: %s", err)
			}
		}
		return pb, nil
	}

	return pb, sdk.ErrNoPipelineBuild
}

// LoadPipelineBuildChildren load triggered pipeline from given build
func LoadPipelineBuildChildren(db *sql.DB, pipelineID int64, applicationID int64, buildNumber int64, environmentID int64) ([]sdk.PipelineBuild, error) {
	pbs := []sdk.PipelineBuild{}
	query := fmt.Sprintf(LoadPipelineBuildRequest, "", "pbTriggerFrom.pipeline_id = $1 AND pbTriggerFrom.build_number = $2 AND pbTriggerFrom.application_id = $3 AND pbTriggerFrom.environment_id = $4", "")

	rows, err := db.Query(query, pipelineID, buildNumber, applicationID, environmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var pb sdk.PipelineBuild
		err = scanPbShort(&pb, rows)
		if err != nil {
			return nil, err
		}
		pbs = append(pbs, pb)
	}

	return pbs, nil
}

// StopPipelineBuild fails all currently building actions
func StopPipelineBuild(db *sql.DB, pbID int64) error {
	query := `UPDATE action_build SET status = $1, done = now() WHERE pipeline_build_id = $2 AND status IN ( $3, $4 )`
	_, err := db.Exec(query, string(sdk.StatusFail), pbID, string(sdk.StatusBuilding), string(sdk.StatusWaiting))
	if err != nil {
		return err
	}

	// TODO: Add log to inform user

	return nil
}

// RestartPipelineBuild restarts failed actions build
func RestartPipelineBuild(db *sql.DB, pb sdk.PipelineBuild) error {
	var actionBuilds []sdk.ActionBuild
	var err error
	if pb.Status == sdk.StatusSuccess {
		actionBuilds, err = loadActionBuildsByStagePosition(db, pb.ID, 1)
	} else {
		actionBuilds, err = loadAllActionBuilds(db, pb.ID)
	}
	if err != nil {
		return fmt.Errorf("RestartPipelineBuild> Cannot load action builds for pipeline ID %d : %s", pb.ID, err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("RestartPipelineBuild> Cannot start tx: %s", err)
	}
	defer tx.Rollback()

	for _, ab := range actionBuilds {
		if ab.Status != sdk.StatusDisabled && ab.Status != sdk.StatusSkipped && (ab.Status == sdk.StatusFail || pb.Status == sdk.StatusSuccess) {
			log.Notice("RestartPipelineBuild: Action %s: restarting\n", ab.ActionName)
			err = RestartActionBuild(tx, ab.ID)
			if err != nil {
				return err
			}
		}
	}

	if pb.Status == sdk.StatusSuccess {
		// Select other actions build
		actionBuildIDs := []int64{}
		query := `SELECT action_build.id FROM action_build
			  JOIN pipeline_action ON pipeline_action.id = action_build.pipeline_action_id
			  JOIN pipeline_stage ON pipeline_stage.id = pipeline_action.pipeline_stage_id
	 		  WHERE action_build.pipeline_build_id = $1 and pipeline_stage.build_order > 1
	 		  	AND action_build.status not in ($2, $3)`
		rows, err := tx.Query(query, pb.ID, sdk.StatusDisabled.String(), sdk.StatusSkipped.String())
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var abID int64
			err = rows.Scan(&abID)
			if err != nil {
				return err
			}
			actionBuildIDs = append(actionBuildIDs, abID)
		}

		for _, id := range actionBuildIDs {
			err := build.DeleteBuildLogs(tx, id)
			if err != nil {
				return err
			}
			queryDelete := `DELETE FROM action_build WHERE id = $1`
			_, err = tx.Exec(queryDelete, id)
			if err != nil {
				log.Warning("RestartPipelineBuild> Cannot remove action builds %d: %s\n", id, err)
				return err
			}
		}

		// Delete test results
		err = build.DeletePipelineTestResults(db, pb.ID)
		if err != nil {
			return err
		}

		// Update start time
		queryUpdateStart := `UPDATE pipeline_build set start = current_timestamp WHERE id = $1`
		_, err = tx.Exec(queryUpdateStart, pb.ID)
		if err != nil {
			return err
		}

	}

	err = UpdatePipelineBuildStatus(tx, pb, sdk.StatusBuilding)
	if err != nil {
		return fmt.Errorf("RestartPipelineBuild> UpdatePipelineBuildStatus> %s", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("RestartPipelineBuild> Cannot commit tx: %s", err)
	}

	return nil
}

// RestartActionBuild destroy action build data and queue it up again
func RestartActionBuild(db *sql.Tx, actionBuildID int64) error {
	var plholder int64

	// Select for update to prevent unwanted update
	query := `SELECT id FROM action_build WHERE id = $1 FOR UPDATE`
	err := db.QueryRow(query, actionBuildID).Scan(&plholder)
	if err != nil {
		return fmt.Errorf("action_build %d: %s", actionBuildID, err)
	}

	// Delete previous build logs
	query = `DELETE FROM build_log WHERE action_build_id = $1`
	_, err = db.Exec(query, actionBuildID)
	if err != nil {
		return err
	}

	// Update status to Waiting
	query = `UPDATE action_build SET status = $1 WHERE id = $2`
	res, err := db.Exec(query, sdk.StatusWaiting.String(), actionBuildID)
	if err != nil {
		return err
	}

	aff, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if aff != 1 {
		return fmt.Errorf("could not restart ab %d: %d rows affected", actionBuildID, aff)
	}

	return nil
}

// LoadCompletePipelineBuildToArchive Load all information about a build
func LoadCompletePipelineBuildToArchive(db database.Querier, pipelineBuildID int64) (sdk.PipelineBuild, error) {
	var pb sdk.PipelineBuild

	query := `SELECT
			 pipeline_build.id,
			 pipeline_build.status,
			 pipeline_build.version,
			 pipeline_build.build_number,
			 pipeline_build.args,
			 pipeline_build.start,
			 pipeline_build.done,
			 pipeline_build.pipeline_id,
			 pipeline_build.application_id,
			 pipeline_build.environment_id,
			 action_build.id,
			 action_build.pipeline_action_id,
			 action_build.args,
			 action_build.status,
			 action_build.queued,
			 action_build.start,
			 action_build.done,
			 action_build.worker_model_name,
			 action.name,
			 pipeline_stage.id,
			 pipeline_stage.name,
			 pipeline_stage.build_order,
			 application.name,
			 pipeline.name,
			 environment.name,
			 pipeline_build.manual_trigger,
			 pipeline_build.scheduled_trigger,
			 pipeline_build.triggered_by,
			 pipeline_build.parent_pipeline_build_id,
			 pipeline_build.vcs_changes_branch,
			 pipeline_build.vcs_changes_hash,
			 pipeline_build.vcs_changes_author,
			 "user".username,
			 triggeredFromPip.name as trigPipName,
			 triggeredFromPb.version as versionTriggerFrom
		FROM pipeline_build
		JOIN application ON application.id = pipeline_build.application_id
		JOIN pipeline ON pipeline.id = pipeline_build.pipeline_id
		JOIN environment ON environment.id = pipeline_build.environment_id
		LEFT JOIN action_build ON action_build.pipeline_build_id = pipeline_build.id
		LEFT JOIN pipeline_action ON pipeline_action.id = action_build.pipeline_action_id
		LEFT JOIN action ON action.id = pipeline_action.action_id
		LEFT JOIN pipeline_stage ON pipeline_stage.id = pipeline_action.pipeline_stage_id
		LEFT JOIN "user" ON "user".id = triggered_by
		LEFT JOIN (
			SELECT id, version, pipeline_id FROM pipeline_build
			UNION
			SELECT pipeline_build_id, version, pipeline_id FROM pipeline_history
		) as triggeredFromPb ON triggeredFromPb.id = pipeline_build.parent_pipeline_build_id
		LEFT JOIN pipeline triggeredFromPip ON triggeredFromPip.id = triggeredFromPb.pipeline_id
	 	WHERE pipeline_build.id = $1 AND (pipeline_build.status = $2 OR pipeline_build.status = $3)`

	rows, err := db.Query(query, pipelineBuildID, string(sdk.StatusSuccess), string(sdk.StatusFail))
	if err != nil {
		log.Warning("LoadCompletePipelineBuildToArchive> Error querying : %s", err)
		return pb, err
	}
	defer rows.Close()

	var pbDone pq.NullTime
	for rows.Next() {
		var actionBuild sdk.ActionBuild
		var pbArgs, abArgs string
		var actionStart, actionDone, actionQueued pq.NullTime
		var pipelineBuildStatus, actionBuildStatus string
		var stage sdk.Stage
		var manual, scheduled sql.NullBool
		var stageBuildOrder, stageID, actionBuildID, actionBuildPipelineActionID, trigBy, parentID sql.NullInt64
		var stageName, actionBuildStatusTmp, actionBuildArgs, actionBuildActionName, branch, hash, author, username, trigPipname, actionBuildWorkerModelName sql.NullString
		var version sql.NullInt64

		err = rows.Scan(
			&pb.ID,
			&pipelineBuildStatus,
			&pb.Version,
			&pb.BuildNumber,
			&pbArgs,
			&pb.Start,
			&pbDone,
			&pb.Pipeline.ID,
			&pb.Application.ID,
			&pb.Environment.ID,
			&actionBuildID,
			&actionBuildPipelineActionID,
			&actionBuildArgs,
			&actionBuildStatusTmp,
			&actionQueued,
			&actionStart,
			&actionDone,
			&actionBuildWorkerModelName,
			&actionBuildActionName,
			&stageID,
			&stageName,
			&stageBuildOrder,
			&pb.Application.Name,
			&pb.Pipeline.Name,
			&pb.Environment.Name,
			&manual,
			&scheduled,
			&trigBy,
			&parentID,
			&branch,
			&hash,
			&author,
			&username,
			&trigPipname,
			&version,
		)
		if err != nil {
			log.Warning("LoadCompletePipelineBuildToArchive> Error scanning : %s", err)
			return pb, err
		}

		if pbDone.Valid {
			pb.Done = pbDone.Time
		}

		if actionBuildID.Valid {
			actionBuild.ID = actionBuildID.Int64
			actionBuild.PipelineActionID = actionBuildPipelineActionID.Int64
			actionBuild.ActionName = actionBuildActionName.String
			actionBuild.Queued = actionQueued.Time
			actionBuildStatus = actionBuildStatusTmp.String
			abArgs = actionBuildArgs.String
		}

		if stageID.Valid {
			stage.ID = stageID.Int64
			stage.Name = stageName.String
			stage.BuildOrder = int(stageBuildOrder.Int64)
		}

		//Skipped and disabled action :
		if actionStart.Valid {
			actionBuild.Start = actionStart.Time
		} else {
			actionBuild.Start = pb.Start
		}
		if actionDone.Valid {
			actionBuild.Done = actionDone.Time
		} else {
			actionBuild.Done = pb.Done
		}

		if actionBuildWorkerModelName.Valid {
			actionBuild.Model = actionBuildWorkerModelName.String
		}

		pb.Trigger = sdk.PipelineBuildTrigger{}
		loadPbTrigger(&pb, manual, scheduled, parentID, branch, hash, author, username, trigPipname, version)

		if trigBy.Valid && pb.Trigger.TriggeredBy != nil {
			pb.Trigger.TriggeredBy.ID = trigBy.Int64
		}

		if actionBuildID.Valid {
			actionBuild.Status = sdk.StatusFromString(actionBuildStatus)
		}

		pb.Status = sdk.StatusFromString(pipelineBuildStatus)

		if err = json.Unmarshal([]byte(pbArgs), &pb.Parameters); err != nil {
			log.Warning("LoadCompletePipelineBuildToArchive> Error unmarshalling : %s", err)
			return pb, err
		}

		if actionBuildID.Valid {
			if err = json.Unmarshal([]byte(abArgs), &actionBuild.Args); err != nil {
				var oa []string
				err = json.Unmarshal([]byte(abArgs), &oa)
				if err != nil {
					log.Warning("LoadCompletePipelineBuildToArchive> Error unmarshalling : %s", err)
					return pb, err
				}
				for _, op := range oa {
					t := strings.SplitN(op, "=", 2)
					p := sdk.Parameter{
						Name:  t[0],
						Type:  sdk.StringParameter,
						Value: t[1],
					}
					actionBuild.Args = append(actionBuild.Args, p)
				}
			}
		}

		// Add stage and action to result
		stageAttached := false
		if actionBuildID.Valid {
			for i := range pb.Stages {
				s := &pb.Stages[i]
				if stage.BuildOrder == s.BuildOrder {
					stageAttached = true
					s.ActionBuilds = append(s.ActionBuilds, actionBuild)
				}
			}
		}

		if stageID.Valid && !stageAttached {
			stage.ActionBuilds = append(stage.ActionBuilds, actionBuild)
			pb.Stages = append(pb.Stages, stage)
		}
	}

	for _, stage := range pb.Stages {
		for i := range stage.ActionBuilds {
			actionBuild := &stage.ActionBuilds[i]
			// add logs
			var logs sql.NullString
			queryLog := `
			SELECT string_agg(log,'')
			FROM (
				SELECT '[' || timestamp || ']' || ' ' || value as log
				FROM build_log
				WHERE action_build_id = $1
				ORDER BY build_log.id ASC
			     ) sub
			`
			if err := db.QueryRow(queryLog, actionBuild.ID).Scan(&logs); err != nil {
				if err != sql.ErrNoRows {
					log.Warning("LoadCompletePipelineBuildToArchive> Error querying build_log : %s", err)
					return pb, err
				}
			} else if logs.Valid {
				actionBuild.Logs = logs.String
			}
		}
	}

	return pb, nil
}

// SelectBuildForUpdate  Select a build and lock a build
func SelectBuildForUpdate(db database.Querier, buildID int64) error {
	var id int64
	query := `SELECT id
	          FROM pipeline_build
	          WHERE id = $1 AND status = $2
						FOR UPDATE NOWAIT`
	return db.QueryRow(query, buildID, sdk.StatusBuilding.String()).Scan(&id)
}

// LoadBuildIDsToArchive Load build to archive
func LoadBuildIDsToArchive(db *sql.DB, hours int) ([]int64, error) {
	log.Debug("LoadBuildIDsToArchive>...")
	var buildIDs []int64
	query := fmt.Sprintf(`SELECT pipeline_build.id
		  FROM pipeline_build
		  WHERE pipeline_build.done < NOW() - INTERVAL '%d hours'
		  AND (status = $1 OR status = $2)`, hours)

	rows, err := db.Query(query, string(sdk.StatusSuccess), string(sdk.StatusFail))
	if err != nil {
		return buildIDs, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return buildIDs, err
		}
		buildIDs = append(buildIDs, id)
	}
	return buildIDs, nil
}

// DeletePipelineBuildArtifact Delete artifact for the current build
func DeletePipelineBuildArtifact(db database.QueryExecuter, pipelineBuildID int64) error {
	// Delete pipeline build artifacts
	query := `SELECT artifact.id FROM artifact
	WHERE artifact.application_id IN (SELECT application_id FROM pipeline_build WHERE id = $1)
	AND artifact.environment_id IN (SELECT environment_id FROM pipeline_build WHERE id = $1)
	AND artifact.pipeline_id IN (SELECT pipeline_id FROM pipeline_build WHERE id = $1)
	AND artifact.build_number IN (SELECT build_number FROM pipeline_build WHERE id = $1);
	`
	rows, err := db.Query(query, pipelineBuildID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var ids []int64
	var id int64
	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			return err
		}
		ids = append(ids, id)
	}
	for _, id := range ids {
		err = artifact.DeleteArtifact(db, id)
		if err != nil {
			return fmt.Errorf("DeletePipelineBuild> cannot delete artifact %d> %s", id, err)
		}
	}
	return nil
}

// DeletePipelineBuild deletes a pipeline build and generated artifacts
func DeletePipelineBuild(db database.QueryExecuter, pipelineBuildID int64) error {
	err := DeletePipelineBuildArtifact(db, pipelineBuildID)
	if err != nil {
		return err
	}

	// Then delete pipeline build data
	err = build.DeleteBuild(db, pipelineBuildID)
	if err != nil {
		return err
	}

	return nil
}

//BuildNumberAndHash represents BuildNumber, Commit Hash and Branch for a Pipeline Build
type BuildNumberAndHash struct {
	BuildNumber int64
	Hash        string
	Branch      string
}

//CurrentAndPreviousPipelineBuildNumberAndHash returns a struct with BuildNumber, Commit Hash and Branch
//for the current pipeline build and the previous one on the same branch.
//Returned pointers may be null if pipeline build are not found
func CurrentAndPreviousPipelineBuildNumberAndHash(db database.Querier, buildNumber, pipelineID, applicationID, environmentID int64) (*BuildNumberAndHash, *BuildNumberAndHash, error) {
	query := `
			SELECT
				current_pipeline.build_number, current_pipeline.vcs_changes_hash, current_pipeline.vcs_changes_branch,
				previous_pipeline.build_number, previous_pipeline.vcs_changes_hash, previous_pipeline.vcs_changes_branch
			FROM
				(
					SELECT    id, pipeline_id, build_number, vcs_changes_branch, vcs_changes_hash
					FROM      pipeline_build
					WHERE 		build_number = $1
					AND				pipeline_id = $2
					AND				application_id = $3
					AND 			environment_id = $4
					UNION ALL (
						SELECT    pipeline_build_id as id, pipeline_id, build_number, vcs_changes_branch, vcs_changes_hash
						FROM      pipeline_history
						WHERE 		build_number = $1
						AND				pipeline_id = $2
						AND				application_id = $3
						AND 			environment_id = $4
					)
				) AS current_pipeline
			LEFT OUTER JOIN (
					SELECT    id, pipeline_id, build_number, vcs_changes_branch, vcs_changes_hash
					FROM      pipeline_build
					WHERE     build_number < $1
					AND				pipeline_id = $2
					AND				application_id = $3
					AND 			environment_id = $4
					UNION ALL (
						SELECT    pipeline_build_id as id, pipeline_id, build_number, vcs_changes_branch, vcs_changes_hash
						FROM      pipeline_history
						WHERE     build_number < $1
						AND				pipeline_id = $2
						AND				application_id = $3
						AND 			environment_id = $4
					)
					ORDER BY  build_number DESC
				) AS previous_pipeline ON (
					previous_pipeline.pipeline_id = current_pipeline.pipeline_id AND previous_pipeline.vcs_changes_branch = current_pipeline.vcs_changes_branch
				)
			WHERE current_pipeline.build_number = $1
			ORDER BY  previous_pipeline.build_number DESC
			LIMIT 1;
	`
	var curBuildNumber, prevBuildNumber sql.NullInt64
	var curHash, prevHash, curBranch, prevBranch sql.NullString
	err := db.QueryRow(query, buildNumber, pipelineID, applicationID, environmentID).Scan(&curBuildNumber, &curHash, &curBranch, &prevBuildNumber, &prevHash, &prevBranch)
	if err == sql.ErrNoRows {
		log.Warning("CurrentAndPreviousPipelineBuildNumberAndHash> no result with %d %d %d %d", buildNumber, pipelineID, applicationID, environmentID)
		return nil, nil, sdk.ErrNoPipelineBuild
	}
	if err != nil {
		return nil, nil, err
	}

	cur := &BuildNumberAndHash{}
	if curBuildNumber.Valid {
		cur.BuildNumber = curBuildNumber.Int64
	}
	if curHash.Valid {
		cur.Hash = curHash.String
	}
	if curBranch.Valid {
		cur.Branch = curBranch.String
	}

	prev := &BuildNumberAndHash{}
	if prevBuildNumber.Valid {
		prev.BuildNumber = prevBuildNumber.Int64
	} else {
		return cur, nil, nil
	}
	if prevHash.Valid {
		prev.Hash = prevHash.String
	}
	if prevBranch.Valid {
		prev.Branch = prevBranch.String
	}
	return cur, prev, nil
}

// GetDeploymentHistory Get all last deployment
func GetDeploymentHistory(db database.Querier, projectKey, appName string) ([]sdk.PipelineBuild, error) {
	pbs := []sdk.PipelineBuild{}
	query := `
		SELECT DISTINCT ON (pipName, envName) pipName, MAX(start),
			appName, envName,
			pb.version, pb.status, pb.done, pb.build_number,
			pb.manual_trigger, pb.scheduled_trigger, username, pb.vcs_changes_branch, pb.vcs_changes_hash, pb.vcs_changes_author
		FROM
		(
			(
				SELECT
					appName, pipName, envName,
					pb.version, pb.status, pb.done, pb.start, pb.build_number,
					pb.manual_trigger, pb.scheduled_trigger, "user".username, pb.vcs_changes_branch, pb.vcs_changes_hash, pb.vcs_changes_author
				FROM pipeline_build pb
				JOIN
				    (SELECT
					MAX(start) AS maxStart,
					application_id, pipeline_id, environment_id,
					application.name as appName, pipeline.name as pipName, environment.name as envName
				    FROM pipeline_build
				    JOIN application ON application.id = application_id
				    JOIN pipeline ON pipeline.id = pipeline_id
				    JOIN environment ON environment.id = environment_id
				    JOIN project ON project.id = application.project_id AND project.id = pipeline.project_id
				    WHERE pipeline.type = 'deployment' AND project.projectkey = $1 AND application.name = $2
				    GROUP BY pipeline_id, environment_id, application_id, appName, pipName, envName
				    ORDER BY MAX(start) DESC ) groupedtt
				ON pb.pipeline_id = groupedtt.pipeline_id AND pb.environment_id = groupedtt.environment_id AND pb.application_id = groupedtt.application_id
				AND pb.start = groupedtt.maxStart
				LEFT JOIN "user" ON "user".id = pb.triggered_by
			)
			UNION
			(
				SELECT
					appName, pipName, envName,
					ph.version, ph.status, ph.done, ph.start, ph.build_number,
					ph.manual_trigger, pb.scheduled_trigger, "user".username, ph.vcs_changes_branch, ph.vcs_changes_hash, ph.vcs_changes_author
				FROM pipeline_history ph
				JOIN
				    (SELECT
					MAX(start) AS maxStart,
					application_id, pipeline_id, environment_id,
					application.name as appName, pipeline.name as pipName, environment.name as envName
				    FROM pipeline_history
				    JOIN application ON application.id = application_id
				    JOIN pipeline ON pipeline.id = pipeline_id
				    JOIN environment ON environment.id = environment_id
				    JOIN project ON project.id = application.project_id AND project.id = pipeline.project_id
				    WHERE pipeline.type = 'deployment' AND project.projectkey = $1 AND application.name = $2
				    GROUP BY pipeline_id, environment_id, application_id, appName, pipName, envName
				    ORDER BY MAX(start) DESC ) groupedtt
				ON ph.pipeline_id = groupedtt.pipeline_id AND ph.environment_id = groupedtt.environment_id AND ph.application_id = groupedtt.application_id
				AND ph.start = groupedtt.maxStart
				LEFT JOIN "user" ON "user".id = ph.triggered_by
			)
		) pb
		GROUP BY pipName, appName, envName,
			pb.version, pb.status, pb.done, pb.build_number,
			pb.manual_trigger, pb.scheduled_trigger, username, pb.vcs_changes_branch, pb.vcs_changes_hash, pb.vcs_changes_author
		ORDER BY pipName ASC, envName ASC, max(start) DESC
	`
	rows, err := db.Query(query, projectKey, appName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var pb sdk.PipelineBuild
		var status string
		var user sdk.User
		var manual sql.NullBool
		var hash, author, username, branch sql.NullString

		err = rows.Scan(&pb.Pipeline.Name, &pb.Start,
			&pb.Application.Name, &pb.Environment.Name,
			&pb.Version, &status, &pb.Done, &pb.BuildNumber,
			&manual, &username, &branch, &hash, &author)
		if err != nil {
			return nil, err
		}

		if username.Valid {
			user.Username = username.String
		}
		pb.Trigger.TriggeredBy = &user
		pb.Status = sdk.StatusFromString(status)

		if branch.Valid {
			pb.Trigger.VCSChangesBranch = branch.String
		}
		if manual.Valid {
			pb.Trigger.ManualTrigger = manual.Bool
		}
		if hash.Valid {
			pb.Trigger.VCSChangesHash = hash.String
		}
		if author.Valid {
			pb.Trigger.VCSChangesAuthor = author.String
		}

		pbs = append(pbs, pb)
	}
	return pbs, nil
}

// GetVersions  Get version for the given application and branch
func GetVersions(db database.Querier, app *sdk.Application, branchName string) ([]int, error) {
	query := `
		SELECT version
		FROM
		(
			SELECT version
			FROM pipeline_build
			WHERE application_id = $1 AND vcs_changes_branch = $2


			UNION

			SELECT version
			FROM pipeline_history
			WHERE application_id = $1 AND vcs_changes_branch = $2
		) sub
		ORDER BY version DESC
		LIMIT 15
	`
	rows, err := db.Query(query, app.ID, branchName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	versions := []int{}
	for rows.Next() {
		var version int
		err = rows.Scan(&version)
		if err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	return versions, nil
}

// GetBranchHistory  Get last build for all branches
func GetBranchHistory(db database.Querier, projectKey, appName string, page, nbPerPage int) ([]sdk.PipelineBuild, error) {
	pbs := []sdk.PipelineBuild{}

	if page < 1 {
		page = 1
	}
	offset := nbPerPage * (page - 1)
	query := `
		WITH lastestBuild AS (
			(
				SELECT
					pb.application_id, pb.pipeline_id, pb.environment_id,
					appName, pipName, envName,
					pb.start, pb.done, pb.status, pb.version, pb.build_number,
					pb.manual_trigger, pb.scheduled_trigger, pb.triggered_by, pb.vcs_changes_branch, pb.vcs_changes_hash, pb.vcs_changes_author
				FROM
					pipeline_build pb
				JOIN (
					SELECT distinct(pipeline_id, environment_id, vcs_changes_branch) record, pipeline_id, environment_id, vcs_changes_branch, max(start) as start,
						application_id, application.name as appName, pipeline.name as pipName, environment.name as envName
					FROM pipeline_build
					JOIN application ON application.id = application_id
					JOIN pipeline ON pipeline.id = pipeline_id
					JOIN project ON project.id = application.project_id AND project.id = pipeline.project_id
					JOIN environment ON environment.id = environment_id AND
					(
						environment.project_id is NULL
						OR
						environment.project_id = project.id
					)
					WHERE vcs_changes_branch != ''
						AND vcs_changes_branch IS NOT NULL
						AND project.projectkey= $1
						AND application.name = $2
						AND pipeline.type = 'build'
					GROUP by pipeline_id, environment_id, application_id, vcs_changes_branch, appName, pipName, envName
					ORDER BY start DESC
				) hh ON hh.pipeline_id = pb.pipeline_id AND hh.application_id =pb.application_id AND hh.environment_id = pb.environment_id AND hh.start = pb.start
			)
			UNION ALL
			(
				SELECT
					ph.application_id, ph.pipeline_id, ph.environment_id,
					appName, pipName, envName,
					ph.start, ph.done, ph.status, ph.version,  ph.build_number,
					ph.manual_trigger, ph.scheduled_trigger, ph.triggered_by, ph.vcs_changes_branch, ph.vcs_changes_hash, ph.vcs_changes_author
				FROM
					pipeline_history ph
				JOIN (
					SELECT distinct(pipeline_id, environment_id, vcs_changes_branch) record, pipeline_id, environment_id, vcs_changes_branch, max(start) as start,
						application_id, application.name as appName, pipeline.name as pipName, environment.name as envName
					FROM pipeline_history
					JOIN application ON application.id = application_id
					JOIN pipeline ON pipeline.id = pipeline_id
					JOIN project ON project.id = application.project_id AND project.id = pipeline.project_id
					JOIN environment ON environment.id = environment_id AND
					(
						environment.project_id is NULL
						OR
						environment.project_id = project.id
					)
					WHERE vcs_changes_branch != ''
						AND vcs_changes_branch IS NOT NULL
						AND project.projectkey= $1
						AND application.name = $2
						AND pipeline.type = 'build'
					GROUP by pipeline_id, environment_id, application_id, vcs_changes_branch, appName, pipName, envName
					ORDER BY start DESC
				) hh ON hh.pipeline_id = ph.pipeline_id AND hh.application_id = ph.application_id AND hh.environment_id = ph.environment_id AND hh.start = ph.start
			)
		)
		SELECT
			lastestBuild.pipeline_id, lastestBuild.application_id, lastestBuild.environment_id,
			lastestBuild.appName, lastestBuild.pipName, lastestBuild.envName,
			lastestBuild.start, lastestBuild.done, lastestBuild.status, lastestBuild.version, lastestBuild.build_number,
			lastestBuild.manual_trigger, "user".username, lastestBuild.vcs_changes_branch, lastestBuild.vcs_changes_hash, lastestBuild.vcs_changes_author
		FROM lastestBuild
		JOIN (
			SELECT max(start) as start , application_id, pipeline_id, environment_id ,vcs_changes_branch
			FROM lastestBuild
			GROUP BY application_id, pipeline_id, environment_id ,vcs_changes_branch
		) m ON
			m.start = lastestBuild.start AND
			m.application_id = lastestBuild.application_id AND
			m.pipeline_id = lastestBuild.pipeline_id AND
			m.environment_id = lastestBuild.environment_id AND
			m.vcs_changes_branch = lastestBuild.vcs_changes_branch
		LEFT JOIN "user" ON "user".id = lastestBuild.triggered_by
		ORDER by lastestBuild.start DESC
		OFFSET $3
		LIMIT $4
	`
	rows, err := db.Query(query, projectKey, appName, offset, nbPerPage)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var pb sdk.PipelineBuild
		var status string
		var user sdk.User
		var manual sql.NullBool
		var hash, author, username sql.NullString

		err = rows.Scan(&pb.Pipeline.ID, &pb.Application.ID, &pb.Environment.ID,
			&pb.Application.Name, &pb.Pipeline.Name, &pb.Environment.Name,
			&pb.Start, &pb.Done, &status, &pb.Version, &pb.BuildNumber,
			&manual, &username, &pb.Trigger.VCSChangesBranch, &hash, &author,
		)
		if err != nil {
			return nil, err
		}

		if username.Valid {
			user.Username = username.String
		}
		pb.Trigger.TriggeredBy = &user

		pb.Status = sdk.StatusFromString(status)

		if manual.Valid {
			pb.Trigger.ManualTrigger = manual.Bool
		}
		if hash.Valid {
			pb.Trigger.VCSChangesHash = hash.String
		}
		if author.Valid {
			pb.Trigger.VCSChangesAuthor = author.String
		}
		pbs = append(pbs, pb)
	}
	return pbs, nil
}

//BuildExists checks if a build already exist
func BuildExists(db database.Querier, appID, pipID, envID int64, trigger *sdk.PipelineBuildTrigger) (bool, error) {
	query := `
		select count(1)
		from pipeline_build
		where application_id = $1
		and pipeline_id = $2
		and environment_id = $3
		and vcs_changes_hash = $4
		and vcs_changes_branch = $5
		and vcs_changes_author = $6
	`
	var count int
	if err := db.QueryRow(query, appID, pipID, envID, trigger.VCSChangesHash, trigger.VCSChangesBranch, trigger.VCSChangesAuthor).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil

}
