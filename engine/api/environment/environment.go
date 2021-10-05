package environment

import (
	"database/sql"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

// LoadAllByIDs load all environment
func LoadAllByIDs(db gorp.SqlExecutor, ids []int64) ([]sdk.Environment, error) {
	var envs []sdk.Environment

	query := `SELECT environment.id, environment.name, environment.project_id, environment.created,
      environment.last_modified, environment.from_repository, project.projectkey
		FROM environment
		JOIN project ON project.id = environment.project_id
		WHERE environment.id = ANY($1)
    ORDER by environment.name
  `
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
		if err := rows.Scan(&env.ID, &env.Name, &env.ProjectID, &env.Created,
			&env.LastModified, &env.FromRepository, &env.ProjectKey); err != nil {
			return envs, sdk.WithStack(err)
		}
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

	query := `
    SELECT environment.id, environment.name, environment.project_id, environment.created,
      environment.last_modified, environment.from_repository, project.projectkey
		FROM environment
		JOIN project ON project.id = environment.project_id
		WHERE project.projectKey = $1
    ORDER by environment.name
  `
	rows, err := db.Query(query, projectKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return envs, sdk.WithStack(sdk.ErrNoEnvironment)
		}
		return envs, sdk.WithStack(err)
	}
	defer rows.Close()

	for rows.Next() {
		var env sdk.Environment
		if err := rows.Scan(&env.ID, &env.Name, &env.ProjectID, &env.Created,
			&env.LastModified, &env.FromRepository, &env.ProjectKey); err != nil {
			return envs, sdk.WithStack(err)
		}
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
	var env sdk.Environment
	query := `
    SELECT environment.id, environment.name, environment.project_id, environment.created,
      environment.last_modified, environment.from_repository, project.projectkey
    FROM environment
    JOIN project ON project.id = environment.project_id
    WHERE environment.id = $1
  `
	if err := db.QueryRow(query, ID).Scan(&env.ID, &env.Name, &env.ProjectID, &env.Created,
		&env.LastModified, &env.FromRepository, &env.ProjectKey); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WithStack(sdk.ErrEnvironmentNotFound)
		}
		return nil, err
	}
	return &env, loadDependencies(db, &env)
}

// LoadEnvironmentByName load the given environment
func LoadEnvironmentByName(db gorp.SqlExecutor, projectKey, envName string) (*sdk.Environment, error) {
	var env sdk.Environment
	query := `
    SELECT environment.id, environment.name, environment.project_id, environment.created,
      environment.last_modified, environment.from_repository, project.projectkey
    FROM environment
    JOIN project ON project.id = environment.project_id
    WHERE project.projectKey = $1 AND environment.name = $2
  `
	if err := db.QueryRow(query, projectKey, envName).Scan(&env.ID, &env.Name, &env.ProjectID, &env.Created,
		&env.LastModified, &env.FromRepository, &env.ProjectKey); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WithData(sdk.ErrEnvironmentNotFound, envName)
		}
		return nil, sdk.WithStack(err)
	}
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
	query := `INSERT INTO environment (name, project_id, from_repository) VALUES($1, $2, $3) RETURNING id, created, last_modified`

	rx := sdk.NamePatternRegex
	if !rx.MatchString(env.Name) {
		return sdk.NewErrorFrom(sdk.ErrInvalidName, "environment name should match pattern %s", sdk.NamePattern)
	}

	err := db.QueryRow(query, env.Name, env.ProjectID, env.FromRepository).Scan(&env.ID, &env.Created, &env.LastModified)
	if err != nil {
		pqerr, ok := err.(*pq.Error)
		if ok {
			if pqerr.Code == "23000" || pqerr.Code == gorpmapper.ViolateUniqueKeyPGCode || pqerr.Code == "23514" {
				return sdk.WithStack(sdk.ErrEnvironmentExist)
			}
		}
		return err
	}
	return nil
}

// UpdateEnvironment Update an environment
func UpdateEnvironment(db gorp.SqlExecutor, env *sdk.Environment) error {
	rx := sdk.NamePatternRegex
	if !rx.MatchString(env.Name) {
		return sdk.NewErrorFrom(sdk.ErrInvalidName, "environment name should match pattern %s", sdk.NamePattern)
	}

	env.LastModified = time.Now()
	query := `UPDATE environment SET name=$1, from_repository=$2, last_modified=$3 WHERE id=$4`
	if _, err := db.Exec(query, env.Name, env.FromRepository, env.LastModified, env.ID); err != nil {
		return sdk.WithStack(err)
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

// LoadAllNamesByFromRepository returns all environment names for a repository
func LoadAllNamesByFromRepository(db gorp.SqlExecutor, projID int64, fromRepository string) (sdk.IDNames, error) {
	if fromRepository == "" {
		return nil, sdk.WithData(sdk.ErrUnknownError, "could not call LoadAllNamesByFromRepository with empty fromRepository")
	}
	query := `SELECT environment.id, environment.name
			  FROM environment
			  WHERE project_id = $1 AND from_repository = $2
			  ORDER BY environment.name`

	var res sdk.IDNames
	if _, err := db.Select(&res, query, projID, fromRepository); err != nil {
		if err == sql.ErrNoRows {
			return res, nil
		}
		return nil, sdk.WrapError(err, "environment.LoadAllNamesByFromRepository")
	}

	return res, nil
}

// ResetFromRepository reset fromRepository for all environments using the same fromRepository in a given project
func ResetFromRepository(db gorp.SqlExecutor, projID int64, fromRepository string) error {
	if fromRepository == "" {
		return sdk.WithData(sdk.ErrUnknownError, "could not call LoadAllNamesByFromRepository with empty fromRepository")
	}
	query := `UPDATE environment SET from_repository='' WHERE project_id = $1 AND from_repository = $2`
	_, err := db.Exec(query, projID, fromRepository)
	return sdk.WithStack(err)
}
