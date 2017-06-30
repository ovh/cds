package cdsclient

import (
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectCreate(p *sdk.Project) error {
	code, err := c.PostJSON("/project", p, nil)
	if code != 201 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func (c *client) ProjectDelete(key string) error {
	code, err := c.DeleteJSON("/project/"+key, nil, nil)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func (c *client) ProjectInfo(key string) (*sdk.Project, error) {
	p := &sdk.Project{}
	code, err := c.GetJSON("/project/"+key, p)
	if code != 200 {
		if err == nil {
			return nil, fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (c *client) ProjectList() ([]sdk.Project, error) {
	p := []sdk.Project{}
	code, err := c.GetJSON("/project", &p)
	if code != 200 {
		if err == nil {
			return nil, fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}
