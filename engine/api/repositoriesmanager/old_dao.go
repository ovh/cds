package repositoriesmanager

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// InsertForApplication associates a repositories manager with an application.
func InsertForApplication(db gorp.SqlExecutor, app *sdk.Application) error {
	query := `UPDATE application SET vcs_server = $1, repo_fullname = $2 WHERE id = $3`
	if _, err := db.Exec(query, app.VCSServer, app.RepositoryFullname, app.ID); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

// DeleteForApplication removes association between a repositories manager and an application.
func DeleteForApplication(db gorp.SqlExecutor, app *sdk.Application) error {
	query := `UPDATE application SET vcs_server = '', repo_fullname = '' WHERE id = $1`
	if _, err := db.Exec(query, app.ID); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

// LoadLinkedApplicationNames loads applications which are linked with this repository manager name.
func LoadLinkedApplicationNames(db gorp.SqlExecutor, projectKey, rmName string) (sdk.IDNames, error) {
	query := `
    SELECT application.id, application.name, application.description, '' AS icon
	  FROM application
		JOIN project ON project.id = application.project_id
    WHERE project.projectkey = $1 AND application.vcs_server = $2
  `
	var idNames sdk.IDNames
	if _, err := db.Select(&idNames, query, projectKey, rmName); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WithStack(err)
	}
	return idNames, nil
}
