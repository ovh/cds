package build

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

var (
	// ErrAlreadyTaken Action already taken by a worker
	ErrAlreadyTaken = fmt.Errorf("cds: action already taken")
)

// LoadGroupWaitingQueue loads action build in queue accessbible to given group
func LoadGroupWaitingQueue(db *sql.DB, groupID int64) ([]sdk.ActionBuild, error) {
	var queue []sdk.ActionBuild

	query := `
			 SELECT action_build.id,
			 action_build.pipeline_action_id,
			 action.id,
			 action.name,
			 action_build.args,
			 action_build.status, action_build.pipeline_build_id,
			 pipeline_build.pipeline_id,
			 pipeline_build.build_number
		  FROM action_build
		  JOIN pipeline_build ON pipeline_build.id = action_build.pipeline_build_id
		  JOIN pipeline_action ON pipeline_action.id = action_build.pipeline_action_id
		  JOIN action ON action.id = pipeline_action.action_id
		  JOIN pipeline ON pipeline.id = pipeline_build.pipeline_id
			JOIN pipeline_group ON pipeline_group.pipeline_id = pipeline.id
			WHERE action_build.status = $1
			AND ( 
					(
						pipeline_group.group_id = $2
						AND
						pipeline_group.role > 4
					)
				OR
				$2 = (SELECT id FROM "group" WHERE name = $3)
			)
			ORDER BY pipeline_build.id,action.name,action_build.pipeline_action_id
			LIMIT 100
			`

	rows, err := db.Query(query, sdk.StatusWaiting.String(), groupID, group.SharedInfraGroup)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		b, err := loadQueue(db, rows)
		if err != nil {
			return nil, err
		}
		queue = append(queue, b)
	}

	return queue, nil

}

// LoadUserWaitingQueue loads action build in queue where user has access
func LoadUserWaitingQueue(db *sql.DB, u *sdk.User) ([]sdk.ActionBuild, error) {
	var queue []sdk.ActionBuild

	// If related user is admin, returns everything
	if u.Admin {
		return LoadWaitingQueue(db)
	}

	// If user is in no group, don't bother
	if len(u.Groups) == 0 {
		log.Warning("LoadUserWaitingQueue> User %s is in no groups, let it go\n", u.Username)
		return queue, nil
	}

	query := `
			 SELECT action_build.id,
			 action_build.pipeline_action_id,
			 action.id,
			 action.name,
			 action_build.args,
			 action_build.status, action_build.pipeline_build_id,
			 pipeline_build.pipeline_id,
			 pipeline_build.build_number
		  FROM action_build
		  JOIN pipeline_build ON pipeline_build.id = action_build.pipeline_build_id
		  JOIN pipeline_action ON pipeline_action.id = action_build.pipeline_action_id
		  JOIN action ON action.id = pipeline_action.action_id
		  JOIN pipeline ON pipeline.id = pipeline_build.pipeline_id
			JOIN pipeline_group ON pipeline_group.pipeline_id = pipeline.id
			JOIN group_user ON group_user.group_id = pipeline_group.group_id
			WHERE action_build.status = $1 AND group_user.user_id = $2
		  ORDER BY pipeline_build.id,action.name,action_build.pipeline_action_id
			LIMIT 100
			`

	rows, err := db.Query(query, sdk.StatusWaiting.String(), u.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		b, err := loadQueue(db, rows)
		if err != nil {
			return nil, err
		}
		queue = append(queue, b)
	}

	return queue, nil
}

func loadQueue(db *sql.DB, s database.Scanner) (sdk.ActionBuild, error) {
	var b sdk.ActionBuild
	var argsJSON, actionName, sStatus string
	var actionID int64
	err := s.Scan(&b.ID, &b.PipelineActionID, &actionID, &actionName, &argsJSON, &sStatus, &b.PipelineBuildID, &b.PipelineID, &b.BuildNumber)
	b.Status = sdk.StatusFromString(sStatus)
	if err != nil {
		return b, err
	}

	err = json.Unmarshal([]byte(argsJSON), &b.Args)
	if err != nil {
		var oa []string
		err = json.Unmarshal([]byte(argsJSON), &oa)
		if err != nil {
			return b, err
		}
		for _, op := range oa {
			t := strings.SplitN(op, "=", 2)
			p := sdk.Parameter{
				Name:  t[0],
				Type:  sdk.StringParameter,
				Value: t[1],
			}
			b.Args = append(b.Args, p)
		}
	}

	// load action requirements
	a, err := action.LoadActionByID(db, actionID)
	if err != nil {
		return b, err
	}
	b.Requirements = a.Requirements
	return b, nil
}

