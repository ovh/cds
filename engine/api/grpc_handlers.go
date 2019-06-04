package api

import (
	"io"

	"github.com/go-gorp/gorp"

	"github.com/golang/protobuf/ptypes/empty"
	"golang.org/x/net/context"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/grpc"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type grpcHandlers struct {
	dbConnectionFactory *database.DBConnectionFactory
	store               cache.Store
	stepMaxLogSize      int64
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

		if err := workflow.AddLog(db, nil, in, h.stepMaxLogSize); err != nil {
			return sdk.WrapError(err, "Unable to insert log ")
		}
	}
}

//SendResult is the WorkflowQueueServer implementation
func (h *grpcHandlers) SendResult(c context.Context, res *sdk.Result) (*empty.Empty, error) {
	log.Debug("grpc.SendResult> begin")
	defer log.Debug("grpc.SendResult> end")

	workerID, ok := c.Value(keyWorkerID).(string)
	if !ok {
		return new(empty.Empty), sdk.ErrForbidden
	}

	db := h.dbConnectionFactory.GetDBMap()

	p, errP := project.LoadProjectByNodeRunID(nil, db, h.store, res.BuildID, project.LoadOptions.WithVariables)
	if errP != nil {
		return new(empty.Empty), sdk.WrapError(errP, "SendResult> Cannot load project")
	}

	wr, errW := worker.LoadByID(c, db, workerID)
	if errW != nil {
		return new(empty.Empty), sdk.WrapError(errW, "SendResult> Cannot load worker info")
	}
	dbFunc := func(c context.Context) *gorp.DbMap {
		return h.dbConnectionFactory.GetDBMap()
	}
	report, err := postJobResult(c, dbFunc, h.store, p, wr, res)
	if err != nil {
		return new(empty.Empty), sdk.WrapError(err, "Cannot post job result")
	}

	workflow.ResyncNodeRunsWithCommits(db, h.store, p, report)
	go workflow.SendEvent(context.Background(), db, p.Key, report)

	return new(empty.Empty), nil
}
