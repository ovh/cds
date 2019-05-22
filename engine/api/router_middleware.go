package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) maintenanceMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
	if rc.MaintenanceAware && api.Maintenance {
		return ctx, sdk.WrapError(sdk.ErrServiceUnavailable, "CDS Maintenance ON")
	}
	return ctx, nil
}
