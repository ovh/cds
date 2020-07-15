package plugin

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gorpmapping"
)

type grpcPlugin sdk.GRPCPlugin

func init() {
	gorpmapping.Register(gorpmapping.New(grpcPlugin{}, "grpc_plugin", true, "id"))
}
