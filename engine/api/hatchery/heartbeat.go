package hatchery

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk/log"
)

// HatcheryHeartbeatTimeout defines the number of seconds allowed for hatcheries to refresh their beat
var HatcheryHeartbeatTimeout = 30.0

// Heartbeat runs in a goroutine and check last beat from all hatcheries
// on a 10s basis
func Heartbeat(c context.Context, DBFunc func(context.Context) *gorp.DbMap) {
	tick := time.NewTicker(5 * time.Second).C

	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting HatcheryHeartbeat: %v", c.Err())
				return
			}
		case <-tick:
			db := DBFunc(c)
			if db != nil {
				w, err := LoadDeadHatcheries(db, HatcheryHeartbeatTimeout)
				if err != nil {
					log.Warning("HatcheryHeartbeat> Cannot load hatcherys: %s", err)
					continue
				}

				for i := range w {
					if err = DeleteHatchery(db, w[i].ID, w[i].Model.ID); err != nil {
						log.Warning("HatcheryHeartbeat> Cannot delete hatchery %d: %s", w[i].ID, err)
						continue
					}
					log.Debug("HatcheryHeartbeat> Hatchery %s removed.", w[i].Name)
				}
			}
		}
	}
}
