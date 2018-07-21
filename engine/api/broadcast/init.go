package broadcast

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk/log"
)

//Initialize starts goroutine for broadcast
func Initialize(c context.Context, DBFunc func() *gorp.DbMap) {
	tickPurge := time.NewTicker(6 * time.Hour)
	defer tickPurge.Stop()

	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting broadcast.Cleaner: %v", c.Err())
				return
			}
		case <-tickPurge.C:
			log.Debug("PurgeBroadcast> Deleting all old broadcast...")
			if err := deleteOldBroadcasts(DBFunc()); err != nil {
				log.Warning("broadcast.Purge> Error : %s", err)
			}
		}
	}
}
