package sdk

// Those are icon for hooks
const (
	GitlabIcon    = "Gitlab"
	GitHubIcon    = "Github"
	BitbucketIcon = "Bitbucket"
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

//WorkflowNodeHook represents a hook which cann trigger the workflow from a given node
type WorkflowNodeHook struct {
	ID                  int64                  `json:"id" db:"id"`
	UUID                string                 `json:"uuid" db:"uuid"`
	WorkflowNodeID      int64                  `json:"workflow_node_id" db:"workflow_node_id"`
	WorkflowHookModelID int64                  `json:"workflow_hook_model_id" db:"workflow_hook_model_id"`
	WorkflowHookModel   WorkflowHookModel      `json:"model" db:"-"`
	Config              WorkflowNodeHookConfig `json:"config" db:"-"`
}

// WorkflowHookModelBuiltin is a constant for the builtin hook models
const WorkflowHookModelBuiltin = "builtin"

//WorkflowNodeHookConfig represents the configguration for a WorkflowNodeHook
type WorkflowNodeHookConfig map[string]WorkflowNodeHookConfigValue

//Values return values of the WorkflowNodeHookConfig
func (cfg WorkflowNodeHookConfig) Values() map[string]string {
	r := make(map[string]string)
	for k, v := range cfg {
		r[k] = v.Value
	}
	return r
}

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
