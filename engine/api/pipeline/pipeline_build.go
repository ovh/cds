package pipeline

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/stats"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/artifact"
)

const (
	SELECT_PB = `
		SELECT
			pb.id as id, pb.application_id as appID, pb.pipeline_id as pipID, pb.environment_id as envID,
			application.name as appName, pipeline.name as pipName, environment.name as envName,
			pb.build_number as build_number, pb.version as version, pb.status as status,
			pb.args as args, pb.stages as stages,
			pb.start as start, pb.done as done,
			pb.manual_trigger as manual_trigger, pb.triggered_by as triggered_by,
			pb.vcs_changes_branch as vcs_branch, pb.vcs_changes_hash as vcs_hash, pb.vcs_changes_author as vcs_author,
			pb.parent_pipeline_build_id as parent_pipeline_build,
			"user".username as username,
			pb.scheduled_trigger as scheduled_trigger
		FROM pipeline_build pb
		JOIN application ON application.id = pb.application_id
		JOIN pipeline ON pipeline.id = pb.pipeline_id
		JOIN environment ON environment.id = pb.environment_id
		LEFT JOIN "user" ON "user".id = pb.triggered_by
	`
)


// SelectBuildForUpdate  Select a build and lock a build
func SelectBuildForUpdate(db gorp.SqlExecutor, buildID int64) error {
	var id int64
	query := `SELECT id
	          FROM pipeline_build
	          WHERE id = $1 AND status = $2
		  FOR UPDATE NOWAIT`
	return db.QueryRow(query, buildID, sdk.StatusBuilding.String()).Scan(&id)
}

// LoadPipelineBuildID Load only id of pipeline build
func LoadPipelineBuildID(db gorp.SqlExecutor, applicationID, pipelineID, environmentID, buildNumber int64) (int64, error) {
	var pbID int64
	query := `SELECT id
	          FROM pipeline_build
	          WHERE application_id = $1, pipeline_id = $2, environment_id = $3, build_number = $4`
	if err := db.QueryRow(query, applicationID, pipelineID, environmentID, buildNumber).Scan(&pbID); err != nil {
		return 0, err
	}
	return pbID, nil
}

// LoadBuildingPipelines Load all building pipeline
func LoadBuildingPipelines(db gorp.SqlExecutor) ([]sdk.PipelineBuild, error) {
	whereCondition := `
		WHERE pb.status = $1
		ORDER by pb.id ASC
	`
	query := fmt.Sprintf("%s %s", SELECT_PB, whereCondition)
	var rows []sdk.PipelineBuildDbResult
	_, err := db.Select(&rows, query, sdk.StatusBuilding.String())
	if err != nil {
		return nil, err
	}

	pbs := []sdk.PipelineBuild{}
	for _, r := range rows {
		pb, errScan := scanPipelineBuild(r)
		if errScan != nil {
			return nil, errScan
		}
		pbs = append(pbs, *pb)
	}
	return pbs, nil
}

// LoadRecentPipelineBuild retrieves pipelines in database having a build running or finished
// less than a minute ago
func LoadRecentPipelineBuild(db gorp.SqlExecutor, args ...FuncArg) ([]sdk.PipelineBuild, error) {
	whereCondition := `
		WHERE pb.status = $1 OR (pb.status != $1 AND pb.done > NOW() - INTERVAL '1 minutes')
		ORDER by pb.id ASC
	`
	query := fmt.Sprintf("%s %s", SELECT_PB, whereCondition)
	var rows []sdk.PipelineBuildDbResult
	_, err := db.Select(&rows, query, sdk.StatusBuilding.String())
	if err != nil {
		return nil, err
	}

	pbs := []sdk.PipelineBuild{}
	for _, r := range rows {
		pb, errScan := scanPipelineBuild(r)
		if errScan != nil {
			return nil, errScan
		}
		pbs = append(pbs, *pb)
	}
	return pbs, nil
}

