package environment

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/artifact"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// LoadEnvironments load all environment from the given project
func LoadEnvironments(db database.Querier, projectKey string, loadDeps bool, user *sdk.User) ([]sdk.Environment, error) {
	envs := []sdk.Environment{}

	var rows *sql.Rows
	var err error
	if user.Admin {
		query := `SELECT environment.id, environment.name, environment.last_modified
		  FROM environment
		  JOIN project ON project.id = environment.project_id
		  WHERE project.projectKey = $1
		  ORDER by environment.name`
		rows, err = db.Query(query, projectKey)
	} else {
		query := `SELECT distinct(environment.id), environment.name, environment.last_modified
			  FROM environment
			  JOIN environment_group ON environment.id = environment_group.environment_id
			  JOIN group_user ON environment_group.group_id = group_user.group_id
			  JOIN project ON project.id = environment.project_id
			  WHERE group_user.user_id = $1
			  	AND project.projectKey = $2
			  ORDER by environment.name`
		rows, err = db.Query(query, user.ID, projectKey)
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return envs, sdk.ErrNoEnvironment
		}
		return envs, err
	}
	defer rows.Close()

	for rows.Next() {
		var env sdk.Environment
		var lastModified time.Time
		err = rows.Scan(&env.ID, &env.Name, &lastModified)
		env.LastModified = lastModified.Unix()
		if err != nil {
			return envs, err
		}
		envs = append(envs, env)
	}
	rows.Close()

	for i := range envs {
		if loadDeps {
			err = loadDependencies(db, &envs[i])
			if err != nil {
				return envs, err
			}
		}
	}
	return envs, nil
}

// LoadEnvironmentByName load the given environment
func LoadEnvironmentByName(db database.Querier, projectKey, envName string) (*sdk.Environment, error) {
	var env sdk.Environment
	query := `SELECT environment.id, environment.name
		  FROM environment
		  JOIN project ON project.id = environment.project_id
		  WHERE project.projectKey = $1 AND environment.name = $2`
	if err := db.QueryRow(query, projectKey, envName).Scan(&env.ID, &env.Name); err != nil {
		if err == sql.ErrNoRows {
			return &env, sdk.ErrNoEnvironment
		}
		return &env, err
	}
	return &env, loadDependencies(db, &env)
}

//Exists checks if an environment already exists on the project
func Exists(db database.Querier, projectKey, envName string) (bool, error) {
	var n int
	query := `SELECT count(1)
		  FROM environment
		  JOIN project ON project.id = environment.project_id
		  WHERE project.projectKey = $1 AND environment.name = $2`
	if err := db.QueryRow(query, projectKey, envName).Scan(&n); err != nil {
		return false, err
	}
	return n == 1, nil
}

// CheckDefaultEnv create default env if not exists
func CheckDefaultEnv(db database.Querier) error {
	var env sdk.Environment
	query := `SELECT environment.id, environment.name FROM environment WHERE environment.id = $1`
	err := db.QueryRow(query, sdk.DefaultEnv.ID).Scan(&env.ID, &env.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			query := `INSERT INTO environment (name) VALUES($1) RETURNING id`
			if err1 := db.QueryRow(query, sdk.DefaultEnv.Name).Scan(&env.ID); err1 != nil {
				return err1
			} else if env.ID != sdk.DefaultEnv.ID {
				return fmt.Errorf("CheckDefaultEnv> default env created, but with wrong id. Please check db")
			}
			return nil
		}
		return err
	} else if env.ID != sdk.DefaultEnv.ID || env.Name != sdk.DefaultEnv.Name {
		return fmt.Errorf("CheckDefaultEnv> default env exists, but with wrong id or name. Please check db")
	}
	return nil
}

func loadDependencies(db database.Querier, env *sdk.Environment) error {
	variables, err := GetAllVariableByID(db, env.ID)
	if err != nil {
		return err
	}
	env.Variable = variables
	return loadGroupByEnvironment(db, env)
}

// InsertEnvironment Insert new environment
func InsertEnvironment(db database.Querier, env *sdk.Environment) error {
	query := `INSERT INTO environment (name, project_id) VALUES($1, $2) RETURNING id, last_modified`

	if env.Name == "" {
		return sdk.ErrInvalidName
	}

	var lastModified time.Time
	err := db.QueryRow(query, env.Name, env.ProjectID).Scan(&env.ID, &lastModified)
	if err != nil {
		pqerr, ok := err.(*pq.Error)
		if ok {
			if pqerr.Code == "23000" || pqerr.Code == "23505" || pqerr.Code == "23514" {
				return sdk.ErrEnvironmentExist
			}
		}
		return err
	}
	env.LastModified = lastModified.Unix()
	return nil
}

// UpdateEnvironment Update an environment
func UpdateEnvironment(db database.QueryExecuter, environment *sdk.Environment) error {
	var lastModified time.Time
	query := `UPDATE environment SET name=$1, last_modified=current_timestamp WHERE id=$2 RETURNING last_modified`
	err := db.QueryRow(query, environment.Name, environment.ID).Scan(&lastModified)
	if err != nil {
		return err
	}
	environment.LastModified = lastModified.Unix()
	return nil
}

