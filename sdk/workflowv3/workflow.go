package workflowv3

import (
	"fmt"
	"sort"

	"github.com/ovh/cds/sdk"
	"github.com/pkg/errors"
)

func NewWorkflow() Workflow {
	return Workflow{
		Repositories:  make(map[string]Repository),
		Hooks:         make(map[string]Hook),
		Deployments:   make(map[string]Deployment),
		Notifications: make(map[string]Notification),
		Stages:        make(map[string]Stage),
		Jobs:          make(map[string]Job),
		Variables:     make(map[string]Variable),
		Secrets:       make(map[string]Secret),
		Actions:       make(map[string]Action),
		Keys:          make(map[string]Key),
	}
}

type Workflow struct {
	Name          string                  `json:"name,omitempty" yaml:"name,omitempty"`
	Repositories  Repositories            `json:"repositories,omitempty" yaml:"repositories,omitempty"`
	Hooks         map[string]Hook         `json:"hooks,omitempty" yaml:"hooks,omitempty"`
	Deployments   Deployments             `json:"deployments,omitempty" yaml:"deployments,omitempty"`
	Notifications map[string]Notification `json:"notifications,omitempty" yaml:"notifications,omitempty"`
	Stages        Stages                  `json:"stages,omitempty" yaml:"stages,omitempty"`
	Jobs          Jobs                    `json:"jobs,omitempty" yaml:"jobs,omitempty"`
	Variables     Variables               `json:"variables,omitempty" yaml:"variables,omitempty"`
	Secrets       Secrets                 `json:"secrets,omitempty" yaml:"secrets,omitempty"`
	Actions       Actions                 `json:"actions,omitempty" yaml:"actions,omitempty"`
	Keys          Keys                    `json:"keys,omitempty" yaml:"keys,omitempty"`
}

func (w Workflow) Validate() (ExternalDependencies, error) {
	var extDep ExternalDependencies

	rx := sdk.NamePatternRegex
	if !rx.MatchString(w.Name) {
		return extDep, fmt.Errorf("workflow name %q do not respect pattern %q", w.Name, sdk.NamePattern)
	}

	if len(w.Jobs) == 0 {
		return extDep, fmt.Errorf("workflow should contains at least one job")
	}

	// Validate repositories
	for rName, r := range w.Repositories {
		dep, err := r.Validate(w)
		if err != nil {
			return extDep, errors.WithMessagef(err, "repository %q", rName)
		}
		extDep.Add(dep)
	}

	// Validate hooks
	for hName, h := range w.Hooks {
		dep, err := h.Validate(w)
		if err != nil {
			return extDep, errors.WithMessagef(err, "hook %q", hName)
		}
		extDep.Add(dep)
	}

	// Validate deployments
	for dName, d := range w.Deployments {
		dep, err := d.Validate()
		if err != nil {
			return extDep, errors.WithMessagef(err, "deployment %q", dName)
		}
		extDep.Add(dep)
	}

	// Validate notification
	for nName, n := range w.Notifications {
		if err := n.Validate(w); err != nil {
			return extDep, errors.WithMessagef(err, "notification %q", nName)
		}
	}

	// Validate actions
	for aName, a := range w.Actions {
		dep, err := a.Validate(w)
		if err != nil {
			return extDep, errors.WithMessagef(err, "action %q", aName)
		}
		extDep.Add(dep)
	}
	if err := w.Actions.ToGraph().DetectLoops(); err != nil {
		return extDep, errors.WithMessagef(err, "can't validate actions")
	}

	// Validate keys
	for kName, k := range w.Keys {
		if err := k.Validate(w); err != nil {
			return extDep, errors.WithMessagef(err, "key %q", kName)
		}
	}

	// Validate stages
	for sName, s := range w.Stages {
		if err := s.Validate(sName, w); err != nil {
			return extDep, errors.WithMessagef(err, "stage %q", sName)
		}
	}
	if err := w.Stages.ToGraph().DetectLoops(); err != nil {
		return extDep, errors.WithMessagef(err, "can't validate stages")
	}

	// Validate jobs
	for jName, j := range w.Jobs {
		dep, err := j.Validate(w)
		if err != nil {
			return extDep, errors.WithMessagef(err, "job %q", jName)
		}
		extDep.Add(dep)
	}
	for _, g := range w.Jobs.ToGraphs() {
		if err := g.DetectLoops(); err != nil {
			return extDep, errors.WithMessagef(err, "can't validate jobs")
		}
	}

	return extDep, nil
}