// LoadRecentPipelineBuild retrieves pipelines in database having a build running or finished
// less than a minute ago
func LoadUserRecentPipelineBuild(db gorp.SqlExecutor, userID int64) ([]sdk.PipelineBuild, error) {
	whereCondition := `
		JOIN pipeline_group ON pipeline_group.pipeline_id = pb.pipeline_id
		JOIN group_user ON group_user.group_id = pipeline_group.group_id
		WHERE pb.status = $1 OR (pb.status != $1 AND pb.done > NOW() - INTERVAL '1 minutes')
		AND group_user.user_id = $2
		ORDER by pb.id ASC
	`
	query := fmt.Sprintf("%s %s", SELECT_PB, whereCondition)
	var rows []sdk.PipelineBuildDbResult
	_, err := db.Select(&rows, query, sdk.StatusBuilding.String(), userID)
	if err != nil {
		return nil, err
	}

	pbs := []sdk.PipelineBuild{}
	for _, r := range rows {
		pb, errScan := scanPipelineBuild(r)
		if errScan != nil {
			return nil, errScan
		}
		pbs = append(pbs, *pb)
	}
	return pbs, nil
}

func LoadPipelineBuildByApplicationPipelineEnvBuildNumber(db gorp.SqlExecutor, applicationID, pipelineID, environmentID, buildNumber int64) (*sdk.PipelineBuild, error) {
	whereCondition := `
		WHERE pb.application_id = $1 AND pb.pipeline_id = $2 AND pb.environment_id = $3  AND pb.build_number = $4
	`
	query := fmt.Sprintf("%s %s", SELECT_PB, whereCondition)

	var row sdk.PipelineBuildDbResult
	_, err := db.Select(&row, query, applicationID, pipelineID, environmentID, buildNumber)
	if err != nil {
		return nil, err
	}
	return scanPipelineBuild(row)
}

// LoadPipelineBuildByHash look for a pipeline build triggered by a change with given hash
func LoadPipelineBuildByHash(db gorp.SqlExecutor, hash string) ([]sdk.PipelineBuild, error) {
	whereCondition := `
		WHERE pb.vcs_changes_hash = $1
	`
	var rows []sdk.PipelineBuildDbResult
	query := fmt.Sprintf("%s %s", SELECT_PB, whereCondition)
	if _, errQuery := db.Select(&rows, query, hash); errQuery != nil {
		return nil, errQuery
	}
	pbs := []sdk.PipelineBuild{}
	for _, r := range rows {
		pb, errScan := scanPipelineBuild(r)
		if errScan != nil {
			return nil, errScan
		}
		pbs = append(pbs, *pb)
	}
	return pbs, nil
}

// LoadPipelineBuildsByApplicationAndPipeline Load pipeline builds from application/pipeline/env status, branchname
func LoadPipelineBuildsByApplicationAndPipeline(db gorp.SqlExecutor, applicationID, pipelineID, environmentID int64, limit int, status, branchName string) ([]sdk.PipelineBuild, error) {
	whereCondition := `
		WHERE pb.application_id = $1 AND pb.pipeline_id = $2 AND pb.environment_id = $3 %s
	`
	query := fmt.Sprintf("%s %s", SELECT_PB, whereCondition)

	var rows []sdk.PipelineBuildDbResult
	var errQuery error
	if status == "" && branchName == "" {
		query = fmt.Sprintf(query, "")
		_, errQuery = db.Select(&rows, query, applicationID, pipelineID, environmentID, limit)
	} else if status != "" && branchName == "" {
		query = fmt.Sprintf(query, " AND pb.status = $5")
		_, errQuery = db.Select(&rows, query, applicationID, pipelineID, environmentID, limit, status)
	} else if status == "" && branchName != "" {
		query = fmt.Sprintf(query, " AND pb.vcs_changes_branch = $5")
		_, errQuery = db.Select(&rows, query, applicationID, pipelineID, environmentID, limit, branchName)
	} else {
		query = fmt.Sprintf(query, " AND pb.status = $5 AND pb.vcs_changes_branch = $6")
		_, errQuery = db.Select(&rows, query, applicationID, pipelineID, environmentID, limit, status, branchName)
	}
	if errQuery != nil {
		return nil, errQuery
	}

	pbs := []sdk.PipelineBuild{}
	for _, r := range rows {
		pb, errScan := scanPipelineBuild(r)
		if errScan != nil {
			return nil, errScan
		}
		pbs = append(pbs, *pb)
	}
	return pbs, nil
}

