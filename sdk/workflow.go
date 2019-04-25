package sdk

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/fsamin/go-dump"
)

// DefaultHistoryLength is the default history length
const (
	DefaultHistoryLength int64 = 20
)

// ColorRegexp represent the regexp for a format to hexadecimal color
var ColorRegexp = regexp.MustCompile(`^#\w{3,8}$`)

//Workflow represents a pipeline based workflow
type Workflow struct {
	ID                      int64                        `json:"id" db:"id" cli:"-"`
	Name                    string                       `json:"name" db:"name" cli:"name,key"`
	Description             string                       `json:"description,omitempty" db:"description" cli:"description"`
	Icon                    string                       `json:"icon,omitempty" db:"icon" cli:"-"`
	LastModified            time.Time                    `json:"last_modified" db:"last_modified" mapstructure:"-"`
	ProjectID               int64                        `json:"project_id,omitempty" db:"project_id" cli:"-"`
	ProjectKey              string                       `json:"project_key" db:"-" cli:"-"`
	RootID                  int64                        `json:"root_id,omitempty" db:"root_node_id" cli:"-"`
	Root                    *WorkflowNode                `json:"root,omitempty" db:"-" cli:"-"`
	Joins                   []WorkflowNodeJoin           `json:"joins,omitempty" db:"-" cli:"-"`
	Groups                  []GroupPermission            `json:"groups,omitempty" db:"-" cli:"-"`
	Permission              int                          `json:"permission,omitempty" db:"-" cli:"-"`
	Metadata                Metadata                     `json:"metadata" yaml:"metadata" db:"-"`
	Usage                   *Usage                       `json:"usage,omitempty" db:"-" cli:"-"`
	HistoryLength           int64                        `json:"history_length" db:"history_length" cli:"-"`
	PurgeTags               []string                     `json:"purge_tags,omitempty" db:"-" cli:"-"`
	Notifications           []WorkflowNotification       `json:"notifications,omitempty" db:"-" cli:"-"`
	FromRepository          string                       `json:"from_repository,omitempty" db:"from_repository" cli:"from"`
	DerivedFromWorkflowID   int64                        `json:"derived_from_workflow_id,omitempty" db:"derived_from_workflow_id" cli:"-"`
	DerivedFromWorkflowName string                       `json:"derived_from_workflow_name,omitempty" db:"derived_from_workflow_name" cli:"-"`
	DerivationBranch        string                       `json:"derivation_branch,omitempty" db:"derivation_branch" cli:"-"`
	Audits                  []AuditWorkflow              `json:"audits" db:"-"`
	Pipelines               map[int64]Pipeline           `json:"pipelines" db:"-" cli:"-"  mapstructure:"-"`
	Applications            map[int64]Application        `json:"applications" db:"-" cli:"-"  mapstructure:"-"`
	Environments            map[int64]Environment        `json:"environments" db:"-" cli:"-"  mapstructure:"-"`
	ProjectIntegrations     map[int64]ProjectIntegration `json:"project_integrations" db:"-" cli:"-"  mapstructure:"-"`
	HookModels              map[int64]WorkflowHookModel  `json:"hook_models" db:"-" cli:"-"  mapstructure:"-"`
	OutGoingHookModels      map[int64]WorkflowHookModel  `json:"outgoing_hook_models" db:"-" cli:"-"  mapstructure:"-"`
	Labels                  []Label                      `json:"labels" db:"-" cli:"labels"`
	ToDelete                bool                         `json:"to_delete" db:"to_delete" cli:"-"`
	Favorite                bool                         `json:"favorite" db:"-" cli:"favorite"`
	WorkflowData            *WorkflowData                `json:"workflow_data" db:"-" cli:"-"`
	AsCodeEvent             []AsCodeEvent                `json:"as_code_events" db:"-" cli:"-"`
	// aggregates
	Template         *WorkflowTemplate         `json:"-" db:"-" cli:"-"`
	TemplateInstance *WorkflowTemplateInstance `json:"-" db:"-" cli:"-"`
	FromTemplate     string                    `json:"from_template,omitempty" db:"-" cli:"-"`
	TemplateUpToDate bool                      `json:"template_up_to_date,omitempty" db:"-" cli:"-"`
	URLs             URL                       `json:"urls" yaml:"-" db:"-" cli:"-"`
}

// AsCodeEvent represents all pending modifications on a workflow
type AsCodeEvent struct {
	ID             int64     `json:"id" db:"id" cli:"-"`
	WorkflowID     int64     `json:"workflow_id" db:"workflow_id" cli:"-"`
	PullRequestID  int64     `json:"pullrequest_id" db:"pullrequest_id" cli:"-"`
	PullRequestURL string    `json:"pullrequest_url" db:"pullrequest_url" cli:"-"`
	Username       string    `json:"username" db:"username" cli:"-"`
	CreationDate   time.Time `json:"creation_date" db:"creation_date" cli:"-"`
}

// GetApplication retrieve application from workflow
func (w *Workflow) GetApplication(ID int64) Application {
	return w.Applications[ID]
}

