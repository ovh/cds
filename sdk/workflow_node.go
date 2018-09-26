package sdk

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
}

//NodeHook represents a hook which cann trigger the workflow from a given node
type NodeHook struct {
	ID          int64                  `json:"id" db:"id"`
	UUID        string                 `json:"uuid" db:"uuid"`
	Ref         string                 `json:"ref" db:"ref"`
	NodeID      int64                  `json:"node_id" db:"node_id"`
	HookModelID int64                  `json:"hook_model_id" db:"hook_model_id"`
	Config      WorkflowNodeHookConfig `json:"config" db:"-"`
}

// NodeContext represents a node linked to a pipeline
type NodeContext struct {
	ID                        int64                  `json:"id" db:"id"`
	NodeID                    int64                  `json:"node_id" db:"node_id"`
	PipelineID                int64                  `json:"pipeline_id" db:"pipeline_id"`
	ApplicationID             int64                  `json:"application_id" db:"application_id"`
	EnvironmentID             int64                  `json:"environment_id" db:"environment_id"`
	ProjectPlatformID         int64                  `json:"project_platform_id" db:"project_platform_id"`
	DefaultPayload            interface{}            `json:"default_payload" db:"-"`
	DefaultPipelineParameters []Parameter            `json:"default_pipeline_parameters" db:"-"`
	Conditions                WorkflowNodeConditions `json:"conditions" db:"-"`
	Mutex                     bool                   `json:"mutex" db:"mutex"`
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
	ID          int64                  `json:"id" db:"id"`
	NodeID      int64                  `json:"node_id" db:"node_id"`
	HookModelID int64                  `json:"hook_model_id" db:"hook_model_id"`
	UUID        string                 `json:"uuid" db:"uuid"`
	Config      WorkflowNodeHookConfig `json:"config" db:"-"`
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

func (n *Node) Ancestors(w *WorkflowData, mapNodes map[int64]*Node, deep bool) []int64 {
	if n == nil {
		return nil
	}

	res, ok := w.Node.ancestor(n.ID, deep)

	if !ok {
	joinLoop:
		for _, j := range w.Joins {
			for _, t := range j.Triggers {
				resAncestor, ok := (&t.ChildNode).ancestor(n.ID, deep)
				if ok {
					if len(resAncestor) == 1 || deep {
						for id := range resAncestor {
							res[id] = true
						}
					}

					if len(resAncestor) == 0 || deep {
						for _, jc := range j.JoinContext {
							res[jc.ParentID] = true
							if deep {
								node := mapNodes[jc.ParentID]
								if node != nil {
									ancerstorRes := node.Ancestors(w, mapNodes, deep)
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
