package broadcast

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// Broadcast is a gorp wrapper around sdk.Broadcast
type broadcast sdk.Broadcast

type broadcastRead struct {
	BroadcastID        int64  `json:"broadcast_id" db:"broadcast_id"`
	AuthentifiedUserID string `json:"user_id" db:"authentified_user_id"`
}

func init() {
	gorpmapping.Register(gorpmapping.New(broadcast{}, "broadcast", true, "id"))
	gorpmapping.Register(gorpmapping.New(broadcastRead{}, "broadcast_read", false, "broadcast_id", "user_id"))
}