// RetroMigrate  temporary method that convert new workflow structure into old workflow structure for backward compatibility
func (w *Workflow) RetroMigrate() {
	root := w.WorkflowData.Node.retroMigrate()
	w.Root = &root

	w.Joins = nil
	if len(w.WorkflowData.Joins) > 0 {
		w.Joins = make([]WorkflowNodeJoin, 0, len(w.WorkflowData.Joins))
		for _, j := range w.WorkflowData.Joins {
			w.Joins = append(w.Joins, j.retroMigrateJoin())
		}
	}

	// Set context on old node
	for _, n := range w.Nodes(true) {
		node := w.GetNodeByName(n.Name)
		if node.Context == nil {
			continue
		}

		if node.Context.ApplicationID != 0 {
			app, ok := w.Applications[node.Context.ApplicationID]
			if ok {
				node.Context.Application = &app
			}
		}
		if node.Context.EnvironmentID != 0 {
			env, ok := w.Environments[node.Context.EnvironmentID]
			if ok {
				node.Context.Environment = &env
			}
		}
		if node.Context.ProjectIntegrationID != 0 {
			pp, ok := w.ProjectIntegrations[node.Context.ProjectIntegrationID]
			if ok {
				node.Context.ProjectIntegration = &pp
			}
		}
	}
}

// Migrate old workflow struct into new workflow struct
func (w *Workflow) Migrate(withID bool) WorkflowData {
	work := WorkflowData{}

	if w != nil && w.Root != nil {
		// Add root node
		work.Node = (*w.Root).migrate(withID)

		// Add Join
		work.Joins = make([]Node, 0, len(w.Joins))
		for _, j := range w.Joins {
			work.Joins = append(work.Joins, j.migrate(withID))
		}
	}
	return work
}

// WorkflowNotification represents notifications on a workflow
type WorkflowNotification struct {
	ID             int64                    `json:"id,omitempty" db:"id"`
	WorkflowID     int64                    `json:"workflow_id,omitempty" db:"workflow_id"`
	SourceNodeRefs []string                 `json:"source_node_ref,omitempty" db:"-"`
	SourceNodeIDs  []int64                  `json:"source_node_id,omitempty" db:"-"`
	NodeIDs        []int64                  `json:"node_id,omitempty" db:"-"`
	Type           string                   `json:"type" db:"type"`
	Settings       UserNotificationSettings `json:"settings" db:"-"`
}

func (w *Workflow) Forks() (map[int64]WorkflowNodeFork, map[int64]string) {
	forkMap := make(map[int64]WorkflowNodeFork, 0)
	forkTriggerMap := make(map[int64]string, 0)
	w.Root.ForksMap(&forkMap, &forkTriggerMap)
	for _, j := range w.Joins {
		for _, t := range j.Triggers {
			(&t.WorkflowDestNode).ForksMap(&forkMap, &forkTriggerMap)
		}
	}
	return forkMap, forkTriggerMap
}

//JoinsID returns joins ID
func (w *Workflow) JoinsID() []int64 {
	res := make([]int64, len(w.Joins))
	for i, j := range w.Joins {
		res[i] = j.ID
	}
	return res
}