// DeleteEnvironment Delete the given environment
func DeleteEnvironment(db *sql.Tx, environmentID int64) error {

	// Delete variables
	err := DeleteAllVariable(db, environmentID)
	if err != nil {
		log.Warning("DeleteEnvironment> Cannot delete environment variable: %s\n", err)
		return err
	}

	// Delete groups
	query := `DELETE FROM environment_group WHERE environment_id = $1`
	_, err = db.Exec(query, environmentID)
	if err != nil {
		log.Warning("DeleteEnvironment> Cannot delete environment gorup: %s\n", err)
		return err
	}

	// Delete history
	query = `DELETE FROM pipeline_history where environment_id = $1`
	_, err = db.Exec(query, environmentID)
	if err != nil {
		log.Warning("DeleteEnvironment> Cannot delete environment related history: %s\n", err)
		return err
	}

	// Delete builds
	query = `DELETE FROM build_log where action_build_id IN (
			SELECT id FROM action_build WHERE pipeline_build_id IN (
					SELECT id FROM pipeline_build WHERE environment_id = $1
			))`
	_, err = db.Exec(query, environmentID)
	if err != nil {
		log.Warning("DeleteEnvironment> Cannot delete environment related builds: %s\n", err)
		return err
	}

	query = `DELETE FROM action_build WHERE pipeline_build_id
			IN (SELECT id FROM pipeline_build WHERE environment_id = $1)`
	_, err = db.Exec(query, environmentID)
	if err != nil {
		log.Warning("DeleteEnvironment> Cannot delete environment related builds: %s\n", err)
		return err
	}

	query = `DELETE FROM pipeline_build where environment_id = $1`
	_, err = db.Exec(query, environmentID)
	if err != nil {
		log.Warning("DeleteEnvironment> Cannot delete environment related builds: %s\n", err)
		return err
	}

	// Delete artifacts related to this environments
	query = `SELECT id FROM artifact where environment_id = $1`
	rows, err := db.Query(query, environmentID)
	if err != nil {
		return fmt.Errorf("DeleteEnvironment> Cannot load related artifacts: %s", err)
	}
	defer rows.Close()
	var ids []int64
	var id int64
	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			return fmt.Errorf("DeleteEnvironment> cannot scan artifact id: %s", err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		err = artifact.DeleteArtifact(db, id)
		if err != nil {
			return fmt.Errorf("DeleteEnvironment> Cannot delete artifact: %s", err)
		}
	}

	//Delete application_pipeline_notif to this environments
	query = `DELETE FROM application_pipeline_notif WHERE environment_id = $1`
	_, err = db.Exec(query, environmentID)
	if err != nil {
		log.Warning("DeleteEnvironment> Cannot delete environment application_pipeline_notif: %s\n", err)
		return err
	}

	// FINALY delete environment
	query = `DELETE FROM environment WHERE id=$1`
	_, err = db.Exec(query, environmentID)
	if err != nil {
		log.Warning("DeleteEnvironment> Cannot delete environment: %s\n", err)
		return err
	}
	return nil
}

// DeleteAllEnvironment Delete all environment attached to the given project
func DeleteAllEnvironment(db database.Executer, projectID int64) error {

	query := `DELETE FROM environment_variable WHERE environment_id IN (SELECT id FROM environment WHERE project_id = $1)`
	_, err := db.Exec(query, projectID)
	if err != nil {
		log.Warning("DeleteAllEnvironment> Cannot delete environment variable: %s\n", err)
		return err
	}

	// Delete groups
	query = `DELETE FROM environment_group WHERE environment_id IN (SELECT id FROM environment WHERE project_id = $1)`
	_, err = db.Exec(query, projectID)
	if err != nil {
		log.Warning("DeleteEnvironment> Cannot delete environment group: %s\n", err)
		return err
	}

	//Delete application_pipeline_notif to this environments
	query = `DELETE FROM application_pipeline_notif WHERE environment_id  IN (SELECT id FROM environment WHERE project_id = $1)`
	_, err = db.Exec(query, projectID)
	if err != nil {
		log.Warning("DeleteEnvironment> Cannot delete environment application_pipeline_notif: %s\n", err)
		return err
	}

	query = `DELETE FROM environment WHERE project_id=$1`
	_, err = db.Exec(query, projectID)
	if err != nil {
		log.Warning("DeleteEnvironment> Cannot delete environment: %s\n", err)
		return err
	}
	return nil
}

func loadGroupByEnvironment(db database.Querier, environment *sdk.Environment) error {
	query := `SELECT "group".id, "group".name, environment_group.role FROM "group"
	 		  JOIN environment_group ON environment_group.group_id = "group".id
	 		  WHERE environment_group.environment_id = $1 ORDER BY "group".name ASC`

	rows, err := db.Query(query, environment.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var group sdk.Group
		var perm int
		err = rows.Scan(&group.ID, &group.Name, &perm)
		if err != nil {
			return err
		}
		environment.EnvironmentGroups = append(environment.EnvironmentGroups, sdk.GroupPermission{
			Group:      group,
			Permission: perm,
		})
	}
	return nil
}

// LoadEnvironmentByGroup loads all environments where group has access
func LoadEnvironmentByGroup(db database.Querier, group *sdk.Group) error {
	query := `SELECT project.projectKey,
			 environment.id,
	                 environment.name,
	                 environment_group.role
	          FROM environment
	          JOIN environment_group ON environment_group.environment_id = environment.id
	 	  JOIN project ON environment.project_id = project.id
	 	  WHERE environment_group.group_id = $1
	 	  ORDER BY environment.name ASC`
	rows, err := db.Query(query, group.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var environment sdk.Environment
		var perm int
		err = rows.Scan(&environment.ProjectKey, &environment.ID, &environment.Name, &perm)
		if err != nil {
			return err
		}
		group.EnvironmentGroups = append(group.EnvironmentGroups, sdk.EnvironmentGroup{
			Environment: environment,
			Permission:  perm,
		})
	}
	return nil
}
