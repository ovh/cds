package main

import (
	"net/http"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getWorkflowJobQueueHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}

func postWorkflowJobRrequirementsErrorHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}

func postTakeWorkflowJobHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}

func postBookWorkflowJobHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}

func postSpawnInfosWorkflowJobHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}

func postWorkflowJobResultHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	id, errc := requestVarInt(r, "id")
	if errc != nil {
		return sdk.WrapError(errc, "postWorkflowJobResultHandler> invalid id")
	}

	//Load workflow node job run
	job, errj := workflow.LoadNodeJobRun(db, id)
	if errj != nil {
		return sdk.WrapError(errj, "postWorkflowJobResultHandler> Unable to load node run job")
	}

	// Unmarshal into results
	var res sdk.Result
	if err := UnmarshalBody(r, &res); err != nil {
		return sdk.WrapError(err, "postWorkflowJobResultHandler> cannot unmarshal request")
	}

	tx, errb := db.Begin()
	if errb != nil {
		return sdk.WrapError(errb, "postWorkflowJobResultHandler> Cannot begin tx")
	}
	defer tx.Rollback()

	//Update worker status
	if err := worker.UpdateWorkerStatus(tx, c.Worker.ID, sdk.StatusWaiting); err != nil {
		log.Warning("postWorkflowJobResultHandler> Cannot update worker status (%s): %s", c.Worker.ID, err)
	}

	// Update action status
	log.Debug("postWorkflowJobResultHandler> Updating %d to %s in queue", id, res.Status)
	if err := workflow.UpdateNodeJobRunStatus(tx, job, res.Status); err != nil {
		return sdk.WrapError(err, "postWorkflowJobResultHandler> Cannot update %d status", id)
	}

	//Update spwan info
	_ = []sdk.SpawnInfo{{
		RemoteTime: res.RemoteTime,
		Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerEnd.ID, Args: []interface{}{c.Worker.Name, res.Duration}},
	}}
	/*
		if _, err := pipeline.AddSpawnInfosPipelineBuildJob(tx, pbJob.ID, infos); err != nil {
			log.Error("addQueueResultHandler> Cannot save spawn info job %d: %s", pbJob.ID, err)
			return err
		}
	*/

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "postWorkflowJobResultHandler> Cannot commit tx")
	}

	return nil
}

//TODO grpc
func postWorkflowJobLogsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}

func postWorkflowJobStepStatusHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}
