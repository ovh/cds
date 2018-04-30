package api

import (
	"io"

	"github.com/golang/protobuf/ptypes/empty"
	"golang.org/x/net/context"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/grpc"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type grpcHandlers struct {
	dbConnectionFactory *database.DBConnectionFactory
	store               cache.Store
}

//AddBuildLog is the BuildLogServer implementation
func (h *grpcHandlers) AddBuildLog(stream grpc.BuildLog_AddBuildLogServer) error {
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

		db := h.dbConnectionFactory.GetDBMap()
		if err := pipeline.AddBuildLog(db, in); err != nil {
			return sdk.WrapError(err, "grpc.AddBuildLog> Unable to insert log ")
		}
	}
}

//SendLog is the WorkflowQueueServer implementation
func (h *grpcHandlers) SendLog(stream grpc.WorkflowQueue_SendLogServer) error {
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

		db := h.dbConnectionFactory.GetDBMap()
		if err := workflow.AddLog(db, nil, in); err != nil {
			return sdk.WrapError(err, "grpc.SendLog> Unable to insert log ")
		}
	}
}

//SendResult is the WorkflowQueueServer implementation
func (h *grpcHandlers) SendResult(c context.Context, res *sdk.Result) (*empty.Empty, error) {
	log.Debug("grpc.SendResult> begin")
	defer log.Debug("grpc.SendResult> end")

	//Get workerName from context
	workerName, ok := c.Value(keyWorkerName).(string)
	if !ok {
		return new(empty.Empty), sdk.ErrForbidden
	}

	workerID, ok := c.Value(keyWorkerID).(string)
	if !ok {
		return new(empty.Empty), sdk.ErrForbidden
	}

	workerUser := &sdk.User{
		Username: workerName,
	}

	db := h.dbConnectionFactory.GetDBMap()

	p, errP := project.LoadProjectByNodeRunID(db, h.store, res.BuildID, workerUser, project.LoadOptions.WithVariables)
	if errP != nil {
		return new(empty.Empty), sdk.WrapError(errP, "SendResult> Cannot load project")
	}

	chanEvent := make(chan interface{}, 1)
	chanError := make(chan error, 1)

	wr, errW := worker.LoadWorker(db, workerID)
	if errW != nil {
		return new(empty.Empty), sdk.WrapError(errW, "SendResult> Cannot load worker info")
	}
	go postJobResult(chanEvent, chanError, db, h.store, p, wr, res)

	workflowRuns, workflowNodeRuns, workflowNodeJobRuns, err := workflow.GetWorkflowRunEventData(chanError, chanEvent, p.Key)
	if err != nil {
		return new(empty.Empty), err
	}
	workflow.ResyncNodeRunsWithCommits(db, h.store, p, workflowNodeRuns)
	go workflow.SendEvent(db, workflowRuns, workflowNodeRuns, workflowNodeJobRuns, p.Key)

	return new(empty.Empty), nil
}
