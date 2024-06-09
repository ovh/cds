package authentication_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

type myPayload struct {
	RandomID string `json:"consumer_id"`
	Nonce    int64  `json:"nonce"`
}

func TestSignJWS(t *testing.T) {
	_, _ = test.SetupPG(t, bootstrap.InitializeDB)

	p := myPayload{
		RandomID: sdk.UUID(),
		Nonce:    time.Now().Unix(),
	}

	token, err := authentication.SignJWS(p, time.Now(), time.Hour)
	require.NoError(t, err)

	var res myPayload
	require.NoError(t, authentication.VerifyJWS(token, &res))

	assert.Equal(t, p.RandomID, res.RandomID)
	assert.Equal(t, p.Nonce, res.Nonce)
}
