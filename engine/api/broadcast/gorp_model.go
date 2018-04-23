package broadcast

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// Broadcast is a gorp wrapper around sdk.Broadcast
type broadcast sdk.Broadcast

func init() {
	gorpmapping.Register(gorpmapping.New(broadcast{}, "broadcast", true, "id"))
}
