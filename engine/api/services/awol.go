package services

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk/log"
)

// KillDeadServices must be run as a goroutine. It Deletes all dead workers
func KillDeadServices(ctx context.Context, dbFunc func() *gorp.DbMap) {
	tick := time.NewTicker(30 * time.Second)
	db := dbFunc()
	for {
		select {
		case <-tick.C:
			services, errdead := FindDeadServices(ctx, db, 3*60*time.Second)
			if errdead != nil {
				log.Error("KillDeadServices> Unable to find dead services: %v", errdead)
				continue
			}
			log.Debug("services.KillDeadServices> %d services to remove", len(services))
			for i := range services {
				if err := Delete(db, &services[i]); err != nil {
					log.Error("KillDeadServices> Unable to find dead services: %v", err)
					continue
				}
			}
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				log.Error("Exiting KillDeadServices: %v", err)
				return
			}
		}
	}
}
