package grpc

import (
	"io"

	"golang.org/x/net/context"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type handlers struct{}

func (h *handlers) AddBuildLog(stream BuildLog_AddBuildLogServer) error {
	log.Debug("grpc.AddBuildLog> started stream")
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		log.Debug("grpc.AddBuildLog> Got %+v", in)

		db := database.GetDBMap()
		if err := pipeline.AddBuildLog(db, in); err != nil {
			log.Warning("grpc.AddBuildLog> Unable to insert log : %s", err)
			return err
		}
	}
}

func (*handlers) SendLog(stream WorkflowQueue_SendLogServer) error {
	log.Debug("grpc.SendLog> begin")
	defer log.Debug("grpc.SendLog> end")
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		log.Debug("grpc.SendLog> Got %+v", in)

		db := database.GetDBMap()
		if err := workflow.AddLog(db, nil, in); err != nil {
			log.Warning("grpc.SendLog> Unable to insert log : %s", err)
			return err
		}
	}
}

func (*handlers) SendResult(c context.Context, res *sdk.Result) (*empty.Empty, error) {
	log.Debug("grpc.SendResult> begin")
	defer log.Debug("grpc.SendResult> end")

	//Get workerID from context
	workerID, ok := c.Value(keyWorkerID).(string)
	if !ok {
		return new(empty.Empty), sdk.ErrForbidden
	}

	//Get workerName from context
	workerName, ok := c.Value(keyWorkerName).(string)
	if !ok {
		return new(empty.Empty), sdk.ErrForbidden
	}

	db := database.GetDBMap()

	//Load workflow node job run
	job, errj := workflow.LoadNodeJobRun(db, res.BuildID)
	if errj != nil {
		return new(empty.Empty), sdk.WrapError(errj, "postWorkflowJobResultHandler> Unable to load node run job")
	}

	//Start the transaction
	tx, errb := db.Begin()
	if errb != nil {
		return new(empty.Empty), sdk.WrapError(errb, "postWorkflowJobResultHandler> Cannot begin tx")
	}
	defer tx.Rollback()

	//Update worker status
	if err := worker.UpdateWorkerStatus(tx, workerID, sdk.StatusWaiting); err != nil {
		log.Warning("postWorkflowJobResultHandler> Cannot update worker status (%s): %s", workerID, err)
	}

	// Update action status
	log.Debug("postWorkflowJobResultHandler> Updating %d to %s in queue", workerID, res.Status)
	if err := workflow.UpdateNodeJobRunStatus(tx, job, sdk.Status(res.Status)); err != nil {
		return new(empty.Empty), sdk.WrapError(err, "postWorkflowJobResultHandler> Cannot update %d status", workerID)
	}

	remoteTime, errt := ptypes.Timestamp(res.RemoteTime)
	if errt != nil {
		return new(empty.Empty), sdk.WrapError(errt, "postWorkflowJobResultHandler> Cannot parse remote time")
	}

	//Update spwan info
	infos := []sdk.SpawnInfo{{
		RemoteTime: remoteTime,
		Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerEnd.ID, Args: []interface{}{workerName, res.Duration}},
	}}

	//Add spawn infos
	if _, err := workflow.AddSpawnInfosNodeJobRun(tx, job.ID, infos); err != nil {
		log.Error("addQueueResultHandler> Cannot save spawn info job %d: %s", job.ID, err)
		return nil, err
	}

	//Commit the transaction
	if err := tx.Commit(); err != nil {
		return new(empty.Empty), sdk.WrapError(err, "postWorkflowJobResultHandler> Cannot commit tx")
	}

	return nil, nil
}
