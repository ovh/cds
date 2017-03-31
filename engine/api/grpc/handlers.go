package grpc

import (
	"io"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/log"
)

type handlers struct{}

func (h *handlers) AddBuildLog(stream BuildLog_AddBuildLogServer) error {
	log.Info("grpc.AddBuildLog> started stream")
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
