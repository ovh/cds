package pipeline

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

var (
	// ErrAlreadyTaken Action already taken by a worker
	ErrAlreadyTaken = fmt.Errorf("cds: action already taken")
)

// DeletePipelineBuildJob Delete all pipeline build job for the current pipeline build
func DeletePipelineBuildJob(db gorp.SqlExecutor, pipelineBuildID int64) error {
	query := "DELETE FROM pipeline_build_job WHERE pipeline_build_id = $1"
	_, err := db.Exec(query, pipelineBuildID)
	return err
}

// InsertPipelineBuildJob Insert a new job in the queue
func InsertPipelineBuildJob(db gorp.SqlExecutor, pbJob *sdk.PipelineBuildJob) error {
	dbmodel := PipelineBuildJob(*pbJob)
	dbmodel.JobJSON = []byte("[]")
	dbmodel.ParametersJSON = []byte("[]")
	if err := db.Insert(&dbmodel); err != nil {
		return err
	}
	*pbJob = sdk.PipelineBuildJob(dbmodel)
	return nil
}

// GetPipelineBuildJobByPipelineBuildID Get all pipeline build job for the given pipeline build
func GetPipelineBuildJobByPipelineBuildID(db gorp.SqlExecutor, pbID int64) ([]sdk.PipelineBuildJob, error) {
	var pbJobsGorp []PipelineBuildJob
	query := `
		SELECT *
		FROM pipeline_build_job
		WHERE pipeline_build_id = $1
	`
	if _, err := db.Select(&pbJobsGorp, query, pbID); err != nil {
		return nil, err
	}

	var pbJobs []sdk.PipelineBuildJob
	for i := range pbJobsGorp {
		if err := pbJobsGorp[i].PostGet(db); err != nil {
			return nil, err
		}
		pbJobs = append(pbJobs, sdk.PipelineBuildJob(pbJobsGorp[i]))
	}
	return pbJobs, nil
}

// GetWaitingPipelineBuildJob Get waiting pipeline build job
func GetWaitingPipelineBuildJob(db gorp.SqlExecutor) ([]sdk.PipelineBuildJob, error) {
	var pbJobsGorp []PipelineBuildJob
	query := `
		SELECT *
		FROM pipeline_build_job
		WHERE status = $1
	`
	if _, err := db.Select(&pbJobsGorp, query, sdk.StatusWaiting.String()); err != nil {
		return nil, err
	}
	var pbJobs []sdk.PipelineBuildJob
	for _, j := range pbJobsGorp {
		if err := j.PostGet(db); err != nil {
			return nil, err
		}
		pbJobs = append(pbJobs, sdk.PipelineBuildJob(j))
	}
	return pbJobs, nil
}

// GetWaitingPipelineBuildJobForGroup Get waiting pipeline build job for the given group
func GetWaitingPipelineBuildJobForGroup(db gorp.SqlExecutor, groupID, sharedInfraGroupID int64) ([]sdk.PipelineBuildJob, error) {
	var pbJobsGorp []PipelineBuildJob
	query := `
		SELECT distinct pipeline_build_job.*
		FROM pipeline_build_job
		JOIN pipeline_build ON pipeline_build.id = pipeline_build_job.pipeline_build_id
		JOIN pipeline_group ON pipeline_group.pipeline_id = pipeline_build.pipeline_id
		WHERE pipeline_build_job.status = $1
		AND (
			pipeline_group.group_id = $2
			OR
			$3 = $2
		)`

	if _, err := db.Select(&pbJobsGorp, query, sdk.StatusWaiting.String(), groupID, sharedInfraGroupID); err != nil {
		return nil, err
	}
	var pbJobs []sdk.PipelineBuildJob
	for _, j := range pbJobsGorp {
		if err := j.PostGet(db); err != nil {
			return nil, err
		}
		pbJobs = append(pbJobs, sdk.PipelineBuildJob(j))
	}
	return pbJobs, nil
}

// GetPipelineBuildJob Get pipeline build job
func GetPipelineBuildJob(db gorp.SqlExecutor, id int64) (*sdk.PipelineBuildJob, error) {
	var pbJobGorp PipelineBuildJob
	if err := db.SelectOne(&pbJobGorp, `
		SELECT *
		FROM pipeline_build_job
		WHERE id = $1
	`, id); err != nil {
		return nil, err
	}
	pbJob := sdk.PipelineBuildJob(pbJobGorp)
	return &pbJob, nil
}

