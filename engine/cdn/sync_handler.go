package cdn

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/service"
)

func (s *Service) syncProjectsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return s.startCDSSync(ctx)
	}
}
