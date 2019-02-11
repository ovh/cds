package sdk

import (
	"fmt"
	"sort"

	"github.com/fsamin/go-dump"
)

const (
	NodeTypePipeline     = "pipeline"
	NodeTypeJoin         = "join"
	NodeTypeOutGoingHook = "outgoinghook"
	NodeTypeFork         = "fork"
)

// Node represents a node in a workflow
type Node struct {
	ID                  int64             `json:"id" db:"id"`
	WorkflowID          int64             `json:"workflow_id" db:"workflow_id"`
	Name                string            `json:"name" db:"name"`
	Ref                 string            `json:"ref" db:"ref"`
	Type                string            `json:"type" db:"type"`
	Triggers            []NodeTrigger     `json:"triggers" db:"-"`
	TriggerID           int64             `json:"-" db:"-"`
	Context             *NodeContext      `json:"context" db:"-"`
	OutGoingHookContext *NodeOutGoingHook `json:"outgoing_hook" db:"-"`
	JoinContext         []NodeJoin        `json:"parents" db:"-"`
	Hooks               []NodeHook        `json:"hooks" db:"-"`
	Groups              []GroupPermission `json:"groups,omitempty" db:"-"`
}

//NodeHook represents a hook which cann trigger the workflow from a given node
type NodeHook struct {
	ID            int64                  `json:"id" db:"id"`
	UUID          string                 `json:"uuid" db:"uuid"`
	Ref           string                 `json:"ref" db:"ref"`
	NodeID        int64                  `json:"node_id" db:"node_id"`
	HookModelID   int64                  `json:"hook_model_id" db:"hook_model_id"`
	HookModelName string                 `json:"-" db:"-"`
	Config        WorkflowNodeHookConfig `json:"config" db:"-"`
}

// NodeContext represents a node linked to a pipeline
type NodeContext struct {
	ID                        int64                  `json:"id" db:"id"`
	NodeID                    int64                  `json:"node_id" db:"node_id"`
	PipelineID                int64                  `json:"pipeline_id" db:"pipeline_id"`
	PipelineName              string                 `json:"-" db:"-"`
	ApplicationID             int64                  `json:"application_id" db:"application_id"`
	ApplicationName           string                 `json:"-" db:"-"`
	EnvironmentID             int64                  `json:"environment_id" db:"environment_id"`
	EnvironmentName           string                 `json:"-" db:"-"`
	ProjectIntegrationID      int64                  `json:"project_integration_id" db:"project_integration_id"`
	ProjectIntegrationName    string                 `json:"-" db:"-"`
	DefaultPayload            interface{}            `json:"default_payload,omitempty" db:"-"`
	DefaultPipelineParameters []Parameter            `json:"default_pipeline_parameters" db:"-"`
	Conditions                WorkflowNodeConditions `json:"conditions" db:"-"`
	Mutex                     bool                   `json:"mutex" db:"mutex"`
}

//AddTrigger adds a trigger to the destination node from the node found by its name
func (n *Node) AddTrigger(name string, dest Node) {
	if n.Name == name {
		n.Triggers = append(n.Triggers, NodeTrigger{
			ChildNode: dest,
		})
		return
	}
	for i := range n.Triggers {
		destNode := &n.Triggers[i].ChildNode
		destNode.AddTrigger(name, dest)
	}
}

//Sort sorts the workflow node
func (n *Node) Sort() {
	sort.Slice(n.Triggers, func(i, j int) bool {
		return n.Triggers[i].ChildNode.Name < n.Triggers[j].ChildNode.Name
	})
}

//VisitNode all the workflow and apply the visitor func on the current node and the children
func (n *Node) VisitNode(w *Workflow, visitor func(node *Node, w *Workflow)) {
	visitor(n, w)
	for i := range n.Triggers {
		d := &n.Triggers[i].ChildNode
		d.VisitNode(w, visitor)
	}
}

func (c *NodeContext) HasDefaultPayload() bool {
	if c == nil {
		return false
	}
	if c.DefaultPayload == nil {
		return false
	}
	dumper := dump.NewDefaultEncoder(nil)
	dumper.ExtraFields.DetailedMap = false
	dumper.ExtraFields.DetailedStruct = false
	dumper.ExtraFields.Len = false
	dumper.ExtraFields.Type = false
	m, _ := dumper.ToStringMap(c.DefaultPayload)
	return len(m) > 0

}
func (c *NodeContext) DefaultPayloadToMap() (map[string]string, error) {
	// DefaultPayloadToMap returns default payload to map
	if c == nil {
		return nil, fmt.Errorf("Workflow node context is nil")
	}
	if c.DefaultPayload == nil {
		return map[string]string{}, nil
	}
	dumper := dump.NewDefaultEncoder(nil)
	dumper.ExtraFields.DetailedMap = false
	dumper.ExtraFields.DetailedStruct = false
	dumper.ExtraFields.Len = false
	dumper.ExtraFields.Type = false
	return dumper.ToStringMap(c.DefaultPayload)

}

