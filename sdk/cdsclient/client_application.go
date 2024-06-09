package cdsclient

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) ApplicationCreate(key string, app *sdk.Application) error {
	_, err := c.PostJSON(context.Background(), "/project/"+key+"/applications", app, nil)
	return err
}

func (c *client) ApplicationUpdate(projectKey string, appName string, app *sdk.Application) error {
	url := fmt.Sprintf("/project/%s/application/%s", url.QueryEscape(projectKey), url.QueryEscape(appName))
	_, err := c.PutJSON(context.Background(), url, app, app)
	return err
}

func (c *client) ApplicationDelete(key string, appName string) error {
	_, err := c.DeleteJSON(context.Background(), "/project/"+key+"/application/"+appName, nil)
	return err
}

func (c *client) ApplicationGet(key string, appName string, mods ...RequestModifier) (*sdk.Application, error) {
	app := &sdk.Application{}
	if _, err := c.GetJSON(context.Background(), "/project/"+key+"/application/"+appName, app, mods...); err != nil {
		return nil, err
	}
	return app, nil
}

func (c *client) ApplicationList(key string) ([]sdk.Application, error) {
	apps := []sdk.Application{}
	if _, err := c.GetJSON(context.Background(), "/project/"+key+"/applications", &apps); err != nil {
		return nil, err
	}
	return apps, nil
}

// ApplicationAttachToReposistoriesManager attaches the application to the repo identified by its fullname in the reposManager
func (c *client) ApplicationAttachToReposistoriesManager(projectKey, appName, reposManager, repoFullname string) error {
	uri := fmt.Sprintf("/project/%s/repositories_manager/%s/application/%s/attach?fullname=%s", projectKey, reposManager, appName, url.QueryEscape(repoFullname))
	_, _, _, err := c.Request(context.Background(), "POST", uri, nil)
	return err
}
