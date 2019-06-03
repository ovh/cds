package services_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
)

func TestDAO(t *testing.T) {
	db, _, end := test.SetupPG(t)
	defer end()

	allSrv, err := services.All(db)
	for _, s := range allSrv {
		if err := services.Delete(db, &s); err != nil {
			t.Fatalf("unable to delete service: %v", err)
		}
	}

	var grp = sdk.Group{
		Name: "services-TestDAO-group",
	}

	u, _ := assets.InsertLambdaUser(db, &grp)

	privateKey, err := jws.NewRandomRSAKey()
	test.NoError(t, err)
	publicKey, err := jws.ExportPublicKey(privateKey)
	test.NoError(t, err)

	var srv = sdk.Service{
		CanonicalService: sdk.CanonicalService{
			Name:       sdk.RandomString(10),
			Type:       "type-service-test",
			PublicKey:  publicKey,
			Maintainer: *u,
		},
	}

	test.NoError(t, services.Insert(db, &srv))

	srv2, err := services.FindByName(db, srv.Name)
	test.NoError(t, err)

	assert.Equal(t, srv.Name, srv2.Name)
	assert.Equal(t, string(srv.PublicKey), string(srv2.PublicKey))

	all, err := services.FindByType(db, srv.Type)
	test.NoError(t, err)

	assert.True(t, len(all) >= 1)

	for _, s := range all {
		test.NoError(t, services.Delete(db, &s))
	}

	_, err = services.FindDeadServices(db, 0)
	test.NoError(t, err)
}
