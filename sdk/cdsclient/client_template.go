package cdsclient

import (
	"fmt"

	"github.com/ovh/cds/sdk"
)

// TemplateList
func (c *client) TemplateList() ([]sdk.Template, error) {
	templates := []sdk.Template{}
	if _, err := c.GetJSON("/template/build", templates); err != nil {
		return nil, err
	}
	return templates, nil
}

// TemplateGet Get the build template corresponding to the given name
func (c *client) TemplateGet(name string) (*sdk.Template, error) {
	tpls, err := c.TemplateList()
	if err != nil {
		return nil, err
	}

	for _, t := range tpls {
		if t.Name == name {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("%s: not found", err)
}

// TemplateApplicationCreate creates given application and apply build template
func (c *client) TemplateApplicationCreate(projectKey, name string, template *sdk.Template) error {
	opts := sdk.ApplyTemplatesOptions{
		ApplicationName: name,
		TemplateName:    template.Name,
		TemplateParams:  template.Params,
	}
	_, err := c.PostJSON(fmt.Sprintf("/project/%s/template", projectKey), opts, nil)
	return err
}
