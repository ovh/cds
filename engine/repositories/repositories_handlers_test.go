package repositories

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/engine/api/test"
	"github.com/stretchr/testify/assert"
)

func Test_postOperationHandler(t *testing.T) {
	//Bootstrap the service
	s, err := newTestService(t)
	test.NoError(t, err)

	op := new(Operation)
	op.URL = "https://github.com/ovh/cds.git"
	op.RepositoryStrategy = sdk.RepositoryStrategy{
		ConnectionType: "https",
		DefaultBranch:  "master",
	}
	op.Setup.Checkout = OperationCheckout{
		Branch: "master",
	}

	//Prepare request
	vars := map[string]string{}
	uri := s.Router.GetRoute("POST", s.postOperationHandler, vars)
	test.NotEmpty(t, uri)
	req := newRequest(t, s, "POST", uri, op)

	//Do the request
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 202, rec.Code)
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), op))

	//Prepare request
	vars = map[string]string{
		"uuid": op.UUID,
	}
	uri = s.Router.GetRoute("GET", s.getOperationsHandler, vars)
	test.NotEmpty(t, uri)
	req = newRequest(t, s, "GET", uri, nil)

	//Do the request
	rec = httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 200, rec.Code)
	t.Logf(rec.Body.String())
}
