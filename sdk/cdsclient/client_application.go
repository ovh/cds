package cdsclient

import (
	"encoding/json"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) ApplicationCreate(key string, app *sdk.Application) error {
	code, err := c.PostJSON("/project/"+key+"/applications", app, nil)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}

func (c *client) ApplicationDelete(key string, appName string) error {
	code, err := c.DeleteJSON("/project/"+key+"/application/"+appName, nil)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}

func (c *client) ApplicationGet(key string, appName string, mods ...RequestModifier) (*sdk.Application, error) {
	app := &sdk.Application{}
	code, err := c.GetJSON("/project/"+key+"/application/"+appName, app, mods...)
	if code != 200 {
		if err == nil {
			return nil, fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (c *client) ApplicationList(key string) ([]sdk.Application, error) {
	apps := []sdk.Application{}
	code, err := c.GetJSON("/project/"+key+"/applications", &apps)
	if code != 200 {
		if err == nil {
			return nil, fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return nil, err
	}
	return apps, nil
}

func (c *client) ApplicationGroupsImport(projectKey, appName string, content []byte, format string, force bool) (sdk.Application, error) {
	var url string
	var app sdk.Application
	url = fmt.Sprintf("/project/%s/application/%s/group/import?format=%s", projectKey, appName, format)

	if force {
		url += "&forceUpdate=true"
	}

	btes, code, errReq := c.Request("POST", url, content)
	if code != 200 {
		if errReq == nil {
			return app, fmt.Errorf("HTTP Code %d", code)
		}
	}

	if err := json.Unmarshal(btes, &app); err != nil {
		return app, errReq
	}

	return app, errReq
}