// LoadWaitingQueue Load Waiting pipeline_build_job
func LoadWaitingQueue(db gorp.SqlExecutor) ([]sdk.PipelineBuildJob, error) {
	var pbJobsGorp []PipelineBuildJob
	if _, err := db.Select(&pbJobsGorp, `
		SELECT distinct pipeline_build_job.* FROM pipeline_build_job
		WHERE status = $1 ORDER BY pipeline_build_id ASC, pipeline_build_job.id ASC
	`, sdk.StatusWaiting.String()); err != nil {
		return nil, err
	}
	var pbJobs []sdk.PipelineBuildJob
	for _, j := range pbJobsGorp {
		if err := j.PostGet(db); err != nil {
			return nil, err
		}
		pbJobs = append(pbJobs, sdk.PipelineBuildJob(j))
	}
	return pbJobs, nil
}

// LoadGroupWaitingQueue loads pipeline_build_job in queue accessbible to given group
func LoadGroupWaitingQueue(db gorp.SqlExecutor, groupID int64) ([]sdk.PipelineBuildJob, error) {
	var pbJobsGorp []PipelineBuildJob
	if _, err := db.Select(&pbJobsGorp, `
		SELECT distinct pipeline_build_job.* FROM pipeline_build_job
		JOIN pipeline_build ON pipeline_build.id = pipeline_build_job.pipeline_build_id
		JOIN pipeline_group ON pipeline_group.pipeline_id = pipeline_build.pipeline_id
		WHERE pipeline_build_job.status = $1 AND
		(
			(
				pipeline_group.group_id = $2
				AND
				pipeline_group.role > 4
			)
			OR $2 =  $3

		)
		 ORDER BY pipeline_build_job.pipeline_build_id ASC, pipeline_build_job.id ASC
	`, sdk.StatusWaiting.String(), groupID, group.SharedInfraGroup.ID); err != nil {
		return nil, err
	}
	var pbJobs []sdk.PipelineBuildJob
	for _, j := range pbJobsGorp {
		if err := j.PostGet(db); err != nil {
			return nil, err
		}
		pbJobs = append(pbJobs, sdk.PipelineBuildJob(j))
	}
	return pbJobs, nil
}

// LoadUserWaitingQueue loads action build in queue where user has access
func LoadUserWaitingQueue(db gorp.SqlExecutor, u *sdk.User) ([]sdk.PipelineBuildJob, error) {
	var pbJobsGorp []PipelineBuildJob

	// If related user is admin, returns everything
	if u.Admin {
		return LoadWaitingQueue(db)
	}

	// If user is in no group, don't bother
	if len(u.Groups) == 0 {
		log.Warning("LoadUserWaitingQueue> User %s is in no groups, let it go\n", u.Username)
		return nil, nil
	}

	if _, err := db.Select(&pbJobsGorp, `
		SELECT distinct pipeline_build_job.* FROM pipeline_build_job
		JOIN pipeline_build ON pipeline_build.id = pipeline_build_job.pipeline_build_id
		JOIN pipeline_group ON pipeline_group.pipeline_id = pipeline_build.pipeline_id
		JOIN group_user ON group_user.group_id = pipeline_group.group_id
		WHERE pipeline_build_job.status = $1 AND group_user.user_id = $2
		ORDER BY pipeline_build_job.pipeline_build_id ASC, pipeline_build_job.id ASC
	`, sdk.StatusWaiting.String(), u.ID); err != nil {
		return nil, err
	}
	var pbJobs []sdk.PipelineBuildJob
	for _, j := range pbJobsGorp {
		if err := j.PostGet(db); err != nil {
			return nil, err
		}
		pbJobs = append(pbJobs, sdk.PipelineBuildJob(j))
	}
	return pbJobs, nil
}

// TakeActionBuild Take an action build for update
func TakeActionBuild(db gorp.SqlExecutor, pbJobID int64, model string, workerName string) (*sdk.PipelineBuildJob, error) {
	var pbJobGorp PipelineBuildJob
	if err := db.SelectOne(&pbJobGorp, `
		SELECT *
		FROM pipeline_build_job
		WHERE id = $1 FOR UPDATE
	`, pbJobID); err != nil {
		return nil, err
	}

	if pbJobGorp.Status != sdk.StatusWaiting.String() {
		return nil, ErrAlreadyTaken
	}

	pbJobGorp.Model = model
	pbJobGorp.Job.WorkerName = workerName
	var errMarshal error
	pbJobGorp.JobJSON, errMarshal = json.Marshal(pbJobGorp.Job)
	if errMarshal != nil {
		return nil, errMarshal
	}
	pbJobGorp.Start = time.Now()
	pbJobGorp.Status = sdk.StatusBuilding.String()
	if _, err := db.Update(&pbJobGorp); err != nil {
		log.Warning("Cannot update model on pipeline build job : %s", err)
		return nil, err
	}

	pbJob := sdk.PipelineBuildJob(pbJobGorp)
	return &pbJob, nil
}