func LoadPipelineBuildByID(db gorp.SqlExecutor, id int64) (*sdk.PipelineBuild, error) {
	whereCondition := `
		WHERE pb.id = $1
	`
	query := fmt.Sprintf("%s %s", SELECT_PB, whereCondition)
	var row sdk.PipelineBuildDbResult
	_, err := db.Select(&row, query, id)
	if err != nil {
		return nil, err
	}
	return scanPipelineBuild(row)
}

// LoadPipelineBuildChildren load triggered pipeline from given build
func LoadPipelineBuildChildren(db gorp.SqlExecutor, pipelineID int64, applicationID int64, buildNumber int64, environmentID int64) ([]sdk.PipelineBuild, error) {
	pbs := []sdk.PipelineBuild{}

	pbID, errLoad := LoadPipelineBuildID(db, applicationID, pipelineID, environmentID, buildNumber)
	if errLoad != nil {
		return nil, errLoad
	}

	whereCondition := `
		WHERE pb.parent_pipeline_build_id = $1
	`
	query := fmt.Sprintf("%s %s", SELECT_PB, whereCondition)
	var rows []sdk.PipelineBuildDbResult
	_, err := db.Select(&rows, query, pbID)
	if err != nil {
		return nil, err
	}

	for _, r := range rows {
		pb, errScan := scanPipelineBuild(r)
		if errScan != nil {
			return nil, errScan
		}
		pbs = append(pbs, *pb)
	}
	return pbs, nil
}

func scanPipelineBuild(pbResult sdk.PipelineBuildDbResult) (*sdk.PipelineBuild, error) {
	pb := sdk.PipelineBuild{
		ID: pbResult.ID,
		Application: sdk.Application{
			ID:   pbResult.ApplicationID,
			Name: pbResult.ApplicatioName,
		},
		Pipeline: sdk.Pipeline{
			ID:   pbResult.PipelineID,
			Name: pbResult.PipelineName,
		},
		Environment: sdk.Environment{
			ID:   pbResult.EnvironmentID,
			Name: pbResult.EnvironmentName,
		},
		BuildNumber: pbResult.BuildNumber,
		Version:     pbResult.Version,
		Status:      sdk.StatusFromString(pbResult.Status),
		Start:       pbResult.Start,
		Done:        pbResult.Done,
		Trigger: sdk.PipelineBuildTrigger{
			ManualTrigger: pbResult.ManualTrigger,
			TriggeredBy: &sdk.User{
				ID:       pbResult.TriggeredBy,
				Username: pbResult.Username,
			},
			VCSChangesBranch: pbResult.VCSChangesBranch,
			VCSChangesAuthor: pbResult.VCSChangesAuthor,
			VCSChangesHash:   pbResult.VCSChangesHash,
		},
	}

	if err := json.Unmarshal([]byte(pbResult.Args), pb.Parameters); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(pbResult.Stages), pb.Stages); err != nil {
		return nil, err
	}

	return &pb, nil
}

// UpdatePipelineBuildStatusAndStage Update pipeline build status + stage
func UpdatePipelineBuildStatusAndStage(db gorp.SqlExecutor, pb *sdk.PipelineBuild) error {
	stagesB, errStage := json.Marshal(pb.Stages)
	if errStage != nil {
		return errStage
	}
	query := `UPDATE pipeline_build set status = $1, stages = $2 WHERE id = $3`
	_, err := db.Exec(query, pb.Status.String(), string(stagesB), pb.ID)
	return err
}

