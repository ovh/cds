package services

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"
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
				log.Error(ctx, "KillDeadServices> Unable to find dead services: %v", errdead)
				continue
			}
			log.Debug(ctx, "services.KillDeadServices> %d services to remove", len(services))
			for i := range services {
				tx, err := db.Begin()
				if err != nil {
					log.Error(ctx, "services.KillDeadServices> unable to start transaction: %v", err)
					continue
				}
				if err := Delete(tx, &services[i]); err != nil {
					_ = tx.Rollback()
					log.Error(ctx, "KillDeadServices> Unable to find dead services: %v", err)
					continue
				}

				if err := tx.Commit(); err != nil {
					_ = tx.Rollback()
					log.Error(ctx, "KillDeadServices> Unable to  commit transaction: %v", err)
					continue
				}
			}
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				log.Error(ctx, "Exiting KillDeadServices: %v", err)
				return
			}
		}
	}
}
