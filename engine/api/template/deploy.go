package template

import (
	"database/sql"

	"github.com/ovh/cds/sdk"
)

func applyDeployTemplate(tx *sql.Tx, p *sdk.Project, buildPip *sdk.Pipeline, app *sdk.Application) error {
	/*switch app.DeployTemplate.ID {
	case 0: // Not all applications needs to be deployed
		return nil
	}*/
	return sdk.ErrUnknownTemplate
}
