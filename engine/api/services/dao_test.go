package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
)

func TestDAO(t *testing.T) {
	db, _ := test.SetupPG(t)

	allSrv, err := services.LoadAll(context.TODO(), db)
	for _, s := range allSrv {
		if err := services.Delete(db, &s); err != nil {
			t.Fatalf("unable to delete service: %v", err)
		}
	}

	privateKey, err := jws.NewRandomRSAKey()
	test.NoError(t, err)
	publicKey, err := jws.ExportPublicKey(privateKey)
	test.NoError(t, err)

	var srv = sdk.Service{
		CanonicalService: sdk.CanonicalService{
			Name:      sdk.RandomString(10),
			Type:      "type-service-test",
			PublicKey: publicKey,
		},
	}

	test.NoError(t, services.Insert(context.TODO(), db, &srv))

	srv2, err := services.LoadByName(context.TODO(), db, srv.Name)
	test.NoError(t, err)

	assert.Equal(t, srv.Name, srv2.Name)
	assert.Equal(t, string(srv.PublicKey), string(srv2.PublicKey))

	all, err := services.LoadAllByType(context.TODO(), db, srv.Type)
	test.NoError(t, err)

	assert.True(t, len(all) >= 1)

	for _, s := range all {
		test.NoError(t, services.Delete(db, &s))
	}

	_, err = services.FindDeadServices(context.TODO(), db, 0)
	test.NoError(t, err)
}

func TestDAOWithStatus(t *testing.T) {
	db, _ := test.SetupPG(t)

	allSrv, err := services.LoadAll(context.TODO(), db)
	for _, s := range allSrv {
		if err := services.Delete(db, &s); err != nil {
			t.Fatalf("unable to delete service: %v", err)
		}
	}

	privateKey, err := jws.NewRandomRSAKey()
	test.NoError(t, err)
	publicKey, err := jws.ExportPublicKey(privateKey)
	test.NoError(t, err)

	theServiceName := sdk.RandomString(10)
	var srv = sdk.Service{
		CanonicalService: sdk.CanonicalService{
			Name:      theServiceName,
			Type:      "type-service-test",
			PublicKey: publicKey,
		},
		MonitoringStatus: sdk.MonitoringStatus{
			Now: time.Now(),
			Lines: []sdk.MonitoringStatusLine{
				{
					Component: "backend/cds-backend/items",
					Hostname:  "foofoo.local",
					Service:   "cds-cdn",
					Status:    "OK",
					Type:      "cdn",
					Value:     "90",
				},
			},
		},
	}

	test.NoError(t, services.Insert(context.TODO(), db, &srv))
	test.NoError(t, services.UpsertStatus(db, srv, ""))

	srv2, err := services.LoadByName(context.TODO(), db, srv.Name)
	test.NoError(t, err)

	assert.Equal(t, srv.Name, srv2.Name)
	assert.Equal(t, string(srv.PublicKey), string(srv2.PublicKey))

	all, err := services.LoadAllByType(context.TODO(), db, srv.Type)
	test.NoError(t, err)

	assert.True(t, len(all) >= 1)

	all2, err := services.LoadAll(context.TODO(), db, services.LoadOptions.WithStatus)
	test.NoError(t, err)
	var found bool
	for _, s := range all2 {
		if s.Name == theServiceName {
			found = true
			require.EqualValues(t, 1, len(s.MonitoringStatus.Lines))
			for _, ss := range s.MonitoringStatus.Lines {
				require.EqualValues(t, "backend/cds-backend/items", ss.Component)
			}
			break
		}
	}

	require.True(t, found)

	srv3, err := services.LoadByName(context.TODO(), db, theServiceName)
	test.NoError(t, err)
	require.EqualValues(t, 0, len(srv3.MonitoringStatus.Lines))

	srv4, err := services.LoadByName(context.TODO(), db, theServiceName, services.LoadOptions.WithStatus)
	test.NoError(t, err)
	require.EqualValues(t, 1, len(srv4.MonitoringStatus.Lines))

	for _, s := range all {
		test.NoError(t, services.Delete(db, &s))
	}

	_, err = services.FindDeadServices(context.TODO(), db, 0)
	test.NoError(t, err)
}
