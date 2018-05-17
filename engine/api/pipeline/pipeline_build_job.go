package pipeline

import (
	"fmt"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
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
	if err := db.Insert(&dbmodel); err != nil {
		return err
	}
	*pbJob = sdk.PipelineBuildJob(dbmodel)
	return nil
}

// GetPipelineBuildJobByPipelineBuildID Get all pipeline build job for the given pipeline build
func GetPipelineBuildJobByPipelineBuildID(db gorp.SqlExecutor, store cache.Store, pbID int64) ([]sdk.PipelineBuildJob, error) {
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
		h := sdk.Hatchery{}
		if store.Get(keyBookJob(pbJobsGorp[i].ID), &h) {
			pbJobsGorp[i].BookedBy = h
		}
		pbJobs = append(pbJobs, sdk.PipelineBuildJob(pbJobsGorp[i]))
	}
	return pbJobs, nil
}

// GetPipelineBuildJobForUpdate Get pipeline build job
func GetPipelineBuildJobForUpdate(db gorp.SqlExecutor, store cache.Store, id int64) (*sdk.PipelineBuildJob, error) {
	var pbJobGorp PipelineBuildJob
	if err := db.SelectOne(&pbJobGorp, `
		SELECT *
		FROM pipeline_build_job
		WHERE id = $1 FOR UPDATE NOWAIT
	`, id); err != nil {
		if pqerr, ok := err.(*pq.Error); ok && pqerr.Code == "55P03" {
			return nil, sdk.WrapError(sdk.ErrAlreadyTaken, "GetPipelineBuildJobForUpdate> Unable to get pipeline_build_job for update (ErrAlreadyTaken)")
		}
		return nil, sdk.WrapError(err, "GetPipelineBuildJobForUpdate> Unable to get pipeline_build_job for update")
	}
	h := sdk.Hatchery{}
	if store.Get(keyBookJob(pbJobGorp.ID), &h) {
		pbJobGorp.BookedBy = h
	}
	pbJob := sdk.PipelineBuildJob(pbJobGorp)
	return &pbJob, nil
}

// GetPipelineBuildJob Get pipeline build job
func GetPipelineBuildJob(db gorp.SqlExecutor, store cache.Store, id int64) (*sdk.PipelineBuildJob, error) {
	var pbJobGorp PipelineBuildJob
	if err := db.SelectOne(&pbJobGorp, `
		SELECT *
		FROM pipeline_build_job
		WHERE id = $1
	`, id); err != nil {
		return nil, err
	}
	h := sdk.Hatchery{}
	if store.Get(keyBookJob(pbJobGorp.ID), &h) {
		pbJobGorp.BookedBy = h
	}
	pbJob := sdk.PipelineBuildJob(pbJobGorp)
	return &pbJob, nil
}

// LoadWaitingQueue Load Waiting pipeline_build_job
func LoadWaitingQueue(db gorp.SqlExecutor, store cache.Store) ([]sdk.PipelineBuildJob, error) {
	var pbJobsGorp []PipelineBuildJob
	if _, err := db.Select(&pbJobsGorp, `
		SELECT distinct pipeline_build_job.* FROM pipeline_build_job
		WHERE status = $1 ORDER BY pipeline_build_id ASC, pipeline_build_job.id ASC
	`, sdk.StatusWaiting.String()); err != nil {
		return nil, err
	}
	var pbJobs []sdk.PipelineBuildJob
	for i := range pbJobsGorp {
		if err := pbJobsGorp[i].PostGet(db); err != nil {
			return nil, err
		}
		h := sdk.Hatchery{}
		if store.Get(keyBookJob(pbJobsGorp[i].ID), &h) {
			pbJobsGorp[i].BookedBy = h
		}
		pbJobs = append(pbJobs, sdk.PipelineBuildJob(pbJobsGorp[i]))
	}
	return pbJobs, nil
}

