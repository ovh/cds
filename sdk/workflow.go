package sdk

import (
	"encoding/json"
	"fmt"
	"time"
)

//Workflow represents a pipeline based workflow
type Workflow struct {
	ID            int64              `json:"id" db:"id" cli:"-"`
	Name          string             `json:"name" db:"name" cli:"name,key"`
	Description   string             `json:"description,omitempty" db:"description" cli:"description"`
	LastModified  time.Time          `json:"last_modified" db:"last_modified"`
	ProjectID     int64              `json:"project_id,omitempty" db:"project_id" cli:"-"`
	ProjectKey    string             `json:"project_key" db:"-" cli:"-"`
	RootID        int64              `json:"root_id,omitempty" db:"root_node_id" cli:"-"`
	Root          *WorkflowNode      `json:"root" db:"-" cli:"-"`
	Joins         []WorkflowNodeJoin `json:"joins,omitempty" db:"-" cli:"-"`
	Groups        []GroupPermission  `json:"groups,omitempty" db:"-" cli:"-"`
	Permission    int                `json:"permission,omitempty" db:"-" cli:"-"`
	Metadata      Metadata           `json:"metadata" yaml:"metadata" db:"-"`
	Usage         *Usage             `json:"usage,omitempty" db:"-" cli:"-"`
	HistoryLength int64              `json:"history_length" db:"history_length" cli:"-"`
	PurgeTags     []string           `json:"purge_tags,omitempty" db:"-" cli:"-"`
}

// FilterHooksConfig filter all hooks configuration and remove somme configuration key
func (w *Workflow) FilterHooksConfig(s ...string) {
	if w.Root == nil {
		return
	}

	w.Root.FilterHooksConfig(s...)
	for i := range w.Joins {
		for j := range w.Joins[i].Triggers {
			w.Joins[i].Triggers[j].WorkflowDestNode.FilterHooksConfig(s...)
		}
	}
}

// GetHooks returns the list of all hooks in the workflow tree
func (w *Workflow) GetHooks() map[string]WorkflowNodeHook {
	if w.Root == nil {
		return nil
	}

	res := map[string]WorkflowNodeHook{}

	a := w.Root.GetHooks()
	for k, v := range a {
		res[k] = v
	}

	for _, j := range w.Joins {
		for _, t := range j.Triggers {
			b := t.WorkflowDestNode.GetHooks()
			for k, v := range b {
				res[k] = v
			}
		}
	}

	return res
}

//JoinsID returns joins ID
func (w *Workflow) JoinsID() []int64 {
	res := []int64{}
	for _, j := range w.Joins {
		res = append(res, j.ID)
	}
	return res
}

//Nodes returns nodes IDs excluding the root ID
func (w *Workflow) Nodes() []int64 {
	if w.Root == nil {
		return nil
	}

	res := []int64{}
	for _, t := range w.Root.Triggers {
		res = append(res, t.WorkflowDestNode.Nodes()...)
	}

	for _, j := range w.Joins {
		for _, t := range j.Triggers {
			res = append(res, t.WorkflowDestNode.Nodes()...)
		}
	}
	return res
}

//GetNode returns the node given its id
func (w *Workflow) GetNode(id int64) *WorkflowNode {
	n := w.Root.GetNode(id)
	if n != nil {
		return n
	}
	for _, j := range w.Joins {
		for _, t := range j.Triggers {
			n = t.WorkflowDestNode.GetNode(id)
			if n != nil {
				return n
			}
		}
	}
	return nil
}

//GetJoin returns the join given its id
func (w *Workflow) GetJoin(id int64) *WorkflowNodeJoin {
	for _, j := range w.Joins {
		if j.ID == id {
			return &j
		}
	}
	return nil
}

//TriggersID returns triggers IDs
func (w *Workflow) TriggersID() []int64 {
	res := w.Root.TriggersID()
	for _, j := range w.Joins {
		for _, t := range j.Triggers {
			res = append(res, t.ID)
			res = append(res, t.WorkflowDestNode.TriggersID()...)
		}
	}
	return res
}

//References returns a slice with all node references
func (w *Workflow) References() []string {
	if w.Root == nil {
		return nil
	}

	res := w.Root.References()
	for _, j := range w.Joins {
		for _, t := range j.Triggers {
			res = append(res, t.WorkflowDestNode.References()...)
		}
	}
	return res
}

//InvolvedApplications returns all applications used in the workflow
func (w *Workflow) InvolvedApplications() []int64 {
	if w.Root == nil {
		return nil
	}

	res := w.Root.InvolvedApplications()
	for _, j := range w.Joins {
		for _, t := range j.Triggers {
			res = append(res, t.WorkflowDestNode.InvolvedApplications()...)
		}
	}
	return res
}

