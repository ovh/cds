package services

import (
	"context"
	"time"

	"github.com/ovh/cds/sdk/log"
)

// KillDeadServices must be run as a goroutine. It Deletes all dead workers
func KillDeadServices(ctx context.Context, r *Repository) {
	tick := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-tick.C:
			services, errdead := r.FindDeadServices(3 * 60 * time.Second)
			if errdead != nil {
				log.Error("KillDeadServices> Unable to find dead services: %v", errdead)
				continue
			}
			for i := range services {
				if err := r.Delete(&services[i]); err != nil {
					log.Error("KillDeadServices> Unable to find dead services: %v", errdead)
					continue
				}
			}
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				log.Error("Exiting KillDeadServices: %v", err)
			}
		}
	}
}
