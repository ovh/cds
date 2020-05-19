package environment

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// LoadAllByIDs load all environment
func LoadAllByIDs(db gorp.SqlExecutor, ids []int64) ([]sdk.Environment, error) {
	var envs []sdk.Environment

	query := `SELECT environment.id, environment.name, environment.last_modified, environment.from_repository, project.projectkey
		  FROM environment
		  JOIN project ON project.id = environment.project_id
		  WHERE environment.id = ANY($1)
		  ORDER by environment.name`
	rows, err := db.Query(query, pq.Int64Array(ids))
	if err != nil {
		if err == sql.ErrNoRows {
			return envs, sdk.WithStack(sdk.ErrNoEnvironment)
		}
		return envs, sdk.WithStack(err)
	}
	defer rows.Close()

	for rows.Next() {
		var env sdk.Environment
		var lastModified time.Time
		var projectKey string
		if err := rows.Scan(&env.ID, &env.Name, &lastModified, &env.FromRepository, &projectKey); err != nil {
			return envs, sdk.WithStack(err)
		}
		env.LastModified = lastModified.Unix()
		env.ProjectKey = projectKey
		envs = append(envs, env)
	}
	rows.Close()

	for i := range envs {
		if err := loadDependencies(db, &envs[i]); err != nil {
			return envs, err
		}
	}
	return envs, nil
}

// LoadEnvironments load all environment from the given project
func LoadEnvironments(db gorp.SqlExecutor, projectKey string) ([]sdk.Environment, error) {
	var envs []sdk.Environment

	query := `SELECT environment.id, environment.name, environment.last_modified, environment.from_repository
		  FROM environment
		  JOIN project ON project.id = environment.project_id
		  WHERE project.projectKey = $1
		  ORDER by environment.name`
	rows, err := db.Query(query, projectKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return envs, sdk.ErrNoEnvironment
		}
		return envs, sdk.WithStack(err)
	}
	defer rows.Close()

	for rows.Next() {
		var env sdk.Environment
		var lastModified time.Time
		if err := rows.Scan(&env.ID, &env.Name, &lastModified, &env.FromRepository); err != nil {
			return envs, sdk.WithStack(err)
		}
		env.LastModified = lastModified.Unix()
		env.ProjectKey = projectKey
		envs = append(envs, env)
	}
	rows.Close()

	for i := range envs {
		if err := loadDependencies(db, &envs[i]); err != nil {
			return envs, err
		}
	}
	return envs, nil
}

// Lock locks an environment given its ID
func Lock(db gorp.SqlExecutor, projectKey, envName string) error {
	_, err := db.Exec(`
	SELECT *
	FROM environment
	WHERE id in (
		SELECT environment.id FROM environment
		JOIN project ON project.id = environment.project_id
		WHERE project.projectKey = $1 AND environment.name = $2
	) FOR UPDATE SKIP LOCKED
	`, projectKey, envName)
	if err == sql.ErrNoRows {
		return sdk.WithStack(sdk.ErrEnvironmentNotFound)
	}
	return err
}

// LoadEnvironmentByID load the given environment
func LoadEnvironmentByID(db gorp.SqlExecutor, ID int64) (*sdk.Environment, error) {
	if ID == sdk.DefaultEnv.ID {
		return &sdk.DefaultEnv, nil
	}
	var env sdk.Environment
	query := `SELECT environment.id, environment.name, environment.project_id, environment.from_repository
		  	FROM environment
		 	WHERE id = $1`
	if err := db.QueryRow(query, ID).Scan(&env.ID, &env.Name, &env.ProjectID, &env.FromRepository); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WithStack(sdk.ErrEnvironmentNotFound)
		}
		return nil, err
	}
	return &env, loadDependencies(db, &env)
}

// LoadEnvironmentByName load the given environment
func LoadEnvironmentByName(db gorp.SqlExecutor, projectKey, envName string) (*sdk.Environment, error) {
	if envName == "" || envName == sdk.DefaultEnv.Name {
		return &sdk.DefaultEnv, nil
	}

	var env sdk.Environment
	query := `SELECT environment.id, environment.name,  environment.project_id, environment.from_repository, environment.last_modified
		  FROM environment
		  JOIN project ON project.id = environment.project_id
		  WHERE project.projectKey = $1 AND environment.name = $2`
	var lastModified time.Time
	if err := db.QueryRow(query, projectKey, envName).Scan(&env.ID, &env.Name, &env.ProjectID, &env.FromRepository, &lastModified); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WithData(sdk.ErrEnvironmentNotFound, envName)
		}
		return nil, sdk.WithStack(err)
	}
	env.LastModified = lastModified.Unix()
	env.ProjectKey = projectKey
	return &env, sdk.WithStack(loadDependencies(db, &env))
}

