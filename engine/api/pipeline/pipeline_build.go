package pipeline

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/artifact"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/stats"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
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
	          WHERE application_id = $1 AND pipeline_id = $2 AND environment_id = $3 AND build_number = $4`
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
		WHERE pb.status = $1 OR (pb.status != $1 AND pb.done > NOW() -  INTERVAL '1 minutes')
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
	if err := db.SelectOne(&row, query, applicationID, pipelineID, environmentID, buildNumber); err != nil {
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
		query = fmt.Sprintf(query, "LIMIT $4")
		_, errQuery = db.Select(&rows, query, applicationID, pipelineID, environmentID, limit)
	} else if status != "" && branchName == "" {
		query = fmt.Sprintf(query, " AND pb.status = $5 LIMIT $4")
		_, errQuery = db.Select(&rows, query, applicationID, pipelineID, environmentID, limit, status)
	} else if status == "" && branchName != "" {
		query = fmt.Sprintf(query, " AND pb.vcs_changes_branch = $5  LIMIT $4")
		_, errQuery = db.Select(&rows, query, applicationID, pipelineID, environmentID, limit, branchName)
	} else {
		query = fmt.Sprintf(query, " AND pb.status = $5 AND pb.vcs_changes_branch = $6  LIMIT $4")
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
	if err := db.SelectOne(&row, query, id); err != nil {
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
		Trigger: sdk.PipelineBuildTrigger{
			ManualTrigger: pbResult.ManualTrigger,
		},
	}

	if pbResult.Done.Valid {
		pb.Done = pbResult.Done.Time
	}
	if pbResult.TriggeredBy.Valid && pbResult.Username.Valid {
		pb.Trigger.TriggeredBy = &sdk.User{
			ID:       pbResult.TriggeredBy.Int64,
			Username: pbResult.Username.String,
		}
	}
	if pbResult.VCSChangesAuthor.Valid {
		pb.Trigger.VCSChangesAuthor = pbResult.VCSChangesAuthor.String
	}
	if pbResult.VCSChangesBranch.Valid {
		pb.Trigger.VCSChangesBranch = pbResult.VCSChangesBranch.String
	}
	if pbResult.VCSChangesHash.Valid {
		pb.Trigger.VCSChangesHash = pbResult.VCSChangesHash.String
	}

	if err := json.Unmarshal([]byte(pbResult.Args), &pb.Parameters); err != nil {
		log.Warning("scanPipelineBuild> Unable to Unmarshal parameter %s", pbResult.Args)
		return nil, err
	}
	if err := json.Unmarshal([]byte(pbResult.Stages), &pb.Stages); err != nil {
		log.Warning("scanPipelineBuild> Unable to Unmarshal stages %s", pbResult.Stages)
		return nil, err
	}

	return &pb, nil
}

// UpdatePipelineBuildStatusAndStage Update pipeline build status + stage
func UpdatePipelineBuildStatusAndStage(db gorp.SqlExecutor, pb *sdk.PipelineBuild, newStatus sdk.Status) error {
	stagesB, errStage := json.Marshal(pb.Stages)
	if errStage != nil {
		return errStage
	}
	query := `UPDATE pipeline_build set status = $1, stages = $2 WHERE id = $3`
	_, err := db.Exec(query, newStatus.String(), string(stagesB), pb.ID)
	if err != nil {
		return err
	}
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
	history, err := LoadPipelineBuildsByApplicationAndPipeline(db, pb.Application.ID, pb.Pipeline.ID, pb.Environment.ID, 2, "", branch)
	if err != nil {
		log.Critical("UpdatePipelineBuildStatusAndStage> error while loading previous pipeline build")
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

	// Load repositorie manager if necessary
	if pb.Application.RepositoriesManager == nil || pb.Application.RepositoryFullname == "" {
		rfn, rm, errl := repositoriesmanager.LoadFromApplicationByID(db, pb.Application.ID)
		if errl != nil {
			log.Critical("UpdatePipelineBuildStatus> error while loading repoManger for appID %s err:%s", pb.Application.ID, errl)
		}
		pb.Application.RepositoryFullname = rfn
		pb.Application.RepositoriesManager = rm
	}

	if pb.Status != newStatus {

		query := `
			SELECT projectkey FROM project
			JOIN application ON application.project_id = project.id
			WHERE application.id = $1
		`
		var key string
		if err := db.QueryRow(query, pb.Application.ID).Scan(&key); err != nil {
			log.Critical("UpdatePipelineBuildStatus> error while loading project key from appID %d err %s", pb.Application.ID, err)
		}
		pb.Pipeline.ProjectKey = key

		event.PublishPipelineBuild(db, pb, previous)
	}

	pb.Status = newStatus
	return nil
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

	lastBuildNumber, errBN := GetLastBuildNumber(tx, pipID, appID, envID)
	if errBN != nil {
		return 0, errBN
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return lastBuildNumber, nil
}

// GetLastBuildNumber Get the last build number
func GetLastBuildNumber(db gorp.SqlExecutor, pipID, appID, envID int64) (int64, error) {
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
func InsertPipelineBuild(tx gorp.SqlExecutor, project *sdk.Project, p *sdk.Pipeline, app *sdk.Application, applicationPipelineArgs []sdk.Parameter, params []sdk.Parameter, env *sdk.Environment, version int64, trigger sdk.PipelineBuildTrigger) (*sdk.PipelineBuild, error) {
	var buildNumber int64
	var pb sdk.PipelineBuild
	var client sdk.RepositoriesManagerClient

	//Initialize client for repository manager
	if app.RepositoriesManager != nil && app.RepositoryFullname != "" {
		client, _ = repositoriesmanager.AuthorizedClient(tx, project.Key, app.RepositoriesManager.Name)
	}

	// Load last finished build
	buildNumber, err := GetLastBuildNumber(tx, p.ID, app.ID, env.ID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
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
		(app.ID != trigger.ParentPipelineBuild.Application.ID && p.Type == sdk.BuildPipeline) {
		log.Debug("InsertPipelineBuild: Set version to buildnumber (provided: %d), has parent (%t), appID (%d)", version, trigger.ParentPipelineBuild != nil, app.ID)
		pb.Version = pb.BuildNumber
	}
	sdk.AddParameter(&params, "cds.pipeline", sdk.StringParameter, p.Name)
	sdk.AddParameter(&params, "cds.project", sdk.StringParameter, p.ProjectKey)
	sdk.AddParameter(&params, "cds.application", sdk.StringParameter, app.Name)
	sdk.AddParameter(&params, "cds.environment", sdk.StringParameter, env.Name)
	sdk.AddParameter(&params, "cds.buildNumber", sdk.StringParameter, strconv.FormatInt(pb.BuildNumber, 10))
	sdk.AddParameter(&params, "cds.version", sdk.StringParameter, strconv.FormatInt(pb.Version, 10))

	if client != nil {
		repo, err := client.RepoByFullname(app.RepositoryFullname)
		if err != nil {
			log.Warning("InsertPipelineBuild> Unable to get repository %s from %s : %s", app.RepositoriesManager.Name, app.RepositoryFullname, err)
			return nil, err
		}
		sdk.AddParameter(&params, "git.url", sdk.StringParameter, repo.SSHCloneURL)
		sdk.AddParameter(&params, "git.http_url", sdk.StringParameter, repo.HTTPCloneURL)
	}

	if pb.Trigger.TriggeredBy != nil {
		//Load user information to store them as args
		sdk.AddParameter(&params, "cds.triggered_by.username", sdk.StringParameter, pb.Trigger.TriggeredBy.Username)
		sdk.AddParameter(&params, "cds.triggered_by.fullname", sdk.StringParameter, pb.Trigger.TriggeredBy.Fullname)
		sdk.AddParameter(&params, "cds.triggered_by.email", sdk.StringParameter, pb.Trigger.TriggeredBy.Email)
	}

	//Set git.Branch and git.Hash
	if pb.Trigger.VCSChangesBranch != "" {
		sdk.AddParameter(&params, "git.branch", sdk.StringParameter, pb.Trigger.VCSChangesBranch)
		sdk.AddParameter(&params, "git.hash", sdk.StringParameter, pb.Trigger.VCSChangesHash)
	} else {
		//We consider default branch is master
		defautlBranch := "master"
		lastGitHash := map[string]string{}
		if client != nil {
			branches, _ := client.Branches(app.RepositoryFullname)
			for _, b := range branches {
				//If application is linked to a repository manager, we try to found de default branch
				if b.Default {
					defautlBranch = b.DisplayID
				}
				//And we store LatestCommit for each branches
				lastGitHash[b.DisplayID] = b.LatestCommit
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
			sdk.AddParameter(&params, "git.branch", sdk.StringParameter, defautlBranch)
			pb.Trigger.VCSChangesBranch = defautlBranch

			//And we try to put the lastestCommit for this branch
			if lastGitHash[defautlBranch] != "" {
				sdk.AddParameter(&params, "git.hash", sdk.StringParameter, lastGitHash[defautlBranch])
				pb.Trigger.VCSChangesHash = lastGitHash[defautlBranch]
			}
		} else {
			//If git.branch was found but git.hash wasn't found in pipeline parameters
			//we try to found the LatestCommit
			if !hashFound && lastGitHash[pb.Trigger.VCSChangesBranch] != "" {
				sdk.AddParameter(&params, "git.hash", sdk.StringParameter, lastGitHash[pb.Trigger.VCSChangesBranch])
				pb.Trigger.VCSChangesHash = lastGitHash[pb.Trigger.VCSChangesBranch]
			}
		}
	}

	//Retreive commit information
	if client != nil && pb.Trigger.VCSChangesHash != "" {
		commit, err := client.Commit(app.RepositoryFullname, pb.Trigger.VCSChangesHash)
		if err != nil {
			log.Warning("InsertPipelineBuild> Cannot get commit: %s\n", err)
			return nil, err
		}
		sdk.AddParameter(&params, "git.author", sdk.StringParameter, commit.Author.Name)
		sdk.AddParameter(&params, "git.message", sdk.StringParameter, commit.Message)
	}

	// Process Pipeline Argument
	mapVar, err := ProcessPipelineBuildVariables(p.Parameter, applicationPipelineArgs, params)
	if err != nil {
		log.Warning("InsertPipelineBuild> Cannot process args: %s\n", err)
		return nil, err
	}

	// sdk.Build should have sdk.Variable instead of []string
	var argsFinal []sdk.Parameter
	for _, v := range mapVar {
		argsFinal = append(argsFinal, v)
	}

	argsJSON, err := json.Marshal(argsFinal)
	if err != nil {
		log.Warning("InsertPipelineBuild> Cannot marshal build parameters: %s\n", err)
		return nil, err
	}

	if err := LoadPipelineStage(tx, p); err != nil {
		return nil, err
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
		return nil, errJSON
	}

	err = insertPipelineBuild(tx, string(argsJSON), app.ID, p.ID, &pb, env.ID, string(stages))
	if err != nil {
		log.Warning("InsertPipelineBuild> Cannot insert pipeline build: %s\n", err)
		return nil, err
	}

	pb.Status = sdk.StatusBuilding
	pb.Pipeline = *p
	pb.Parameters = params
	pb.Application = *app
	pb.Environment = *env

	// Update stats
	stats.PipelineEvent(tx, p.Type, project.ID, app.ID)

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
		log.Critical("InsertPipelineBuild> error while loading previous pipeline build: %s", err)
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
	return &pb, nil
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

func GetAllLastBuildByApplication(db gorp.SqlExecutor, applicationID int64, branchName string, version int) ([]sdk.PipelineBuild, error) {
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

				) AS current_pipeline
			LEFT OUTER JOIN (
					SELECT    id, pipeline_id, build_number, vcs_changes_branch, vcs_changes_hash
					FROM      pipeline_build
					WHERE     build_number < $1
					AND				pipeline_id = $2
					AND				application_id = $3
					AND 			environment_id = $4

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

	if err := StopBuildingPipelineBuildJob(db, pbID); err != nil {
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
			if stage.Status != sdk.StatusFail {
				continue
			}
			stage.Status = sdk.StatusWaiting
			// Delete logs
			for _, pbJob := range stage.PipelineBuildJobs {
				if pbJob.Status == sdk.StatusFail.String() {
					if err := DeleteBuildLogs(db, pbJob.ID); err != nil {
						return err
					}
				}
			}
			stage.PipelineBuildJobs = nil
		}
		pb.Done = time.Time{}
	}

	if err := UpdatePipelineBuildStatusAndStage(db, pb, sdk.StatusBuilding); err != nil {
		return err
	}

	return nil
}
