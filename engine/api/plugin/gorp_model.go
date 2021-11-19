package plugin

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func init() {
	gorpmapping.Register(gorpmapping.New(sdk.GRPCPlugin{}, "grpc_plugin", true, "id"))
}
