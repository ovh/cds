package hooks

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (s *Service) authMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
	if !rc.NeedAuth {
		return ctx, nil
	}

	// Check that request are signed by public key

	return ctx, sdk.ErrUnauthorized
}