// RestartPipelineBuildJob destroy pipeline build job data and queue it up again
func RestartPipelineBuildJob(db gorp.SqlExecutor, pbJobID int64) error {
	var pbJobGorp PipelineBuildJob
	if err := db.SelectOne(&pbJobGorp, `
		SELECT *
		FROM pipeline_build_job
		WHERE id = $1 FOR UPDATE
	`, pbJobID); err != nil {
		return fmt.Errorf("RestartPipelineBuildJob> Cannot get pipeline build job %d: %s", pbJobID, err)
	}

	// Delete previous build logs
	if err := DeleteBuildLogs(db, pbJobID); err != nil {
		return err
	}

	pbJobGorp.Status = sdk.StatusWaiting.String()
	pbJob := sdk.PipelineBuildJob(pbJobGorp)
	if err := UpdatePipelineBuildJob(db, &pbJob); err != nil {
		return err
	}
	return nil
}

// StopBuildingPipelineBuildJob Stop running pipeline build job
func StopBuildingPipelineBuildJob(db gorp.SqlExecutor, pbID int64) error {
	query := `UPDATE pipeline_build_job SET status = $1, done = now() WHERE pipeline_build_id = $2 AND status IN ( $3, $4 )`
	_, err := db.Exec(query, string(sdk.StatusFail), pbID, string(sdk.StatusBuilding), string(sdk.StatusWaiting))
	if err != nil {
		return err
	}
	return nil
}

// UpdatePipelineBuildJob Update pipeline build job
func UpdatePipelineBuildJob(db gorp.SqlExecutor, pbJob *sdk.PipelineBuildJob) error {
	// Update pipeline build job
	pbJobGorp := PipelineBuildJob(*pbJob)
	_, errUpdate := db.Update(&pbJobGorp)
	return errUpdate
}

// UpdatePipelineBuildJobStatus Update status of an pipeline_build_job
func UpdatePipelineBuildJobStatus(db gorp.SqlExecutor, pbJob *sdk.PipelineBuildJob, status sdk.Status) error {
	var query string
	query = `SELECT status FROM pipeline_build_job WHERE id = $1 FOR UPDATE`
	var currentStatus string
	if err := db.QueryRow(query, pbJob.ID).Scan(&currentStatus); err != nil {
		log.Warning("UpdatePipelineBuildJobStatus> Cannot lock pipeline build job %s: %s", pbJob.ID, err)
		return err
	}

	switch status {
	case sdk.StatusBuilding:
		if currentStatus != sdk.StatusWaiting.String() {
			return fmt.Errorf("Cannot update status of PipelineBuildJob %d to %s, expected current status %s, got %s",
				pbJob.ID, status, sdk.StatusWaiting, currentStatus)
		}
		pbJob.Start = time.Now()
		pbJob.Status = status.String()

	case sdk.StatusFail, sdk.StatusSuccess, sdk.StatusDisabled, sdk.StatusSkipped:
		if currentStatus != string(sdk.StatusBuilding) && status != sdk.StatusDisabled && status != sdk.StatusSkipped {
			log.Info("Status is %s, cannot update %d to %s", currentStatus, pbJob.ID, status)
			// too late, Nate
			return nil
		}
		pbJob.Done = time.Now()
		pbJob.Status = status.String()
	default:
		return fmt.Errorf("Cannot update PipelineBuildJob %d to status %v", pbJob.ID, status.String())
	}

	if err := UpdatePipelineBuildJob(db, pbJob); err != nil {
		log.Warning("UpdatePipelineBuildJobStatus> Cannot update pipeline build job %d: %s", pbJob.ID, err)
		return err
	}

	pb, errLoad := LoadPipelineBuildByID(db, pbJob.PipelineBuildID)
	if errLoad != nil {
		log.Warning("UpdatePipelineBuildJobStatus> Cannot load pipeline build %d: %s", pbJob.PipelineBuildID, errLoad)
		return errLoad
	}

	event.PublishActionBuild(pb, pbJob)
	return nil
}
