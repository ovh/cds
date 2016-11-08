package sdk

import (
	"net/http"
	"time"

	"io/ioutil"

	"fmt"

	"github.com/facebookgo/httpcontrol"
	"github.com/hashicorp/hcl"
)

type ActionScript struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	Requirements map[string]Requirement `json:"requirement,omitempty"`
	Parameters   map[string]Parameter   `json:"parameters,omitempty"`
	Steps        []struct {
		ArtifactUpload   map[string]string            `json:"artifactUpload,omitempty"`
		ArtifactDownload map[string]string            `json:"artifactDownload,omitempty"`
		Script           string                       `json:"script,omitempty"`
		JUnitReport      string                       `json:"jUnitReport,omitempty"`
		Plugin           map[string]map[string]string `json:"plugin,omitempty"`
	} `json:"steps"`
}

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
			newAction = Action{
				Name: ScriptAction,
				Type: BuiltinAction,
				Parameters: []Parameter{
					{
						Name:  "script",
						Value: v.Script,
						Type:  TextParameter,
					},
				},
				Enabled: true,
			}
			goto next
		}

		//Action builtin =JUnitReport
		if v.JUnitReport != "" {
			newAction = Action{
				Name: JUnitAction,
				Type: BuiltinAction,
				Parameters: []Parameter{
					{
						Name:  "path",
						Value: v.JUnitReport,
						Type:  StringParameter,
					},
				},
				Enabled: true,
			}
			goto next
		}

		//Action builtin = ArtifactUpload
		if v.ArtifactUpload != nil {
			newAction = Action{
				Name: ArtifactUpload,
				Type: BuiltinAction,
				Parameters: []Parameter{
					{
						Name:  "path",
						Value: v.ArtifactUpload["path"],
						Type:  StringParameter,
					},
					{
						Name:  "tag",
						Value: v.ArtifactUpload["tag"],
						Type:  StringParameter,
					},
				},
				Enabled: true,
			}
			goto next
		}

		//Action builtin = ArtifactDownload
		if v.ArtifactDownload != nil {
			newAction = Action{
				Name: ArtifactDownload,
				Type: BuiltinAction,
				Parameters: []Parameter{
					{
						Name:  "path",
						Value: v.ArtifactDownload["path"],
						Type:  StringParameter,
					},
					{
						Name:  "tag",
						Value: v.ArtifactDownload["tag"],
						Type:  StringParameter,
					},
				},
				Enabled: true,
			}
			goto next
		}

		//Action builtin = Plugin
		if v.Plugin != nil {
			for k, v := range v.Plugin {
				newAction = Action{
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
				goto next
			}
		}

		return nil, fmt.Errorf("Unsupported action : %s", string(btes))

	next:
		a.Actions = append(a.Actions, newAction)
	}

	return &a, nil
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
