package api

import (
	"testing"

	"github.com/ovh/cds/engine/api/test/assets"

	"github.com/stretchr/testify/assert"
)

func TestNewTokenThenParse(t *testing.T) {
	api, db, _ := newTestAPI(t)
	u, _ := assets.InsertAdminUser(db)
	token, err := api.newToken(api.claimsForUser(u))
	assert.NoError(t, err)
	t.Logf("token is %s", token)

	claims, err := api.parseToken(token)
	assert.NoError(t, err)
	t.Logf("claims are %#v", claims)

	assert.NotNil(t, claimsUser(claims))
}
