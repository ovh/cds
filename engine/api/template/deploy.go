package template

import (
	"database/sql"

	"github.com/ovh/cds/sdk"
)

func applyDeployTemplate(tx *sql.Tx, p *sdk.Project, buildPip *sdk.Pipeline, app *sdk.Application) error {

	return sdk.ErrUnknownTemplate
}
