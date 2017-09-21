package workflow

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
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
func Load(db gorp.SqlExecutor, store cache.Store, projectKey, name string, u *sdk.User) (*sdk.Workflow, error) {
	query := `
		select workflow.* 
		from workflow
		join project on project.id = workflow.project_id
		where project.projectkey = $1
		and workflow.name = $2`
	res, err := load(db, store, u, query, projectKey, name)
	if err != nil {
		return nil, sdk.WrapError(err, "Load> Unable to load workflow %s in project %s", name, projectKey)
	}
	return res, nil
}

// LoadByID loads a workflow for a given user (ie. checking permissions)
func LoadByID(db gorp.SqlExecutor, store cache.Store, id int64, u *sdk.User) (*sdk.Workflow, error) {
	query := `
		select * 
		from workflow
		where id = $1`
	res, err := load(db, store, u, query, id)
	if err != nil {
		return nil, sdk.WrapError(err, "Load> Unable to load workflow %d", id)
	}
	return res, nil
}

func load(db gorp.SqlExecutor, store cache.Store, u *sdk.User, query string, args ...interface{}) (*sdk.Workflow, error) {
	t0 := time.Now()
	dbRes := Workflow{}
	if err := db.SelectOne(&dbRes, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrWorkflowNotFound
		}
		return nil, sdk.WrapError(err, "Load> Unable to load workflow")
	}

	res := sdk.Workflow(dbRes)
	res.ProjectKey, _ = db.SelectStr("select projectkey from project where id = $1", res.ProjectID)
	if err := loadWorkflowRoot(db, store, &res, u); err != nil {
		return nil, sdk.WrapError(err, "Load> Unable to load workflow root")
	}

	joins, errJ := loadJoins(db, store, &res, u)
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

