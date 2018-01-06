package sdk

import (
	"encoding/json"
	"fmt"
	"time"
)

//Workflow represents a pipeline based workflow
type Workflow struct {
	ID            int64                  `json:"id" db:"id" cli:"-"`
	Name          string                 `json:"name" db:"name" cli:"name,key"`
	Description   string                 `json:"description,omitempty" db:"description" cli:"description"`
	LastModified  time.Time              `json:"last_modified" db:"last_modified"`
	ProjectID     int64                  `json:"project_id,omitempty" db:"project_id" cli:"-"`
	ProjectKey    string                 `json:"project_key" db:"-" cli:"-"`
	RootID        int64                  `json:"root_id,omitempty" db:"root_node_id" cli:"-"`
	Root          *WorkflowNode          `json:"root" db:"-" cli:"-"`
	Joins         []WorkflowNodeJoin     `json:"joins,omitempty" db:"-" cli:"-"`
	Groups        []GroupPermission      `json:"groups,omitempty" db:"-" cli:"-"`
	Permission    int                    `json:"permission,omitempty" db:"-" cli:"-"`
	Metadata      Metadata               `json:"metadata" yaml:"metadata" db:"-"`
	Usage         *Usage                 `json:"usage,omitempty" db:"-" cli:"-"`
	HistoryLength int64                  `json:"history_length" db:"history_length" cli:"-"`
	PurgeTags     []string               `json:"purge_tags,omitempty" db:"-" cli:"-"`
	Notifications []WorkflowNotification `json:"notifications,omitempty" db:"-" cli:"-"`
}

// WorkflowNotifications represents notifications on a workflow
type WorkflowNotification struct {
	ID             int64                        `json:"id,omitempty" db:"id"`
	WorkflowID     int64                        `json:"workflow_id,omitempty" db:"workflow_id"`
	SourceNodeRefs []string                     `json:"source_node_ref,omitempty" db:"-"`
	SourceNodeIDs  []int64                      `json:"source_node_id,omitempty" db:"-"`
	Type           UserNotificationSettingsType `json:"type"  db:"type"`
	Settings       UserNotificationSettings     `json:"settings"  db:"-"`
}

//UnmarshalJSON parses the JSON-encoded data and stores the result in n
func (n *WorkflowNotification) UnmarshalJSON(b []byte) error {
	notif, err := parseWorkflowNotification(b)
	if err != nil {
		return err
	}
	*n = *notif
	return nil
}

//workflowNotificationInput is a way to parse notification
type workflowNotificationInput struct {
	Notification   interface{}                  `json:"settings"`
	ID             int64                        `json:"id,omitempty"`
	WorkflowID     int64                        `json:"workflow_id,omitempty"`
	SourceNodeRefs []string                     `json:"source_node_ref,omitempty"`
	SourceNodeIDs  []int64                      `json:"source_node_id,omitempty"`
	Type           UserNotificationSettingsType `json:"type"`
}

//parseWorkflowNotification transform jsons to UserNotificationSettings map
func parseWorkflowNotification(body []byte) (*WorkflowNotification, error) {
	var input = &workflowNotificationInput{}
	if err := json.Unmarshal(body, &input); err != nil {
		return nil, err
	}
	settingsBody, err := json.Marshal(input.Notification)
	if err != nil {
		return nil, err
	}

	var notif1 = &WorkflowNotification{
		ID:             input.ID,
		SourceNodeIDs:  input.SourceNodeIDs,
		SourceNodeRefs: input.SourceNodeRefs,
		WorkflowID:     input.WorkflowID,
		Type:           UserNotificationSettingsType(input.Type),
	}

	var errParse error
	notif1.Settings, errParse = ParseWorkflowUserNotificationSettings(notif1.Type, settingsBody)
	return notif1, errParse
}

