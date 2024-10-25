package exportentities

import "github.com/ovh/cds/sdk"

// GRPCPlugin represents exported sdk.GRPCPlugin
type GRPCPlugin struct {
	Name        string                     `json:"name" yaml:"name" cli:"name,key"`
	Type        string                     `json:"type" yaml:"type" cli:"type"`
	Integration string                     `json:"integration" yaml:"integration" cli:"integration"`
	Author      string                     `json:"author" yaml:"author" cli:"author"`
	Description string                     `json:"description" yaml:"description" cli:"description"`
	Inputs      map[string]sdk.PluginInput `json:"inputs,omitempty" yaml:"inputs,omitempty"`
	Post        sdk.PluginPost             `json:"post" yaml:"post"`
}

// NewGRPCPlugin returns a ready to export action
func NewGRPCPlugin(p sdk.GRPCPlugin) (plg GRPCPlugin) {
	plg.Name = p.Name
	plg.Type = p.Type
	plg.Integration = p.Integration
	plg.Author = p.Author
	plg.Description = p.Description
	plg.Inputs = p.Inputs
	plg.Post = p.Post
	return plg
}

// GRPCPlugin returns an sdk.GRPCPlugin
func (plg *GRPCPlugin) GRPCPlugin() *sdk.GRPCPlugin {
	p := new(sdk.GRPCPlugin)
	p.Name = plg.Name
	p.Type = plg.Type
	p.Integration = plg.Integration
	p.Author = plg.Author
	p.Description = plg.Description
	p.Inputs = plg.Inputs
	p.Post = plg.Post
	return p
}