//InvolvedPipelines returns all pipelines used in the workflow
func (w *Workflow) InvolvedPipelines() []int64 {
	if w.Root == nil {
		return nil
	}

	res := w.Root.InvolvedPipelines()
	for _, j := range w.Joins {
		for _, t := range j.Triggers {
			res = append(res, t.WorkflowDestNode.InvolvedPipelines()...)
		}
	}
	return res
}

//GetPipelines returns all pipelines used in the workflow
func (w *Workflow) GetPipelines() []Pipeline {
	if w.Root == nil {
		return nil
	}

	res := w.Root.GetPipelines()
	for _, j := range w.Joins {
		for _, t := range j.Triggers {
			res = append(res, t.WorkflowDestNode.GetPipelines()...)
		}
	}
	return res
}

//InvolvedEnvironments returns all environments used in the workflow
func (w *Workflow) InvolvedEnvironments() []int64 {
	if w.Root == nil {
		return nil
	}

	res := w.Root.InvolvedEnvironments()
	for _, j := range w.Joins {
		for _, t := range j.Triggers {
			res = append(res, t.WorkflowDestNode.InvolvedEnvironments()...)
		}
	}
	return res
}

//WorkflowNodeJoin aims to joins multiple node into multiple triggers
type WorkflowNodeJoin struct {
	ID             int64                     `json:"id" db:"id"`
	WorkflowID     int64                     `json:"workflow_id" db:"workflow_id"`
	SourceNodeIDs  []int64                   `json:"source_node_id,omitempty" db:"-"`
	SourceNodeRefs []string                  `json:"source_node_ref,omitempty" db:"-"`
	Triggers       []WorkflowNodeJoinTrigger `json:"triggers,omitempty" db:"-"`
}

//WorkflowNodeJoinTrigger is a trigger for joins
type WorkflowNodeJoinTrigger struct {
	ID                 int64        `json:"id" db:"id"`
	WorkflowNodeJoinID int64        `json:"join_id" db:"workflow_node_join_id"`
	WorkflowDestNodeID int64        `json:"workflow_dest_node_id" db:"workflow_dest_node_id"`
	WorkflowDestNode   WorkflowNode `json:"workflow_dest_node" db:"-"`
}

//WorkflowNode represents a node in w workflow tree
type WorkflowNode struct {
	ID               int64                 `json:"id" db:"id"`
	Name             string                `json:"name" db:"name"`
	Ref              string                `json:"ref,omitempty" db:"-"`
	WorkflowID       int64                 `json:"workflow_id" db:"workflow_id"`
	PipelineID       int64                 `json:"pipeline_id" db:"pipeline_id"`
	Pipeline         Pipeline              `json:"pipeline" db:"-"`
	Context          *WorkflowNodeContext  `json:"context" db:"-"`
	TriggerSrcID     int64                 `json:"-" db:"-"`
	TriggerJoinSrcID int64                 `json:"-" db:"-"`
	Hooks            []WorkflowNodeHook    `json:"hooks,omitempty" db:"-"`
	Triggers         []WorkflowNodeTrigger `json:"triggers,omitempty" db:"-"`
}

// FilterHooksConfig filter all hooks configuration and remove somme configuration key
func (n *WorkflowNode) FilterHooksConfig(s ...string) {
	if n == nil {
		return
	}

	for _, h := range n.Hooks {
		for i := range s {
			for k := range h.Config {
				if k == s[i] {
					delete(h.Config, k)
					break
				}
			}
		}
	}
}

//GetHooks returns all hooks for the node and its children
func (n *WorkflowNode) GetHooks() map[string]WorkflowNodeHook {
	res := map[string]WorkflowNodeHook{}

	for _, h := range n.Hooks {
		res[h.UUID] = h
	}

	for _, t := range n.Triggers {
		b := t.WorkflowDestNode.GetHooks()
		for k, v := range b {
			res[k] = v
		}
	}

	return res
}

// EqualsTo returns true if a node has the same pipeline and context than another
func (n *WorkflowNode) EqualsTo(n1 *WorkflowNode) bool {
	if n.PipelineID != n1.PipelineID {
		return false
	}
	if n.Context == nil && n1.Context != nil {
		return false
	}
	if n.Context != nil && n1.Context == nil {
		return false
	}
	if n.Context.ApplicationID != n1.Context.ApplicationID {
		return false
	}
	if n.Context.EnvironmentID != n1.Context.EnvironmentID {
		return false
	}
	return true
}

