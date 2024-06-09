package v1

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/ovh/cds/sdk"
)

// Workflow is the "as code" representation of a sdk.Workflow
type Workflow struct {
	Name        string `json:"name" yaml:"name" jsonschema_description:"The name of the workflow."`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Version     string `json:"version,omitempty" yaml:"version,omitempty" jsonschema_description:"Version for the yaml syntax, latest is v1.0."`
	// this will be filled for complex workflows
	Workflow map[string]NodeEntry   `json:"workflow,omitempty" yaml:"workflow,omitempty" jsonschema_description:"Workflow nodes list."`
	Hooks    map[string][]HookEntry `json:"hooks,omitempty" yaml:"hooks,omitempty" jsonschema_description:"Workflow hooks list."`
	// this will be filled for simple workflows
	OneAtATime             *bool                  `json:"one_at_a_time,omitempty" yaml:"one_at_a_time,omitempty" jsonschema_description:"Set to true if you want to limit the execution of this node to one at a time."`
	Conditions             *ConditionEntry        `json:"conditions,omitempty" yaml:"conditions,omitempty" jsonschema_description:"Conditions to run this node.\nhttps://ovh.github.io/cds/docs/concepts/workflow/run-conditions."`
	When                   []string               `json:"when,omitempty" yaml:"when,omitempty" jsonschema_description:"Set manual and status condition (ex: 'success')."` //This is used only for manual and success condition
	PipelineName           string                 `json:"pipeline,omitempty" yaml:"pipeline,omitempty" jsonschema_description:"The name of a pipeline used for pipeline node."`
	Payload                map[string]interface{} `json:"payload,omitempty" yaml:"payload,omitempty"`
	Parameters             map[string]string      `json:"parameters,omitempty" yaml:"parameters,omitempty" jsonschema_description:"List of parameters for the workflow."`
	ApplicationName        string                 `json:"application,omitempty" yaml:"application,omitempty" jsonschema_description:"The application to use in the context of the node.\nhttps://ovh.github.io/cds/docs/concepts/workflow/pipeline-context"`
	EnvironmentName        string                 `json:"environment,omitempty" yaml:"environment,omitempty" jsonschema_description:"The environment to use in the context of the node.\nhttps://ovh.github.io/cds/docs/concepts/workflow/pipeline-context"`
	ProjectIntegrationName string                 `json:"integration,omitempty" yaml:"integration,omitempty" jsonschema_description:"The integration to use in the context of the node.\nhttps://ovh.github.io/cds/docs/concepts/workflow/pipeline-context"`
	PipelineHooks          []HookEntry            `json:"pipeline_hooks,omitempty" yaml:"pipeline_hooks,omitempty"`
	// extra workflow data
	Permissions      map[string]int                 `json:"permissions,omitempty" yaml:"permissions,omitempty" jsonschema_description:"The permissions for the workflow (ex: myGroup: 7).\nhttps://ovh.github.io/cds/docs/concepts/permissions"`
	Metadata         map[string]string              `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	PurgeTags        []string                       `json:"purge_tags,omitempty" yaml:"purge_tags,omitempty"`
	Notifications    []NotificationEntry            `json:"notify,omitempty" yaml:"notify,omitempty"` // This is used when the workflow have only one pipeline
	HistoryLength    *int64                         `json:"history_length,omitempty" yaml:"history_length,omitempty"`
	MapNotifications map[string][]NotificationEntry `json:"notifications,omitempty" yaml:"notifications,omitempty"` // This is used when the workflow have more than one pipeline
}

// NodeEntry represents a node as code
type NodeEntry struct {
	ID                     int64                  `json:"-" yaml:"-"`
	DependsOn              []string               `json:"depends_on,omitempty" yaml:"depends_on,omitempty" jsonschema_description:"Names of the parent nodes, can be pipelines, forks or joins."`
	Conditions             *ConditionEntry        `json:"conditions,omitempty" yaml:"conditions,omitempty" jsonschema_description:"Conditions to run this node.\nhttps://ovh.github.io/cds/docs/concepts/workflow/run-conditions."`
	When                   []string               `json:"when,omitempty" yaml:"when,omitempty" jsonschema_description:"Set manual and status condition (ex: 'success')."` //This is used only for manual and success condition
	PipelineName           string                 `json:"pipeline,omitempty" yaml:"pipeline,omitempty" jsonschema_description:"The name of a pipeline used for pipeline node."`
	ApplicationName        string                 `json:"application,omitempty" yaml:"application,omitempty" jsonschema_description:"The application to use in the context of the node.\nhttps://ovh.github.io/cds/docs/concepts/workflow/pipeline-context"`
	EnvironmentName        string                 `json:"environment,omitempty" yaml:"environment,omitempty" jsonschema_description:"The environment to use in the context of the node.\nhttps://ovh.github.io/cds/docs/concepts/workflow/pipeline-context"`
	ProjectIntegrationName string                 `json:"integration,omitempty" yaml:"integration,omitempty" jsonschema_description:"The integration to use in the context of the node.\nhttps://ovh.github.io/cds/docs/concepts/workflow/pipeline-context"`
	OneAtATime             *bool                  `json:"one_at_a_time,omitempty" yaml:"one_at_a_time,omitempty" jsonschema_description:"Set to true if you want to limit the execution of this node to one at a time."`
	Payload                map[string]interface{} `json:"payload,omitempty" yaml:"payload,omitempty"`
	Parameters             map[string]string      `json:"parameters,omitempty" yaml:"parameters,omitempty" jsonschema_description:"List of parameters for the workflow."`
	OutgoingHookModelName  string                 `json:"trigger,omitempty" yaml:"trigger,omitempty"`
	OutgoingHookConfig     map[string]string      `json:"config,omitempty" yaml:"config,omitempty"`
	Permissions            map[string]int         `json:"permissions,omitempty" yaml:"permissions,omitempty" jsonschema_description:"The permissions for the node (ex: myGroup: 7).\nhttps://ovh.github.io/cds/docs/concepts/permissions"`
}

type ConditionEntry struct {
	PlainConditions []PlainConditionEntry `json:"plain,omitempty" yaml:"check,omitempty"`
	LuaScript       string                `json:"script,omitempty" yaml:"script,omitempty"`
}

// WorkflowNodeCondition represents a condition to trigger or not a pipeline in a workflow. Operator can be =, !=, regex
type PlainConditionEntry struct {
	Variable string `json:"variable" yaml:"variable"`
	Operator string `json:"operator" yaml:"operator"`
	Value    string `json:"value" yaml:"value"`
}

// HookEntry represents a hook as code
type HookEntry struct {
	Model      string                      `json:"type,omitempty" yaml:"type,omitempty" jsonschema_description:"Model of the hook.\nhttps://ovh.github.io/cds/docs/concepts/workflow/hooks"`
	Config     map[string]string           `json:"config,omitempty" yaml:"config,omitempty"`
	Conditions *sdk.WorkflowNodeConditions `json:"conditions,omitempty" yaml:"conditions,omitempty" jsonschema_description:"Conditions to run this hook.\nhttps://ovh.github.io/cds/docs/concepts/workflow/run-conditions."`
}

// Entries returns the map of all workflow entries
func (w Workflow) Entries() map[string]NodeEntry {
	if len(w.Workflow) != 0 {
		return w.Workflow
	}

	if w.PipelineName != "" {
		singleEntry := NodeEntry{
			ApplicationName:        w.ApplicationName,
			EnvironmentName:        w.EnvironmentName,
			ProjectIntegrationName: w.ProjectIntegrationName,
			PipelineName:           w.PipelineName,
			Conditions:             w.Conditions,
			When:                   w.When,
			Payload:                w.Payload,
			Parameters:             w.Parameters,
			OneAtATime:             w.OneAtATime,
		}
		return map[string]NodeEntry{
			w.PipelineName: singleEntry,
		}
	}
	return nil
}

func (w Workflow) CheckValidity() error {
	mError := new(sdk.MultiError)

	//Check valid application name
	rx := sdk.NamePatternRegex
	if !rx.MatchString(w.Name) {
		mError.Append(sdk.NewErrorFrom(sdk.ErrWrongRequest, "workflow name %s do not respect pattern %s", w.Name, sdk.NamePattern))
	}

	if len(w.Workflow) != 0 {
		if w.ApplicationName != "" {
			mError.Append(sdk.NewErrorFrom(sdk.ErrWrongRequest, "application %s not allowed here", w.ApplicationName))
		}
		if w.EnvironmentName != "" {
			mError.Append(sdk.NewErrorFrom(sdk.ErrWrongRequest, "environment %s not allowed here", w.EnvironmentName))
		}
		if w.ProjectIntegrationName != "" {
			mError.Append(sdk.NewErrorFrom(sdk.ErrWrongRequest, "integration %s not allowed here", w.ProjectIntegrationName))
		}
		if w.PipelineName != "" {
			mError.Append(sdk.NewErrorFrom(sdk.ErrWrongRequest, "pipeline %s not allowed here", w.PipelineName))
		}
		if w.Conditions != nil {
			mError.Append(sdk.NewErrorFrom(sdk.ErrWrongRequest, "conditions not allowed here"))
		}
		if len(w.When) != 0 {
			mError.Append(sdk.NewErrorFrom(sdk.ErrWrongRequest, "when not allowed here"))
		}
		if len(w.PipelineHooks) != 0 {
			mError.Append(sdk.NewErrorFrom(sdk.ErrWrongRequest, "pipeline_hooks not allowed here"))
		}
	} else {
		if len(w.Hooks) > 0 {
			mError.Append(sdk.NewErrorFrom(sdk.ErrWrongRequest, "hooks not allowed here"))
		}
	}

	for name := range w.Hooks {
		if _, ok := w.Workflow[name]; !ok {
			mError.Append(sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid hook on %s", name))
		}
	}

	//Checks map notifications validity
	mError.Append(CheckWorkflowNotificationsValidity(w))

	if mError.IsEmpty() {
		return nil
	}
	return mError
}

func (w Workflow) CheckDependencies() error {
	mError := new(sdk.MultiError)
	for s, e := range w.Entries() {
		if err := e.checkDependencies(s, w); err != nil {
			mError.Append(err)
		}
	}

	if mError.IsEmpty() {
		return nil
	}
	return mError
}

func (e NodeEntry) checkDependencies(nodeName string, w Workflow) error {
	mError := new(sdk.MultiError)
nextDep:
	for _, d := range e.DependsOn {
		for s := range w.Workflow {
			if s == d {
				continue nextDep
			}
		}
		mError.Append(sdk.NewErrorFrom(sdk.ErrWrongRequest, "the pipeline %s depends on an unknown pipeline: %s", nodeName, d))
	}
	if mError.IsEmpty() {
		return nil
	}
	return mError
}

// GetWorkflow returns a fresh sdk.Workflow
func (w Workflow) GetWorkflow() (*sdk.Workflow, error) {
	var wf = new(sdk.Workflow)
	wf.Name = w.Name
	wf.Description = w.Description
	wf.WorkflowData = sdk.WorkflowData{}
	// Init map
	wf.Applications = make(map[int64]sdk.Application)
	wf.Pipelines = make(map[int64]sdk.Pipeline)
	wf.Environments = make(map[int64]sdk.Environment)
	wf.ProjectIntegrations = make(map[int64]sdk.ProjectIntegration)

	if err := w.CheckValidity(); err != nil {
		return nil, sdk.WrapError(err, "unable to check validity")
	}
	if err := w.CheckDependencies(); err != nil {
		return nil, sdk.WrapError(err, "unable to check dependencies")
	}
	wf.PurgeTags = w.PurgeTags
	if len(w.Metadata) > 0 {
		wf.Metadata = make(map[string]string, len(w.Metadata))
		for k, v := range w.Metadata {
			wf.Metadata[k] = v
		}
	}
	if w.HistoryLength != nil && *w.HistoryLength > 0 {
		wf.HistoryLength = *w.HistoryLength
	}

	rand.Seed(time.Now().Unix())
	entries := w.Entries()

	if len(entries) == 0 {
		return nil, sdk.NewErrorFrom(sdk.ErrWorkflowInvalid, "a workflow must contains at least 1 pipeline")
	}

	var attempt int
	fakeID := rand.Int63n(5000)
	// attempt is there to avoid infinite loop, but it should not happened because we check validity and dependencies earlier
	for len(entries) != 0 && attempt < 10000 {
		for name, entry := range entries {
			entry.ID = fakeID
			ok, err := entry.processNode(name, wf)
			if err != nil {
				return nil, sdk.WrapError(err, "unable to process node")
			}
			if ok {
				delete(entries, name)
				fakeID++
			}
		}
		attempt++
	}
	if len(entries) > 0 {
		return nil, sdk.WithStack(fmt.Errorf("unable to process %+v", entries))
	}

	//Process hooks
	wf.VisitNode(w.processHooks)

	//Compute permissions
	wf.Groups = make([]sdk.GroupPermission, 0, len(w.Permissions))
	for g, p := range w.Permissions {
		perm := sdk.GroupPermission{Group: sdk.Group{Name: g}, Permission: p}
		wf.Groups = append(wf.Groups, perm)
	}

	//Compute notifications
	if err := w.processNotifications(wf); err != nil {
		return nil, err
	}

	wf.SortNode()

	return wf, nil
}

func (e *NodeEntry) getNode(name string) (*sdk.Node, error) {
	var mutex bool
	if e.OneAtATime != nil && *e.OneAtATime {
		mutex = true
	}
	node := &sdk.Node{
		Name: name,
		Ref:  name,
		Type: sdk.NodeTypeFork,
		Context: &sdk.NodeContext{
			PipelineName:           e.PipelineName,
			ApplicationName:        e.ApplicationName,
			EnvironmentName:        e.EnvironmentName,
			ProjectIntegrationName: e.ProjectIntegrationName,
			Mutex:                  mutex,
		},
	}

	if e.PipelineName != "" {
		node.Type = sdk.NodeTypePipeline
	} else if e.OutgoingHookModelName != "" {
		node.Type = sdk.NodeTypeOutGoingHook
	} else if len(e.DependsOn) > 1 {
		node.Type = sdk.NodeTypeJoin
		node.JoinContext = make([]sdk.NodeJoin, 0, len(e.DependsOn))
		for _, parent := range e.DependsOn {
			node.JoinContext = append(node.JoinContext, sdk.NodeJoin{ParentName: parent})
		}
	}

	if len(e.Permissions) > 0 {
		//Compute permissions
		node.Groups = make([]sdk.GroupPermission, 0, len(e.Permissions))
		for g, p := range e.Permissions {
			perm := sdk.GroupPermission{Group: sdk.Group{Name: g}, Permission: p}
			node.Groups = append(node.Groups, perm)
		}
	}

	if e.Conditions != nil {
		node.Context.Conditions = sdk.WorkflowNodeConditions{
			PlainConditions: make([]sdk.WorkflowNodeCondition, 0, len(e.Conditions.PlainConditions)),
			LuaScript:       e.Conditions.LuaScript,
		}
		for _, c := range e.Conditions.PlainConditions {
			node.Context.Conditions.PlainConditions = append(node.Context.Conditions.PlainConditions, sdk.WorkflowNodeCondition{
				Variable: c.Variable,
				Operator: c.Operator,
				Value:    c.Value,
			})
		}
	}

	if len(e.Payload) > 0 {
		if len(e.DependsOn) > 0 {
			return nil, sdk.NewErrorFrom(sdk.ErrInvalidNodeDefaultPayload, "default payload cannot be set on another node than the first one (node: %s)", name)
		}
		node.Context.DefaultPayload = e.Payload
	}

	mapPipelineParameters := sdk.ParametersFromMap(e.Parameters)
	node.Context.DefaultPipelineParameters = mapPipelineParameters

	for _, w := range e.When {
		switch w {
		case "success":
			node.Context.Conditions.PlainConditions = append(node.Context.Conditions.PlainConditions, sdk.WorkflowNodeCondition{
				Operator: sdk.WorkflowConditionsOperatorEquals,
				Value:    sdk.StatusSuccess,
				Variable: "cds.status",
			})
		case "manual":
			node.Context.Conditions.PlainConditions = append(node.Context.Conditions.PlainConditions, sdk.WorkflowNodeCondition{
				Operator: sdk.WorkflowConditionsOperatorEquals,
				Value:    "true",
				Variable: "cds.manual",
			})
		default:
			return nil, sdk.NewErrorFrom(sdk.ErrWrongRequest, "unsupported when condition %s", w)
		}
	}

	if e.OneAtATime != nil {
		node.Context.Mutex = *e.OneAtATime
	}

	if e.OutgoingHookModelName != "" {
		node.Type = sdk.NodeTypeOutGoingHook
		config := sdk.WorkflowNodeHookConfig{}
		for k, v := range e.OutgoingHookConfig {
			config[k] = sdk.WorkflowNodeHookConfigValue{
				Value: v,
			}
		}
		node.OutGoingHookContext = &sdk.NodeOutGoingHook{
			Config:        config,
			HookModelName: e.OutgoingHookModelName,
		}
	}
	return node, nil
}

func (w *Workflow) processHooks(n *sdk.Node, wf *sdk.Workflow) {
	var addHooks = func(hooks []HookEntry) {
		for _, h := range hooks {
			cfg := make(sdk.WorkflowNodeHookConfig, len(h.Config))
			for k, v := range h.Config {
				var hType string
				switch h.Model {
				case sdk.KafkaHookModelName, sdk.RabbitMQHookModelName:
					if k == sdk.HookModelIntegration {
						hType = sdk.HookConfigTypeIntegration
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

			hook := sdk.NodeHook{
				Config:        cfg,
				HookModelName: h.Model,
			}

			if h.Conditions != nil {
				hook.Conditions = *h.Conditions
			}
			n.Hooks = append(n.Hooks, hook)
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
	// Find WorkflowNodeAncestors
	exist, err := e.processNodeAncestors(name, w)
	if err != nil {
		return false, err
	}

	if exist {
		return true, nil
	}

	return false, nil
}

func (e *NodeEntry) processNodeAncestors(name string, w *sdk.Workflow) (bool, error) {
	var ancestorsExist = true
	var ancestors []*sdk.Node

	if len(e.DependsOn) == 1 {
		a := e.DependsOn[0]
		//Looking for the ancestor
		ancestor := w.WorkflowData.NodeByName(a)
		if ancestor == nil {
			ancestorsExist = false
		}
		ancestors = append(ancestors, ancestor)
	} else {
		for _, a := range e.DependsOn {
			//Looking for the ancestor
			ancestor := w.WorkflowData.NodeByName(a)
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
		w.WorkflowData.Node = *n
		return true, nil
	case 1:
		w.AddTrigger(ancestors[0].Name, *n)
		return true, nil
	default:
		if n != nil && n.Type == sdk.NodeTypeJoin && joinAsNode(n) {
			w.WorkflowData.Joins = append(w.WorkflowData.Joins, *n)
			return true, nil
		}
	}

	// Compute join

	// Try to find an existing join with the same references
	var join *sdk.Node
	for i := range w.WorkflowData.Joins {
		j := &w.WorkflowData.Joins[i]

		if len(e.DependsOn) != len(j.JoinContext) {
			continue
		}

		var joinFound = true
		for _, ref := range j.JoinContext {
			var refFound bool
			for _, a := range e.DependsOn {
				if ref.ParentName == a {
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
		joinContext := make([]sdk.NodeJoin, 0, len(e.DependsOn))
		for _, d := range e.DependsOn {
			joinContext = append(joinContext, sdk.NodeJoin{
				ParentName: d,
			})
		}
		join = &sdk.Node{
			JoinContext: joinContext,
			Type:        sdk.NodeTypeJoin,
			Ref:         fmt.Sprintf("fakeRef%d", e.ID),
		}
		appendJoin = true
	}

	join.Triggers = append(join.Triggers, sdk.NodeTrigger{
		ChildNode: *n,
	})

	if appendJoin {
		w.WorkflowData.Joins = append(w.WorkflowData.Joins, *join)
	}
	return true, nil
}

func joinAsNode(n *sdk.Node) bool {
	return n.Context != nil && (n.Context.Conditions.LuaScript != "" || len(n.Context.Conditions.PlainConditions) > 0)
}

func (w Workflow) GetName() string {
	return w.Name
}

func (w Workflow) GetVersion() string {
	return w.Version
}
