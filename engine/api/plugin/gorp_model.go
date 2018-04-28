package plugin

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"

	"github.com/ovh/cds/sdk"
)

type grpcPlugin sdk.GRPCPlugin

func init() {
	gorpmapping.Register(gorpmapping.New(grpcPlugin{}, "grpc_plugin", true, "id"))
}