func (w *Workflow) Add(wf Workflow) error {
	for rName, r := range wf.Repositories {
		if _, ok := w.Repositories[rName]; ok {
			return fmt.Errorf("repository %q already declared", rName)
		}
		w.Repositories[rName] = r
	}
	for hName, h := range wf.Hooks {
		if _, ok := w.Hooks[hName]; ok {
			return fmt.Errorf("hook %q already declared", hName)
		}
		w.Hooks[hName] = h
	}
	for dName, d := range wf.Deployments {
		if _, ok := w.Deployments[dName]; ok {
			return fmt.Errorf("deployment %q already declared", dName)
		}
		w.Deployments[dName] = d
	}
	for nName, n := range wf.Notifications {
		if _, ok := w.Notifications[nName]; ok {
			return fmt.Errorf("notification %q already declared", nName)
		}
		w.Notifications[nName] = n
	}
	for sName, s := range wf.Stages {
		if _, ok := w.Stages[sName]; ok {
			return fmt.Errorf("stage %q already declared", sName)
		}
		w.Stages[sName] = s
	}
	for jName, j := range wf.Jobs {
		if _, ok := w.Jobs[jName]; ok {
			return fmt.Errorf("job %q already declared", jName)
		}
		w.Jobs[jName] = j
	}
	for vName, v := range wf.Variables {
		if _, ok := w.Variables[vName]; ok {
			return fmt.Errorf("variable %q already declared", vName)
		}
		w.Variables[vName] = v
	}
	for sName, s := range wf.Secrets {
		if _, ok := w.Secrets[sName]; ok {
			return fmt.Errorf("secret %q already declared", sName)
		}
		w.Secrets[sName] = s
	}
	for aName, a := range wf.Actions {
		if _, ok := w.Actions[aName]; ok {
			return fmt.Errorf("action %q already declared", aName)
		}
		w.Actions[aName] = a
	}
	for kName, k := range wf.Keys {
		if _, ok := w.Keys[kName]; ok {
			return fmt.Errorf("key %q already declared", kName)
		}
		w.Keys[kName] = k
	}
	return nil
}

type ExternalDependencies struct {
	Repositories []string `json:"repositories,omitempty"`
	Variables    []string `json:"variables,omitempty"`
	Secrets      []string `json:"secrets,omitempty"`
	VCSServers   []string `json:"vcs_servers,omitempty"`
	Actions      []string `json:"actions,omitempty"`
	Integrations []string `json:"integrations,omitempty"`
	Deployments  []string `json:"deployments,omitempty"`
	SSHKeys      []string `json:"ssh_keys,omitempty"`
	PGPKeys      []string `json:"pgp_keys,omitempty"`
}

func (e *ExternalDependencies) Add(d ExternalDependencies) {
	e.Repositories = deduplicateStrings(append(e.Repositories, d.Repositories...))
	e.Variables = deduplicateStrings(append(e.Variables, d.Variables...))
	e.Secrets = deduplicateStrings(append(e.Secrets, d.Secrets...))
	e.VCSServers = deduplicateStrings(append(e.VCSServers, d.VCSServers...))
	e.Actions = deduplicateStrings(append(e.Actions, d.Actions...))
	e.Integrations = deduplicateStrings(append(e.Integrations, d.Integrations...))
	e.SSHKeys = deduplicateStrings(append(e.SSHKeys, d.SSHKeys...))
	e.PGPKeys = deduplicateStrings(append(e.PGPKeys, d.PGPKeys...))
}

func deduplicateStrings(in []string) []string {
	m := make(map[string]struct{})
	for i := range in {
		m[in[i]] = struct{}{}
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

type Condition struct {
	Checks []Check `json:"checks,omitempty" yaml:"checks,omitempty"`
	Lua    string  `json:"lua_script,omitempty" yaml:"script,omitempty"`
}

func (c *Condition) Merge(o Condition) {
	c.Checks = append(c.Checks, o.Checks...)
	if c.Lua != "" && o.Lua != "" {
		c.Lua = `
    local function c1()
      ` + c.Lua + `
    end
    local function c2()
      ` + o.Lua + `
    end
    return c1() and c2();
    `
	} else if o.Lua != "" {
		c.Lua = o.Lua
	}
}

type Check struct {
	Variable string `json:"variable,omitempty" yaml:"variable,omitempty"`
	Operator string `json:"operator,omitempty" yaml:"operator,omitempty"`
	Value    string `json:"value,omitempty" yaml:"value,omitempty"`
}

type Graph []Node

func (g Graph) RemoveRootNodes() (int, Graph) {
	// Select root nodes
	rootNodes := make(map[string]struct{})
	var subGraph Graph
	for i := range g {
		if len(g[i].DependsOn) == 0 {
			rootNodes[g[i].Name] = struct{}{}
		} else {
			subGraph = append(subGraph, g[i])
		}
	}

	// Remove root nodes dependencies
	for i := range subGraph {
		var filteredDependsOn []string
		for _, d := range subGraph[i].DependsOn {
			if _, ok := rootNodes[d]; !ok {
				filteredDependsOn = append(filteredDependsOn, d)
			}
		}
		subGraph[i].DependsOn = filteredDependsOn
	}

	return len(rootNodes), subGraph
}

func (g Graph) DetectLoops() error {
	if len(g) == 0 {
		return nil
	}

	rootNodesCount, subGraph := g.RemoveRootNodes()

	// Return an error if no root nodes removed but there are still nodes in the graph
	if rootNodesCount == 0 && len(subGraph) > 0 {
		var nodeNames []string
		for _, n := range subGraph {
			nodeNames = append(nodeNames, n.Name)
		}
		return fmt.Errorf("dependency loop detected for %q", nodeNames)
	}

	return subGraph.DetectLoops()
}

type Node struct {
	Name      string
	DependsOn []string
}
