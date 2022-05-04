package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) maintenanceMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
	if api.Maintenance && !rc.MaintenanceAware && rc.Method != http.MethodGet {
		isAllowed := isMaintainer(ctx) || isService(ctx) || isWorker(ctx)
		if !isAllowed {
			return ctx, sdk.WrapError(sdk.ErrServiceUnavailable, "CDS Maintenance ON")
		}
	}
	return ctx, nil
}
