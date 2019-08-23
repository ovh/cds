package api

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/migrate"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func TestPostAdminMigrationCancelHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()

	//Create admin user
	_, jwt := assets.InsertAdminUser(t, api.mustDB())

	//Load all migration
	uri := router.GetRoute("GET", api.getAdminMigrationsHandler, nil)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var migrations []sdk.Migration
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &migrations))
	numberOfMigrations := len(migrations)

	mig := sdk.Migration{
		Name:   "TestMigration",
		Status: sdk.MigrationStatusInProgress,
	}
	test.NoError(t, migrate.Insert(db, &mig))
	defer func() {
		_ = migrate.Delete(db, &mig)
	}()

	uri = router.GetRoute("GET", api.getAdminMigrationsHandler, nil)
	req = assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)

	// Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &migrations))

	assert.NotNil(t, migrations)
	assert.Equal(t, numberOfMigrations+1, len(migrations))

	//Prepare post request
	uri = router.GetRoute("POST", api.postAdminMigrationCancelHandler, map[string]string{"id": fmt.Sprintf("%d", migrations[0].ID)})
	req = assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, nil)

	// Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 204, w.Code)

	migUpdated, errM := migrate.GetByName(db, migrations[0].Name)
	test.NoError(t, errM)

	assert.Equal(t, sdk.MigrationStatusCanceled, migUpdated.Status)
}
