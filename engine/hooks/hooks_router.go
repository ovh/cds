package hooks

import (
	"context"

	"github.com/ovh/cds/engine/api"
)

func (s *Service) initRouter(ctx context.Context) {
	r := s.Router
	r.Background = ctx
	r.URL = s.Cfg.URL
	r.SetHeaderFunc = api.DefaultHeaders
	r.Middlewares = append(r.Middlewares, s.authMiddleware)

	r.Handle("/mon/version", r.GET(api.VersionHandler, api.Auth(false)))
	r.Handle("/mon/status", r.GET(s.statusHandler, api.Auth(false)))

	r.Handle("/admin/maintenance", r.POST(s.postMaintenanceHandler))

	r.Handle("/webhook/{uuid}", r.POST(s.webhookHandler, api.Auth(false)), r.GET(s.webhookHandler, api.Auth(false)), r.DELETE(s.webhookHandler, api.Auth(false)), r.PUT(s.webhookHandler, api.Auth(false)))
	r.Handle("/task", r.POST(s.postTaskHandler), r.GET(s.getTasksHandler))
	r.Handle("/task/bulk/start", r.GET(s.startTasksHandler))
	r.Handle("/task/bulk/stop", r.GET(s.stopTasksHandler))
	r.Handle("/task/bulk", r.POST(s.postTaskBulkHandler), r.DELETE(s.deleteTaskBulkHandler))
	r.Handle("/task/execute", r.POST(s.postAndExecuteTaskHandler))
	r.Handle("/task/{uuid}", r.GET(s.getTaskHandler), r.PUT(s.putTaskHandler), r.DELETE(s.deleteTaskHandler))
	r.Handle("/task/{uuid}/start", r.GET(s.startTaskHandler))
	r.Handle("/task/{uuid}/stop", r.GET(s.stopTaskHandler))
	r.Handle("/task/{uuid}/execution", r.GET(s.getTaskExecutionsHandler), r.DELETE(s.deleteAllTaskExecutionsHandler))
	r.Handle("/task/{uuid}/execution/{timestamp}", r.GET(s.getTaskExecutionHandler))
	r.Handle("/task/{uuid}/execution/{timestamp}/stop", r.POST(s.postStopTaskExecutionHandler))

}
