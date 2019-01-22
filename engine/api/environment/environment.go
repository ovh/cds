package environment

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/sdk"
)

// LoadEnvironments load all environment from the given project
func LoadEnvironments(db gorp.SqlExecutor, projectKey string, loadDeps bool, u *sdk.User) ([]sdk.Environment, error) {
	envs := []sdk.Environment{}

	var rows *sql.Rows
	var err error
	if u == nil || u.Admin {
		query := `SELECT environment.id, environment.name, environment.last_modified, 7 as "perm"
		  FROM environment
		  JOIN project ON project.id = environment.project_id
		  WHERE project.projectKey = $1
		  ORDER by environment.name`
		rows, err = db.Query(query, projectKey)
	} else {
		query := `SELECT environment.id, environment.name, environment.last_modified, max(environment_group.role) as "perm"
			  FROM environment
			  JOIN environment_group ON environment.id = environment_group.environment_id
			  JOIN group_user ON environment_group.group_id = group_user.group_id
			  JOIN project ON project.id = environment.project_id
			  WHERE group_user.user_id = $1
			  AND project.projectKey = $2
			  GROUP BY environment.id, environment.name, environment.last_modified
			  ORDER by environment.name`
		rows, err = db.Query(query, u.ID, projectKey)
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
		err = rows.Scan(&env.ID, &env.Name, &lastModified, &env.Permission)
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

// LockByID locks an environment given its ID
func LockByID(db gorp.SqlExecutor, envID int64) error {
	_, err := db.Exec(`
	SELECT *
	FROM environment
	WHERE id = $1 FOR UPDATE NOWAIT
	`, envID)
	if err == sql.ErrNoRows {
		return sdk.ErrNoEnvironment
	}
	return err
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
	) FOR UPDATE NOWAIT
	`, projectKey, envName)
	if err == sql.ErrNoRows {
		return sdk.ErrNoEnvironment
	}
	return err
}

// LoadEnvironmentByID load the given environment
func LoadEnvironmentByID(db gorp.SqlExecutor, ID int64) (*sdk.Environment, error) {
	if ID == sdk.DefaultEnv.ID {
		return &sdk.DefaultEnv, nil
	}
	var env sdk.Environment
	query := `SELECT environment.id, environment.name, environment.project_id
		  	FROM environment
		 	WHERE id = $1`
	if err := db.QueryRow(query, ID).Scan(&env.ID, &env.Name, &env.ProjectID); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNoEnvironment
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
	query := `SELECT environment.id, environment.name,  environment.project_id
		  FROM environment
		  JOIN project ON project.id = environment.project_id
		  WHERE project.projectKey = $1 AND environment.name = $2`
	if err := db.QueryRow(query, projectKey, envName).Scan(&env.ID, &env.Name, &env.ProjectID); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNoEnvironment
		}
		return nil, err
	}
	return &env, loadDependencies(db, &env)
}

// LoadByWorkflowID loads environments from database for a given workflow id
func LoadByWorkflowID(db gorp.SqlExecutor, workflowID int64) ([]sdk.Environment, error) {
	envs := []sdk.Environment{}
	query := `SELECT DISTINCT environment.* FROM environment
	JOIN workflow_node_context ON workflow_node_context.environment_id = environment.id
	JOIN workflow_node ON workflow_node.id = workflow_node_context.workflow_node_id
	JOIN workflow ON workflow.id = workflow_node.workflow_id
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
	variables, err := GetAllVariableByID(db, env.ID)
	if err != nil {
		return sdk.WrapError(err, "Cannot load environment variables")
	}
	env.Variable = variables

	if errK := LoadAllKeys(db, env); errK != nil {
		return sdk.WrapError(errK, "loadDependencies> Cannot load environment dependencies")
	}

	return loadGroupByEnvironment(db, env)
}

// InsertEnvironment Insert new environment
func InsertEnvironment(db gorp.SqlExecutor, env *sdk.Environment) error {
	query := `INSERT INTO environment (name, project_id) VALUES($1, $2) RETURNING id, last_modified`

	rx := sdk.NamePatternRegex
	if !rx.MatchString(env.Name) {
		return sdk.NewError(sdk.ErrInvalidName, fmt.Errorf("Invalid environment name. It should match %s", sdk.NamePattern))
	}

	var lastModified time.Time
	err := db.QueryRow(query, env.Name, env.ProjectID).Scan(&env.ID, &lastModified)
	if err != nil {
		pqerr, ok := err.(*pq.Error)
		if ok {
			if pqerr.Code == "23000" || pqerr.Code == gorpmapping.ViolateUniqueKeyPGCode || pqerr.Code == "23514" {
				return sdk.ErrEnvironmentExist
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

	query := `UPDATE environment SET name=$1 WHERE id=$2`
	if _, err := db.Exec(query, environment.Name, environment.ID); err != nil {
		return err
	}
	return nil
}

// DeleteEnvironment Delete the given environment
func DeleteEnvironment(db gorp.SqlExecutor, environmentID int64) error {
	// Delete variables
	if err := DeleteAllVariable(db, environmentID); err != nil {
		return sdk.WrapError(err, "Cannot delete environment variable")
	}

	// Delete groups
	query := `DELETE FROM environment_group WHERE environment_id = $1`
	if _, err := db.Exec(query, environmentID); err != nil {
		return sdk.WrapError(err, "Cannot delete environment gorup")
	}

	// FINALLY delete environment
	query = `DELETE FROM environment WHERE id=$1`
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

	// Delete groups
	query = `DELETE FROM environment_group WHERE environment_id IN (SELECT id FROM environment WHERE project_id = $1)`
	if _, err := db.Exec(query, projectID); err != nil {
		return sdk.WrapError(err, "Cannot delete environment group")
	}

	query = `DELETE FROM environment WHERE project_id=$1`
	if _, err := db.Exec(query, projectID); err != nil {
		return sdk.WrapError(err, "Cannot delete environment")
	}
	return nil
}

func loadGroupByEnvironment(db gorp.SqlExecutor, environment *sdk.Environment) error {
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
func LoadEnvironmentByGroup(db gorp.SqlExecutor, groupID int64) ([]sdk.EnvironmentGroup, error) {
	res := []sdk.EnvironmentGroup{}
	query := `SELECT project.projectKey,
			 	environment.id,
	        	environment.name,
	    		environment_group.role
	          FROM environment
	          JOIN environment_group ON environment_group.environment_id = environment.id
	 	  JOIN project ON environment.project_id = project.id
	 	  WHERE environment_group.group_id = $1
	 	  ORDER BY environment.name ASC`
	rows, err := db.Query(query, groupID)
	if err != nil {
		return res, err
	}
	defer rows.Close()

	for rows.Next() {
		var environment sdk.Environment
		var perm int
		err = rows.Scan(&environment.ProjectKey, &environment.ID, &environment.Name, &perm)
		if err != nil {
			return nil, err
		}
		res = append(res, sdk.EnvironmentGroup{
			Environment: environment,
			Permission:  perm,
		})
	}
	return res, nil
}

// Permission Get the permission for the given environment and user
func Permission(key string, envName string, u *sdk.User) int {
	if u == nil {
		return 0
	}

	if u.Admin {
		return permission.PermissionReadWriteExecute
	}

	return u.Permissions.EnvironmentsPerm[sdk.UserPermissionKey(key, envName)]
}

// CountEnvironmentByVarValue counts how many time a pattern is in variable value for the given project
func CountEnvironmentByVarValue(db gorp.SqlExecutor, projectKey string, value string) ([]string, error) {
	query := `
		SELECT DISTINCT environment.name
		FROM environment_variable
		JOIN environment ON environment.id = environment_variable.environment_id
		JOIN project ON project.id = environment.project_id
		WHERE value like $2 AND project.projectkey = $1;
	`

	var envsName []string
	if _, err := db.Select(&envsName, query, projectKey, fmt.Sprintf("%%%s%%", value)); err != nil {
		return nil, sdk.WrapError(err, "Unable to count usage")
	}
	return envsName, nil
}

// AddKeyPairToEnvironment generate a ssh key pair and add them as env variables
func AddKeyPairToEnvironment(db gorp.SqlExecutor, envID int64, keyname string, u *sdk.User) error {
	if !strings.HasPrefix(keyname, "env-") {
		keyname = "env-" + keyname
	}

	k, errGenerate := keys.GenerateSSHKey(keyname)
	if errGenerate != nil {
		return errGenerate
	}

	v := &sdk.Variable{
		Name:  keyname,
		Type:  sdk.KeyVariable,
		Value: k.Private,
	}

	if err := InsertVariable(db, envID, v, u); err != nil {
		return err
	}

	p := &sdk.Variable{
		Name:  keyname + ".pub",
		Type:  sdk.TextVariable,
		Value: k.Public,
	}

	return InsertVariable(db, envID, p, u)
}
