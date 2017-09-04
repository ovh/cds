package sdk

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"time"

	"github.com/facebookgo/httpcontrol"
	"github.com/hashicorp/hcl"
)

//ActionScript represents the structure of a HCL action file
type ActionScript struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	Requirements map[string]Requirement `json:"requirement,omitempty"`
	Parameters   map[string]Parameter   `json:"parameters,omitempty"`
	Steps        []struct {
		Enabled          *bool                        `json:"enabled"`
		AlwaysExecuted   bool                         `json:"always_executed"`
		ArtifactUpload   map[string]string            `json:"artifactUpload,omitempty"`
		ArtifactDownload map[string]string            `json:"artifactDownload,omitempty"`
		GitClone         map[string]string            `json:"gitClone,omitempty"`
		GitTag           map[string]string            `json:"gitTag,omitempty"`
		Script           string                       `json:"script,omitempty"`
		JUnitReport      string                       `json:"jUnitReport,omitempty"`
		Plugin           map[string]map[string]string `json:"plugin,omitempty"`
	} `json:"steps"`
}

// NewStepScript returns an action (basically used as a step of a job) of Script type
func NewStepScript(s string) Action {
	newAction := Action{
		Name: ScriptAction,
		Type: BuiltinAction,
		Parameters: []Parameter{
			{
				Name:  "script",
				Value: s,
				Type:  TextParameter,
			},
		},
	}
	return newAction
}

// NewStepJUnitReport returns an action (basically used as a step of a job) of JUnitReport type
func NewStepJUnitReport(s string) Action {
	newAction := Action{
		Name: JUnitAction,
		Type: BuiltinAction,
		Parameters: []Parameter{
			{
				Name:  "path",
				Value: s,
				Type:  StringParameter,
			},
		},
	}
	return newAction
}

// NewStepGitClone returns an action (basically used as a step of a job) of GitClone type
func NewStepGitClone(v map[string]string) Action {
	newAction := Action{
		Name:       GitCloneAction,
		Type:       BuiltinAction,
		Parameters: ParametersFromMap(v),
	}
	return newAction
}

// NewStepGitTag returns an action (basically used as a step of a job) of GitTag type
func NewStepGitTag(v map[string]string) Action {
	newAction := Action{
		Name:       GitTagAction,
		Type:       BuiltinAction,
		Parameters: ParametersFromMap(v),
	}
	return newAction
}

// NewStepArtifactUpload returns an action (basically used as a step of a job) of artifact upload type
func NewStepArtifactUpload(v map[string]string) Action {
	newAction := Action{
		Name:       ArtifactUpload,
		Type:       BuiltinAction,
		Parameters: ParametersFromMap(v),
	}
	return newAction
}

// NewStepArtifactDownload returns an action (basically used as a step of a job) of artifact download type
func NewStepArtifactDownload(v map[string]string) Action {
	newAction := Action{
		Name:       ArtifactDownload,
		Type:       BuiltinAction,
		Parameters: ParametersFromMap(v),
	}
	return newAction
}

// NewStepPlugin returns an action (basically used as a step of a job) of plugin type
func NewStepPlugin(v map[string]map[string]string) (*Action, error) {
	if len(v) != 1 {
		return nil, fmt.Errorf("Malformatted plugin step")
	}
	for k, v := range v {
		newAction := Action{
			Name:       k,
			Type:       PluginAction,
			Parameters: []Parameter{},
		}
		for p, val := range v {
			newAction.Parameters = append(newAction.Parameters, Parameter{
				Name:  p,
				Value: val,
			})
		}
		return &newAction, nil
	}
	return nil, nil
}

// NewStepDefault returns an action (basically used as a step of a job) of default type
func NewStepDefault(n string, args map[string]string) (*Action, error) {
	newAction := Action{
		Name:       n,
		Parameters: []Parameter{},
	}
	for p, val := range args {
		newAction.Parameters = append(newAction.Parameters, Parameter{
			Name:  p,
			Value: val,
		})
	}
	return &newAction, nil
}

