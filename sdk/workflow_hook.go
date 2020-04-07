package sdk

import (
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
)

// Those are icon for hooks
const (
	GitlabIcon    = "Gitlab"
	GitHubIcon    = "Github"
	BitbucketIcon = "Bitbucket"
	GerritIcon    = "git"
)

//NodeHook represents a hook which cann trigger the workflow from a given node
type NodeHook struct {
	ID            int64                  `json:"id" db:"id"`
	UUID          string                 `json:"uuid" db:"uuid"`
	NodeID        int64                  `json:"node_id" db:"node_id"`
	HookModelID   int64                  `json:"hook_model_id" db:"hook_model_id"`
	HookModelName string                 `json:"hook_model_name" db:"-"`
	Config        WorkflowNodeHookConfig `json:"config" db:"config"`
	Conditions    WorkflowNodeConditions `json:"conditions" db:"conditions"`
}

func (h NodeHook) Ref() string {
	s := "model:" + h.HookModelName + ";"

	var mapKeys = reflect.ValueOf(h.Config).MapKeys()
	sort.Slice(mapKeys, func(i, j int) bool {
		return mapKeys[i].String() < mapKeys[j].String()
	})
	for _, k := range mapKeys {
		cfg := h.Config[k.String()]
		if cfg.Configurable {
			s += k.String() + ":" + cfg.Value + ";"
		}
	}

	return base64.StdEncoding.EncodeToString([]byte(s))
}

//Equals checks functional equality between two hooks
func (h NodeHook) Equals(h1 NodeHook) bool {
	if h.UUID != h1.UUID {
		return false
	}
	if h.HookModelID != h1.HookModelID {
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

// FilterHooksConfig filter all hooks configuration and remove some configuration key
func (w *Workflow) FilterHooksConfig(s ...string) {
	w.WorkflowData.Node.FilterHooksConfig(s...)
	for i := range w.WorkflowData.Joins {
		w.WorkflowData.Joins[i].FilterHooksConfig(s...)
	}
}

// WorkflowHookModelBuiltin is a constant for the builtin hook models
const WorkflowHookModelBuiltin = "builtin"

//WorkflowNodeHookConfig represents the configguration for a WorkflowNodeHook
type WorkflowNodeHookConfig map[string]WorkflowNodeHookConfigValue

// Value returns driver.Value from WorkflowNodeHookConfig request.
func (w WorkflowNodeHookConfig) Value() (driver.Value, error) {
	j, err := json.Marshal(w)
	return j, WrapError(err, "cannot marshal WorkflowNodeHookConfig")
}

// Scan workflow template request.
func (w *WorkflowNodeHookConfig) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(json.Unmarshal(source, w), "cannot unmarshal WorkflowNodeHookConfig")
}

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