// DeletePipelineBuild Delete a pipeline build
func DeletePipelineBuild(db gorp.SqlExecutor, applicationID, pipelineID, environmentID, buildNumber int64) error {
	query := `
		DELETE FROM pipeline_build
		WHERE application_id = $1 AND pipeline_id = $2 AND environment_id = $3
		AND build_number = $4
	`

	_, errDelete := db.Query(query, applicationID, pipelineID, environmentID, buildNumber)
	return errDelete
}

// DeletePipelineBuildByID  Delete pipeline build by his ID
func DeletePipelineBuildByID(db gorp.SqlExecutor, pbID int64) error {
	query := `
		DELETE FROM pipeline_build
		WHERE id = $1
	`

	_, errDelete := db.Query(query, pbID)
	return errDelete
}

// GetLastBuildNumber returns the last build number at the time of query.
// Should be used only for non-sensitive query
func GetLastBuildNumberInTx(db *gorp.DbMap, pipID, appID, envID int64) (int64, error) {
	// JIRA CD-1164: When starting a lot of pipeline in a short time,
	// there is a race condition when fetching the last build number used.
	// The solution implemented here is to lock the actual last build.
	// We then try to select build number twice until we got the same value locked
	// This is why GetLastBuildNumber now requires a transaction.
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	lastBuildNumber, errBN := GetLastBuildNumber(tx,  pipID, appID, envID)
	if errBN != nil {
		return 0, errBN
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return lastBuildNumber, nil
}

// GetLastBuildNumber Get the last build number
func GetLastBuildNumber(db gorp.SqlExecutor,pipID, appID, envID int64 ) (int64, error) {
	var lastBuildNumber int64
	query := `SELECT build_number FROM pipeline_build WHERE pipeline_id = $1 AND application_id = $2 AND environment_id = $3 ORDER BY build_number DESC LIMIT 1 FOR UPDATE`
	if err := db.QueryRow(query, pipID, appID, envID).Scan(&lastBuildNumber); err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	return lastBuildNumber, nil
}

// InsertBuildVariable adds a variable exported in user scripts and forwarded by building worker
func InsertBuildVariable(db gorp.SqlExecutor, pbID int64, v sdk.Variable) error {

	// Load args from pipeline build and lock it
	query := `SELECT args FROM pipeline_build WHERE id = $1 FOR UPDATE`
	var argsJSON string
	if err := db.QueryRow(query, pbID).Scan(&argsJSON); err != nil {
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
	data, errJson := json.Marshal(params)
	if errJson != nil {
		return errJson
	}

	query = `UPDATE pipeline_build SET args = $1 WHERE id = $2`
	if _, err := db.Exec(query, string(data), pbID); err != nil {
		return err
	}

	// now load all related action build
	pbJobs, errJobs := GetPipelineBuildJobByPipelineBuildID(db, pbID)
	if errJobs != nil {
		return errJobs
	}

	for _, j := range pbJobs {
		j.Parameters = append(j.Parameters, sdk.Parameter{
			Name:  "cds.build." + v.Name,
			Type:  sdk.StringParameter,
			Value: v.Value,
		})

		// Update
		if err := UpdatePipelineBuildJob(db, &j); err != nil {
			return err
		}
	}
	return nil
}

// InsertPipelineBuild insert build informations in database so Scheduler can pick it up
func InsertPipelineBuild(tx gorp.SqlExecutor, project *sdk.Project, p *sdk.Pipeline, applicationData *sdk.Application, applicationPipelineArgs []sdk.Parameter, params []sdk.Parameter, env *sdk.Environment, version int64, trigger sdk.PipelineBuildTrigger) (sdk.PipelineBuild, error) {
	var buildNumber int64
	var pb sdk.PipelineBuild
	var client sdk.RepositoriesManagerClient

	// Load last finished build
	buildNumber, err := GetLastBuildNumber(tx, p.ID, applicationData.ID, env.ID)
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

	if err := LoadPipelineStage(tx, p); err != nil {
		return pb, err
	}

	// Init Action build
	for stageIndex := range p.Stages {
		stage := &p.Stages[stageIndex]
		if stageIndex == 0 {
			stage.Status = sdk.StatusWaiting
		}
	}

	stages, errJSON := json.Marshal(p.Stages)
	if errJSON != nil {
		return pb, errJSON
	}

	err = insertPipelineBuild(tx, string(argsJSON), applicationData.ID, p.ID, &pb, env.ID, string(stages))
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
	history, err := LoadPipelineBuildsByApplicationAndPipeline(tx, pb.Application.ID, pb.Pipeline.ID, pb.Environment.ID, 2, "", branch)
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

	event.PublishPipelineBuild(tx, &pb, previous)
	return pb, nil
}

func insertPipelineBuild(db gorp.SqlExecutor, args string, applicationID, pipelineID int64, pb *sdk.PipelineBuild, envID int64, stages string) error {
	query := `INSERT INTO pipeline_build (pipeline_id, build_number, version, status, args, start, application_id,environment_id, done, manual_trigger, triggered_by, parent_pipeline_build_id, vcs_changes_branch, vcs_changes_hash, vcs_changes_author, scheduled_trigger, stages)
						VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17) RETURNING id`

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
		pb.Trigger.VCSChangesBranch, pb.Trigger.VCSChangesHash, pb.Trigger.VCSChangesAuthor, pb.Trigger.ScheduledTrigger, stages)
	err := statement.Scan(&pb.ID)
	if err != nil {
		return fmt.Errorf("App:%d,Pip:%d,Env:%d> %s", applicationID, pipelineID, envID, err)
	}

	return nil
}

//BuildExists checks if a build already exist
func BuildExists(db gorp.SqlExecutor, appID, pipID, envID int64, trigger *sdk.PipelineBuildTrigger) (bool, error) {
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

// GetBranchHistory  Get last build for all branches
// TODO REFACTOR
func GetBranchHistory(db gorp.SqlExecutor, projectKey, appName string, page, nbPerPage int) ([]sdk.PipelineBuild, error) {
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

// GetDeploymentHistory Get all last deployment
// TODO Refactor
func GetDeploymentHistory(db gorp.SqlExecutor, projectKey, appName string) ([]sdk.PipelineBuild, error) {
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
func GetVersions(db gorp.SqlExecutor, app *sdk.Application, branchName string) ([]int, error) {
	query := `
		SELECT version
		FROM pipeline_build
		WHERE application_id = $1 AND vcs_changes_branch = $2
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

func GetAllLastBuildByApplication(db gorp.SqlExecutor, applicationID int64, branchName string, version int)([]sdk.PipelineBuild, error) {
	whereCondition := `
		WHERE pb.id IN (
			select max(id)
			FROM pipeline_build
			WHERE application_id = $1 %s
			GROUP BY pipeline_id, environment_id
		) AND application_id = $1;
	`
	var rows []sdk.PipelineBuildDbResult
	var errSelect error
	if branchName == "" && version == 0 {
		query := fmt.Sprintf("%s %s", SELECT_PB, fmt.Sprintf(whereCondition, ""))
		_, errSelect = db.Select(&rows, query, applicationID)
	} else if branchName != "" && version == 0 {
		query := fmt.Sprintf("%s %s", SELECT_PB, fmt.Sprintf(whereCondition, " AND vcs_changes_branch = $2"))
		_, errSelect = db.Select(&rows, query, applicationID, branchName)
	} else if branchName == "" && version != 0 {
		query := fmt.Sprintf("%s %s", SELECT_PB, fmt.Sprintf(whereCondition, " AND version = $2"))
		_, errSelect = db.Select(&rows, query, applicationID, version)
	} else {
		query := fmt.Sprintf("%s %s", SELECT_PB, fmt.Sprintf(whereCondition, " AND vcs_changes_branch = $2 AND version = $3"))
		_, errSelect = db.Select(&rows, query, applicationID, branchName, version)
	}

	if errSelect != nil {
		return nil, errSelect
	}

	pbs := []sdk.PipelineBuild{}
	for _, r := range rows {
		pb, errScan := scanPipelineBuild(r)
		if errScan != nil {
			return nil, errScan
		}
		pbs = append(pbs, *pb)
	}
	return pbs, nil
}

// GetBranches from pipeline build and pipeline history for the given application
func GetBranches(db gorp.SqlExecutor, app *sdk.Application) ([]sdk.VCSBranch, error) {
	branches := []sdk.VCSBranch{}
	query := `
		SELECT DISTINCT vcs_changes_branch
		FROM pipeline_build
		WHERE application_id = $1
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

//BuildNumberAndHash represents BuildNumber, Commit Hash and Branch for a Pipeline Build
type BuildNumberAndHash struct {
	BuildNumber int64
	Hash        string
	Branch      string
}

//CurrentAndPreviousPipelineBuildNumberAndHash returns a struct with BuildNumber, Commit Hash and Branch
//for the current pipeline build and the previous one on the same branch.
//Returned pointers may be null if pipeline build are not found
func CurrentAndPreviousPipelineBuildNumberAndHash(db gorp.SqlExecutor, buildNumber, pipelineID, applicationID, environmentID int64) (*BuildNumberAndHash, *BuildNumberAndHash, error) {
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

// StopPipelineBuild fails all currently building actions
func StopPipelineBuild(db gorp.SqlExecutor, pbID int64) error {
	query := `UPDATE action_build SET status = $1, done = now() WHERE pipeline_build_id = $2 AND status IN ( $3, $4 )`
	_, err := db.Exec(query, string(sdk.StatusFail), pbID, string(sdk.StatusBuilding), string(sdk.StatusWaiting))
	if err != nil {
		return err
	}

	// TODO: Add log to inform user

	return nil
}

func RestartPipelineBuild(db gorp.SqlExecutor, pb *sdk.PipelineBuild) error {
	if pb.Status == sdk.StatusSuccess {
		// Remove all pipeline build jobs
		for i := range pb.Stages {
			stage := &pb.Stages[i]
			if i == 0 {
				stage.Status = sdk.StatusWaiting
			}
			// Delete logs
			for _, pbJob := range stage.PipelineBuildJobs {
				if err := DeleteBuildLogs(db, pbJob.ID); err != nil {
					return err
				}

			}
			stage.PipelineBuildJobs = nil
		}
		pb.Start = time.Now()
		pb.Done = time.Time{}

		// Delete artifacts
		arts, errArts := artifact.LoadArtifactsByBuildNumber(db, pb.Pipeline.ID, pb.Application.ID, pb.BuildNumber, pb.Environment.ID)
		if errArts != nil {
			return errArts
		}
		for _, a := range arts {
			if err := artifact.DeleteArtifact(db, a.ID); err != nil {
				return err
			}
		}

		// Delete test results
		if err := DeletePipelineTestResults(db, pb.ID); err != nil {
			return err
		}

	} else {
		for i := range pb.Stages {
			stage := &pb.Stages[i]
			if (stage.Status != sdk.StatusFail) {
				continue
			}
			stage.Status = sdk.StatusWaiting
			// Delete logs
			for _, pbJob := range stage.PipelineBuildJobs {
				if err := DeleteBuildLogs(db, pbJob.ID); err != nil {
					return err
				}

			}
			stage.PipelineBuildJobs = nil
		}
		pb.Done = time.Time{}
	}

	pb.Status = sdk.StatusBuilding

	if err := UpdatePipelineBuildStatusAndStage(db, pb); err != nil {
		return err
	}

	return nil
}


/*
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

// LoadPipelineBuildByID look for a pipeline build by pipelineBuildID
func LoadPipelineBuildByID(tx *sql.Tx, pipelineBuildID int64) (sdk.PipelineBuild, error) {
	var pb sdk.PipelineBuild

	query := fmt.Sprintf(LoadPipelineBuildRequest, "", "pb.id= $1", "")
	row := tx.QueryRow(query, pipelineBuildID)
	if err := scanPbShort(&pb, row); err != nil {
		return pb, err
	}

	queryParams := "SELECT args FROM pipeline_build WHERE id =$1"
	var params sql.NullString

	if err := tx.QueryRow(queryParams, pipelineBuildID).Scan(&params); err != nil {
		return pb, fmt.Errorf("LoadPipelineBuildByID> Cannot load pipeline build parameters for pb.ID:%d: %s", pipelineBuildID, err)
	}
	if params.Valid {
		if err := json.Unmarshal([]byte(params.String), &pb.Parameters); err != nil {
			return pb, fmt.Errorf("LoadPipelineBuildByID> Cannot unmarshal pipeline build parameters: %s", err)
		}
	}
	return pb, nil
}





// LoadUserRecentPipelineBuild retrieves all user accessible pipeline build finished less than a minute ago
func LoadUserRecentPipelineBuild(db gorp.SqlExecutor, userID int64) ([]sdk.PipelineBuild, error) {
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


func scanPbWithStagesAndActions(rows *sql.Rows) ([]sdk.PipelineBuild, error) {
	var pb sdk.PipelineBuild
	var pbs []sdk.PipelineBuild

	var pbI, stageI, abI int
	var pbStatus, pbArgs, pipType, stageName string
	var stageID int64
	var manual, scheduled sql.NullBool
	var trigBy, parentID, abID sql.NullInt64
	var branch, hash, author, actionName, abStatus sql.NullString
	for rows.Next() {
		err := rows.Scan(&pb.Pipeline.ProjectID, &pb.Pipeline.ProjectKey,
			&pb.Application.ID, &pb.Application.Name,
			&pb.Environment.ID, &pb.Environment.Name,
			&pb.ID, &pbStatus, &pb.Version,
			&pb.BuildNumber, &pbArgs, &manual, &scheduled,
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
		if scheduled.Valid {
			pb.Trigger.ScheduledTrigger = scheduled.Bool
		}
		// moar info on automatic trigger
		if manual.Valid && trigBy.Valid && parentID.Valid && branch.Valid && hash.Valid && author.Valid {
			pb.Trigger.TriggeredBy = &sdk.User{ID: trigBy.Int64}
			pb.Trigger.ParentPipelineBuild = &sdk.PipelineBuild{ID: parentID.Int64}
			pb.Trigger.VCSChangesHash = hash.String
			pb.Trigger.VCSChangesAuthor = author.String
			pb.Trigger.ManualTrigger = manual.Bool
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
func InsertBuildVariable(db gorp.SqlExecutor, pbID int64, v sdk.Variable) error {

	tx, errDb := db.Begin()
	if errDb != nil {
		return errDb
	}
	defer tx.Rollback()

	// Load args from pipeline build and lock it
	query := `SELECT args FROM pipeline_build WHERE id = $1 FOR UPDATE`
	var argsJSON string
	if err := tx.QueryRow(query, pbID).Scan(&argsJSON); err != nil {
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
	data, errJson := json.Marshal(params)
	if errJson != nil {
		return errJson
	}

	query = `UPDATE pipeline_build SET args = $1 WHERE id = $2`
	if _, err := tx.Exec(query, string(data), pbID); err != nil {
		return err
	}

	// now load all related action build
	query = `SELECT id, args FROM action_build WHERE pipeline_build_id = $1 FOR UPDATE`
	rows, errQuery := tx.Query(query, pbID)
	if errQuery != nil {
		return errQuery
	}
	defer rows.Close()
	var abs []sdk.ActionBuild
	for rows.Next() {
		var ab sdk.ActionBuild
		if err := rows.Scan(&ab.ID, &argsJSON); err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(argsJSON), &ab.Args); err != nil {
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

		data, errMarshal := json.Marshal(ab.Args)
		if errMarshal != nil {
			return errMarshal
		}

		if _, err := tx.Exec(query, string(data), ab.ID); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// LoadBuildingPipelines retrieves pipelines in database having a build running
func LoadBuildingPipelines(db gorp.SqlExecutor, args ...FuncArg) ([]sdk.PipelineBuild, error) {
	query := `
SELECT DISTINCT ON (project.projectkey, application.name, pb.application_id, pb.pipeline_id, pb.environment_id, pb.vcs_changes_branch)
	pb.pipeline_id, pb.application_id, pb.environment_id, pb.id, project.id as project_id,
	environment.name as envName, application.name as appName, pipeline.name as pipName, project.projectkey,
	pipeline.type,
	pb.build_number, pb.version, pb.status, pb.args,
	pb.start, pb.done,
	pb.manual_trigger, pb.scheduled_trigger, pb.triggered_by, pb.parent_pipeline_build_id, pb.vcs_changes_branch, pb.vcs_changes_hash, pb.vcs_changes_author,
	"user".username, pipTriggerFrom.name as pipTriggerFrom, pbTriggerFrom.version as versionTriggerFrom,
	pb.stages
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

		var status, typePipeline, argsJSON, stages string
		var manual, scheduled sql.NullBool
		var trigBy, pPbID, version sql.NullInt64
		var branch, hash, author, fromUser, fromPipeline sql.NullString

		err := rows.Scan(&p.Pipeline.ID, &p.Application.ID, &p.Environment.ID, &p.ID, &p.Pipeline.ProjectID,
			&p.Environment.Name, &p.Application.Name, &p.Pipeline.Name, &p.Pipeline.ProjectKey,
			&typePipeline,
			&p.BuildNumber, &p.Version, &status, &argsJSON,
			&p.Start, &p.Done,
			&manual, &scheduled, &trigBy, &pPbID, &branch, &hash, &author,
			&fromUser, &fromPipeline, &version, &stages)
		if err != nil {
			log.Warning("LoadBuildingPipelines> Error while loading build information: %s", err)
			return nil, err
		}
		p.Status = sdk.StatusFromString(status)
		p.Pipeline.Type = sdk.PipelineTypeFromString(typePipeline)
		p.Application.ProjectKey = p.Pipeline.ProjectKey
		loadPbTrigger(&p, manual, scheduled, pPbID, branch, hash, author, fromUser, fromPipeline, version)

		if err := json.Unmarshal([]byte(stages), &p.Stages); err != nil {
			return nil, err
		}

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
func UpdatePipelineBuildStatus(db gorp.SqlExecutor, pb sdk.PipelineBuild, status sdk.Status) error {
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
	event.PublishPipelineBuild(db, &pb, previous)
	return nil
}











/



// LoadPipelineBuildChildren load triggered pipeline from given build
func LoadPipelineBuildChildren(db gorp.SqlExecutor, pipelineID int64, applicationID int64, buildNumber int64, environmentID int64) ([]sdk.PipelineBuild, error) {
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








// DeletePipelineBuildArtifact Delete artifact for the current build
func DeletePipelineBuildArtifact(db gorp.SqlExecutor, pipelineBuildID int64) error {
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
func DeletePipelineBuild(db gorp.SqlExecutor, pipelineBuildID int64) error {

	if err := DeletePipelineBuildArtifact(db, pipelineBuildID); err != nil {
		return err
	}

	// Then delete pipeline build data
	if err := DeleteBuild(db, pipelineBuildID); err != nil {
		return err
	}

	return nil
}





// GetDeploymentHistory Get all last deployment
func GetDeploymentHistory(db gorp.SqlExecutor, projectKey, appName string) ([]sdk.PipelineBuild, error) {
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



// GetBranchHistory  Get last build for all branches
func GetBranchHistory(db gorp.SqlExecutor, projectKey, appName string, page, nbPerPage int) ([]sdk.PipelineBuild, error) {
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


*/
