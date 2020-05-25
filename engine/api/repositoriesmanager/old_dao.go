package repositoriesmanager

import (
	"database/sql"

	"github.com/go-gorp/gorp"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
)

//DeprecatedLoadAllForProject loads all repomanager link for a project
func DeprecatedLoadAllForProject(db gorp.SqlExecutor, projectKey string) ([]sdk.ProjectVCSServer, error) {
	vcsServerStr := []byte{}
	if err := db.QueryRow("select vcs_servers from project where projectkey = $1", projectKey).Scan(&vcsServerStr); err != nil {
		return nil, err
	}

	if len(vcsServerStr) == 0 {
		return []sdk.ProjectVCSServer{}, nil
	}

	clearVCSServer, err := secret.Decrypt(vcsServerStr)
	if err != nil {
		return nil, err
	}
	vcsServer := []sdk.ProjectVCSServer{}

	if err := yaml.Unmarshal(clearVCSServer, &vcsServer); err != nil {
		return nil, err
	}

	return vcsServer, nil
}

//InsertForApplication associates a repositories manager with an application
func InsertForApplication(db gorp.SqlExecutor, app *sdk.Application, projectKey string) error {
	query := `UPDATE application SET vcs_server = $1, repo_fullname = $2 WHERE id = $3`
	if _, err := db.Exec(query, app.VCSServer, app.RepositoryFullname, app.ID); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

//DeleteForApplication removes association between  a repositories manager and an application
//it deletes the corresponding line in repositories_manager_project
func DeleteForApplication(db gorp.SqlExecutor, app *sdk.Application) error {
	query := `UPDATE application SET vcs_server = '', repo_fullname = '' WHERE id = $1`
	if _, err := db.Exec(query, app.ID); err != nil {
		return err
	}
	return nil
}

//LoadLinkedApplicationNames loads applications which are linked with this repository manager name
func LoadLinkedApplicationNames(db gorp.SqlExecutor, projectKey, rmName string) (sdk.IDNames, error) {
	query := `SELECT application.id, application.name, application.description, '' AS icon
	FROM application
		JOIN project ON project.id = application.project_id
	WHERE project.projectkey = $1 AND application.vcs_server = $2`
	var idNames sdk.IDNames
	if _, err := db.Select(&idNames, query, projectKey, rmName); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WithStack(err)
	}

	return idNames, nil
}
