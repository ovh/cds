package cdn

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/service"
)

func (s *Service) syncBufferHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		s.GoRoutines.Exec(context.Background(), "Buffer sync", func(ctx context.Context) {
			s.Units.SyncBuffer(ctx)
		})
		return nil
	}
}
