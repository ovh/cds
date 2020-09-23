package cdn

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/cdn/storage/cds"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) syncProjectsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Find CDS Backend
		for _, st := range s.Units.Storages {
			cdsStorage, ok := st.(*cds.CDS)
			if !ok {
				continue
			}
			s.GoRoutines.Exec(ctx, "cdn-cds-backend-migration", func(ctx context.Context) {
				if err := s.SyncLogs(ctx, cdsStorage); err != nil {
					log.Error(ctx, "unable to sync logs: %v", err)
				}
			})
			break
		}
		return nil
	}
}
