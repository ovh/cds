package exportentities

import (
	"archive/tar"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/fsamin/go-dump"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Name pattern for pull files.
const (
	PullWorkflowName    = "%s.yml"
	PullPipelineName    = "%s.pip.yml"
	PullApplicationName = "%s.app.yml"
	PullEnvironmentName = "%s.env.yml"
)

// Workflow is the "as code" representation of a sdk.Workflow
type Workflow struct {
	Name        string  `json:"name" yaml:"name"`
	Description string  `json:"description,omitempty" yaml:"description,omitempty"`
	Version     string  `json:"version,omitempty" yaml:"version,omitempty"`
	Template    *string `json:"template,omitempty" yaml:"template,omitempty"`
	// This will be filled for complex workflows
	Workflow map[string]NodeEntry   `json:"workflow,omitempty" yaml:"workflow,omitempty"`
	Hooks    map[string][]HookEntry `json:"hooks,omitempty" yaml:"hooks,omitempty"`
	// This will be filled for simple workflows
	DependsOn              []string                       `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Conditions             *sdk.WorkflowNodeConditions    `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	When                   []string                       `json:"when,omitempty" yaml:"when,omitempty"` //This is used only for manual and success condition
	PipelineName           string                         `json:"pipeline,omitempty" yaml:"pipeline,omitempty"`
	Payload                map[string]interface{}         `json:"payload,omitempty" yaml:"payload,omitempty"`
	Parameters             map[string]string              `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	ApplicationName        string                         `json:"application,omitempty" yaml:"application,omitempty"`
	EnvironmentName        string                         `json:"environment,omitempty" yaml:"environment,omitempty"`
	ProjectIntegrationName string                         `json:"integration,omitempty" yaml:"integration,omitempty"`
	PipelineHooks          []HookEntry                    `json:"pipeline_hooks,omitempty" yaml:"pipeline_hooks,omitempty"`
	Permissions            map[string]int                 `json:"permissions,omitempty" yaml:"permissions,omitempty"`
	Metadata               map[string]string              `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	PurgeTags              []string                       `json:"purge_tags,omitempty" yaml:"purge_tags,omitempty"`
	HistoryLength          *int64                         `json:"history_length,omitempty" yaml:"history_length,omitempty"`
	Notifications          []NotificationEntry            `json:"notify,omitempty" yaml:"notify,omitempty"`               // This is used when the workflow have only one pipeline
	MapNotifications       map[string][]NotificationEntry `json:"notifications,omitempty" yaml:"notifications,omitempty"` // This is used when the workflow have more than one pipeline
}

// WorkflowPulled contains all the yaml base64 that are needed to generate a workflow tar file.
type WorkflowPulled struct {
	Workflow     WorkflowPulledItem   `json:"workflow"`
	Pipelines    []WorkflowPulledItem `json:"pipelines"`
	Applications []WorkflowPulledItem `json:"applications"`
	Environments []WorkflowPulledItem `json:"environments"`
}

// WorkflowPulledItem contains data for a workflow item.
type WorkflowPulledItem struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Tar returns a tar containing all files for a pulled workflow.
func (w WorkflowPulled) Tar(writer io.Writer) error {
	tw := tar.NewWriter(writer)
	defer func() {
		if err := tw.Close(); err != nil {
			log.Error("%v", sdk.WrapError(err, "unable to close tar writer"))
		}
	}()

	bs, err := base64.StdEncoding.DecodeString(w.Workflow.Value)
	if err != nil {
		return sdk.WithStack(err)
	}
	if err := tw.WriteHeader(&tar.Header{
		Name: fmt.Sprintf(PullWorkflowName, w.Workflow.Name),
		Mode: 0644,
		Size: int64(len(bs)),
	}); err != nil {
		return sdk.WrapError(err, "unable to write workflow header for %s", w.Workflow.Name)
	}
	if _, err := tw.Write(bs); err != nil {
		return sdk.WrapError(err, "unable to write workflow value")
	}

	for _, a := range w.Applications {
		bs, err := base64.StdEncoding.DecodeString(a.Value)
		if err != nil {
			return sdk.WithStack(err)
		}
		if err := tw.WriteHeader(&tar.Header{
			Name: fmt.Sprintf(PullApplicationName, a.Name),
			Mode: 0644,
			Size: int64(len(bs)),
		}); err != nil {
			return sdk.WrapError(err, "unable to write application header for %s", a.Name)
		}
		if _, err := tw.Write(bs); err != nil {
			return sdk.WrapError(err, "unable to write application value")
		}
	}

	for _, e := range w.Environments {
		bs, err := base64.StdEncoding.DecodeString(e.Value)
		if err != nil {
			return sdk.WithStack(err)
		}
		if err := tw.WriteHeader(&tar.Header{
			Name: fmt.Sprintf(PullEnvironmentName, e.Name),
			Mode: 0644,
			Size: int64(len(bs)),
		}); err != nil {
			return sdk.WrapError(err, "unable to write env header for %s", e.Name)
		}
		if _, err := tw.Write(bs); err != nil {
			return sdk.WrapError(err, "unable to copy env buffer")
		}
	}

	for _, p := range w.Pipelines {
		bs, err := base64.StdEncoding.DecodeString(p.Value)
		if err != nil {
			return sdk.WithStack(err)
		}
		if err := tw.WriteHeader(&tar.Header{
			Name: fmt.Sprintf(PullPipelineName, p.Name),
			Mode: 0644,
			Size: int64(len(bs)),
		}); err != nil {
			return sdk.WrapError(err, "unable to write pipeline header for %s", p.Name)
		}
		if _, err := tw.Write(bs); err != nil {
			return sdk.WrapError(err, "unable to write pipeline value")
		}
	}

	return nil
}

// NodeEntry represents a node as code
type NodeEntry struct {
	ID                     int64                       `json:"-" yaml:"-"`
	DependsOn              []string                    `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Conditions             *sdk.WorkflowNodeConditions `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	When                   []string                    `json:"when,omitempty" yaml:"when,omitempty"` //This is used only for manual and success condition
	PipelineName           string                      `json:"pipeline,omitempty" yaml:"pipeline,omitempty"`
	ApplicationName        string                      `json:"application,omitempty" yaml:"application,omitempty"`
	EnvironmentName        string                      `json:"environment,omitempty" yaml:"environment,omitempty"`
	ProjectIntegrationName string                      `json:"integration,omitempty" yaml:"integration,omitempty"`
	OneAtATime             *bool                       `json:"one_at_a_time,omitempty" yaml:"one_at_a_time,omitempty"`
	Payload                map[string]interface{}      `json:"payload,omitempty" yaml:"payload,omitempty"`
	Parameters             map[string]string           `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	OutgoingHookModelName  string                      `json:"trigger,omitempty" yaml:"trigger,omitempty"`
	OutgoingHookConfig     map[string]string           `json:"config,omitempty" yaml:"config,omitempty"`
	Permissions            map[string]int              `json:"permissions,omitempty" yaml:"permissions,omitempty"`
}

// HookEntry represents a hook as code
type HookEntry struct {
	Model  string            `json:"type,omitempty" yaml:"type,omitempty"`
	Ref    string            `json:"ref,omitempty" yaml:"ref,omitempty"`
	Config map[string]string `json:"config,omitempty" yaml:"config,omitempty"`
}

// WorkflowVersion is the type for the version
type WorkflowVersion string

// There are the supported versions
const (
	WorkflowVersion1 = "v1.0"
)

func craftNodeEntry(w sdk.Workflow, n sdk.Node) (NodeEntry, error) {
	entry := NodeEntry{}

	ancestors := []string{}

	if n.Type != sdk.NodeTypeJoin {
		nodes := w.WorkflowData.Array()
		for _, node := range nodes {
			if n.Name == node.Name {
				continue
			}
			for _, t := range node.Triggers {
				if t.ChildNode.Name == n.Name {
					if node.Type == sdk.NodeTypeJoin && !joinAsNode(node) {
						for _, jp := range node.JoinContext {
							parentNode := w.WorkflowData.NodeByRef(jp.ParentName)
							if parentNode == nil {
								return entry, sdk.WithStack(sdk.ErrWorkflowNodeNotFound)
							}
							ancestors = append(ancestors, parentNode.Name)
						}
					} else {
						ancestors = append(ancestors, node.Name)
					}
				}
			}
		}
	} else {
		for _, jc := range n.JoinContext {
			ancestors = append(ancestors, jc.ParentName)
		}
	}

	sort.Strings(ancestors)
	entry.DependsOn = ancestors

	if n.Context != nil && n.Context.PipelineName != "" {
		entry.PipelineName = n.Context.PipelineName
	}

	if n.Context != nil {
		conditions := []sdk.WorkflowNodeCondition{}
		for _, c := range n.Context.Conditions.PlainConditions {
			if c.Operator == sdk.WorkflowConditionsOperatorEquals &&
				c.Value == sdk.StatusSuccess.String() &&
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

		if n.Context.ApplicationName != "" {
			entry.ApplicationName = n.Context.ApplicationName
		}
		if n.Context.EnvironmentName != "" {
			entry.EnvironmentName = n.Context.EnvironmentName
		}
		if n.Context.ProjectIntegrationName != "" {
			entry.ProjectIntegrationName = n.Context.ProjectIntegrationName
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
				return entry, sdk.WrapError(err, "Unable to encode payload")
			}
			entry.Payload = m
		}

		if len(n.Context.DefaultPipelineParameters) > 0 {
			entry.Parameters = sdk.ParametersToMap(n.Context.DefaultPipelineParameters)
		}
	}

	if len(n.Groups) > 0 {
		entry.Permissions = map[string]int{}
		for _, gr := range n.Groups {
			entry.Permissions[gr.Group.Name] = gr.Permission
		}
	}

	if n.OutGoingHookContext != nil {
		entry.OutgoingHookModelName = n.OutGoingHookContext.HookModelName

		m := sdk.GetBuiltinOutgoingHookModelByName(entry.OutgoingHookModelName)
		if m == nil {
			return entry, sdk.WrapError(sdk.ErrNotFound, "unable to find outgoing hook model %s", entry.OutgoingHookModelName)
		}
		entry.OutgoingHookConfig = n.OutGoingHookContext.Config.Values(m.DefaultConfig)
	}

	return entry, nil
}

// WorkflowOptions is the type for several workflow-as-code options
type WorkflowOptions func(sdk.Workflow, *Workflow) error

// WorkflowWithPermissions export workflow with permissions
func WorkflowWithPermissions(w sdk.Workflow, exportedWorkflow *Workflow) error {
	exportedWorkflow.Permissions = make(map[string]int, len(w.Groups))
	for _, p := range w.Groups {
		exportedWorkflow.Permissions[p.Group.Name] = p.Permission
	}

	for _, node := range w.WorkflowData.Array() {
		entries := exportedWorkflow.Entries()
		if len(entries) > 1 { // Else the permissions are the same than the workflow
			for exportedNodeName, entry := range entries {
				if entry.Permissions == nil {
					entry.Permissions = map[string]int{}
				}
				if node.Name == exportedNodeName {
					for _, p := range node.Groups {
						entry.Permissions[p.Group.Name] = p.Permission
					}
					exportedWorkflow.Workflow[exportedNodeName] = entry
				}
			}
		}
	}

	return nil
}

// WorkflowSkipIfOnlyOneRepoWebhook skips the repo webhook if it's the only one
func WorkflowSkipIfOnlyOneRepoWebhook(w sdk.Workflow, exportedWorkflow *Workflow) error {
	if len(exportedWorkflow.Workflow) == 0 {
		if len(exportedWorkflow.PipelineHooks) == 1 && exportedWorkflow.PipelineHooks[0].Model == sdk.RepositoryWebHookModelName {
			exportedWorkflow.PipelineHooks = nil
		}
		return nil
	}

	for nodeName, hs := range exportedWorkflow.Hooks {
		if nodeName == w.Root.Name && len(hs) == 1 {
			if hs[0].Model == sdk.RepositoryWebHookModelName {
				delete(exportedWorkflow.Hooks, nodeName)
				break
			}
		}
	}

	return nil
}

func joinAsNode(n *sdk.Node) bool {
	return n.Context != nil && (n.Context.Conditions.LuaScript != "" || len(n.Context.Conditions.PlainConditions) > 0)
}

//NewWorkflow creates a new exportable workflow
func NewWorkflow(w sdk.Workflow, opts ...WorkflowOptions) (Workflow, error) {
	exportedWorkflow := Workflow{}
	exportedWorkflow.Name = w.Name
	exportedWorkflow.Description = w.Description
	exportedWorkflow.Version = WorkflowVersion1
	exportedWorkflow.Workflow = map[string]NodeEntry{}
	exportedWorkflow.Hooks = map[string][]HookEntry{}
	if len(w.Metadata) > 0 {
		exportedWorkflow.Metadata = make(map[string]string, len(w.Metadata))
		for k, v := range w.Metadata {
			// don't export empty metadata
			if v != "" {
				exportedWorkflow.Metadata[k] = v
			}
		}
	}

	if w.HistoryLength > 0 && w.HistoryLength != sdk.DefaultHistoryLength {
		exportedWorkflow.HistoryLength = &w.HistoryLength
	}

	exportedWorkflow.PurgeTags = w.PurgeTags

	nodes := w.WorkflowData.Array()

	if len(nodes) == 1 {
		n := w.WorkflowData.Node
		entry, err := craftNodeEntry(w, n)
		if err != nil {
			return exportedWorkflow, err
		}
		exportedWorkflow.ApplicationName = entry.ApplicationName
		exportedWorkflow.PipelineName = entry.PipelineName
		exportedWorkflow.EnvironmentName = entry.EnvironmentName
		exportedWorkflow.ProjectIntegrationName = entry.ProjectIntegrationName
		exportedWorkflow.DependsOn = entry.DependsOn
		if entry.Conditions != nil && (len(entry.Conditions.PlainConditions) > 0 || entry.Conditions.LuaScript != "") {
			exportedWorkflow.When = entry.When
			exportedWorkflow.Conditions = entry.Conditions
		}
		for _, h := range n.Hooks {
			if exportedWorkflow.Hooks == nil {
				exportedWorkflow.Hooks = make(map[string][]HookEntry)
			}

			m := sdk.GetBuiltinHookModelByName(h.HookModelName)
			if m == nil {
				return exportedWorkflow, sdk.WrapError(sdk.ErrNotFound, "unable to find hook model %s", h.HookModelName)
			}

			exportedWorkflow.PipelineHooks = append(exportedWorkflow.PipelineHooks, HookEntry{
				Model:  h.HookModelName,
				Ref:    h.Ref,
				Config: h.Config.Values(m.DefaultConfig),
			})
		}
		exportedWorkflow.Payload = entry.Payload
		exportedWorkflow.Parameters = entry.Parameters
	} else {
		for _, n := range nodes {
			if n.Type == sdk.NodeTypeJoin && !joinAsNode(n) {
				continue
			}

			entry, err := craftNodeEntry(w, *n)
			if err != nil {
				return exportedWorkflow, sdk.WrapError(err, "Unable to craft Node entry %s", n.Name)
			}
			exportedWorkflow.Workflow[n.Name] = entry

			for _, h := range n.Hooks {
				if exportedWorkflow.Hooks == nil {
					exportedWorkflow.Hooks = make(map[string][]HookEntry)
				}

				m := sdk.GetBuiltinHookModelByName(h.HookModelName)
				if m == nil {
					return exportedWorkflow, sdk.WrapError(sdk.ErrNotFound, "unable to find hook model %s", h.HookModelName)
				}

				exportedWorkflow.Hooks[n.Name] = append(exportedWorkflow.Hooks[n.Name], HookEntry{
					Model:  h.HookModelName,
					Ref:    h.Ref,
					Config: h.Config.Values(m.DefaultConfig),
				})
			}
		}
	}

	//Notifications
	if err := craftNotifications(w, &exportedWorkflow); err != nil {
		return exportedWorkflow, err
	}

	for _, f := range opts {
		if err := f(w, &exportedWorkflow); err != nil {
			return exportedWorkflow, sdk.WrapError(err, "Unable to run function")
		}
	}

	if w.Template != nil {
		path := fmt.Sprintf("%s/%s", w.Template.Group.Name, w.Template.Slug)
		exportedWorkflow.Template = &path
	}

	return exportedWorkflow, nil
}

// Entries returns the map of all workflow entries
func (w Workflow) Entries() map[string]NodeEntry {
	if len(w.Workflow) != 0 {
		return w.Workflow
	}

	singleEntry := NodeEntry{
		ApplicationName:        w.ApplicationName,
		EnvironmentName:        w.EnvironmentName,
		ProjectIntegrationName: w.ProjectIntegrationName,
		PipelineName:           w.PipelineName,
		Conditions:             w.Conditions,
		DependsOn:              w.DependsOn,
		When:                   w.When,
		Payload:                w.Payload,
		Parameters:             w.Parameters,
	}
	return map[string]NodeEntry{
		w.PipelineName: singleEntry,
	}
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
		if w.ProjectIntegrationName != "" {
			mError.Append(fmt.Errorf("Error: wrong usage: integration %s not allowed here", w.ProjectIntegrationName))
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

	//Checks map notifications validity
	mError.Append(checkWorkflowNotificationsValidity(w))

	if mError.IsEmpty() {
		return nil
	}
	return mError
}

func (w Workflow) checkDependencies() error {
	mError := new(sdk.MultiError)
	for s, e := range w.Entries() {
		if err := e.checkDependencies(s, w); err != nil {
			mError.Append(fmt.Errorf("Error: %s invalid: %v", s, err))
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
		mError.Append(fmt.Errorf("the pipeline %s depends on an unknown pipeline: %s", nodeName, d))
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
	wf.WorkflowData = &sdk.WorkflowData{}
	// Init map
	wf.Applications = make(map[int64]sdk.Application)
	wf.Pipelines = make(map[int64]sdk.Pipeline)
	wf.Environments = make(map[int64]sdk.Environment)
	wf.ProjectIntegrations = make(map[int64]sdk.ProjectIntegration)

	if err := w.checkValidity(); err != nil {
		return nil, sdk.WrapError(err, "Unable to check validity")
	}
	if err := w.checkDependencies(); err != nil {
		return nil, sdk.WrapError(err, "Unable to check dependencies")
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
	} else {
		wf.HistoryLength = sdk.DefaultHistoryLength
	}

	rand.Seed(time.Now().Unix())
	entries := w.Entries()
	var attempt int
	fakeID := rand.Int63n(5000)
	// attempt is there to avoid infinit loop, but it should not happend becase we check validty and dependencies earlier
	for len(entries) != 0 && attempt < 10000 {
		for name, entry := range entries {
			entry.ID = fakeID
			ok, err := entry.processNode(name, wf)
			if err != nil {
				return nil, sdk.WrapError(err, "Unable to process node")
			}
			if ok {
				delete(entries, name)
				fakeID++
			}
		}
		attempt++
	}
	if len(entries) > 0 {
		return nil, sdk.WithStack(fmt.Errorf("Unable to process %+v", entries))
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

	// if there is a template instance id on the workflow export, add it
	if w.Template != nil {
		templatePath := strings.Split(*w.Template, "/")
		if len(templatePath) != 2 {
			return nil, sdk.WithStack(fmt.Errorf("Invalid template path"))
		}
		wf.Template = &sdk.WorkflowTemplate{
			Group: &sdk.Group{Name: templatePath[0]},
			Slug:  templatePath[1],
		}
	}

	wf.SortNode()

	return wf, nil
}

func (e *NodeEntry) getNode(name string, w *sdk.Workflow) (*sdk.Node, error) {
	node := &sdk.Node{
		Name: name,
		Ref:  name,
		Type: sdk.NodeTypeFork,
		Context: &sdk.NodeContext{
			PipelineName:           e.PipelineName,
			ApplicationName:        e.ApplicationName,
			EnvironmentName:        e.EnvironmentName,
			ProjectIntegrationName: e.ProjectIntegrationName,
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
		node.Context.Conditions = *e.Conditions
	}

	if len(e.Payload) > 0 {
		if len(e.DependsOn) > 0 {
			return nil, sdk.WrapError(sdk.ErrInvalidNodeDefaultPayload, "Default payload cannot be set on another node than the first one (node : %s)", name)
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
				Value:    sdk.StatusSuccess.String(),
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
			if h.Ref == "" {
				h.Ref = fmt.Sprintf("%d", time.Now().Unix())
			}

			n.Hooks = append(n.Hooks, sdk.NodeHook{
				Config:        cfg,
				Ref:           h.Ref,
				HookModelName: h.Model,
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

	n, err := e.getNode(name, w)
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
	case 2:
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