// LoadGroupWaitingQueue loads pipeline_build_job in queue accessbible to given group
func LoadGroupWaitingQueue(db gorp.SqlExecutor, store cache.Store, groupID int64) ([]sdk.PipelineBuildJob, error) {
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
	for i := range pbJobsGorp {
		if err := pbJobsGorp[i].PostGet(db); err != nil {
			return nil, err
		}
		h := sdk.Hatchery{}
		if store.Get(keyBookJob(pbJobsGorp[i].ID), &h) {
			pbJobsGorp[i].BookedBy = h
		}
		pbJobs = append(pbJobs, sdk.PipelineBuildJob(pbJobsGorp[i]))
	}
	return pbJobs, nil
}

// LoadUserWaitingQueue loads action build in queue where user has access
func LoadUserWaitingQueue(db gorp.SqlExecutor, store cache.Store, u *sdk.User) ([]sdk.PipelineBuildJob, error) {
	var pbJobsGorp []PipelineBuildJob

	// If related user is admin, returns everything
	if u.Admin {
		return LoadWaitingQueue(db, store)
	}

	// If user is in no group, don't bother
	if len(u.Groups) == 0 {
		log.Warning("LoadUserWaitingQueue> User %s is in no groups, let it go", u.Username)
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

// TakePipelineBuildJob Take an action build for update
func TakePipelineBuildJob(db gorp.SqlExecutor, store cache.Store, pbJobID int64, model string, workerName string, infos []sdk.SpawnInfo) (*sdk.PipelineBuildJob, error) {
	pbJob, err := GetPipelineBuildJobForUpdate(db, store, pbJobID)
	if err != nil {
		return nil, sdk.WrapError(err, "TakePipelineBuildJob> Cannot load pipeline build job")
	}
	if pbJob.Status != sdk.StatusWaiting.String() {
		k := keyBookJob(pbJobID)
		h := sdk.Hatchery{}
		if store.Get(k, &h) {
			return nil, sdk.WrapError(sdk.ErrAlreadyTaken, "TakePipelineBuildJob> job %d is not waiting status and was booked by hatchery %d. Current status:%s", pbJobID, h.ID, pbJob.Status)
		}
		return nil, sdk.WrapError(sdk.ErrAlreadyTaken, "TakePipelineBuildJob> job %d is not waiting status. Current status:%s", pbJobID, pbJob.Status)
	}

	pbJob.Model = model
	pbJob.Job.WorkerName = workerName
	pbJob.Start = time.Now()
	pbJob.Status = sdk.StatusBuilding.String()

	if err := prepareSpawnInfos(pbJob, infos); err != nil {
		return nil, sdk.WrapError(err, "TakePipelineBuildJob> Cannot prepare spawn infos")
	}

	if err := UpdatePipelineBuildJob(db, pbJob); err != nil {
		return nil, sdk.WrapError(err, "TakePipelineBuildJob>Cannot update model on pipeline build job")
	}
	return pbJob, nil
}

func keyBookJob(pbJobID int64) string {
	return cache.Key("book", "job", strconv.FormatInt(pbJobID, 10))
}

// BookPipelineBuildJob Book an action for a hatchery
func BookPipelineBuildJob(store cache.Store, pbJobID int64, hatchery *sdk.Hatchery) (*sdk.Hatchery, error) {
	k := keyBookJob(pbJobID)
	h := sdk.Hatchery{}
	if !store.Get(k, &h) {
		// job not already booked, book it for 2 min
		store.SetWithTTL(k, hatchery, 120)
		return nil, nil
	}
	return &h, sdk.WrapError(sdk.ErrJobAlreadyBooked, "BookPipelineBuildJob> job %d already booked by %s (%d)", pbJobID, h.Name, h.ID)
}

// AddSpawnInfosPipelineBuildJob saves spawn info before starting worker
func AddSpawnInfosPipelineBuildJob(db gorp.SqlExecutor, store cache.Store, pbJobID int64, infos []sdk.SpawnInfo) (*sdk.PipelineBuildJob, error) {
	pbJob, err := GetPipelineBuildJobForUpdate(db, store, pbJobID)
	if err != nil {
		return nil, sdk.WrapError(err, "AddSpawnInfosPipelineBuildJob> Cannot load pipeline build job")
	}
	if err := prepareSpawnInfos(pbJob, infos); err != nil {
		return nil, sdk.WrapError(err, "AddSpawnInfosPipelineBuildJob> Cannot prepare swpan infos")
	}

	if err := UpdatePipelineBuildJob(db, pbJob); err != nil {
		return nil, sdk.WrapError(err, "AddSpawnInfosPipelineBuildJob> Cannot update pipeline build job")
	}
	return pbJob, nil
}

func prepareSpawnInfos(pbJob *sdk.PipelineBuildJob, infos []sdk.SpawnInfo) error {
	now := time.Now()
	for _, info := range infos {
		pbJob.SpawnInfos = append(pbJob.SpawnInfos, sdk.SpawnInfo{
			APITime:    now,
			RemoteTime: info.RemoteTime,
			Message:    info.Message,
		})
	}
	return nil
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
	return UpdatePipelineBuildJob(db, &pbJob)
}

// StopBuildingPipelineBuildJob Stop running pipeline build job
func StopBuildingPipelineBuildJob(db gorp.SqlExecutor, store cache.Store, pb *sdk.PipelineBuild) error {
	pbJobs, err := GetPipelineBuildJobByPipelineBuildID(db, store, pb.ID)
	if err != nil {
		return sdk.WrapError(err, "StopBuildingPipelineBuildJob> Cannot get pipeline build job")
	}
	for j := range pbJobs {
		pbJ := &pbJobs[j]
		if pbJ.Status == string(sdk.StatusBuilding) || pbJ.Status == string(sdk.StatusWaiting) {
			pbJ.Job.Reason = "Job has been stopped"
			pbJ.Status = string(sdk.StatusFail)
		}

		for i := range pbJ.Job.StepStatus {
			ss := &pbJ.Job.StepStatus[i]
			if ss.Status == sdk.StatusBuilding.String() {
				ss.Status = string(sdk.StatusStopped)
			}
		}

		if err := UpdatePipelineBuildJobStatus(db, pbJ, sdk.StatusStopped); err != nil {
			return sdk.WrapError(err, "StopBuildingPipelineBuildJob> Cannot stop pipeline build job")
		}
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
		return sdk.WrapError(err, "UpdatePipelineBuildJobStatus> Cannot lock pipeline build job %d", pbJob.ID)
	}

	switch status {
	case sdk.StatusBuilding:
		if currentStatus != sdk.StatusWaiting.String() {
			return fmt.Errorf("UpdatePipelineBuildJobStatus> Cannot update status of PipelineBuildJob %d to %s, expected current status %s, got %s",
				pbJob.ID, status, sdk.StatusWaiting, currentStatus)
		}
		pbJob.Start = time.Now()
		pbJob.Status = status.String()

	case sdk.StatusFail, sdk.StatusSuccess, sdk.StatusDisabled, sdk.StatusSkipped, sdk.StatusStopped:
		if currentStatus != string(sdk.StatusWaiting) && currentStatus != string(sdk.StatusBuilding) && status != sdk.StatusDisabled && status != sdk.StatusSkipped {
			log.Debug("UpdatePipelineBuildJobStatus> Status is %s, cannot update %d to %s", currentStatus, pbJob.ID, status)
			// too late, Nate
			return nil
		}
		pbJob.Done = time.Now()
		pbJob.Status = status.String()
	default:
		return fmt.Errorf("UpdatePipelineBuildJobStatus> Cannot update PipelineBuildJob %d to status %v", pbJob.ID, status.String())
	}

	if err := UpdatePipelineBuildJob(db, pbJob); err != nil {
		return sdk.WrapError(err, "UpdatePipelineBuildJobStatus> Cannot update pipeline build job %d", pbJob.ID)
	}

	pb, errLoad := LoadPipelineBuildByID(db, pbJob.PipelineBuildID)
	if errLoad != nil {
		return sdk.WrapError(errLoad, "UpdatePipelineBuildJobStatus> Cannot load pipeline build %d: %s", pbJob.PipelineBuildID, errLoad)
	}

	event.PublishActionBuild(pb, pbJob)
	return nil
}
