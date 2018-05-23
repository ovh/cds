package exportentities

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/fsamin/go-dump"

	"github.com/ovh/cds/sdk"
)

type Workflow struct {
	Name    string `json:"name" yaml:"name"`
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
	// This will be filled for complex workflows
	Workflow map[string]NodeEntry   `json:"workflow,omitempty" yaml:"workflow,omitempty"`
	Hooks    map[string][]HookEntry `json:"hooks,omitempty" yaml:"hooks,omitempty"`
	// This will be filled for simple workflows
	DependsOn           []string                    `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Conditions          *sdk.WorkflowNodeConditions `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	When                []string                    `json:"when,omitempty" yaml:"when,omitempty"` //This is use only for manual and success condition
	PipelineName        string                      `json:"pipeline,omitempty" yaml:"pipeline,omitempty"`
	Payload             map[string]interface{}      `json:"payload,omitempty" yaml:"payload,omitempty"`
	Parameters          map[string]string           `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	ApplicationName     string                      `json:"application,omitempty" yaml:"application,omitempty"`
	EnvironmentName     string                      `json:"environment,omitempty" yaml:"environment,omitempty"`
	ProjectPlatformName string                      `json:"platform,omitempty" yaml:"platform,omitempty"`
	PipelineHooks       []HookEntry                 `json:"pipeline_hooks,omitempty" yaml:"pipeline_hooks,omitempty"`
	Permissions         map[string]int              `json:"permissions,omitempty" yaml:"permissions,omitempty"`
	Metadata            map[string]string           `json:"metadata,omitempty" yaml:"metadata,omitempty" db:"-"`
	PurgeTags           []string                    `json:"purge_tags,omitempty" yaml:"purge_tags,omitempty" db:"-"`
}

type NodeEntry struct {
	ID                  int64                       `json:"-" yaml:"-"`
	DependsOn           []string                    `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Conditions          *sdk.WorkflowNodeConditions `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	When                []string                    `json:"when,omitempty" yaml:"when,omitempty"` //This is use only for manual and success condition
	PipelineName        string                      `json:"pipeline,omitempty" yaml:"pipeline,omitempty"`
	ApplicationName     string                      `json:"application,omitempty" yaml:"application,omitempty"`
	EnvironmentName     string                      `json:"environment,omitempty" yaml:"environment,omitempty"`
	ProjectPlatformName string                      `json:"platform,omitempty" yaml:"platform,omitempty"`
	OneAtATime          *bool                       `json:"one_at_a_time,omitempty" yaml:"one_at_a_time,omitempty"`
	Payload             map[string]interface{}      `json:"payload,omitempty" yaml:"payload,omitempty"`
	Parameters          map[string]string           `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type HookEntry struct {
	Model  string            `json:"type,omitempty" yaml:"type,omitempty"`
	Ref    string            `json:"ref,omitempty" yaml:"ref,omitempty"`
	Config map[string]string `json:"config,omitempty" yaml:"config,omitempty"`
}

type WorkflowVersion string

const WorkflowVersion1 = "v1.0"

//NewWorkflow creates a new exportable workflow
func NewWorkflow(w sdk.Workflow, withPermission bool) (Workflow, error) {
	exportedWorkflow := Workflow{}
	exportedWorkflow.Name = w.Name
	exportedWorkflow.Version = WorkflowVersion1
	exportedWorkflow.Workflow = map[string]NodeEntry{}
	exportedWorkflow.Hooks = map[string][]HookEntry{}
	if len(w.Metadata) > 0 {
		exportedWorkflow.Metadata = make(map[string]string, len(w.Metadata))
		for k, v := range w.Metadata {
			exportedWorkflow.Metadata[k] = v
		}
	}

	exportedWorkflow.PurgeTags = w.PurgeTags
	nodes := w.Nodes(false)

	if withPermission {
		exportedWorkflow.Permissions = make(map[string]int, len(w.Groups))
		for _, p := range w.Groups {
			exportedWorkflow.Permissions[p.Group.Name] = p.Permission
		}
	}

	var craftNodeEntry = func(n *sdk.WorkflowNode) (NodeEntry, error) {
		entry := NodeEntry{}

		ancestorIDs := n.Ancestors(&w, false)
		ancestors := make([]string, 0, len(ancestorIDs))
		for _, aID := range ancestorIDs {
			a := w.GetNode(aID)
			if a == nil {
				return entry, sdk.ErrWorkflowNodeNotFound
			}
			ancestors = append(ancestors, a.Name)
		}

		entry.DependsOn = ancestors
		entry.PipelineName = n.Pipeline.Name
		conditions := []sdk.WorkflowNodeCondition{}
		for _, c := range n.Context.Conditions.PlainConditions {
			if c.Operator == sdk.WorkflowConditionsOperatorEquals &&
				c.Value == "Success" &&
				c.Variable == "cds.status" {
				entry.When = append(entry.When, "success")
			} else if c.Operator == sdk.WorkflowConditionsOperatorEquals &&
				c.Value == "true" &&
				c.Variable == "cds.manual" {
				entry.When = append(entry.When, "manual")
			} else {
				conditions = append(conditions, c)
			}
		}

		if len(conditions) > 0 || n.Context.Conditions.LuaScript != "" {
			entry.Conditions = &sdk.WorkflowNodeConditions{
				PlainConditions: conditions,
				LuaScript:       n.Context.Conditions.LuaScript,
			}
		}

		if n.Context.Application != nil {
			entry.ApplicationName = n.Context.Application.Name
		}
		if n.Context.Environment != nil {
			entry.EnvironmentName = n.Context.Environment.Name
		}
		if n.Context.ProjectPlatform != nil {
			entry.ProjectPlatformName = n.Context.ProjectPlatform.Name
		}

		if n.Context.Mutex {
			entry.OneAtATime = &n.Context.Mutex
		}

		if n.Context.HasDefaultPayload() {
			enc := dump.NewDefaultEncoder(nil)
			enc.ExtraFields.DetailedMap = false
			enc.ExtraFields.DetailedStruct = false
			enc.ExtraFields.Len = false
			enc.ExtraFields.Type = false
			enc.Formatters = nil
			m, err := enc.ToMap(n.Context.DefaultPayload)
			if err != nil {
				return entry, err
			}
			entry.Payload = m
		}

		if len(n.Context.DefaultPipelineParameters) > 0 {
			entry.Parameters = sdk.ParametersToMap(n.Context.DefaultPipelineParameters)
		}

		return entry, nil
	}

	hooks := w.GetHooks()

	if len(nodes) == 0 {
		n := w.Root
		if n == nil {
			return exportedWorkflow, sdk.ErrWorkflowNodeNotFound
		}
		entry, err := craftNodeEntry(n)
		if err != nil {
			return exportedWorkflow, err
		}
		exportedWorkflow.ApplicationName = entry.ApplicationName
		exportedWorkflow.PipelineName = entry.PipelineName
		exportedWorkflow.EnvironmentName = entry.EnvironmentName
		exportedWorkflow.ProjectPlatformName = entry.ProjectPlatformName
		exportedWorkflow.DependsOn = entry.DependsOn
		if entry.Conditions != nil && (len(entry.Conditions.PlainConditions) > 0 || entry.Conditions.LuaScript != "") {
			exportedWorkflow.When = entry.When
			exportedWorkflow.Conditions = entry.Conditions
		}
		for _, h := range hooks {
			if exportedWorkflow.Hooks == nil {
				exportedWorkflow.Hooks = make(map[string][]HookEntry)
			}
			exportedWorkflow.PipelineHooks = append(exportedWorkflow.PipelineHooks, HookEntry{
				Model:  h.WorkflowHookModel.Name,
				Ref:    h.Ref,
				Config: h.Config.Values(),
			})
		}
		exportedWorkflow.Payload = entry.Payload
		exportedWorkflow.Parameters = entry.Parameters
	} else {
		nodes = append(nodes, *w.Root)
		for i := range nodes {
			n := &nodes[i]
			if n == nil {
				return exportedWorkflow, sdk.ErrWorkflowNodeNotFound
			}
			entry, err := craftNodeEntry(n)
			if err != nil {
				return exportedWorkflow, err
			}
			exportedWorkflow.Workflow[n.Name] = entry
		}

		for _, h := range hooks {
			if exportedWorkflow.Hooks == nil {
				exportedWorkflow.Hooks = make(map[string][]HookEntry)
			}
			exportedWorkflow.Hooks[w.GetNode(h.WorkflowNodeID).Name] = append(exportedWorkflow.Hooks[w.GetNode(h.WorkflowNodeID).Name], HookEntry{
				Model:  h.WorkflowHookModel.Name,
				Ref:    h.Ref,
				Config: h.Config.Values(),
			})
		}
	}

	return exportedWorkflow, nil
}

// Entries returns the map of all workflow entries
func (w Workflow) Entries() map[string]NodeEntry {
	if len(w.Workflow) != 0 {
		return w.Workflow
	}

	singleEntry := NodeEntry{
		ApplicationName:     w.ApplicationName,
		EnvironmentName:     w.EnvironmentName,
		ProjectPlatformName: w.ProjectPlatformName,
		PipelineName:        w.PipelineName,
		Conditions:          w.Conditions,
		DependsOn:           w.DependsOn,
		When:                w.When,
		Payload:             w.Payload,
		Parameters:          w.Parameters,
	}
	return map[string]NodeEntry{
		w.PipelineName: singleEntry,
	}
}

func (e NodeEntry) checkValidity(w sdk.Workflow) error {
	return nil
}

func (w Workflow) checkValidity() error {
	mError := new(sdk.MultiError)

	if len(w.Workflow) != 0 {
		if w.ApplicationName != "" {
			mError.Append(fmt.Errorf("Error: wrong usage: application %s not allowed here", w.ApplicationName))
		}
		if w.EnvironmentName != "" {
			mError.Append(fmt.Errorf("Error: wrong usage: environment %s not allowed here", w.EnvironmentName))
		}
		if w.ProjectPlatformName != "" {
			mError.Append(fmt.Errorf("Error: wrong usage: platform %s not allowed here", w.ProjectPlatformName))
		}
		if w.PipelineName != "" {
			mError.Append(fmt.Errorf("Error: wrong usage: pipeline %s not allowed here", w.PipelineName))
		}
		if w.Conditions != nil {
			mError.Append(fmt.Errorf("Error: wrong usage: conditions not allowed here"))
		}
		if len(w.When) != 0 {
			mError.Append(fmt.Errorf("Error: wrong usage: when not allowed here"))
		}
		if len(w.DependsOn) != 0 {
			mError.Append(fmt.Errorf("Error: wrong usage: depends_on not allowed here"))
		}
		if len(w.PipelineHooks) != 0 {
			mError.Append(fmt.Errorf("Error: wrong usage: pipeline_hooks not allowed here"))
		}
	} else {
		if len(w.Hooks) > 0 {
			mError.Append(fmt.Errorf("Error: wrong usage: hooks not allowed here"))
		}
	}

	for name := range w.Hooks {
		if _, ok := w.Workflow[name]; !ok {
			mError.Append(fmt.Errorf("Error: wrong usage: invalid hook on %s", name))
		}
	}

	if mError.IsEmpty() {
		return nil
	}
	return mError
}

func (w Workflow) checkDependencies() error {
	mError := new(sdk.MultiError)
	for s, e := range w.Entries() {
		if err := e.checkDependencies(w); err != nil {
			mError.Append(fmt.Errorf("Error: %s invalid: %v", s, err))
		}
	}

	if mError.IsEmpty() {
		return nil
	}
	return mError
}

func (e NodeEntry) checkDependencies(w Workflow) error {
	mError := new(sdk.MultiError)
nextDep:
	for _, d := range e.DependsOn {
		for s := range w.Workflow {
			if s == d {
				continue nextDep
			}
		}
		mError.Append(fmt.Errorf("%s not found", d))
	}
	if mError.IsEmpty() {
		return nil
	}
	return mError
}

//GetWorkflow returns a fresh sdk.Workflow
func (w Workflow) GetWorkflow() (*sdk.Workflow, error) {
	var wf = new(sdk.Workflow)
	wf.Name = w.Name
	if err := w.checkValidity(); err != nil {
		return nil, err
	}
	if err := w.checkDependencies(); err != nil {
		return nil, err
	}
	wf.PurgeTags = w.PurgeTags
	if len(w.Metadata) > 0 {
		wf.Metadata = make(map[string]string, len(w.Metadata))
		for k, v := range w.Metadata {
			wf.Metadata[k] = v
		}
	}

	rand.Seed(time.Now().Unix())
	entries := w.Entries()
	var attempt int
	fakeID := rand.Int63n(5000)
	// attempt is there to avoid infinit loop, but it should not happend becase we check validty and dependencies earlier
	for len(entries) != 0 && attempt < 1000 {
		for name, entry := range entries {
			entry.ID = fakeID
			ok, err := entry.processNode(name, wf)
			if err != nil {
				return nil, err
			}
			if ok {
				delete(entries, name)
				fakeID++
			}
		}
		attempt++
	}
	if len(entries) > 0 {
		return nil, fmt.Errorf("Unable to process %+v", entries)
	}

	//Process hooks
	wf.Visit(w.processHooks)

	//Compute permissions
	wf.Groups = make([]sdk.GroupPermission, 0, len(w.Permissions))
	for g, p := range w.Permissions {
		perm := sdk.GroupPermission{Group: sdk.Group{Name: g}, Permission: p}
		wf.Groups = append(wf.Groups, perm)
	}

	return wf, nil
}

func (e *NodeEntry) getNode(name string) (*sdk.WorkflowNode, error) {
	node := &sdk.WorkflowNode{
		Name: name,
		Ref:  name,
		Pipeline: sdk.Pipeline{
			Name: e.PipelineName,
		},
	}

	if e.ApplicationName != "" {
		node.Context = new(sdk.WorkflowNodeContext)
		node.Context.Application = &sdk.Application{
			Name: e.ApplicationName,
		}
	}

	if e.EnvironmentName != "" {
		if node.Context == nil {
			node.Context = new(sdk.WorkflowNodeContext)
		}

		node.Context.Environment = &sdk.Environment{
			Name: e.EnvironmentName,
		}
	}

	if e.ProjectPlatformName != "" {
		if node.Context == nil {
			node.Context = new(sdk.WorkflowNodeContext)
		}

		node.Context.ProjectPlatform = &sdk.ProjectPlatform{
			Name: e.ProjectPlatformName,
		}
	}

	if e.Conditions != nil {
		if node.Context == nil {
			node.Context = new(sdk.WorkflowNodeContext)
		}
		node.Context.Conditions = *e.Conditions
	}

	if len(e.Payload) > 0 {
		if node.Context == nil {
			node.Context = new(sdk.WorkflowNodeContext)
		}
		node.Context.DefaultPayload = e.Payload
	}

	mapPipelineParameters := sdk.ParametersFromMap(e.Parameters)
	if node.Context == nil {
		node.Context = new(sdk.WorkflowNodeContext)
	}
	node.Context.DefaultPipelineParameters = mapPipelineParameters

	for _, w := range e.When {
		if node.Context == nil {
			node.Context = new(sdk.WorkflowNodeContext)
		}

		switch w {
		case "success":
			node.Context.Conditions.PlainConditions = append(node.Context.Conditions.PlainConditions, sdk.WorkflowNodeCondition{
				Operator: sdk.WorkflowConditionsOperatorEquals,
				Value:    "Success",
				Variable: "cds.status",
			})
		case "manual":
			node.Context.Conditions.PlainConditions = append(node.Context.Conditions.PlainConditions, sdk.WorkflowNodeCondition{
				Operator: sdk.WorkflowConditionsOperatorEquals,
				Value:    "true",
				Variable: "cds.manual",
			})
		default:
			return nil, fmt.Errorf("Unsupported when condition %s", w)
		}
	}

	if e.OneAtATime != nil {
		if node.Context == nil {
			node.Context = &sdk.WorkflowNodeContext{}
		}
		node.Context.Mutex = *e.OneAtATime
	}

	return node, nil
}

func (w *Workflow) processHooks(n *sdk.WorkflowNode) {
	var addHooks = func(hooks []HookEntry) {
		for _, h := range hooks {
			cfg := make(sdk.WorkflowNodeHookConfig, len(h.Config))
			for k, v := range h.Config {
				var hType string
				switch h.Model {
				case sdk.KafkaHookModelName:
					if k == sdk.KafkaHookModelPlatform {
						hType = sdk.HookConfigTypePlatform
					} else {
						hType = sdk.HookConfigTypeString
					}
				default:
					hType = sdk.HookConfigTypeString
				}

				cfg[k] = sdk.WorkflowNodeHookConfigValue{
					Value:        v,
					Configurable: true,
					Type:         hType,
				}
			}
			if h.Ref == "" {
				h.Ref = fmt.Sprintf("%d", time.Now().Unix())
			}
			n.Hooks = append(n.Hooks, sdk.WorkflowNodeHook{
				WorkflowHookModel: sdk.GetDefaultHookModel(h.Model),
				Ref:               h.Ref,
				Config:            cfg,
			})
		}
	}

	if len(w.PipelineHooks) > 0 {
		//Only one node workflow
		addHooks(w.PipelineHooks)
		return
	}

	addHooks(w.Hooks[n.Name])
}

func (e *NodeEntry) processNode(name string, w *sdk.Workflow) (bool, error) {
	if err := e.checkValidity(*w); err != nil {
		return false, err
	}

	var ancestorsExist = true
	var ancestors []*sdk.WorkflowNode

	if len(e.DependsOn) == 1 {
		a := e.DependsOn[0]
		//Looking for the ancestor
		ancestor := w.GetNodeByName(a)
		if ancestor == nil {
			ancestorsExist = false
		}
		ancestors = append(ancestors, ancestor)
	} else {
		for _, a := range e.DependsOn {
			//Looking for the ancestor
			ancestor := w.GetNodeByName(a)
			if ancestor == nil {
				ancestorsExist = false
				break
			}
			ancestors = append(ancestors, ancestor)
		}
	}

	if !ancestorsExist {
		return false, nil
	}

	n, err := e.getNode(name)
	if err != nil {
		return false, err
	}

	switch len(ancestors) {
	case 0:
		w.Root = n
		return true, nil
	case 1:
		w.AddTrigger(ancestors[0].Name, *n)
		return true, nil
	}

	//Try to find an existing join with the same references
	var join *sdk.WorkflowNodeJoin
	for i := range w.Joins {
		j := &w.Joins[i]
		var joinFound = true

		for _, ref := range j.SourceNodeRefs {
			var refFound bool
			for _, a := range e.DependsOn {
				if ref == a {
					refFound = true
					break
				}
			}
			if !refFound {
				joinFound = false
				break
			}
		}

		if joinFound {
			j.Ref = fmt.Sprintf("fakeRef%d", e.ID)
			join = j
		}
	}

	var appendJoin bool
	if join == nil {
		join = &sdk.WorkflowNodeJoin{
			SourceNodeRefs: e.DependsOn,
			Ref:            fmt.Sprintf("fakeRef%d", e.ID),
		}
		appendJoin = true
	}

	join.Triggers = append(join.Triggers, sdk.WorkflowNodeJoinTrigger{
		WorkflowDestNode: *n,
	})

	if appendJoin {
		w.Joins = append(w.Joins, *join)
	}
	return true, nil

}