// LoadByWorkflowID loads environments from database for a given workflow id
func LoadByWorkflowID(db gorp.SqlExecutor, workflowID int64) ([]sdk.Environment, error) {
	envs := []sdk.Environment{}
	query := `SELECT DISTINCT environment.* FROM environment
	JOIN w_node_context ON w_node_context.environment_id = environment.id
	JOIN w_node ON w_node.id = w_node_context.node_id
	JOIN workflow ON workflow.id = w_node.workflow_id
	WHERE workflow.id = $1`

	if _, err := db.Select(&envs, query, workflowID); err != nil {
		if err == sql.ErrNoRows {
			return envs, nil
		}
		return nil, sdk.WrapError(err, "Unable to load environments linked to workflow id %d", workflowID)
	}

	return envs, nil
}

//Exists checks if an environment already exists on the project
func Exists(db gorp.SqlExecutor, projectKey, envName string) (bool, error) {
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
func CheckDefaultEnv(db gorp.SqlExecutor) error {
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

func loadDependencies(db gorp.SqlExecutor, env *sdk.Environment) error {
	variables, err := LoadAllVariables(db, env.ID)
	if err != nil {
		return sdk.WrapError(err, "Cannot load environment variables")
	}
	env.Variables = variables

	keys, err := LoadAllKeys(db, env.ID)
	if err != nil {
		return sdk.WrapError(err, "loadDependencies> Cannot load environment dependencies")
	}

	env.Keys = keys

	return nil
}

// InsertEnvironment Insert new environment
func InsertEnvironment(db gorp.SqlExecutor, env *sdk.Environment) error {
	query := `INSERT INTO environment (name, project_id, from_repository) VALUES($1, $2, $3) RETURNING id, last_modified`

	rx := sdk.NamePatternRegex
	if !rx.MatchString(env.Name) {
		return sdk.NewError(sdk.ErrInvalidName, fmt.Errorf("Invalid environment name. It should match %s", sdk.NamePattern))
	}

	var lastModified time.Time
	err := db.QueryRow(query, env.Name, env.ProjectID, env.FromRepository).Scan(&env.ID, &lastModified)
	if err != nil {
		pqerr, ok := err.(*pq.Error)
		if ok {
			if pqerr.Code == "23000" || pqerr.Code == gorpmapping.ViolateUniqueKeyPGCode || pqerr.Code == "23514" {
				return sdk.WithStack(sdk.ErrEnvironmentExist)
			}
		}
		return err
	}
	env.LastModified = lastModified.Unix()
	return nil
}

// UpdateEnvironment Update an environment
func UpdateEnvironment(db gorp.SqlExecutor, environment *sdk.Environment) error {
	rx := sdk.NamePatternRegex
	if !rx.MatchString(environment.Name) {
		return sdk.NewError(sdk.ErrInvalidName, fmt.Errorf("Invalid environment name. It should match %s", sdk.NamePattern))
	}

	query := `UPDATE environment SET name=$1, from_repository=$3 WHERE id=$2`
	if _, err := db.Exec(query, environment.Name, environment.ID, environment.FromRepository); err != nil {
		return err
	}
	return nil
}

// DeleteEnvironment Delete the given environment
func DeleteEnvironment(db gorp.SqlExecutor, environmentID int64) error {
	// Delete variables
	if err := DeleteAllVariables(db, environmentID); err != nil {
		return sdk.WrapError(err, "Cannot delete environment variable")
	}

	query := `DELETE FROM environment WHERE id=$1`
	if _, err := db.Exec(query, environmentID); err != nil {
		if err, ok := err.(*pq.Error); ok {
			if err.Code.Name() == "foreign_key_violation" {
				return sdk.WrapError(sdk.ErrEnvironmentCannotBeDeleted, "DeleteEnvironment> Cannot delete environment %d", environmentID)
			}
		}
		return sdk.WrapError(err, "Cannot delete environment %d", environmentID)
	}

	return nil
}

// DeleteAllEnvironment Delete all environment attached to the given project
func DeleteAllEnvironment(db gorp.SqlExecutor, projectID int64) error {
	query := `DELETE FROM environment_variable WHERE environment_id IN (SELECT id FROM environment WHERE project_id = $1)`
	if _, err := db.Exec(query, projectID); err != nil {
		return sdk.WrapError(err, "Cannot delete environment variable")
	}

	query = `DELETE FROM environment WHERE project_id=$1`
	if _, err := db.Exec(query, projectID); err != nil {
		return sdk.WrapError(err, "Cannot delete environment")
	}
	return nil
}

// LoadAllNames returns all environment names
func LoadAllNames(db gorp.SqlExecutor, projID int64) (sdk.IDNames, error) {
	query := `SELECT environment.id, environment.name
			  FROM environment
			  WHERE project_id = $1
			  ORDER BY environment.name`

	var res sdk.IDNames
	if _, err := db.Select(&res, query, projID); err != nil {
		if err == sql.ErrNoRows {
			return res, nil
		}
		return nil, sdk.WithStack(err)
	}

	return res, nil
}