//GetNode returns the node given its id
func (n *WorkflowNode) GetNode(id int64) *WorkflowNode {
	if n == nil {
		return nil
	}
	if n.ID == id {
		return n
	}
	for _, t := range n.Triggers {
		n = t.WorkflowDestNode.GetNode(id)
		if n != nil {
			return n
		}
	}
	return nil
}

//Nodes returns a slice with all node IDs
func (n *WorkflowNode) Nodes() []int64 {
	res := []int64{n.ID}
	for _, t := range n.Triggers {
		res = append(res, t.WorkflowDestNode.Nodes()...)
	}
	return res
}

func ancestor(id int64, node *WorkflowNode, deep bool) (map[int64]bool, bool) {
	res := map[int64]bool{}
	if id == node.ID {
		return res, true
	}
	for _, t := range node.Triggers {
		if t.WorkflowDestNode.ID == id {
			res[node.ID] = true
			return res, true
		}
		ids, ok := ancestor(id, &t.WorkflowDestNode, deep)
		if ok {
			if len(ids) == 1 || deep {
				for k := range ids {
					res[k] = true
				}
			}
			if deep {
				res[node.ID] = true
			}
			return res, true
		}
	}
	return res, false
}

// Ancestors returns  all node ancestors if deep equal true, and only his direct ancestors if deep equal false
func (n *WorkflowNode) Ancestors(w *Workflow, deep bool) []int64 {
	if n == nil {
		return nil
	}

	res, ok := ancestor(n.ID, w.Root, deep)

	if !ok {
	joinLoop:
		for _, j := range w.Joins {
			for _, t := range j.Triggers {
				resAncestor, ok := ancestor(n.ID, &t.WorkflowDestNode, deep)
				if ok {
					if len(resAncestor) == 1 || deep {
						for id := range resAncestor {
							res[id] = true
						}
					}

					if len(resAncestor) == 0 || deep {
						for _, id := range j.SourceNodeIDs {
							res[id] = true
							if deep {
								node := w.GetNode(id)
								if node != nil {
									ancerstorRes := node.Ancestors(w, deep)
									for _, id := range ancerstorRes {
										res[id] = true
									}
								}
							}
						}
					}
					break joinLoop
				}
			}
		}
	}

	keys := make([]int64, len(res))
	i := 0
	for k := range res {
		keys[i] = k
		i++
	}
	return keys
}

//TriggersID returns a slides of triggers IDs
func (n *WorkflowNode) TriggersID() []int64 {
	res := []int64{}
	for _, t := range n.Triggers {
		res = append(res, t.ID)
		res = append(res, t.WorkflowDestNode.TriggersID()...)
	}
	return res
}

//References returns a slice with all node references
func (n *WorkflowNode) References() []string {
	res := []string{}
	if n.Ref != "" {
		res = []string{n.Ref}
	}
	for _, t := range n.Triggers {
		res = append(res, t.WorkflowDestNode.References()...)
	}
	return res
}

//InvolvedApplications returns all applications used in the workflow
func (n *WorkflowNode) InvolvedApplications() []int64 {
	res := []int64{}
	if n.Context != nil {
		if n.Context.ApplicationID == 0 && n.Context.Application != nil {
			n.Context.ApplicationID = n.Context.Application.ID
		}
		if n.Context.ApplicationID != 0 {
			res = []int64{n.Context.ApplicationID}
		}
	}
	for _, t := range n.Triggers {
		res = append(res, t.WorkflowDestNode.InvolvedApplications()...)
	}
	return res
}

//InvolvedPipelines returns all pipelines used in the workflow
func (n *WorkflowNode) InvolvedPipelines() []int64 {
	res := []int64{}

	if n.PipelineID == 0 {
		n.PipelineID = n.Pipeline.ID
	}
	res = []int64{n.PipelineID}

	for _, t := range n.Triggers {
		res = append(res, t.WorkflowDestNode.InvolvedPipelines()...)
	}
	return res
}

//GetPipelines returns all pipelines used in the workflow
func (n *WorkflowNode) GetPipelines() []Pipeline {
	res := []Pipeline{n.Pipeline}
	for _, t := range n.Triggers {
		res = append(res, t.WorkflowDestNode.GetPipelines()...)
	}
	return res
}

//InvolvedEnvironments returns all environments used in the workflow
func (n *WorkflowNode) InvolvedEnvironments() []int64 {
	res := []int64{}
	if n.Context != nil {
		if n.Context.EnvironmentID == 0 && n.Context.Environment != nil {
			n.Context.EnvironmentID = n.Context.Environment.ID
		}
		if n.Context.EnvironmentID != 0 {
			res = []int64{n.Context.EnvironmentID}
		}
	}
	for _, t := range n.Triggers {
		res = append(res, t.WorkflowDestNode.InvolvedEnvironments()...)
	}
	return res
}

