package repositoriesmanager

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

//LoadAll Load all RepositoriesManager from the database
func LoadAll(db *sql.DB) ([]sdk.RepositoriesManager, error) {
	rms := []sdk.RepositoriesManager{}
	query := `SELECT id, type, name, url, data FROM repositories_manager`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var t, name, URL, data string

		err = rows.Scan(&id, &t, &name, &URL, &data)
		if err != nil {
			log.Warning("LoadAll> Error %s", err)
		}
		rm, err := New(sdk.RepositoriesManagerType(t), id, name, URL, map[string]string{}, data)
		if err != nil {
			log.Warning("LoadAll> Error %s", err)
		}
		if rm != nil {
			rms = append(rms, *rm)
		}
	}
	return rms, nil
}

//LoadByID loads the specified RepositoriesManager from the database
func LoadByID(db *sql.DB, id int64) (*sdk.RepositoriesManager, error) {
	var rm *sdk.RepositoriesManager
	var rmid int64
	var t, name, URL, data string

	query := `SELECT id, type, name, url, data FROM repositories_manager WHERE id=$1`
	if err := db.QueryRow(query, id).Scan(&rmid, &t, &name, &URL, &data); err != nil {
		log.Warning("LoadByID> Error %s", err)
		return nil, err
	}

	rm, err := New(sdk.RepositoriesManagerType(t), rmid, name, URL, map[string]string{}, data)
	if err != nil {
		log.Warning("LoadByID> Error %s", err)
	}
	return rm, nil
}

//LoadByName loads the specified RepositoriesManager from the database
func LoadByName(db *sql.DB, repositoriesManagerName string) (*sdk.RepositoriesManager, error) {
	var rm *sdk.RepositoriesManager
	var id int64
	var t, name, URL, data string

	query := `SELECT id, type, name, url, data FROM repositories_manager WHERE name=$1`
	if err := db.QueryRow(query, repositoriesManagerName).Scan(&id, &t, &name, &URL, &data); err != nil {
		log.Warning("LoadByName> Error %s", err)
		return nil, err
	}

	rm, err := New(sdk.RepositoriesManagerType(t), id, name, URL, map[string]string{}, data)
	if err != nil {
		log.Warning("LoadByName> Error %s", err)
	}
	return rm, nil
}

//LoadForProject load the specified repositorymanager for the project
func LoadForProject(db database.Querier, projectkey, repositoriesManagerName string) (*sdk.RepositoriesManager, error) {
	query := `SELECT 	repositories_manager.id,
										repositories_manager.type,
										repositories_manager.name,
										repositories_manager.url,
										repositories_manager.data
						FROM 		repositories_manager
						JOIN 	  repositories_manager_project ON repositories_manager.id = repositories_manager_project.id_repositories_manager
						JOIN	  project ON repositories_manager_project.id_project = project.id
						WHERE 	project.projectkey = $1
						and			repositories_manager.name = $2
						`

	var id int64
	var t, name, URL, data string
	if err := db.QueryRow(query, projectkey, repositoriesManagerName).Scan(&id, &t, &name, &URL, &data); err != nil {
		return nil, err
	}
	rm, err := New(sdk.RepositoriesManagerType(t), id, name, URL, map[string]string{}, data)
	if err != nil {
		log.Warning("LoadForProject> Error %s", err)
	}

	return rm, nil
}

//LoadAllForProject Load RepositoriesManager for a project from the database
func LoadAllForProject(db *sql.DB, projectkey string) ([]sdk.RepositoriesManager, error) {
	rms := []sdk.RepositoriesManager{}
	query := `SELECT repositories_manager.id,
			 repositories_manager.type,
			 repositories_manager.name,
			 repositories_manager.url,
			 repositories_manager.data
		  FROM 	 repositories_manager
		  JOIN 	 repositories_manager_project ON repositories_manager.id = repositories_manager_project.id_repositories_manager
		  JOIN	 project ON repositories_manager_project.id_project = project.id
		  WHERE  project.projectkey = $1 AND repositories_manager_project.data is not null
						`
	rows, err := db.Query(query, projectkey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var t, name, URL, data string

		err = rows.Scan(&id, &t, &name, &URL, &data)
		if err != nil {
			log.Warning("LoadAllForProject> Error %s", err)
			return rms, nil
		}
		rm, err := New(sdk.RepositoriesManagerType(t), id, name, URL, map[string]string{}, data)
		if err != nil {
			log.Warning("LoadAllForProject> Error %s", err)
			return rms, nil
		}
		if rm != nil {
			rms = append(rms, *rm)
		}
	}
	return rms, nil
}

//Insert insert a new InsertRepositoriesManager in database
//FIXME: Invalid name: it can only contain lowercase letters, numbers, dots or dashes, and run between 1 and 99 characters long not valid
func Insert(db *sql.DB, rm *sdk.RepositoriesManager) error {
	query := `INSERT INTO repositories_manager (type, name, url, data) VALUES ($1, $2, $3, $4) RETURNING id`
	if err := db.QueryRow(query, string(rm.Type), rm.Name, rm.URL, rm.Consumer.Data()).Scan(&rm.ID); err != nil {
		return err
	}
	return nil
}

//Update update repositories_manager url and data only
func Update(db *sql.DB, rm *sdk.RepositoriesManager) error {
	query := `UPDATE 	repositories_manager
						SET			url = $1,
						 				data = 	$2
						WHERE 	id = $3
						RETURNING id`
	if err := db.QueryRow(query, rm.URL, rm.Consumer.Data(), rm.ID).Scan(&rm.ID); err != nil {
		return err
	}
	return nil
}

