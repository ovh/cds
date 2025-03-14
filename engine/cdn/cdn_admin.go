package cdn

import (
	"github.com/ovh/cds/engine/database"
	"github.com/ovh/cds/engine/service"
)

func (s *Service) deleteDatabaseMigrationHandler() service.Handler {
	return database.AdminDeleteDatabaseMigration(s.mustDB)
}

func (s *Service) postDatabaseMigrationUnlockedHandler() service.Handler {
	return database.AdminDatabaseMigrationUnlocked(s.mustDB)
}

func (s *Service) getDatabaseMigrationHandler() service.Handler {
	return database.AdminGetDatabaseMigration(s.mustDB)
}

func (s *Service) getAdminDatabaseSignatureResume() service.Handler {
	return database.AdminDatabaseSignatureResume(s.mustDB, s.Mapper)
}

func (s *Service) getAdminDatabaseSignatureTuplesBySigner() service.Handler {
	return database.AdminDatabaseSignatureTuplesBySigner(s.mustDB, s.Mapper)
}

func (s *Service) postAdminDatabaseSignatureRollEntityByPrimaryKey() service.Handler {
	return database.AdminDatabaseSignatureRollEntityByPrimaryKey(s.mustDB, s.Mapper)
}

func (s *Service) getAdminDatabaseSignatureInfoEntityByPrimaryKey() service.Handler {
	return database.AdminDatabaseSignatureInfoEntityByPrimaryKey(s.mustDB, s.Mapper)
}

func (s *Service) getAdminDatabaseEncryptedEntities() service.Handler {
	return database.AdminDatabaseEncryptedEntities(s.mustDB, s.Mapper)
}

func (s *Service) getAdminDatabaseTuplesByEntity() service.Handler {
	return database.AdminDatabaseTuplesByEntity(s.mustDB, s.Mapper)
}

func (s *Service) postAdminDatabaseRollEncryptedEntityByPrimaryKey() service.Handler {
	return database.AdminDatabaseRollEncryptedEntityByPrimaryKey(s.mustDB, s.Mapper)
}

func (s *Service) getAdminDatabaseInfoEncryptedEntityByPrimaryKey() service.Handler {
	return database.AdminDatabaseInfoEncryptedEntityByPrimaryKey(s.mustDB, s.Mapper)
}
