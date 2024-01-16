package project_secret

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func init() {
	gorpmapping.Register(gorpmapping.New(dbProjectSecret{}, "project_secret", false, "id"))
}

type dbProjectSecret struct {
	sdk.ProjectSecret
}
