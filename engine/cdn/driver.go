package cdn

import (
	"strings"

	"github.com/ovh/cds/engine/cdn/objectstore"
	"github.com/ovh/cds/sdk"
)

func (s *Service) getDriver(projectKey, integrationName string) (objectstore.Driver, error) {
	var storageDriver objectstore.Driver
	if strings.HasPrefix(integrationName, sdk.DefaultStorageIntegrationName) || integrationName == "" {
		storageDriver = s.DefaultDriver
	} else {
		projectIntegration, err := s.Client.ProjectIntegrationGet(projectKey, integrationName, true)
		if err != nil {
			return nil, err
		}

		var errD error
		storageDriver, errD = objectstore.InitDriver(projectIntegration)
		if errD != nil {
			return nil, sdk.WrapError(errD, "cannot init storage driver")
		}
	}

	return storageDriver, nil
}
