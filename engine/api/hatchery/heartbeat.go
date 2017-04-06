package hatchery

import (
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk/log"
)

// HatcheryHeartbeatTimeout defines the number of seconds allowed for hatcheries to refresh their beat
var HatcheryHeartbeatTimeout = 30.0

// Heartbeat runs in a goroutine and check last beat from all hatcheries
// on a 10s basis
func Heartbeat(DBFunc func() *gorp.DbMap) {
	// If this goroutine exit, then it's a crash
	defer log.Fatalf("Goroutine of hatchery.Heartbeat exited - Exit CDS Engine")

	for {
		db := DBFunc()
		if db != nil {
			w, err := LoadDeadHatcheries(db, HatcheryHeartbeatTimeout)
			if err != nil {
				log.Warning("HatcheryHeartbeat> Cannot load hatcherys: %s\n", err)
				// add extra sleep if db is unavailable
				time.Sleep(5 * time.Second)
				continue
			}

			for i := range w {
				err = DeleteHatchery(db, w[i].ID, w[i].Model.ID)
				if err != nil {
					log.Warning("HatcheryHeartbeat> Cannot delete hatchery %d: %s\n", w[i].ID, err)
					continue
				}
				log.Debug("HatcheryHeartbeat> Hatchery %s removed.\n", w[i].Name)
			}
		}
		time.Sleep(5 * time.Second)
	}
}
