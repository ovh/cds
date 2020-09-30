package hooks

import (
	"context"

	"github.com/ovh/cds/engine/service"
)

func (s *Service) initRouter(ctx context.Context) {
	r := s.Router
	r.Background = ctx
	r.URL = s.Cfg.URL
	r.SetHeaderFunc = service.DefaultHeaders
	r.DefaultAuthMiddleware = service.CheckRequestSignatureMiddleware(s.ParsedAPIPublicKey)

	r.Handle("/admin/maintenance", nil, r.POST(s.postMaintenanceHandler))

	r.Handle("/mon/version", nil, r.GET(service.VersionHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/status", nil, r.GET(s.statusHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/metrics", nil, r.GET(service.GetPrometheustMetricsHandler(s), service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/metrics/all", nil, r.GET(service.GetMetricsHandler, service.OverrideAuth(service.NoAuthMiddleware)))

	r.Handle("/webhook/{uuid}", nil, r.POST(s.webhookHandler, service.OverrideAuth(service.NoAuthMiddleware)), r.GET(s.webhookHandler, service.OverrideAuth(service.NoAuthMiddleware)), r.DELETE(s.webhookHandler, service.OverrideAuth(service.NoAuthMiddleware)), r.PUT(s.webhookHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/task", nil, r.POST(s.postTaskHandler), r.GET(s.getTasksHandler))
	r.Handle("/task/bulk/start", nil, r.GET(s.startTasksHandler))
	r.Handle("/task/bulk/stop", nil, r.GET(s.stopTasksHandler))
	r.Handle("/task/bulk", nil, r.POST(s.postTaskBulkHandler), r.DELETE(s.deleteTaskBulkHandler))
	r.Handle("/task/execute", nil, r.POST(s.postAndExecuteTaskHandler))
	r.Handle("/task/{uuid}", nil, r.GET(s.getTaskHandler), r.PUT(s.putTaskHandler), r.DELETE(s.deleteTaskHandler))
	r.Handle("/task/{uuid}/start", nil, r.GET(s.startTaskHandler))
	r.Handle("/task/{uuid}/stop", nil, r.GET(s.stopTaskHandler))
	r.Handle("/task/{uuid}/execution", nil, r.GET(s.getTaskExecutionsHandler), r.DELETE(s.deleteAllTaskExecutionsHandler))
	r.Handle("/task/{uuid}/execution/{timestamp}", nil, r.GET(s.getTaskExecutionHandler))
	r.Handle("/task/{uuid}/execution/{timestamp}/stop", nil, r.POST(s.postStopTaskExecutionHandler))
}
