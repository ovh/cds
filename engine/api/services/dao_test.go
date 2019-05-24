package services_test

import (
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
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

	var grp = sdk.Group{
		Name: "services-TestDAO-group",
	}

	u, _ := assets.InsertLambdaUser(db, &grp)

	privateKey, err := jws.NewRandomRSAKey()
	test.NoError(t, err)
	publicKey, err := jws.ExportPublicKey(privateKey)
	test.NoError(t, err)

	id := sdk.UUID()
	claims := sdk.AccessTokenJWTClaims{
		ID:     id,
		Groups: sdk.GroupsToIDs([]sdk.Group{grp}),
		StandardClaims: jwt.StandardClaims{
			Issuer:    "services-TestDAO-token",
			Subject:   "services-TestDAO-token",
			Id:        id,
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		},
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodRS512, claims)
	signedToken, err := jwtToken.SignedString(privateKey)
	test.NoError(t, err)

	var srv = sdk.Service{
		CanonicalService: sdk.CanonicalService{
			Name:       sdk.RandomString(10),
			Type:       "type-service-test",
			PublicKey:  publicKey,
			Maintainer: *u,
		},
		ClearJWT: signedToken,
	}

	test.NoError(t, services.Insert(db, &srv))

	srv2, err := services.FindByName(db, srv.Name)
	test.NoError(t, err)

	assert.Equal(t, srv.Name, srv2.Name)
	assert.Equal(t, string(srv.PublicKey), string(srv2.PublicKey))

	jwt, err := LoadClearJWT(db, srv2.ID)
	test.NoError(t, err)
	assert.Equal(t, signedToken, jwt)

	all, err := services.FindByType(db, srv.Type)
	test.NoError(t, err)

	assert.True(t, len(all) >= 1)

	for _, s := range all {
		test.NoError(t, services.Delete(db, &s))
	}

	_, err = services.FindDeadServices(db, 0)
	test.NoError(t, err)
}
