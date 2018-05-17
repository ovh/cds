package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/broadcast"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
)

func Test_addBroadcastHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(db)

	br := sdk.Broadcast{
		Title:   "maintenance swarm",
		Content: "bad news",
	}
	jsonBody, _ := json.Marshal(br)
	body := bytes.NewBuffer(jsonBody)

	uri := api.Router.GetRoute("POST", api.addBroadcastHandler, nil)
	req, err := http.NewRequest("POST", uri, body)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)
	var newBr sdk.Broadcast
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &newBr))
	assert.NotEmpty(t, newBr.ID)
	assert.Equal(t, "maintenance swarm", newBr.Title)
	assert.False(t, newBr.Read)
}

func Test_postMarkAsReadBroadcastHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(db)
	uLambda, passLambda := assets.InsertLambdaUser(db)

	br := sdk.Broadcast{
		Title:   "maintenance swarm",
		Content: "bad news",
	}

	test.NoError(t, broadcast.Insert(db, &br))

	uri := api.Router.GetRoute("POST", api.postMarkAsReadBroadcastHandler, map[string]string{"id": fmt.Sprintf("%d", br.ID)})
	req, err := http.NewRequest("POST", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	uri2 := api.Router.GetRoute("GET", api.getBroadcastHandler, map[string]string{"id": fmt.Sprintf("%d", br.ID)})
	req2, err := http.NewRequest("GET", uri2, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req2, u, pass)

	// Do the request
	w2 := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w2, req2)
	assert.Equal(t, 200, w2.Code)

	var newBr sdk.Broadcast
	test.NoError(t, json.Unmarshal(w2.Body.Bytes(), &newBr))
	assert.NotZero(t, newBr.ID)
	assert.Equal(t, "maintenance swarm", newBr.Title)
	assert.True(t, newBr.Read)

	uri3 := api.Router.GetRoute("GET", api.getBroadcastHandler, map[string]string{"id": fmt.Sprintf("%d", br.ID)})
	req3, err := http.NewRequest("GET", uri3, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req3, uLambda, passLambda)

	// Do the request
	w3 := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w3, req3)
	assert.Equal(t, 200, w3.Code)

	test.NoError(t, json.Unmarshal(w3.Body.Bytes(), &newBr))
	assert.NotZero(t, newBr.ID)
	assert.Equal(t, "maintenance swarm", newBr.Title)
	assert.False(t, newBr.Read)
}
