package cdsclient

import (
	"fmt"

	"github.com/ovh/cds/sdk"
)

// TemplateList
func (c *client) TemplateList() ([]sdk.Template, error) {
	templates := []sdk.Template{}
	code, err := c.GetJSON("/template/build", templates)
	if code != 200 {
		if err == nil {
			return nil, fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
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
	code, err := c.PostJSON(fmt.Sprintf("/project/%s/template", projectKey), opts, nil)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}
