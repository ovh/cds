package test

import (
	"testing"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/sessionstore"
)

func LocalAuth(t *testing.T) auth.Driver {
	authDriver, err := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local"})
	NoError(t, err)
	return authDriver
}