// ResetIDs resets all nodes and joins ids
func (w *Workflow) ResetIDs() {
	if w.Root == nil {
		return
	}
	(w.Root).ResetIDs()
	for i := range w.Joins {
		j := &w.Joins[i]
		j.ID = 0
		j.SourceNodeIDs = nil
		for tid := range j.Triggers {
			t := &j.Triggers[tid]
			(&t.WorkflowDestNode).ResetIDs()
		}
	}

	for _, n := range w.WorkflowData.Array() {
		n.ID = 0
	}
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
		for _, f := range w.Root.Forks {
			for _, t := range f.Triggers {
				res = append(res, t.WorkflowDestNode.Nodes()...)
			}
		}
		for i := range w.Root.OutgoingHooks {
			for j := range w.Root.OutgoingHooks[i].Triggers {
				res = append(res, w.Root.OutgoingHooks[i].Triggers[j].WorkflowDestNode.Nodes()...)
			}
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
func (w *Workflow) AddTrigger(name string, dest Node) {
	if w.WorkflowData == nil || w.WorkflowData.Node.Name == "" {
		return
	}

	(&w.WorkflowData.Node).AddTrigger(name, dest)
	for i := range w.WorkflowData.Joins {
		(&w.WorkflowData.Joins[i]).AddTrigger(name, dest)
	}
}

//GetNodeByRef returns the node given its ref
func (w *Workflow) GetNodeByRef(ref string) *WorkflowNode {
	n := w.Root.GetNodeByRef(ref)
	if n != nil {
		return n
	}
	for ji := range w.Joins {
		j := &w.Joins[ji]
		for ti := range j.Triggers {
			t := &j.Triggers[ti]
			n2 := (&t.WorkflowDestNode).GetNodeByRef(ref)
			if n2 != nil {
				return n2
			}
		}
	}
	return nil
}

func (w *Workflow) GetForkByName(name string) *WorkflowNodeFork {
	n := w.Root.GetForkByName(name)
	if n != nil {
		return n
	}
	for _, j := range w.Joins {
		for _, t := range j.Triggers {
			n = t.WorkflowDestNode.GetForkByName(name)
			if n != nil {
				return n
			}
		}
	}
	return nil
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

	res := make([]Pipeline, len(w.Pipelines))
	var i int
	for _, p := range w.Pipelines {
		res[i] = p
		i++
	}
	return res
}

// GetRepositories returns the list of repositories from applications
func (w *Workflow) GetRepositories() []string {
	apps := w.GetApplications()
	repos := map[string]struct{}{}
	for _, a := range apps {
		if a.RepositoryFullname != "" {
			repos[a.RepositoryFullname] = struct{}{}
		}
	}
	res := make([]string, len(repos))
	var i int
	for repo := range repos {
		res[i] = repo
		i++
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

//InvolvedIntegrations returns all integrations used in the workflow
func (w *Workflow) InvolvedIntegrations() []int64 {
	if w.Root == nil {
		return nil
	}

	res := w.Root.InvolvedIntegrations()
	for _, j := range w.Joins {
		for _, t := range j.Triggers {
			res = append(res, t.WorkflowDestNode.InvolvedIntegrations()...)
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

//Visit all the workflow and apply the visitor func on all nodes
func (w *Workflow) VisitNode(visitor func(*Node, *Workflow)) {
	w.WorkflowData.Node.VisitNode(w, visitor)
	for i := range w.WorkflowData.Joins {
		for j := range w.WorkflowData.Joins[i].Triggers {
			n := &w.WorkflowData.Joins[i].Triggers[j].ChildNode
			n.VisitNode(w, visitor)
		}
	}
}

//Sort sorts the workflow
func (w *Workflow) SortNode() {
	w.VisitNode(func(n *Node, w *Workflow) {
		n.Sort()
	})
	for _, join := range w.WorkflowData.Joins {
		sort.Slice(join.Triggers, func(i, j int) bool {
			return join.Triggers[i].ChildNode.Name < join.Triggers[j].ChildNode.Name
		})
	}
}

//Sort sorts the workflow
func (w *Workflow) Sort() {
	w.Visit(func(n *WorkflowNode) {
		n.Sort()
	})
	for _, join := range w.Joins {
		sort.Slice(join.Triggers, func(i, j int) bool {
			return join.Triggers[i].WorkflowDestNode.Name < join.Triggers[j].WorkflowDestNode.Name
		})
	}
}

// AssignEmptyType fill node type field
func (w *Workflow) AssignEmptyType() {
	// set node type for join
	for i := range w.WorkflowData.Joins {
		j := &w.WorkflowData.Joins[i]
		j.Type = NodeTypeJoin
	}

	nodesArray := w.WorkflowData.Array()
	for i := range nodesArray {
		n := nodesArray[i]
		if n.Type == "" {
			if n.Context != nil && n.Context.PipelineID != 0 {
				n.Type = NodeTypePipeline
			} else if n.OutGoingHookContext != nil && n.OutGoingHookContext.HookModelID != 0 {
				n.Type = NodeTypeOutGoingHook
			} else {
				n.Type = NodeTypeFork
			}
		}
	}
}

// ValidateType check if nodes have a correct nodeType
func (w *Workflow) ValidateType() error {
	namesInError := make([]string, 0)

	for _, n := range w.WorkflowData.Array() {
		switch n.Type {
		case NodeTypePipeline:
			if n.Context == nil || (n.Context.PipelineID == 0 && n.Context.PipelineName == "") {
				namesInError = append(namesInError, n.Name)
			}
		case NodeTypeOutGoingHook:
			if n.OutGoingHookContext == nil || (n.OutGoingHookContext.HookModelID == 0 && n.OutGoingHookContext.HookModelName == "") {
				namesInError = append(namesInError, n.Name)
			}
		case NodeTypeJoin:
			if n.JoinContext == nil || len(n.JoinContext) == 0 {
				namesInError = append(namesInError, n.Name)
			}
		case NodeTypeFork:
			if (n.Context != nil && (n.Context.PipelineID != 0 || n.Context.PipelineName != "")) ||
				(n.OutGoingHookContext != nil && (n.OutGoingHookContext.HookModelID != 0 || n.OutGoingHookContext.HookModelName != "")) ||
				(n.JoinContext != nil && len(n.JoinContext) > 0) {
				namesInError = append(namesInError, n.Name)
			}
		default:
			namesInError = append(namesInError, n.Name)
		}
	}
	if len(namesInError) > 0 {
		return WithStack(fmt.Errorf("wrong type for nodes %v", namesInError))
	}
	return nil
}

//Visit all the workflow and apply the visitor func on the current node and the children
func (n *WorkflowNode) Visit(visitor func(*WorkflowNode)) {
	visitor(n)
	for i := range n.Triggers {
		d := &n.Triggers[i].WorkflowDestNode
		d.Visit(visitor)
	}
	for i := range n.OutgoingHooks {
		for j := range n.OutgoingHooks[i].Triggers {
			d := &n.OutgoingHooks[i].Triggers[j].WorkflowDestNode
			d.Visit(visitor)
		}
	}
	for i := range n.Forks {
		for j := range n.Forks[i].Triggers {
			d := &n.Forks[i].Triggers[j].WorkflowDestNode
			d.Visit(visitor)
		}
	}
}

//Sort sorts the workflow node
func (n *WorkflowNode) Sort() {
	sort.Slice(n.Triggers, func(i, j int) bool {
		return n.Triggers[i].WorkflowDestNode.Name < n.Triggers[j].WorkflowDestNode.Name
	})
}

//WorkflowNodeJoin aims to joins multiple node into multiple triggers
type WorkflowNodeJoin struct {
	ID             int64                     `json:"id" db:"id"`
	Ref            string                    `json:"ref" db:"-"`
	WorkflowID     int64                     `json:"workflow_id" db:"workflow_id"`
	SourceNodeIDs  []int64                   `json:"source_node_id,omitempty" db:"-"`
	SourceNodeRefs []string                  `json:"source_node_ref,omitempty" db:"-"`
	Triggers       []WorkflowNodeJoinTrigger `json:"triggers,omitempty" db:"-"`
}

func (j WorkflowNodeJoin) migrate(withID bool) Node {
	newNode := Node{
		Name:        j.Ref,
		Ref:         j.Ref,
		Type:        NodeTypeJoin,
		JoinContext: make([]NodeJoin, 0, len(j.SourceNodeIDs)),
		Triggers:    make([]NodeTrigger, 0, len(j.Triggers)),
	}
	if newNode.Ref == "" {
		newNode.Ref = RandomString(5)
	}
	if withID {
		newNode.ID = j.ID
	}
	for i := range j.SourceNodeIDs {
		newNode.JoinContext = append(newNode.JoinContext, NodeJoin{
			ParentID:   j.SourceNodeIDs[i],
			ParentName: j.SourceNodeRefs[i],
		})
	}

	for _, t := range j.Triggers {
		child := t.WorkflowDestNode.migrate(withID)
		newNode.Triggers = append(newNode.Triggers, NodeTrigger{
			ParentNodeName: newNode.Name,
			ChildNode:      child,
		})
	}
	return newNode
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
	ID                 int64                      `json:"id" db:"id"`
	Name               string                     `json:"name" db:"name"`
	Ref                string                     `json:"ref,omitempty" db:"-"`
	WorkflowID         int64                      `json:"workflow_id" db:"workflow_id"`
	PipelineID         int64                      `json:"pipeline_id" db:"pipeline_id"`
	PipelineName       string                     `json:"pipeline_name" db:"-"`
	DeprecatedPipeline Pipeline                   `json:"pipeline" db:"-"`
	Context            *WorkflowNodeContext       `json:"context" db:"-"`
	TriggerSrcID       int64                      `json:"-" db:"-"`
	TriggerJoinSrcID   int64                      `json:"-" db:"-"`
	TriggerHookSrcID   int64                      `json:"-" db:"-"`
	TriggerSrcForkID   int64                      `json:"-" db:"-"`
	Hooks              []WorkflowNodeHook         `json:"hooks,omitempty" db:"-"`
	Forks              []WorkflowNodeFork         `json:"forks,omitempty" db:"-"`
	Triggers           []WorkflowNodeTrigger      `json:"triggers,omitempty" db:"-"`
	OutgoingHooks      []WorkflowNodeOutgoingHook `json:"outgoing_hooks,omitempty" db:"-"`
}

func (n Node) retroMigrate() WorkflowNode {
	newNode := WorkflowNode{
		Ref:        n.Ref,
		Name:       n.Name,
		WorkflowID: n.WorkflowID,
		Context: &WorkflowNodeContext{
			ProjectIntegrationID:      n.Context.ProjectIntegrationID,
			EnvironmentID:             n.Context.EnvironmentID,
			ApplicationID:             n.Context.ApplicationID,
			DefaultPipelineParameters: n.Context.DefaultPipelineParameters,
			DefaultPayload:            n.Context.DefaultPayload,
			Mutex:                     n.Context.Mutex,
			Conditions:                n.Context.Conditions,
		},
		PipelineID:    n.Context.PipelineID,
		OutgoingHooks: nil,
		Hooks:         make([]WorkflowNodeHook, 0, len(n.Hooks)),
		Triggers:      nil,
		Forks:         nil,
	}

	for _, h := range n.Hooks {
		hook := WorkflowNodeHook{
			UUID:                h.UUID,
			Ref:                 h.Ref,
			WorkflowHookModelID: h.HookModelID,
			Config:              h.Config,
		}
		newNode.Hooks = append(newNode.Hooks, hook)
	}

	for _, t := range n.Triggers {
		switch t.ChildNode.Type {
		case NodeTypePipeline:
			trig := WorkflowNodeTrigger{
				WorkflowDestNode: t.ChildNode.retroMigrate(),
			}
			newNode.Triggers = append(newNode.Triggers, trig)
		case NodeTypeFork:
			newNode.Forks = append(newNode.Forks, t.ChildNode.retroMigrateFork())
		case NodeTypeOutGoingHook:
			newNode.OutgoingHooks = append(newNode.OutgoingHooks, t.ChildNode.retroMigrateOutGoingHook())
		}
	}
	return newNode
}

func (n Node) retroMigrateFork() WorkflowNodeFork {
	fork := WorkflowNodeFork{
		Name: n.Name,
	}
	if len(n.Triggers) > 0 {
		fork.Triggers = make([]WorkflowNodeForkTrigger, 0, len(n.Triggers))
	}
	for _, t := range n.Triggers {
		trig := WorkflowNodeForkTrigger{}
		switch t.ChildNode.Type {
		case NodeTypePipeline:
			trig.WorkflowDestNode = t.ChildNode.retroMigrate()
		default:
			continue
		}
		fork.Triggers = append(fork.Triggers, trig)
	}
	return fork
}

func (n Node) retroMigrateOutGoingHook() WorkflowNodeOutgoingHook {
	h := WorkflowNodeOutgoingHook{
		Config:              n.OutGoingHookContext.Config,
		WorkflowHookModelID: n.OutGoingHookContext.HookModelID,
		Ref:                 n.Ref,
		Name:                n.Name,
	}
	if len(n.Triggers) > 0 {
		h.Triggers = make([]WorkflowNodeOutgoingHookTrigger, 0, len(n.Triggers))
		for _, t := range n.Triggers {
			trig := WorkflowNodeOutgoingHookTrigger{}
			switch t.ChildNode.Type {
			case NodeTypePipeline:
				trig.WorkflowDestNode = t.ChildNode.retroMigrate()
			default:
				continue
			}
			h.Triggers = append(h.Triggers, trig)
		}
	}
	return h
}

func (n Node) retroMigrateJoin() WorkflowNodeJoin {
	j := WorkflowNodeJoin{
		Ref: n.Ref,
	}

	j.SourceNodeRefs = make([]string, 0, len(n.JoinContext))
	for _, jc := range n.JoinContext {
		j.SourceNodeRefs = append(j.SourceNodeRefs, jc.ParentName)
	}

	if len(n.Triggers) > 0 {
		j.Triggers = make([]WorkflowNodeJoinTrigger, 0, len(n.Triggers))
		for _, t := range n.Triggers {
			trig := WorkflowNodeJoinTrigger{}
			switch t.ChildNode.Type {
			case NodeTypePipeline:
				trig.WorkflowDestNode = t.ChildNode.retroMigrate()
			default:
				continue
			}
			j.Triggers = append(j.Triggers, trig)
		}
	}

	return j
}

func (n WorkflowNode) migrate(withID bool) Node {
	newNode := Node{
		WorkflowID: n.WorkflowID,
		Type:       NodeTypePipeline,
		Name:       n.Name,
		Ref:        n.Ref,
		Context: &NodeContext{
			PipelineID:                n.PipelineID,
			ApplicationID:             n.Context.ApplicationID,
			EnvironmentID:             n.Context.EnvironmentID,
			ProjectIntegrationID:      n.Context.ProjectIntegrationID,
			Conditions:                n.Context.Conditions,
			DefaultPayload:            n.Context.DefaultPayload,
			DefaultPipelineParameters: n.Context.DefaultPipelineParameters,
			Mutex:                     n.Context.Mutex,
		},
		Hooks:    make([]NodeHook, 0, len(n.Hooks)),
		Triggers: make([]NodeTrigger, 0, len(n.Triggers)+len(n.Forks)+len(n.OutgoingHooks)),
	}
	if n.Context.ApplicationID == 0 && n.Context.Application != nil {
		newNode.Context.ApplicationID = n.Context.Application.ID
	}
	if n.Context.EnvironmentID == 0 && n.Context.Environment != nil {
		newNode.Context.EnvironmentID = n.Context.Environment.ID
	}
	if n.Context.ProjectIntegrationID == 0 && n.Context.ProjectIntegration != nil {
		newNode.Context.ProjectIntegrationID = n.Context.ProjectIntegration.ID
	}
	if withID {
		newNode.ID = n.ID
	}
	if n.Ref == "" {
		n.Ref = n.Name
	}

	for _, h := range n.Hooks {
		nh := NodeHook{
			Ref:         h.Ref,
			HookModelID: h.WorkflowHookModelID,
			Config:      h.Config,
			UUID:        h.UUID,
		}
		if withID {
			nh.ID = h.ID
		}
		newNode.Hooks = append(newNode.Hooks, nh)
	}

	for _, t := range n.Triggers {
		triggeredNode := t.WorkflowDestNode.migrate(withID)
		newNode.Triggers = append(newNode.Triggers, NodeTrigger{
			ParentNodeName: n.Name,
			ChildNode:      triggeredNode,
		})
	}

	for _, f := range n.Forks {
		forkNode := f.migrate(withID)
		newNode.Triggers = append(newNode.Triggers, NodeTrigger{
			ParentNodeName: n.Name,
			ChildNode:      forkNode,
		})
	}

	for _, h := range n.OutgoingHooks {
		ogh := h.migrate(withID)
		newNode.Triggers = append(newNode.Triggers, NodeTrigger{
			ParentNodeName: n.Name,
			ChildNode:      ogh,
		})
	}

	return newNode
}

func (n *WorkflowNode) ForksMap(forkMap *map[int64]WorkflowNodeFork, triggerMap *map[int64]string) {
	for _, f := range n.Forks {
		(*forkMap)[f.ID] = f
		for _, t := range f.Triggers {
			(*triggerMap)[t.ID] = f.Name
			(&t.WorkflowDestNode).ForksMap(forkMap, triggerMap)
		}
	}
	for _, t := range n.Triggers {
		(&t.WorkflowDestNode).ForksMap(forkMap, triggerMap)
	}
	for _, o := range n.OutgoingHooks {
		for _, t := range o.Triggers {
			(&t.WorkflowDestNode).ForksMap(forkMap, triggerMap)
		}
	}
}

// IsLinkedToRepo returns boolean to know if the node is linked to an application which is also linked to a repository
func (n *WorkflowNode) IsLinkedToRepo() bool {
	if n == nil {
		return false
	}
	return n.Context != nil && n.Context.Application != nil && n.Context.Application.RepositoryFullname != ""
}

// Application return an application and a boolean (false if no application)
func (n *WorkflowNode) Application() (a Application, b bool) {
	if n == nil {
		return a, false
	}
	if n.Context == nil {
		return a, false
	}
	if n.Context.Application == nil {
		return a, false
	}
	return *n.Context.Application, true
}

// Environment return an environment and a boolean (false if no environment)
func (n *WorkflowNode) Environment() (e Environment, b bool) {
	if n == nil {
		return e, false
	}
	if n.Context == nil {
		return e, false
	}
	if n.Context.Environment == nil {
		return e, false
	}
	return *n.Context.Environment, true
}

// ProjectIntegration return an ProjectIntegration and a boolean (false if no ProjectIntegration)
func (n *WorkflowNode) ProjectIntegration() (p ProjectIntegration, b bool) {
	if n == nil {
		return p, false
	}
	if n.Context == nil {
		return p, false
	}
	if n.Context.ProjectIntegration == nil {
		return p, false
	}
	return *n.Context.ProjectIntegration, true
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

//GetNodeByRef returns the node given its ref
func (n *WorkflowNode) GetNodeByRef(ref string) *WorkflowNode {
	if n == nil {
		return nil
	}
	if n.Ref == ref {
		return n
	}
	for i := range n.Triggers {
		t := &n.Triggers[i]
		n2 := (&t.WorkflowDestNode).GetNodeByRef(ref)
		if n2 != nil {
			return n2
		}
	}
	for i := range n.OutgoingHooks {
		for j := range n.OutgoingHooks[i].Triggers {
			n2 := (&n.OutgoingHooks[i].Triggers[j].WorkflowDestNode).GetNodeByRef(ref)
			if n2 != nil {
				return n2
			}
		}
	}
	for i := range n.Forks {
		for j := range n.Forks[i].Triggers {
			n2 := (&n.Forks[i].Triggers[j].WorkflowDestNode).GetNodeByRef(ref)
			if n2 != nil {
				return n2
			}
		}
	}

	return nil
}

func (n *WorkflowNode) GetForkByName(name string) *WorkflowNodeFork {
	if n == nil {
		return nil
	}
	for i := range n.Forks {
		f := &n.Forks[i]
		if f.Name == name {
			return f
		}

		for j := range f.Triggers {
			f2 := (&f.Triggers[j].WorkflowDestNode).GetForkByName(name)
			if f2 != nil {
				return f2
			}
		}
	}

	for j := range n.Triggers {
		n2 := (&n.Triggers[j].WorkflowDestNode).GetForkByName(name)
		if n2 != nil {
			return n2
		}
	}

	for i := range n.OutgoingHooks {
		for j := range n.OutgoingHooks[i].Triggers {
			n2 := (&n.OutgoingHooks[i].Triggers[j].WorkflowDestNode).GetForkByName(name)
			if n2 != nil {
				return n2
			}
		}
	}
	return nil
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
		n2 := t.WorkflowDestNode.GetNodeByName(name)
		if n2 != nil {
			return n2
		}
	}
	for i := range n.OutgoingHooks {
		for j := range n.OutgoingHooks[i].Triggers {
			n2 := (&n.OutgoingHooks[i].Triggers[j].WorkflowDestNode).GetNodeByName(name)
			if n2 != nil {
				return n2
			}
		}
	}
	for i := range n.Forks {
		for j := range n.Forks[i].Triggers {
			n2 := (&n.Forks[i].Triggers[j].WorkflowDestNode).GetNodeByName(name)
			if n2 != nil {
				return n2
			}
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
		n1 := t.WorkflowDestNode.GetNode(id)
		if n1 != nil {
			return n1
		}
	}
	for i := range n.OutgoingHooks {
		for j := range n.OutgoingHooks[i].Triggers {
			n2 := (&n.OutgoingHooks[i].Triggers[j].WorkflowDestNode).GetNode(id)
			if n2 != nil {
				return n2
			}
		}
	}
	for i := range n.Forks {
		for j := range n.Forks[i].Triggers {
			n2 := (&n.Forks[i].Triggers[j].WorkflowDestNode).GetNode(id)
			if n2 != nil {
				return n2
			}
		}
	}
	return nil
}

// ResetIDs resets node id for the following node and its triggers
func (n *WorkflowNode) ResetIDs() {
	n.ID = 0
	for i := range n.Triggers {
		t := &n.Triggers[i]
		(&t.WorkflowDestNode).ResetIDs()
	}
	for i := range n.OutgoingHooks {
		for j := range n.OutgoingHooks[i].Triggers {
			(&n.OutgoingHooks[i].Triggers[j].WorkflowDestNode).ResetIDs()
		}
	}

	for i := range n.Forks {
		for j := range n.Forks[i].Triggers {
			(&n.Forks[i].Triggers[j].WorkflowDestNode).ResetIDs()
		}
	}
}

//Nodes returns a slice with all node IDs
func (n *WorkflowNode) Nodes() []WorkflowNode {
	res := []WorkflowNode{*n}
	for _, t := range n.Triggers {
		res = append(res, t.WorkflowDestNode.Nodes()...)
	}
	for i := range n.OutgoingHooks {
		for j := range n.OutgoingHooks[i].Triggers {
			res = append(res, n.OutgoingHooks[i].Triggers[j].WorkflowDestNode.Nodes()...)
		}
	}
	for i := range n.Forks {
		for j := range n.Forks[i].Triggers {
			res = append(res, n.Forks[i].Triggers[j].WorkflowDestNode.Nodes()...)
		}
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
	for i := range node.Forks {
		for j := range node.Forks[i].Triggers {
			destNode := &node.Forks[i].Triggers[j].WorkflowDestNode
			if destNode.ID == id {
				res[node.ID] = true
				return res, true
			}
			ids, ok := ancestor(id, destNode, deep)
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
	for i := range n.OutgoingHooks {
		for j := range n.OutgoingHooks[i].Triggers {
			res = append(res, n.OutgoingHooks[i].Triggers[j].WorkflowDestNode.References()...)
		}
	}
	for i := range n.Forks {
		for j := range n.Forks[i].Triggers {
			res = append(res, n.Forks[i].Triggers[j].WorkflowDestNode.References()...)
		}
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
	for i := range n.OutgoingHooks {
		for j := range n.OutgoingHooks[i].Triggers {
			res = append(res, n.OutgoingHooks[i].Triggers[j].WorkflowDestNode.InvolvedApplications()...)
		}
	}
	for i := range n.Forks {
		for j := range n.Forks[i].Triggers {
			res = append(res, n.Forks[i].Triggers[j].WorkflowDestNode.InvolvedApplications()...)
		}
	}
	return res
}

//InvolvedPipelines returns all pipelines used in the workflow
func (n *WorkflowNode) InvolvedPipelines() []int64 {
	res := []int64{}
	if n.PipelineID != 0 {
		res = append(res, n.PipelineID)
	}
	for _, t := range n.Triggers {
		res = append(res, t.WorkflowDestNode.InvolvedPipelines()...)
	}
	for i := range n.OutgoingHooks {
		for j := range n.OutgoingHooks[i].Triggers {
			res = append(res, n.OutgoingHooks[i].Triggers[j].WorkflowDestNode.InvolvedPipelines()...)
		}
	}
	for i := range n.Forks {
		for j := range n.Forks[i].Triggers {
			res = append(res, n.Forks[i].Triggers[j].WorkflowDestNode.InvolvedPipelines()...)
		}
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
	for i := range n.OutgoingHooks {
		for j := range n.OutgoingHooks[i].Triggers {
			res = append(res, n.OutgoingHooks[i].Triggers[j].WorkflowDestNode.GetApplications()...)
		}
	}
	for i := range n.Forks {
		for j := range n.Forks[i].Triggers {
			res = append(res, n.Forks[i].Triggers[j].WorkflowDestNode.GetApplications()...)
		}
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
	for i := range n.OutgoingHooks {
		for j := range n.OutgoingHooks[i].Triggers {
			res = append(res, n.OutgoingHooks[i].Triggers[j].WorkflowDestNode.GetEnvironments()...)
		}
	}
	for i := range n.Forks {
		for j := range n.Forks[i].Triggers {
			res = append(res, n.Forks[i].Triggers[j].WorkflowDestNode.GetEnvironments()...)
		}
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
	for i := range n.OutgoingHooks {
		for j := range n.OutgoingHooks[i].Triggers {
			res = append(res, n.OutgoingHooks[i].Triggers[j].WorkflowDestNode.InvolvedEnvironments()...)
		}
	}
	for i := range n.Forks {
		for j := range n.Forks[i].Triggers {
			res = append(res, n.Forks[i].Triggers[j].WorkflowDestNode.InvolvedEnvironments()...)
		}
	}
	return res
}

//InvolvedIntegrations returns all integrations used in the workflow
func (n *WorkflowNode) InvolvedIntegrations() []int64 {
	res := []int64{}
	if n.Context != nil {
		if n.Context.ProjectIntegrationID == 0 && n.Context.ProjectIntegration != nil {
			n.Context.ProjectIntegrationID = n.Context.ProjectIntegration.ID
		}
		if n.Context.ProjectIntegrationID != 0 {
			res = []int64{n.Context.ProjectIntegrationID}
		}
	}
	for _, t := range n.Triggers {
		res = append(res, t.WorkflowDestNode.InvolvedIntegrations()...)
	}
	for i := range n.OutgoingHooks {
		for j := range n.OutgoingHooks[i].Triggers {
			res = append(res, n.OutgoingHooks[i].Triggers[j].WorkflowDestNode.InvolvedIntegrations()...)
		}
	}
	for i := range n.Forks {
		for j := range n.Forks[i].Triggers {
			res = append(res, n.Forks[i].Triggers[j].WorkflowDestNode.InvolvedIntegrations()...)
		}
	}
	return res
}

// CheckApplicationDeploymentStrategies checks application deployment strategies
func (n *WorkflowNode) CheckApplicationDeploymentStrategies(proj *Project) error {
	if n.Context == nil {
		return nil
	}
	if n.Context.Application == nil {
		return nil
	}

	var id = n.Context.ProjectIntegrationID
	if id == 0 && n.Context.ProjectIntegration != nil {
		id = n.Context.ProjectIntegration.ID
	}

	if id == 0 {
		return nil
	}

	pf := proj.GetIntegrationByID(id)
	if pf == nil {
		return fmt.Errorf("integration unavailable")
	}

	for _, a := range proj.Applications {
		if a.ID == n.Context.ApplicationID || (n.Context.Application != nil && n.Context.Application.ID == a.ID) {
			if _, has := a.DeploymentStrategies[pf.Name]; !has {
				return fmt.Errorf("integration %s unavailable", pf.Name)
			}
		}
	}

	return nil
}

//WorkflowNodeTrigger is a link between two pipelines in a workflow
type WorkflowNodeTrigger struct {
	ID                 int64        `json:"id" db:"id"`
	WorkflowNodeID     int64        `json:"workflow_node_id" db:"workflow_node_id"`
	WorkflowDestNodeID int64        `json:"workflow_dest_node_id" db:"workflow_dest_node_id"`
	WorkflowDestNode   WorkflowNode `json:"workflow_dest_node" db:"-"`
}

// WorkflowNodeForkTrigger is a link between a fork and a node
type WorkflowNodeForkTrigger struct {
	ID                 int64        `json:"id" db:"id"`
	WorkflowForkID     int64        `json:"workflow_node_fork_id" db:"workflow_node_fork_id"`
	WorkflowDestNodeID int64        `json:"workflow_dest_node_id" db:"workflow_dest_node_id"`
	WorkflowDestNode   WorkflowNode `json:"workflow_dest_node" db:"-"`
}

//WorkflowNodeOutgoingHookTrigger is a link between an outgoing hook and pipeline in a workflow
type WorkflowNodeOutgoingHookTrigger struct {
	ID                         int64        `json:"id" db:"id"`
	WorkflowNodeOutgoingHookID int64        `json:"workflow_node_outgoing_hook_id" db:"workflow_node_outgoing_hook_id"`
	WorkflowDestNodeID         int64        `json:"workflow_dest_node_id" db:"workflow_dest_node_id"`
	WorkflowDestNode           WorkflowNode `json:"workflow_dest_node" db:"-"`
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
	ProjectIntegration        *ProjectIntegration    `json:"project_integration" db:"-"`
	ProjectIntegrationID      int64                  `json:"project_integration_id" db:"project_integration_id"`
	DefaultPayload            interface{}            `json:"default_payload,omitempty" db:"-"`
	DefaultPipelineParameters []Parameter            `json:"default_pipeline_parameters,omitempty" db:"-"`
	Conditions                WorkflowNodeConditions `json:"conditions,omitempty" db:"-"`
	Mutex                     bool                   `json:"mutex"`
}

// HasDefaultPayload returns true if the node has a default payload
func (c *WorkflowNodeContext) HasDefaultPayload() bool {
	if c == nil {
		return false
	}
	if c.DefaultPayload == nil {
		return false
	}
	dumper := dump.NewDefaultEncoder()
	dumper.ExtraFields.DetailedMap = false
	dumper.ExtraFields.DetailedStruct = false
	dumper.ExtraFields.Len = false
	dumper.ExtraFields.Type = false
	m, _ := dumper.ToStringMap(c.DefaultPayload)
	return len(m) > 0
}

// DefaultPayloadToMap returns default payload to map
func (c *WorkflowNodeContext) DefaultPayloadToMap() (map[string]string, error) {
	if c == nil {
		return nil, fmt.Errorf("Workflow node context is nil")
	}
	if c.DefaultPayload == nil {
		return map[string]string{}, nil
	}
	dumper := dump.NewDefaultEncoder()
	dumper.ExtraFields.DetailedMap = false
	dumper.ExtraFields.DetailedStruct = false
	dumper.ExtraFields.Len = false
	dumper.ExtraFields.Type = false
	return dumper.ToStringMap(c.DefaultPayload)
}

//WorkflowNodeContextDefaultPayloadVCS represents a default payload when a workflow is attached to a repository Webhook
type WorkflowNodeContextDefaultPayloadVCS struct {
	GitBranch     string `json:"git.branch" db:"-"`
	GitTag        string `json:"git.tag" db:"-"`
	GitHash       string `json:"git.hash" db:"-"`
	GitAuthor     string `json:"git.author" db:"-"`
	GitHashBefore string `json:"git.hash.before" db:"-"`
	GitRepository string `json:"git.repository" db:"-"`
	GitMessage    string `json:"git.message" db:"-"`
}

// IsWorkflowNodeContextDefaultPayloadVCS checks with several way if the workflow node context has a default vcs payloas
func IsWorkflowNodeContextDefaultPayloadVCS(i interface{}) bool {
	_, ok := i.(WorkflowNodeContextDefaultPayloadVCS)
	if ok {
		return true
	}

	dumper := dump.NewDefaultEncoder()
	dumper.ExtraFields.DetailedMap = false
	dumper.ExtraFields.DetailedStruct = false
	dumper.ExtraFields.Len = false
	dumper.ExtraFields.Type = false

	mI, _ := dumper.ToMap(i)
	mD, _ := dumper.ToMap(WorkflowNodeContextDefaultPayloadVCS{})

	// compare interface keys with default payload keys
	hasKey := func(s string) bool {
		_, has := mI[s]
		return has
	}

	if len(mI) == len(mD) {
		for k := range mD {
			if !hasKey(k) {
				goto checkGitKey
			}
		}
		return true
	}

checkGitKey:
	return hasKey("git.branch") &&
		hasKey("git.hash") &&
		hasKey("git.author") &&
		hasKey("git.hash.before") &&
		hasKey("git.repository") &&
		hasKey("git.message")
}

// WorkflowNodeJobRunCount return nb workflow run job since 'since'
type WorkflowNodeJobRunCount struct {
	Count int64     `json:"version"`
	Since time.Time `json:"since"`
	Until time.Time `json:"until"`
}

// Label represent a label linked to a workflow
type Label struct {
	ID         int64  `json:"id" db:"id"`
	Name       string `json:"name" db:"name"`
	Color      string `json:"color" db:"color"`
	ProjectID  int64  `json:"project_id" db:"project_id"`
	WorkflowID int64  `json:"workflow_id,omitempty" db:"-"`
}

//Validate return error or update label if it is not valid
func (label *Label) Validate() error {
	if label.Name == "" {
		return WrapError(fmt.Errorf("Label must have a name"), "IsValid>")
	}
	if label.Color == "" {
		bytes := make([]byte, 4)
		if _, err := rand.Read(bytes); err != nil {
			return WrapError(err, "IsValid> Cannot create random color")
		}
		label.Color = "#" + hex.EncodeToString(bytes)
	} else {
		if !ColorRegexp.Match([]byte(label.Color)) {
			return ErrIconBadFormat
		}
	}

	return nil
}

// WorkflowToIDs returns ids of given workflows.
func WorkflowToIDs(ws []*Workflow) []int64 {
	ids := make([]int64, len(ws))
	for i := range ws {
		ids[i] = ws[i].ID
	}
	return ids
}