//NewActionFromScript creates an action from a HCL file as bytes
func NewActionFromScript(btes []byte) (*Action, error) {
	as := ActionScript{}
	if err := hcl.Decode(&as, string(btes)); err != nil {
		return nil, err
	}

	a := Action{
		Name:         as.Name,
		Description:  as.Description,
		Requirements: []Requirement{},
		Parameters:   []Parameter{},
		Actions:      []Action{},
		Enabled:      true,
	}

	for k, v := range as.Requirements {
		a.Requirements = append(a.Requirements, Requirement{
			Name:  k,
			Type:  v.Type,
			Value: v.Value,
		})
	}

	for k, v := range as.Parameters {
		a.Parameters = append(a.Parameters, Parameter{
			Name:        k,
			Type:        v.Type,
			Description: v.Description,
			Value:       v.Value,
		})
	}

	for _, v := range as.Steps {
		var newAction Action
		//Action builtin = Script
		if v.Script != "" {
			newAction = NewStepScript(v.Script)
			goto next
		}

		//Action builtin =JUnitReport
		if v.JUnitReport != "" {
			newAction = NewStepJUnitReport(v.JUnitReport)
			goto next
		}

		//Action builtin = ArtifactUpload
		if v.ArtifactUpload != nil {
			newAction = NewStepArtifactUpload(v.ArtifactUpload)
			goto next
		}

		//Action builtin = ArtifactDownload
		if v.ArtifactDownload != nil {
			newAction = NewStepArtifactDownload(v.ArtifactDownload)
			goto next
		}

		//Action builtin = GitClone
		if v.GitClone != nil {
			newAction = NewStepGitClone(v.GitClone)
			goto next
		}

		if v.GitTag != nil {
			newAction = NewStepGitTag(v.GitTag)
		}

		//Action builtin = Plugin
		if v.Plugin != nil {
			a, err := NewStepPlugin(v.Plugin)
			if err != nil {
				return nil, err
			}
			newAction = *a
			goto next
		}

		return nil, fmt.Errorf("Unsupported action : %s", string(btes))

	next:
		if v.Enabled != nil {
			newAction.Enabled = *v.Enabled
		} else {
			newAction.Enabled = true
		}
		newAction.AlwaysExecuted = v.AlwaysExecuted
		a.Actions = append(a.Actions, newAction)
	}

	return &a, nil
}

// ActionInfoMarkdown returns string formatted with markdown
func ActionInfoMarkdown(a *Action, filename string) string {
	var sp, rq string
	ps := a.Parameters
	sort.Slice(ps, func(i, j int) bool { return ps[i].Name < ps[j].Name })
	for _, p := range ps {
		sp += fmt.Sprintf("* **%s**: %s\n", p.Name, p.Description)
	}
	if sp == "" {
		sp = "No Parameter"
	}

	rs := a.Requirements
	sort.Slice(rs, func(i, j int) bool { return rs[i].Name < rs[j].Name })
	for _, r := range rs {
		rq += fmt.Sprintf("* **%s**: type: %s Value: %s\n", r.Name, r.Type, r.Value)
	}

	if rq == "" {
		rq = "No Requirement"
	}

	info := fmt.Sprintf(`
%s

## Parameters

%s

## Requirements

%s

More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/actions/%s)

`,
		a.Description,
		sp,
		rq,
		filename)

	return info
}

func loadRemoteScript(url string) (*Action, error) {
	client := &http.Client{
		Transport: &httpcontrol.Transport{
			RequestTimeout: time.Minute,
			MaxTries:       3,
		},
	}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return NewActionFromScript(body)
}

//NewActionFromRemoteScript creates an action from an URL giving an HCL file
func NewActionFromRemoteScript(url string, params []Parameter) (*Action, error) {
	a, err := loadRemoteScript(url)
	if err != nil {
		return nil, err
	}
	//Override params value
	for _, p := range params {
		for _, pp := range a.Parameters {
			if p.Name == pp.Name {
				pp.Value = p.Value
			}
		}
	}
	return a, nil
}

//NewActionScript creates a builtin action script
func NewActionScript(script string, requirements []Requirement) Action {
	return Action{
		Name: ScriptAction,
		Type: BuiltinAction,
		Parameters: []Parameter{
			{
				Name:  "script",
				Value: script,
				Type:  TextParameter,
			},
		},
		Requirements: requirements,
	}
}

//NewActionArtifactDownload creates a builtin action artifactDownload
func NewActionArtifactDownload(path, tag string) Action {
	return Action{
		Name: ArtifactDownload,
		Type: BuiltinAction,
		Parameters: []Parameter{
			{
				Name:  "path",
				Value: path,
				Type:  StringParameter,
			},
			{
				Name:  "tag",
				Value: tag,
				Type:  StringParameter,
			},
		},
	}
}

//NewActionArtifactUpload creates a builtin action artifactUpload
func NewActionArtifactUpload(path, tag string) Action {
	return Action{
		Name: ArtifactUpload,
		Type: BuiltinAction,
		Parameters: []Parameter{
			{
				Name:  "path",
				Value: path,
				Type:  StringParameter,
			},
			{
				Name:  "tag",
				Value: tag,
				Type:  StringParameter,
			},
		},
	}
}

//NewActionJUnit  creates a builtin action junit
func NewActionJUnit(path string) Action {
	return Action{
		Name: JUnitAction,
		Type: BuiltinAction,
		Parameters: []Parameter{
			{
				Name:  "path",
				Value: path,
				Type:  StringParameter,
			},
		},
	}
}

//NewActionPlugin  creates a plugin action
func NewActionPlugin(pluginname string, parameters []Parameter) Action {
	return Action{
		Name:       pluginname,
		Type:       PluginAction,
		Parameters: parameters,
		Requirements: []Requirement{
			{
				Name:  pluginname,
				Type:  PluginRequirement,
				Value: pluginname,
			},
		},
	}
}
