package exportentities

import "github.com/ovh/cds/sdk"

// GRPCPlugin represents exported sdk.GRPCPlugin
type GRPCPlugin struct {
	Name        string                    `json:"name" yaml:"name" cli:"name,key"`
	Type        string                    `json:"type" yaml:"type" cli:"type"`
	Author      string                    `json:"author" yaml:"author" cli:"author"`
	Description string                    `json:"description" yaml:"description" cli:"description"`
	Parameters  map[string]ParameterValue `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

// NewGRPCPlugin returns a ready to export action
func NewGRPCPlugin(p sdk.GRPCPlugin) (plg GRPCPlugin) {
	plg.Name = p.Name
	plg.Type = p.Type
	plg.Author = p.Author
	plg.Description = p.Description
	plg.Parameters = make(map[string]ParameterValue, len(p.Parameters))
	for k, v := range p.Parameters {
		param := ParameterValue{
			Type:         string(v.Type),
			DefaultValue: v.Value,
			Description:  v.Description,
		}
		// no need to export it if "Advanced" is false
		if v.Advanced {
			param.Advanced = &p.Parameters[k].Advanced
		}
		plg.Parameters[v.Name] = param
	}
	return plg
}

// GRPCPlugin returns an sdk.GRPCPlugin
func (plg *GRPCPlugin) GRPCPlugin() *sdk.GRPCPlugin {
	p := new(sdk.GRPCPlugin)
	p.Name = plg.Name
	p.Type = plg.Type
	p.Author = plg.Author
	p.Description = plg.Description

	//Compute parameters
	p.Parameters = make([]sdk.Parameter, len(plg.Parameters))
	var i int
	for paramName, v := range plg.Parameters {
		param := sdk.Parameter{
			Name:        paramName,
			Type:        v.Type,
			Value:       v.DefaultValue,
			Description: v.Description,
		}
		if param.Type == "" {
			param.Type = sdk.StringParameter
		}
		if v.Advanced != nil && *v.Advanced {
			param.Advanced = true
		}
		p.Parameters[i] = param
		i++
	}

	return p
}