//InsertForProject associates a repositories manager with a project
func InsertForProject(db database.QueryExecuter, rm *sdk.RepositoriesManager, projectKey string) (time.Time, error) {
	var lastModified time.Time
	query := `INSERT INTO
							repositories_manager_project (id_repositories_manager, id_project)
						VALUES (
							$1,
							(select id from project where projectkey = $2)
						)`

	_, err := db.Exec(query, rm.ID, projectKey)
	if err != nil {
		return lastModified, err
	}
	// Update project
	query = `
		UPDATE project 
		SET last_modified = current_timestamp
		WHERE projectkey = $1 RETURNING last_modified
	`
	if err = db.QueryRow(query, projectKey).Scan(&lastModified); err != nil {
		return lastModified, err
	}
	return lastModified, nil
}

//DeleteForProject removes association between  a repositories manager and a project
//it deletes the corresponding line in repositories_manager_project
func DeleteForProject(db database.QueryExecuter, rm *sdk.RepositoriesManager, project *sdk.Project) error {
	query := `DELETE 	FROM  repositories_manager_project
						WHERE 	id_repositories_manager = $1
						AND 		id_project IN (
							select id from project where projectkey = $2
						)`

	_, err := db.Exec(query, rm.ID, project.Key)
	if err != nil {
		return err
	}
	// Update project
	query = `
		UPDATE project 
		SET last_modified = current_timestamp
		WHERE projectkey = $1 RETURNING last_modified
	`
	if err = db.QueryRow(query, project.Key).Scan(&project.LastModified); err != nil {
		return err
	}
	return nil
}

//SaveDataForProject updates the jsonb value computed at the end the oauth process
func SaveDataForProject(db *sql.DB, rm *sdk.RepositoriesManager, projectKey string, data map[string]string) error {
	query := `UPDATE 	repositories_manager_project
						SET 		data = $1
						WHERE 	id_repositories_manager = $2
						AND 		id_project IN (
							select id from project where projectkey = $3
						)`

	b, _ := json.Marshal(data)
	_, err := db.Exec(query, string(b), rm.ID, projectKey)
	if err != nil {
		return err
	}
	// Update project
	query = `
		UPDATE project 
		SET last_modified = current_timestamp
		WHERE projectkey = $1
	`
	if _, err = db.Exec(query, projectKey); err != nil {
		return err
	}
	return nil
}

//AuthorizedClient returns instance of client with the granted token
func AuthorizedClient(db database.Querier, projectKey, rmName string) (sdk.RepositoriesManagerClient, error) {

	rm, err := LoadForProject(db, projectKey, rmName)
	if err != nil {
		return nil, err
	}

	var data string
	query := `SELECT 	repositories_manager_project.data
			FROM 	repositories_manager_project
			JOIN	project ON repositories_manager_project.id_project = project.id
			JOIN 	repositories_manager on repositories_manager_project.id_repositories_manager = repositories_manager.id
			WHERE 	project.projectkey = $1
			AND		repositories_manager.name = $2`

	if err := db.QueryRow(query, projectKey, rmName).Scan(&data); err != nil {
		return nil, err
	}

	var clientData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &clientData); err != nil {
		return nil, err
	}

	if len(clientData) > 0 && clientData["access_token"] != nil && clientData["access_token_secret"] != nil {
		return rm.Consumer.GetAuthorized(clientData["access_token"].(string), clientData["access_token_secret"].(string))
	}

	return nil, sdk.ErrNoReposManagerClientAuth

}

//InsertForApplication associates a repositories manager with an application
func InsertForApplication(db database.QueryExecuter, app *sdk.Application, projectKey string) error {
	query := `UPDATE application
						SET
							repositories_manager_id =  $1,
							repo_fullname = $2,
							last_modified = current_timestamp
						WHERE
							id = $3
						RETURNING last_modified
						`

	var lastModified time.Time
	err := db.QueryRow(query, app.RepositoriesManager.ID, app.RepositoryFullname, app.ID).Scan(&lastModified)
	if err != nil {
		return err
	}
	app.LastModified = lastModified.Unix()

	k := cache.Key("application", projectKey, "*"+app.Name+"*")
	cache.DeleteAll(k)
	return nil
}

//DeleteForApplication removes association between  a repositories manager and an application
//it deletes the corresponding line in repositories_manager_project
func DeleteForApplication(db database.QueryExecuter, projectKey string, app *sdk.Application) error {
	query := `UPDATE application
						SET
							repositories_manager_id =  NULL,
							repo_fullname = NULL,
							last_modified = current_timestamp
						WHERE
							id = $1
						RETURNING last_modified
						`
	var lastModified time.Time
	err := db.QueryRow(query, app.ID).Scan(&lastModified)
	if err != nil {
		return err
	}
	app.LastModified = lastModified.Unix()

	k := cache.Key("application", projectKey, "*"+app.Name+"*")
	cache.DeleteAll(k)
	return nil
}

//CheckApplicationIsAttached check if the application is properly attached
func CheckApplicationIsAttached(db database.Querier, rmName, projectKey, applicationName string) (bool, error) {
	query := ` SELECT 1
						 FROM 	application
						 JOIN	  project ON application.project_id = project.id
						 JOIN 	repositories_manager ON repositories_manager.id = application.repositories_manager_id
						 WHERE 	project.projectkey = $1
						 AND 		application.name = $2
						 AND 		repositories_manager.name = $3`
	var found int
	err := db.QueryRow(query, projectKey, applicationName, rmName).Scan(&found)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}
