package cdn

import (
	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
)

func (s *Service) getAdminDatabaseSignatureResume() service.Handler {
	return api.AdminDatabaseSignatureResume(s.mustDB, s.Mapper)
}

func (s *Service) getAdminDatabaseSignatureTuplesBySigner() service.Handler {
	return api.AdminDatabaseSignatureTuplesBySigner(s.mustDB, s.Mapper)
}

func (s *Service) postAdminDatabaseSignatureRollEntityByPrimaryKey() service.Handler {
	return api.AdminDatabaseSignatureRollEntityByPrimaryKey(s.mustDB, s.Mapper)
}

func (s *Service) getAdminDatabaseEncryptedEntities() service.Handler {
	return api.AdminDatabaseEncryptedEntities(s.mustDB, s.Mapper)
}

func (s *Service) getAdminDatabaseEncryptedTuplesByEntity() service.Handler {
	return api.AdminDatabaseEncryptedTuplesByEntity(s.mustDB, s.Mapper)
}

func (s *Service) postAdminDatabaseRollEncryptedEntityByPrimaryKey() service.Handler {
	return api.AdminDatabaseRollEncryptedEntityByPrimaryKey(s.mustDB, s.Mapper)
}