func loadWorkflowRoot(db gorp.SqlExecutor, store cache.Store, w *sdk.Workflow, u *sdk.User) error {
	var err error
	w.Root, err = loadNode(db, store, w, w.RootID, u)
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
func Insert(db gorp.SqlExecutor, store cache.Store, w *sdk.Workflow, u *sdk.User) error {
	if err := IsValid(db, store, w, u); err != nil {
		return err
	}

	w.LastModified = time.Now()
	if err := db.QueryRow("INSERT INTO workflow (name, description, project_id) VALUES ($1, $2, $3) RETURNING id", w.Name, w.Description, w.ProjectID).Scan(&w.ID); err != nil {
		return sdk.WrapError(err, "Insert> Unable to insert workflow %s/%s", w.ProjectKey, w.Name)
	}

	if w.Root == nil {
		return sdk.ErrWorkflowInvalidRoot
	}

	if err := renameNode(db, w); err != nil {
		return sdk.WrapError(err, "Insert> Cannot rename node")
	}

	if err := insertNode(db, w, w.Root, u, false); err != nil {
		return sdk.WrapError(err, "Insert> Unable to insert workflow root node")
	}

	if _, err := db.Exec("UPDATE workflow SET root_node_id = $2 WHERE id = $1", w.ID, w.Root.ID); err != nil {
		return sdk.WrapError(err, "Insert> Unable to insert workflow (%#v, %d)", w.Root, w.ID)
	}

	for i := range w.Joins {
		j := &w.Joins[i]
		if err := insertJoin(db, w, j, u); err != nil {
			return sdk.WrapError(err, "Insert> Unable to insert update workflow(%d) join (%#v)", w.ID, j)
		}
	}

	return updateLastModified(db, store, w, u)
}

func renameNode(db gorp.SqlExecutor, w *sdk.Workflow) error {
	nameByPipeline := map[int64][]*sdk.WorkflowNode{}
	maxNumberByPipeline := map[int64]int64{}

	// browse node
	if err := saveNodeByPipeline(db, &nameByPipeline, &maxNumberByPipeline, w.Root); err != nil {
		return err
	}

	// browse join
	for i := range w.Joins {
		join := &w.Joins[i]
		for j := range join.Triggers {
			if err := saveNodeByPipeline(db, &nameByPipeline, &maxNumberByPipeline, &join.Triggers[j].WorkflowDestNode); err != nil {
				return err
			}
		}
	}

	// Generate node name
	for _, v := range nameByPipeline {
		for _, n := range v {
			if n.Name == "" {
				nextNumber := maxNumberByPipeline[n.Pipeline.ID] + 1
				n.Name = fmt.Sprintf("%s_%d", n.Pipeline.Name, nextNumber)
				maxNumberByPipeline[n.Pipeline.ID] = nextNumber
			}
		}
	}

	return nil
}

func saveNodeByPipeline(db gorp.SqlExecutor, dict *map[int64][]*sdk.WorkflowNode, mapMaxNumber *map[int64]int64, n *sdk.WorkflowNode) error {
	// get pipeline ID
	pipID := n.PipelineID
	if pipID == 0 {
		pipID = n.Pipeline.ID
	}

	// Load pipeline to have name
	pip := &n.Pipeline
	if pip.Name == "" {
		var errorP error
		pip, errorP = pipeline.LoadPipelineByID(db, pipID, false)
		if errorP != nil {
			return sdk.WrapError(errorP, "saveNodeByPipeline> Cannot load pipeline %d", pipID)
		}
		n.Pipeline = *pip
	}

	// Save node in pipeline node map
	if _, ok := (*dict)[pipID]; !ok {
		(*dict)[pipID] = []*sdk.WorkflowNode{}
	}
	(*dict)[pipID] = append((*dict)[pipID], n)

	// Check max number for current pipeline
	if n.Name != "" && strings.HasPrefix(n.Name, pip.Name+"_") {
		pipNumber, errI := strconv.ParseInt(strings.Replace(n.Name, pip.Name+"_", "", 1), 10, 64)
		if errI == nil {
			currentMax, ok := (*mapMaxNumber)[pipID]
			if !ok || currentMax < pipNumber {
				(*mapMaxNumber)[pipID] = pipNumber
			}
		}
	}

	for k := range n.Triggers {
		if err := saveNodeByPipeline(db, dict, mapMaxNumber, &n.Triggers[k].WorkflowDestNode); err != nil {
			return err
		}
	}
	return nil
}

// Update updates a workflow
func Update(db gorp.SqlExecutor, store cache.Store, w *sdk.Workflow, oldWorkflow *sdk.Workflow, u *sdk.User) error {
	if err := IsValid(db, store, w, u); err != nil {
		return err
	}

	if err := renameNode(db, w); err != nil {
		return sdk.WrapError(err, "Update> cannot check pipeline name")
	}

	// Delete all OLD JOIN
	for _, j := range oldWorkflow.Joins {
		if err := deleteJoin(db, j); err != nil {
			return sdk.WrapError(err, "Update> unable to delete all join on workflow(%d)", w.ID)
		}
	}

	// Delete old Root Node
	if oldWorkflow.Root != nil {
		if _, err := db.Exec("update workflow set root_node_id = null where id = $1", w.ID); err != nil {
			return sdk.WrapError(err, "Delete> Unable to detache workflow root")
		}
		if err := deleteNode(db, oldWorkflow, oldWorkflow.Root, u); err != nil {
			return sdk.WrapError(err, "Update> unable to delete root node on workflow(%d)", w.ID)
		}
	}

	// Inser new Root Node
	if err := insertNode(db, w, w.Root, u, false); err != nil {
		return sdk.WrapError(err, "Update> unable to update root node on workflow(%d)", w.ID)
	}

	w.RootID = w.Root.ID

	// Insert new JOIN
	for i := range w.Joins {
		j := &w.Joins[i]
		if err := insertJoin(db, w, j, u); err != nil {
			return sdk.WrapError(err, "Insert> Unable to insert update workflow(%d) join (%#v)", w.ID, j)
		}
	}

	w.LastModified = time.Now()
	dbw := Workflow(*w)
	if _, err := db.Update(&dbw); err != nil {
		return sdk.WrapError(err, "Update> Unable to update workflow")
	}

	return updateLastModified(db, store, w, u)
}

// Delete workflow
func Delete(db gorp.SqlExecutor, w *sdk.Workflow, u *sdk.User) error {
	//Detach root from workflow
	if _, err := db.Exec("update workflow set root_node_id = null where id = $1", w.ID); err != nil {
		return sdk.WrapError(err, "Delete> Unable to detache workflow root")
	}

	// Delete all JOINs
	for _, j := range w.Joins {
		if err := deleteJoin(db, j); err != nil {
			return sdk.WrapError(err, "Update> unable to delete all join on workflow(%d)", w.ID)
		}
	}

	//Delete root
	if err := deleteNode(db, w, w.Root, u); err != nil {
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
func updateLastModified(db gorp.SqlExecutor, store cache.Store, w *sdk.Workflow, u *sdk.User) error {
	t := time.Now()
	if u != nil {
		store.SetWithTTL(cache.Key("lastModified", "workflow", fmt.Sprintf("%d", w.ID)), sdk.LastModification{
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
func IsValid(db gorp.SqlExecutor, store cache.Store, w *sdk.Workflow, u *sdk.User) error {
	//Check project is not empty
	if w.ProjectKey == "" {
		return sdk.NewError(sdk.ErrWorkflowInvalid, fmt.Errorf("Invalid project key"))
	}

	//Check duplicate refs
	refs := w.References()
	for i, ref1 := range refs {
		for j, ref2 := range refs {
			if ref1 == ref2 && i != j {
				return sdk.NewError(sdk.ErrWorkflowInvalid, fmt.Errorf("Duplicate reference %s", ref1))
			}
		}
	}

	//Check refs
	for _, j := range w.Joins {
		if len(j.SourceNodeRefs) == 0 {
			return sdk.NewError(sdk.ErrWorkflowInvalid, fmt.Errorf("Source node references is mandatory"))
		}
	}

	//Load the project
	proj, err := project.Load(db, store, w.ProjectKey, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments)
	if err != nil {
		return sdk.NewError(sdk.ErrWorkflowInvalid, err)
	}

	//Checks application are in the current project
	apps := w.InvolvedApplications()
	for _, appID := range apps {
		var found bool
		for _, a := range proj.Applications {
			if appID == a.ID {
				found = true
				break
			}
		}
		if !found {
			return sdk.NewError(sdk.ErrWorkflowInvalid, fmt.Errorf("Unknown application %d", appID))
		}
	}

	//Checks pipelines are in the current project
	pips := w.InvolvedPipelines()
	for _, pipID := range pips {
		var found bool
		for _, p := range proj.Pipelines {
			if pipID == p.ID {
				found = true
				break
			}
		}
		if !found {
			return sdk.NewError(sdk.ErrWorkflowInvalid, fmt.Errorf("Unknown pipeline %d", pipID))
		}
	}

	//Checks environments are in the current project
	envs := w.InvolvedEnvironments()
	for _, envID := range envs {
		var found bool
		for _, e := range proj.Environments {
			if envID == e.ID {
				found = true
				break
			}
		}
		if !found {
			return sdk.NewError(sdk.ErrWorkflowInvalid, fmt.Errorf("Unknown environments %d", envID))
		}
	}

	return nil
}
