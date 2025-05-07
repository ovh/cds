package cdn

import (
	"github.com/ovh/cds/engine/database"
	"github.com/ovh/cds/engine/service"
)

func (s *Service) getAdminDatabaseMigrationHandler() service.Handler {
	return database.AdminGetDatabaseMigration(s.mustDB)
}

func (s *Service) deleteAdminDatabaseMigrationHandler() service.Handler {
	return database.AdminDeleteDatabaseMigration(s.mustDB)
}

func (s *Service) postAdminDatabaseMigrationUnlockHandler() service.Handler {
	return database.AdminPostDatabaseMigrationUnlock(s.mustDB)
}

func (s *Service) getAdminDatabaseEntityList() service.Handler {
	return database.AdminGetDatabaseEntityList(s.mustDB, s.mustMapper)
}

func (s *Service) getAdminDatabaseEntity() service.Handler {
	return database.AdminGetDatabaseEntity(s.mustDB, s.mustMapper)
}

func (s *Service) postAdminDatabaseEntityInfo() service.Handler {
	return database.AdminPostDatabaseEntityInfo(s.mustDB, s.mustMapper)
}

func (s *Service) postAdminDatabaseEntityRoll() service.Handler {
	return database.AdminPostDatabaseEntityRoll(s.mustDB, s.mustMapper)
}
