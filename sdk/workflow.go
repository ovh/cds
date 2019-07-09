package sdk

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"time"
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

// WorkflowNotification represents notifications on a workflow
type WorkflowNotification struct {
	ID             int64                    `json:"id,omitempty" db:"id"`
	WorkflowID     int64                    `json:"workflow_id,omitempty" db:"workflow_id"`
	SourceNodeRefs []string                 `json:"source_node_ref,omitempty" db:"-"`
	NodeIDs        []int64                  `json:"node_id,omitempty" db:"-"`
	Type           string                   `json:"type" db:"type"`
	Settings       UserNotificationSettings `json:"settings" db:"-"`
}

// ResetIDs resets all nodes and joins ids
func (w *Workflow) ResetIDs() {
	for _, n := range w.WorkflowData.Array() {
		n.ID = 0
	}
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

// GetRepositories returns the list of repositories from applications
func (w *Workflow) GetRepositories() []string {
	repos := map[string]struct{}{}
	for _, a := range w.Applications {
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

//WorkflowNodeConditions is either an array of WorkflowNodeCondition or a lua script
type WorkflowNodeConditions struct {
	PlainConditions []WorkflowNodeCondition `json:"plain,omitempty" yaml:"check,omitempty"`
	LuaScript       string                  `json:"lua_script,omitempty" yaml:"script,omitempty"`
}

//WorkflowNodeCondition represents a condition to trigger ot not a pipeline in a workflow. Operator can be =, !=, regex
type WorkflowNodeCondition struct {
	Variable string `json:"variable" yaml:"variable"`
	Operator string `json:"operator" yaml:"operator"`
	Value    string `json:"value" yaml:"value"`
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
		bytes := make([]byte, 3)
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