//ParseUserNotificationSettings transforms json to UserNotificationSettings map
func ParseWorkflowUserNotificationSettings(t UserNotificationSettingsType, userNotif []byte) (UserNotificationSettings, error) {
	switch t {
	case EmailUserNotification, JabberUserNotification:
		var x JabberEmailUserNotificationSettings
		if err := json.Unmarshal(userNotif, &x); err != nil {
			return nil, ErrParseUserNotification
		}
		return &x, nil
	default:
		return nil, ErrNotSupportedUserNotification
	}
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
func (w *Workflow) Nodes(withRoot bool) []WorkflowNode {
	if w.Root == nil {
		return nil
	}

	res := []WorkflowNode{}
	if withRoot {
		res = append(res, w.Root.Nodes()...)
	} else {
		for _, t := range w.Root.Triggers {
			res = append(res, t.WorkflowDestNode.Nodes()...)
		}
	}

	for _, j := range w.Joins {
		for _, t := range j.Triggers {
			res = append(res, t.WorkflowDestNode.Nodes()...)
		}
	}
	return res
}

//AddTrigger adds a trigger to the destination node from the node found by its name
func (w *Workflow) AddTrigger(name string, dest WorkflowNode) {
	if w.Root == nil {
		return
	}

	w.Root.AddTrigger(name, dest)
	for i := range w.Joins {
		for j := range w.Joins[i].Triggers {
			w.Joins[i].Triggers[j].WorkflowDestNode.AddTrigger(name, dest)
		}
	}
}

//AddTrigger adds a trigger to the destination node from the node found by its name
func (n *WorkflowNode) AddTrigger(name string, dest WorkflowNode) {
	if n.Name == name {
		n.Triggers = append(n.Triggers, WorkflowNodeTrigger{
			WorkflowDestNode: dest,
		})
		return
	}
	for i := range n.Triggers {
		destNode := &n.Triggers[i].WorkflowDestNode
		destNode.AddTrigger(name, dest)
	}
}

//GetNodeByName returns the node given its name
func (w *Workflow) GetNodeByName(name string) *WorkflowNode {
	n := w.Root.GetNodeByName(name)
	if n != nil {
		return n
	}
	for _, j := range w.Joins {
		for _, t := range j.Triggers {
			n = t.WorkflowDestNode.GetNodeByName(name)
			if n != nil {
				return n
			}
		}
	}
	return nil
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

//GetApplications returns all applications used in the workflow
func (w *Workflow) GetApplications() []Application {
	if w.Root == nil {
		return nil
	}

	res := w.Root.GetApplications()
	for _, j := range w.Joins {
		for _, t := range j.Triggers {
			res = append(res, t.WorkflowDestNode.GetApplications()...)
		}
	}

	withoutDuplicates := []Application{}
	for _, a := range res {
		var found bool
		for _, d := range withoutDuplicates {
			if a.Name == d.Name {
				found = true
				break
			}
		}
		if !found {
			withoutDuplicates = append(withoutDuplicates, a)
		}
	}

	return withoutDuplicates
}

//GetEnvironments returns all environments used in the workflow
func (w *Workflow) GetEnvironments() []Environment {
	if w.Root == nil {
		return nil
	}

	res := w.Root.GetEnvironments()
	for _, j := range w.Joins {
		for _, t := range j.Triggers {
			res = append(res, t.WorkflowDestNode.GetEnvironments()...)
		}
	}

	withoutDuplicates := []Environment{}
	for _, a := range res {
		var found bool
		for _, d := range withoutDuplicates {
			if a.Name == d.Name {
				found = true
				break
			}
		}
		if !found {
			withoutDuplicates = append(withoutDuplicates, a)
		}
	}

	return withoutDuplicates
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

	withoutDuplicates := []Pipeline{}
	for _, a := range res {
		var found bool
		for _, d := range withoutDuplicates {
			if a.Name == d.Name {
				found = true
				break
			}
		}
		if !found {
			withoutDuplicates = append(withoutDuplicates, a)
		}
	}

	return withoutDuplicates
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

//Visit all the workflow and apply the visitor func on all nodes
func (w *Workflow) Visit(visitor func(*WorkflowNode)) {
	w.Root.Visit(visitor)
	for i := range w.Joins {
		for j := range w.Joins[i].Triggers {
			n := &w.Joins[i].Triggers[j].WorkflowDestNode
			n.Visit(visitor)
		}
	}
}

//Visit all the workflow and apply the visitor func on the current node and the children
func (n *WorkflowNode) Visit(visitor func(*WorkflowNode)) {
	visitor(n)
	for i := range n.Triggers {
		d := &n.Triggers[i].WorkflowDestNode
		d.Visit(visitor)
	}
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

//GetNodeByName returns the node given its name
func (n *WorkflowNode) GetNodeByName(name string) *WorkflowNode {
	if n == nil {
		return nil
	}
	if n.Name == name {
		return n
	}
	for _, t := range n.Triggers {
		n = t.WorkflowDestNode.GetNodeByName(name)
		if n != nil {
			return n
		}
	}
	return nil
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
func (n *WorkflowNode) Nodes() []WorkflowNode {
	res := []WorkflowNode{*n}
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

//GetApplications returns all applications used in the workflow
func (n *WorkflowNode) GetApplications() []Application {
	res := []Application{}
	if n.Context != nil && n.Context.Application != nil {
		res = append(res, *n.Context.Application)
	}
	for _, t := range n.Triggers {
		res = append(res, t.WorkflowDestNode.GetApplications()...)
	}
	return res
}

//GetEnvironments returns all Environments used in the workflow
func (n *WorkflowNode) GetEnvironments() []Environment {
	res := []Environment{}
	if n.Context != nil && n.Context.Environment != nil {
		res = append(res, *n.Context.Environment)
	}
	for _, t := range n.Triggers {
		res = append(res, t.WorkflowDestNode.GetEnvironments()...)
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
	PlainConditions []WorkflowNodeCondition `json:"plain,omitempty" yaml:"check,omitempty"`
	LuaScript       string                  `json:"lua_script,omitempty" yaml:"script,omitempty"`
}

//WorkflowNodeCondition represents a condition to trigger ot not a pipeline in a workflow. Operator can be =, !=, regex
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
	Mutex                     bool                   `json:"mutex"`
}

//WorkflowNodeContextDefaultPayloadVCS represents a default payload when a workflow is attached to a repository Webhook
type WorkflowNodeContextDefaultPayloadVCS struct {
	GitBranch     string `json:"git.branch" db:"-"`
	GitHash       string `json:"git.hash" db:"-"`
	GitAuthor     string `json:"git.author" db:"-"`
	GitHashBefore string `json:"git.hash.before" db:"-"`
	GitRepository string `json:"git.repository" db:"-"`
	GitMessage    string `json:"git.message" db:"-"`
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

// WorkflowNodeJobRunCount return nb workflow run job since 'since'
type WorkflowNodeJobRunCount struct {
	Count int64     `json:"version"`
	Since time.Time `json:"since"`
}
