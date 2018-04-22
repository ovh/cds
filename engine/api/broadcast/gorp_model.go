package broadcast

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// Broadcast is a gorp wrapper around sdk.Broadcast
type Broadcast sdk.Broadcast

func init() {
	gorpmapping.Register(gorpmapping.New(Broadcast{}, "broadcast", true, "id"))
}
