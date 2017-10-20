package repositoriesmanager

import (
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
)

//LoadAll Load all RepositoriesManager from the database
func LoadAll(db *gorp.DbMap, store cache.Store) ([]string, error) {
	serviceDAO := services.NewRepository(func() *gorp.DbMap { return db }, store)
	srvs, err := serviceDAO.FindByType("vcs")
	if err != nil {
		return nil, sdk.WrapError(err, "repositoriesmanager.LoadAll> Unable to load services")
	}

	vcsServers := []string{}
	if _, err := services.DoJSONRequest(srvs, "GET", "/vcs", nil, &vcsServers); err != nil {
		return nil, err
	}
	return vcsServers, nil
}

//LoadForProject loads a repomanager link for a project
func LoadForProject(db gorp.SqlExecutor, projectKey string, repomanagerName string) (*sdk.ProjectVCSServer, error) {
	vcsServerStr, err := db.SelectNullStr("select vcs_servers from project where projectkey = $1", projectKey)
	if err != nil {
		return nil, err
	}

	if !vcsServerStr.Valid {
		return nil, sdk.ErrNotFound
	}

	clearVCSServer, err := secret.Decrypt([]byte(vcsServerStr.String))
	vcsServer := &sdk.ProjectVCSServer{}

	if err := json.Unmarshal(clearVCSServer, vcsServer); err != nil {
		return nil, err
	}

	return vcsServer, nil
}

//InsertForApplication associates a repositories manager with an application
func InsertForApplication(db gorp.SqlExecutor, app *sdk.Application, projectKey string) error {
	query := `UPDATE application SET repositories_manager_id =  $1, repo_fullname = $2 WHERE id = $3`
	if _, err := db.Exec(query, app.RepositoriesManager, app.RepositoryFullname, app.ID); err != nil {
		return err
	}
	return nil
}

//DeleteForApplication removes association between  a repositories manager and an application
//it deletes the corresponding line in repositories_manager_project
func DeleteForApplication(db gorp.SqlExecutor, app *sdk.Application) error {
	query := `UPDATE application SET repositories_manager_id = NULL, repo_fullname = '' WHERE id = $3`
	if _, err := db.Exec(query, app.ID); err != nil {
		return err
	}
	return nil
}
