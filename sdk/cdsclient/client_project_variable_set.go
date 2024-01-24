package cdsclient

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectVariableSetCreate(ctx context.Context, pKey string, vs *sdk.ProjectVariableSet) error {
	path := fmt.Sprintf("/v2/project/%s/variableset", pKey)
	_, err := c.PostJSON(ctx, path, vs, vs)
	return err
}

func (c *client) ProjectVariableSetDelete(ctx context.Context, pKey string, vsName string) error {
	path := fmt.Sprintf("/v2/project/%s/variableset/%s", pKey, vsName)
	_, err := c.DeleteJSON(ctx, path, nil)
	return err
}

func (c *client) ProjectVariableSetList(ctx context.Context, pKey string) ([]sdk.ProjectVariableSet, error) {
	var vss []sdk.ProjectVariableSet
	path := fmt.Sprintf("/v2/project/%s/variableset", pKey)
	_, err := c.GetJSON(ctx, path, &vss)
	return vss, err
}

func (c *client) ProjectVariableSetShow(ctx context.Context, pKey string, vsName string) (*sdk.ProjectVariableSet, error) {
	var vs sdk.ProjectVariableSet
	path := fmt.Sprintf("/v2/project/%s/variableset/%s", pKey, vsName)
	_, err := c.GetJSON(ctx, path, &vs)
	return &vs, err
}

func (c *client) ProjectVariableSetItemAdd(ctx context.Context, pKey string, vsName string, item *sdk.ProjectVariableSetItem) error {
	path := fmt.Sprintf("/v2/project/%s/variableset/%s/item", pKey, vsName)
	_, err := c.PostJSON(ctx, path, item, item)
	return err
}

func (c *client) ProjectVariableSetItemUpdate(ctx context.Context, pKey string, vsName string, item *sdk.ProjectVariableSetItem) error {
	path := fmt.Sprintf("/v2/project/%s/variableset/%s/item/%s", pKey, vsName, item.Name)
	_, err := c.PutJSON(ctx, path, item, item)
	return err
}

func (c *client) ProjectVariableSetItemDelete(ctx context.Context, pKey string, vsName string, itemName string) error {
	path := fmt.Sprintf("/v2/project/%s/variableset/%s/item/%s", pKey, vsName, itemName)
	_, err := c.DeleteJSON(ctx, path, nil)
	return err
}

func (c *client) ProjectVariableSetItemGet(ctx context.Context, pKey string, vsName string, itemName string) (*sdk.ProjectVariableSetItem, error) {
	var item sdk.ProjectVariableSetItem
	path := fmt.Sprintf("/v2/project/%s/variableset/%s/item/%s", pKey, vsName, itemName)
	_, err := c.GetJSON(ctx, path, &item)
	return &item, err
}
