package broadcast

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"
)

//Initialize starts goroutine for broadcast
func Initialize(ctx context.Context, DBFunc func() *gorp.DbMap) {
	tickPurge := time.NewTicker(6 * time.Hour)
	defer tickPurge.Stop()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Exiting broadcast.Cleaner: %v", ctx.Err())
				return
			}
		case <-tickPurge.C:
			log.Debug(ctx, "PurgeBroadcast> Deleting all old broadcast...")
			if err := deleteOldBroadcasts(DBFunc()); err != nil {
				log.Warn(ctx, "broadcast.Purge> Error : %s", err)
			}
		}
	}
}