// TakeActionBuild Take an action build for update
func TakeActionBuild(db *sql.DB, buildID string, worker *sdk.Worker) (sdk.ActionBuild, error) {
	var b sdk.ActionBuild
	var argsJSON string

	tx, err := db.Begin()
	if err != nil {
		return b, err
	}
	defer tx.Rollback()

	query := `SELECT action_build.id,
			 action_build.pipeline_action_id,
			 action_build.args,
			 action_build.status,
			 action_build.pipeline_build_id,
			 pipeline_build.build_number
	     FROM action_build
	     JOIN pipeline_build ON pipeline_build.id = action_build.pipeline_build_id
			 WHERE action_build.id = $1 FOR UPDATE`

	var sStatus string
	err = tx.QueryRow(query, buildID).Scan(&b.ID, &b.PipelineActionID, &argsJSON, &sStatus, &b.PipelineBuildID, &b.BuildNumber)
	b.Status = sdk.StatusFromString(sStatus)
	if err != nil {
		return b, err
	}

	err = json.Unmarshal([]byte(argsJSON), &b.Args)
	if err != nil {
		return b, err
	}

	if b.Status != sdk.StatusWaiting {
		return b, ErrAlreadyTaken
	}

	query = ` update action_build set worker_model_name = worker_model.name from worker_model where worker_model.id=$2 and action_build.id = $1`
	if _, err := tx.Exec(query, b.ID, worker.Model); err != nil {
		log.Warning("Cannot update model on action_build : %s", err)
	}

	// Update queue status to "building"
	if err := UpdateActionBuildStatus(tx, &b, sdk.StatusBuilding); err != nil {
		return b, err
	}

	return b, tx.Commit()
}

// DeleteActionBuild Delete Action Build
func DeleteActionBuild(db database.QueryExecuter, pipelineActionIDs []int64) error {
	for _, id := range pipelineActionIDs {
		actionBuildIDs, err := selectAllActionBuildIDsByPipelineAction(db, id)
		if err != nil {
			return err
		}

		for _, abID := range actionBuildIDs {
			err := DeleteBuildLogs(db, abID)
			if err != nil {
				return err
			}
		}

		queryDelete := `DELETE FROM action_build WHERE pipeline_action_id = $1`
		_, err = db.Exec(queryDelete, id)
		if err != nil {
			log.Warning("DeleteActionBuild> Cannot remove action builds for PipelineAction %d\n", id)
			return err
		}
	}
	return nil
}

func selectAllActionBuildIDsByPipelineAction(db database.QueryExecuter, pipelineActionID int64) ([]int64, error) {
	var actionBuildIDs []int64
	query := `SELECT id FROM "action_build"
	 		  WHERE pipeline_action_id = $1`
	rows, err := db.Query(query, pipelineActionID)
	if err != nil {
		return actionBuildIDs, err
	}
	defer rows.Close()

	for rows.Next() {
		var abID int64
		err = rows.Scan(&abID)
		if err != nil {
			return actionBuildIDs, err
		}
		actionBuildIDs = append(actionBuildIDs, abID)
	}
	return actionBuildIDs, nil
}

// DeleteBuild Delete a build
func DeleteBuild(db database.QueryExecuter, buildID int64) error {

	queryDeleteBuildLog := `DELETE FROM build_log WHERE action_build_id IN
		(SELECT id from action_build WHERE pipeline_build_id = $1)`
	_, err := db.Exec(queryDeleteBuildLog, buildID)
	if err != nil {
		log.Warning("DeleteBuild> Cannot delete build log: %s\n", err)
		return err
	}

	// if we are deleting a building pipeline, delete workers as well
	query := `UPDATE worker SET status = $1, action_build_id = NULL WHERE action_build_id IN
		(SELECT id FROM action_build WHERE pipeline_build_id = $2)`
	_, err = db.Exec(query, string(sdk.StatusDisabled), buildID)
	if err != nil {
		log.Warning("DeleteBuild> Cannot delete building workers: %s\n", err)
		return err
	}

	queryDeleteActionBuild := `DELETE FROM action_build WHERE pipeline_build_id=$1`
	_, err = db.Exec(queryDeleteActionBuild, buildID)
	if err != nil {
		log.Warning("DeleteBuild> Cannot delete action build: %s", err)
		return err
	}

	// delete test results
	err = DeleteTestResults(db, buildID)
	if err != nil {
		return err
	}

	// delete pipeline build
	queryDeletePipelineBuild := `DELETE FROM pipeline_build WHERE id=$1`
	_, err = db.Exec(queryDeletePipelineBuild, buildID)
	if err != nil {
		log.Warning("DeleteBuild> Cannot delete pipeline build: %s", err)
		return err
	}
	return nil
}
