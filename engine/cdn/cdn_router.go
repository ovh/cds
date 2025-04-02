package cdn

import (
	"context"

	"github.com/ovh/cds/engine/service"
)

func (s *Service) initRouter(ctx context.Context) {
	r := s.Router
	r.Background = ctx
	r.URL = s.Cfg.URL
	r.SetHeaderFunc = service.DefaultHeaders
	r.Middlewares = append(r.Middlewares, service.TracingMiddlewareFunc(s), s.jwtMiddleware)
	r.DefaultAuthMiddleware = service.CheckRequestSignatureMiddleware(s.ParsedAPIPublicKey)
	r.PostMiddlewares = append(r.PostMiddlewares, service.TracingPostMiddleware)

	r.Handle("/mon/version", nil, r.GET(service.VersionHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/status", nil, r.GET(s.statusHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/metrics", nil, r.GET(service.GetPrometheustMetricsHandler(s), service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/metrics/all", nil, r.GET(service.GetMetricsHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/profile", nil, r.GET(service.GetAllProfilesHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/profile/{name}", nil, r.GET(service.GetProfileHandler, service.OverrideAuth(service.NoAuthMiddleware)))

	r.Handle("/cache", nil, r.DELETE(s.deleteCacheHandler))
	r.Handle("/cache/status", nil, r.GET(s.getStatusCacheHandler))

	r.Handle("/bulk/item/delete", nil, r.POST(s.bulkDeleteItemsHandler))

	r.Handle("/item/duplicate", nil, r.POST(s.postDuplicateItemForJobHandler))
	r.Handle("/item/upload", nil, r.POST(s.postUploadHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/item/stream", nil, r.GET(s.getItemLogsStreamHandler, service.OverrideAuth(s.validJWTMiddleware)))
	r.Handle("/v2/item/stream", nil, r.GET(s.getItemLogsStreamV2Handler, service.OverrideAuth(s.validJWTMiddleware)))
	r.Handle("/item/{type}", nil, r.GET(s.getItemsHandler))
	r.Handle("/item/{type}/lines", nil, r.GET(s.getItemsAllLogsLinesHandler, service.OverrideAuth(s.validJWTMiddleware)))
	r.Handle("/item/{type}/{apiRef}", nil, r.GET(s.getItemHandler, service.OverrideAuth(s.itemAccessMiddleware)), r.DELETE(s.deleteItemHandler))
	r.Handle("/item/{type}/{apiRef}/checksync", nil, r.GET(s.getItemCheckSyncHandler, service.OverrideAuth(s.itemAccessMiddleware)))
	r.Handle("/item/{type}/{apiRef}/download", nil, r.GET(s.getItemDownloadHandler, service.OverrideAuth(s.itemAccessMiddleware)))
	r.Handle("/item/{type}/{apiRef}/download/{unit}", nil, r.GET(s.getItemDownloadInUnitHandler, service.OverrideAuth(s.itemAccessMiddleware)))
	r.Handle("/item/{type}/{apiRef}/lines", nil, r.GET(s.getItemLogsLinesHandler, service.OverrideAuth(s.itemAccessMiddleware)))

	r.Handle("/unit", nil, r.GET(s.getUnitsHandler))
	r.Handle("/unit/{id}", nil, r.DELETE(s.deleteUnitHandler))
	r.Handle("/unit/{id}/item", nil, r.DELETE(s.markItemUnitAsDeleteHandler))

	r.Handle("/sync/buffer", nil, r.POST(s.syncBufferHandler))

	r.Handle("/size/item/project/{projectKey}", nil, r.GET(s.getSizeByProjectHandler))

	r.Handle("/admin/database/migration", nil, r.GET(s.getAdminDatabaseMigrationHandler))
	r.Handle("/admin/database/migration/delete/{id}", nil, r.DELETE(s.deleteAdminDatabaseMigrationHandler))
	r.Handle("/admin/database/migration/unlock/{id}", nil, r.POST(s.postAdminDatabaseMigrationUnlockHandler))
	r.Handle("/admin/database/entity", nil, r.GET(s.getAdminDatabaseEntityList))
	r.Handle("/admin/database/entity/{entity}", nil, r.GET(s.getAdminDatabaseEntity))
	r.Handle("/admin/database/entity/{entity}/info", nil, r.POST(s.postAdminDatabaseEntityInfo))
	r.Handle("/admin/database/entity/{entity}/roll", nil, r.POST(s.postAdminDatabaseEntityRoll))

	r.Handle("/admin/backend/{id}/resync/{type}", nil, r.POST(s.postAdminResyncBackendWithDatabaseHandler))
}
