package workflow

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// LoadAll loads all workflows for a project. All users in a project can list all workflows in a project
func LoadAll(db gorp.SqlExecutor, projectKey string) ([]sdk.Workflow, error) {
	res := []sdk.Workflow{}
	dbRes := []Workflow{}

	query := `
		select workflow.* 
		from workflow
		join project on project.id = workflow.project_id
		where project.projectkey = $1
		order by workflow.name asc`

	if _, err := db.Select(&dbRes, query, projectKey); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrWorkflowNotFound
		}
		return nil, sdk.WrapError(err, "LoadAll> Unable to load workflows project %s", projectKey)
	}

	for _, w := range dbRes {
		w.ProjectKey = projectKey
		res = append(res, sdk.Workflow(w))
	}

	return res, nil
}

// Load loads a workflow for a given user (ie. checking permissions)
func Load(db gorp.SqlExecutor, projectKey, name string, u *sdk.User) (*sdk.Workflow, error) {
	query := `
		select workflow.* 
		from workflow
		join project on project.id = workflow.project_id
		where project.projectkey = $1
		and workflow.name = $2`
	res, err := load(db, u, query, projectKey, name)
	if err != nil {
		return nil, sdk.WrapError(err, "Load> Unable to load workflow %s in project %s", name, projectKey)
	}
	return res, nil
}

// LoadByID loads a workflow for a given user (ie. checking permissions)
func LoadByID(db gorp.SqlExecutor, id int64, u *sdk.User) (*sdk.Workflow, error) {
	query := `
		select * 
		from workflow
		where id = $1`
	res, err := load(db, u, query, id)
	if err != nil {
		return nil, sdk.WrapError(err, "Load> Unable to load workflow %d", id)
	}
	return res, nil
}

func load(db gorp.SqlExecutor, u *sdk.User, query string, args ...interface{}) (*sdk.Workflow, error) {
	t0 := time.Now()
	dbRes := Workflow{}
	if err := db.SelectOne(&dbRes, query, args...); err != nil {
		return nil, sdk.WrapError(err, "Load> Unable to load workflow")
	}

	res := sdk.Workflow(dbRes)
	res.ProjectKey, _ = db.SelectStr("select projectkey from project where id = $1", res.ProjectID)
	if err := loadWorkflowRoot(db, &res, u); err != nil {
		return nil, sdk.WrapError(err, "Load> Unable to load workflow root")
	}

	joins, errJ := loadJoins(db, &res, u)
	if errJ != nil {
		return nil, sdk.WrapError(errJ, "Load> Unable to load workflow joins")
	}

	res.Joins = joins

	delta := time.Since(t0).Seconds()

	log.Debug("Load> Load workflow (%s/%s)%d took %.3f seconds", res.ProjectKey, res.Name, res.ID, delta)
	w := &res
	Sort(w)
	return w, nil
}

func loadWorkflowRoot(db gorp.SqlExecutor, w *sdk.Workflow, u *sdk.User) error {
	var err error
	w.Root, err = loadNode(db, w, w.RootID, u)
	if err != nil {
		if err == sdk.ErrWorkflowNodeNotFound {
			log.Debug("Load> Unable to load root %d for workflow %d", w.RootID, w.ID)
			return nil
		}
		return sdk.WrapError(err, "Load> Unable to load workflow root %d", w.RootID)
	}
	return nil
}

// Insert inserts a new workflow
func Insert(db gorp.SqlExecutor, w *sdk.Workflow, u *sdk.User) error {
	if err := IsValid(db, w, u); err != nil {
		return err
	}

	w.LastModified = time.Now()
	if err := db.QueryRow("INSERT INTO workflow (name, description, project_id) VALUES ($1, $2, $3) RETURNING id", w.Name, w.Description, w.ProjectID).Scan(&w.ID); err != nil {
		return sdk.WrapError(err, "Insert> Unable to insert workflow %s/%s", w.ProjectKey, w.Name)
	}

	if w.Root == nil {
		return sdk.ErrWorkflowInvalidRoot
	}

	if err := insertOrUpdateNode(db, w, w.Root, u, false); err != nil {
		return sdk.WrapError(err, "Insert> Unable to insert workflow root node")
	}

	if _, err := db.Exec("UPDATE workflow SET root_node_id = $2 WHERE id = $1", w.ID, w.Root.ID); err != nil {
		return sdk.WrapError(err, "Insert> Unable to insert workflow (%#v, %d)", w.Root, w.ID)
	}

	for _, j := range w.Joins {
		if err := insertOrUpdateJoin(db, w, &j, u); err != nil {
			return sdk.WrapError(err, "Insert> Unable to insert update workflow(%d) join (%#v)", w.ID, j)
		}
	}

	return updateLastModified(db, w, u)
}

// Update updates a workflow
func Update(db gorp.SqlExecutor, w *sdk.Workflow, u *sdk.User) error {
	if err := IsValid(db, w, u); err != nil {
		return err
	}

	w.LastModified = time.Now()
	dbw := Workflow(*w)
	if _, err := db.Update(&dbw); err != nil {
		return sdk.WrapError(err, "Update> Unable to update workflow")
	}
	if w.Root != nil {
		return insertOrUpdateNode(db, w, w.Root, u, false)
	}
	for _, j := range w.Joins {
		if err := insertOrUpdateJoin(db, w, &j, u); err != nil {
			return sdk.WrapError(err, "Insert> Unable to insert update workflow(%d) join (%#v)", w.ID, j)
		}
	}
	return updateLastModified(db, w, u)
}

// Delete workflow
func Delete(db gorp.SqlExecutor, w *sdk.Workflow) error {
	//Detach root from workflow
	if _, err := db.Exec("update workflow set root_node_id = null where id = $1", w.ID); err != nil {
		return sdk.WrapError(err, "Delete> Unable to detache workflow root")
	}

	//Delete root
	if err := deleteNode(db, w.Root); err != nil {
		return sdk.WrapError(err, "Delete> Unable to delete workflow root")
	}

	//Delete workflow
	dbw := Workflow(*w)
	if _, err := db.Delete(&dbw); err != nil {
		return sdk.WrapError(err, "Delete> Unable to delete workflow")
	}

	return nil
}

// UpdateLastModified updates the workflow
func updateLastModified(db gorp.SqlExecutor, w *sdk.Workflow, u *sdk.User) error {
	t := time.Now()
	if u != nil {
		cache.SetWithTTL(cache.Key("lastModified", "workflow", fmt.Sprintf("%d", w.ID)), sdk.LastModification{
			Name:         w.Name,
			Username:     u.Username,
			LastModified: t.Unix(),
		}, 0)
	}
	return nil
}

// HasAccessTo checks if user has full r, rx or rwx access to the workflow
func HasAccessTo(db gorp.SqlExecutor, w *sdk.Workflow, u *sdk.User) (bool, error) {
	return true, nil
}

// IsValid cheks workflow validity
func IsValid(db gorp.SqlExecutor, w *sdk.Workflow, u *sdk.User) error {
	//Check duplicate refs
	return nil
}
