package sdk

// Those are icon for hooks
const (
	GitlabIcon    = "Gitlab"
	GitHubIcon    = "Github"
	BitbucketIcon = "Bitbucket"
	GerritIcon    = "git"
)

// FilterHooksConfig filter all hooks configuration and remove some configuration key
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
	if w == nil {
		return nil
	}

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

// WorkflowNodeOutgoingHook represents a outgoing hook
type WorkflowNodeOutgoingHook struct {
	ID                  int64                             `json:"id" db:"id"`
	Name                string                            `json:"name" db:"name"`
	Ref                 string                            `json:"ref" db:"-"`
	WorkflowNodeID      int64                             `json:"workflow_node_id" db:"workflow_node_id"`
	WorkflowHookModelID int64                             `json:"workflow_hook_model_id" db:"workflow_hook_model_id"`
	WorkflowHookModel   WorkflowHookModel                 `json:"model" db:"-"`
	Config              WorkflowNodeHookConfig            `json:"config" db:"-"`
	Triggers            []WorkflowNodeOutgoingHookTrigger `json:"triggers,omitempty" db:"-"`
}

func (h WorkflowNodeOutgoingHook) migrate(withID bool) Node {
	newNode := Node{
		Name: h.Name,
		Ref:  h.Ref,
		Type: NodeTypeOutGoingHook,
		OutGoingHookContext: &NodeOutGoingHook{
			Config:      h.Config,
			HookModelID: h.WorkflowHookModelID,
		},
		Triggers: make([]NodeTrigger, 0, len(h.Triggers)),
	}
	if withID {
		newNode.ID = h.ID
	}
	if h.Ref == "" {
		h.Ref = h.Name
	}
	for _, t := range h.Triggers {
		child := t.WorkflowDestNode.migrate(withID)
		newNode.Triggers = append(newNode.Triggers, NodeTrigger{
			ParentNodeName: newNode.Name,
			ChildNode:      child,
		})
	}
	return newNode
}

//WorkflowNodeFork represents a hook which cann trigger the workflow from a given node
type WorkflowNodeFork struct {
	ID             int64                     `json:"id" db:"id"`
	Name           string                    `json:"name" db:"name"`
	WorkflowNodeID int64                     `json:"workflow_node_id" db:"workflow_node_id"`
	Triggers       []WorkflowNodeForkTrigger `json:"triggers,omitempty" db:"-"`
}

func (f WorkflowNodeFork) migrate(withID bool) Node {
	newNode := Node{
		Name:     f.Name,
		Ref:      f.Name,
		Type:     NodeTypeFork,
		Triggers: make([]NodeTrigger, 0, len(f.Triggers)),
	}
	if withID {
		newNode.ID = f.ID
	}
	if newNode.Ref == "" {
		newNode.Ref = newNode.Name
	}
	for _, t := range f.Triggers {
		child := t.WorkflowDestNode.migrate(withID)
		newNode.Triggers = append(newNode.Triggers, NodeTrigger{
			ParentNodeName: newNode.Name,
			ChildNode:      child,
		})
	}
	return newNode
}

//WorkflowNodeHook represents a hook which cann trigger the workflow from a given node
type WorkflowNodeHook struct {
	ID                  int64                  `json:"id" db:"id"`
	UUID                string                 `json:"uuid" db:"uuid"`
	Ref                 string                 `json:"ref" db:"ref"`
	WorkflowNodeID      int64                  `json:"workflow_node_id" db:"workflow_node_id"`
	WorkflowHookModelID int64                  `json:"workflow_hook_model_id" db:"workflow_hook_model_id"`
	WorkflowHookModel   WorkflowHookModel      `json:"model" db:"-"`
	Config              WorkflowNodeHookConfig `json:"config" db:"-"`
}

//Equals checks functionnal equality between two hooks
func (h WorkflowNodeHook) Equals(h1 WorkflowNodeHook) bool {
	if h.UUID != h1.UUID {
		return false
	}
	if h.WorkflowHookModelID != h1.WorkflowHookModelID {
		return false
	}
	for k, cfg := range h.Config {
		cfg1, has := h1.Config[k]
		if !has {
			return false
		}
		if cfg.Value != cfg1.Value {
			return false
		}
	}
	for k, cfg1 := range h1.Config {
		cfg, has := h.Config[k]
		if !has {
			return false
		}
		if cfg.Value != cfg1.Value {
			return false
		}
	}
	return true
}

// WorkflowHookModelBuiltin is a constant for the builtin hook models
const WorkflowHookModelBuiltin = "builtin"

//WorkflowNodeHookConfig represents the configguration for a WorkflowNodeHook
type WorkflowNodeHookConfig map[string]WorkflowNodeHookConfigValue

// GetBuiltinHookModelByName retrieve the hook model
func GetBuiltinHookModelByName(name string) *WorkflowHookModel {
	for _, m := range BuiltinHookModels {
		if m.Name == name {
			return m
		}
	}
	return nil
}

// GetBuiltinOutgoingHookModelByName retrieve the outgoing hook model
func GetBuiltinOutgoingHookModelByName(name string) *WorkflowHookModel {
	for _, m := range BuiltinOutgoingHookModels {
		if m.Name == name {
			return m
		}
	}
	return nil
}

//Values return values of the WorkflowNodeHookConfig
func (cfg WorkflowNodeHookConfig) Values(model WorkflowNodeHookConfig) map[string]string {
	r := make(map[string]string)
	for k, v := range cfg {
		if model[k].Configurable {
			r[k] = v.Value
		}
	}
	return r
}

// Clone returns a copied dinstance of cfg
func (cfg WorkflowNodeHookConfig) Clone() WorkflowNodeHookConfig {
	m := WorkflowNodeHookConfig(make(map[string]WorkflowNodeHookConfigValue, len(cfg)))
	for k, v := range cfg {
		m[k] = v
	}
	return m
}

// WorkflowNodeHookConfigValue represents the value of a node hook config
type WorkflowNodeHookConfigValue struct {
	Value              string   `json:"value"`
	Configurable       bool     `json:"configurable"`
	Type               string   `json:"type"`
	MultipleChoiceList []string `json:"multiple_choice_list"`
}

const (
	// HookConfigTypeString type string
	HookConfigTypeString = "string"
	// HookConfigTypeIntegration type integration
	HookConfigTypeIntegration = "integration"
	// HookConfigTypeProject type project
	HookConfigTypeProject = "project"
	// HookConfigTypeWorkflow type workflow
	HookConfigTypeWorkflow = "workflow"
	// HookConfigTypeHook type hook
	HookConfigTypeHook = "hook"
	// HookConfigTypeMultiChoice type multiple
	HookConfigTypeMultiChoice = "multiple"
)

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