// NodeTrigger represents the link between 2 nodes
type NodeTrigger struct {
	ID             int64  `json:"id" db:"id"`
	ParentNodeID   int64  `json:"parent_node_id" db:"parent_node_id"`
	ChildNodeID    int64  `json:"child_node_id" db:"child_node_id"`
	ParentNodeName string `json:"parent_node_name" db:"-"`
	ChildNode      Node   `json:"child_node" db:"-"`
}

// NodeOutGoingHook represents the link between a node a its outgoings hooks
type NodeOutGoingHook struct {
	ID            int64                  `json:"id" db:"id"`
	NodeID        int64                  `json:"node_id" db:"node_id"`
	HookModelID   int64                  `json:"hook_model_id" db:"hook_model_id"`
	HookModelName string                 `json:"-" db:"-"`
	Config        WorkflowNodeHookConfig `json:"config" db:"-"`
}

// NodeJoin represents a join type node
type NodeJoin struct {
	ID         int64  `json:"id" db:"id"`
	NodeID     int64  `json:"node_id" db:"node_id"`
	ParentName string `json:"parent_name,omitempty" db:"-"`
	ParentID   int64  `json:"parent_id,omitempty" db:"parent_id"`
}

func (n *Node) nodeByRef(ref string) *Node {
	if n.Ref == ref {
		return n
	}
	for i := range n.Triggers {
		node := (&n.Triggers[i].ChildNode).nodeByRef(ref)
		if node != nil {
			return node
		}
	}
	return nil
}

func (n *Node) nodeByID(ID int64) *Node {
	if n.ID == ID {
		return n
	}
	for i := range n.Triggers {
		node := (&n.Triggers[i].ChildNode).nodeByID(ID)
		if node != nil {
			return node
		}
	}
	return nil
}

func (n *Node) array(a []*Node) []*Node {
	a = append(a, n)
	for i := range n.Triggers {
		a = (&n.Triggers[i].ChildNode).array(a)
	}
	return a
}

func (n *Node) maps(m map[int64]*Node) map[int64]*Node {
	m[n.ID] = n
	for i := range n.Triggers {
		m = (&n.Triggers[i].ChildNode).maps(m)
	}
	return m
}

func (n *Node) Ancestors(w *WorkflowData) []int64 {
	if n == nil {
		return nil
	}

	IDs := make([]int64, 0)
	if n.Type != NodeTypeJoin {
		for _, node := range w.Array() {
			for _, t := range node.Triggers {
				if t.ChildNode.ID == n.ID {
					IDs = append(IDs, node.ID)
				}
			}
		}
	} else {
		for _, jc := range n.JoinContext {
			IDs = append(IDs, jc.ParentID)
		}
	}

	return IDs
}

func (n *Node) ancestorNames(name string) ([]string, bool) {
	res := make([]string, 0)
	if name == n.Name {
		return res, true
	}
	for _, t := range n.Triggers {
		if t.ChildNode.Name == name {
			// If current node is a join
			if n.Type != NodeTypeJoin {
				res = append(res, n.Name)
				return res, true
			}

			parents := make([]string, 0, len(n.JoinContext))
			for _, jp := range n.JoinContext {
				parents = append(parents, jp.ParentName)
			}
			return parents, true
		}
		trigRes, ok := (&t.ChildNode).ancestorNames(name)
		if ok {
			res = append(res, trigRes...)
			return res, true
		}
	}
	return res, false
}

func (n *Node) ancestor(id int64, deep bool) (map[int64]bool, bool) {
	res := map[int64]bool{}
	if id == n.ID {
		return res, true
	}
	for _, t := range n.Triggers {
		if t.ChildNode.ID == id {
			res[n.ID] = true
			return res, true
		}
		ids, ok := (&t.ChildNode).ancestor(id, deep)
		if ok {
			if len(ids) == 1 || deep {
				for k := range ids {
					res[k] = true
				}
			}
			if deep {
				res[n.ID] = true
			}
			return res, true
		}
	}
	return res, false
}

// IsLinkedToRepo returns boolean to know if the node is linked to an application which is also linked to a repository
func (n *Node) IsLinkedToRepo(w *Workflow) bool {
	if n == nil {
		return false
	}
	return n.Context != nil && n.Context.ApplicationID != 0 && w.Applications[n.Context.ApplicationID].RepositoryFullname != ""
}

// CheckApplicationDeploymentStrategies checks application deployment strategies
func (n Node) CheckApplicationDeploymentStrategies(proj *Project, w *Workflow) error {
	if n.Context == nil {
		return nil
	}
	if n.Context.ApplicationID == 0 {
		return nil
	}

	if n.Context.ProjectIntegrationID == 0 {
		return nil
	}

	pf := proj.GetIntegrationByID(n.Context.ProjectIntegrationID)
	if pf == nil {
		return WithStack(fmt.Errorf("integration unavailable"))
	}

	app := w.Applications[n.Context.ApplicationID]
	if _, has := app.DeploymentStrategies[pf.Name]; !has {
		return WithStack(fmt.Errorf("integration %s unavailable on application %s", pf.Name, app.Name))
	}
	return nil
}
