package migrate

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/engine/api/test"
)

func TestRefactorAuthentication(t *testing.T) {
	db, cache, _ := test.SetupPG(t)
	mail.Init("", "", "", "", "", false, true)
	require.NoError(t, RefactorAuthentication(context.TODO(), db, cache, "http://localhost:8080", "http://localhost:4200"))
}