//WorkflowNodeTrigger is a ling betweeb two pipelines in a workflow
type WorkflowNodeTrigger struct {
	ID                 int64        `json:"id" db:"id"`
	WorkflowNodeID     int64        `json:"workflow_node_id" db:"workflow_node_id"`
	WorkflowDestNodeID int64        `json:"workflow_dest_node_id" db:"workflow_dest_node_id"`
	WorkflowDestNode   WorkflowNode `json:"workflow_dest_node" db:"-"`
}

//WorkflowNodeConditions is either an array of WorkflowNodeCondition or a lua script
type WorkflowNodeConditions struct {
	PlainConditions []WorkflowNodeCondition `json:"plain"`
	LuaScript       string                  `json:"lua_script"`
}

//WorkflowTriggerCondition represents a condition to trigger ot not a pipeline in a workflow. Operator can be =, !=, regex
type WorkflowNodeCondition struct {
	Variable string `json:"variable"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

//WorkflowNodeContext represents a context attached on a node
type WorkflowNodeContext struct {
	ID                        int64                  `json:"id" db:"id"`
	WorkflowNodeID            int64                  `json:"workflow_node_id" db:"workflow_node_id"`
	ApplicationID             int64                  `json:"application_id" db:"application_id"`
	Application               *Application           `json:"application,omitempty" db:"-"`
	Environment               *Environment           `json:"environment,omitempty" db:"-"`
	EnvironmentID             int64                  `json:"environment_id" db:"environment_id"`
	DefaultPayload            interface{}            `json:"default_payload,omitempty" db:"-"`
	DefaultPipelineParameters []Parameter            `json:"default_pipeline_parameters,omitempty" db:"-"`
	Conditions                WorkflowNodeConditions `json:"conditions,omitempty" db:"-"`
}

//WorkflowNodeHook represents a hook which cann trigger the workflow from a given node
type WorkflowNodeHook struct {
	ID                  int64                  `json:"id" db:"id"`
	UUID                string                 `json:"uuid" db:"uuid"`
	WorkflowNodeID      int64                  `json:"workflow_node_id" db:"workflow_node_id"`
	WorkflowHookModelID int64                  `json:"workflow_hook_model_id" db:"workflow_hook_model_id"`
	WorkflowHookModel   WorkflowHookModel      `json:"model" db:"-"`
	Config              WorkflowNodeHookConfig `json:"config" db:"-"`
}

var WorkflowHookModelBuiltin = "builtin"

//WorkflowNodeHookConfig represents the configguration for a WorkflowNodeHook
type WorkflowNodeHookConfig map[string]WorkflowNodeHookConfigValue

// WorkflowNodeHookConfigValue represents the value of a node hook config
type WorkflowNodeHookConfigValue struct {
	Value        string `json:"value"`
	Configurable bool   `json:"configurable"`
}

//WorkflowHookModel represents a hook which can be used in workflows.
type WorkflowHookModel struct {
	ID            int64                  `json:"id" db:"id" cli:"-"`
	Name          string                 `json:"name" db:"name" cli:"name"`
	Type          string                 `json:"type"  db:"type"`
	Author        string                 `json:"author" db:"author"`
	Description   string                 `json:"description" db:"description"`
	Identifier    string                 `json:"identifier" db:"identifier"`
	Icon          string                 `json:"icon" db:"icon"`
	Command       string                 `json:"command" db:"command"`
	DefaultConfig WorkflowNodeHookConfig `json:"default_config" db:"-"`
	Disabled      bool                   `json:"disabled" db:"disabled"`
}

//WorkflowList return the list of the workflows for a project
func WorkflowList(projectkey string) ([]Workflow, error) {
	path := fmt.Sprintf("/project/%s/workflows", projectkey)
	body, _, err := Request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var ws = []Workflow{}
	if err := json.Unmarshal(body, &ws); err != nil {
		return nil, err
	}

	return ws, nil
}

//WorkflowGet returns a workflow given its name
func WorkflowGet(projectkey, name string) (*Workflow, error) {
	path := fmt.Sprintf("/project/%s/workflows/%s", projectkey, name)
	body, _, err := Request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var w = Workflow{}
	if err := json.Unmarshal(body, &w); err != nil {
		return nil, err
	}

	return &w, nil
}

// WorkflowDelete Call API to delete a workflow
func WorkflowDelete(projectkey, name string) error {
	path := fmt.Sprintf("/project/%s/workflows/%s", projectkey, name)
	_, _, err := Request("DELETE", path, nil)
	return err
}
