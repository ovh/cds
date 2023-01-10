package cdsclient

import (
	"context"
	"github.com/ovh/cds/sdk"
)

func (c *client) RBACImport(ctx context.Context, rbacRule sdk.RBAC, mods ...RequestModifier) (sdk.RBAC, error) {
	path := "/v2/rbac/import"
	_, err := c.PostJSON(ctx, path, &rbacRule, &rbacRule, mods...)
	return rbacRule, err
}

func (c *client) RBACDelete(ctx context.Context, permissionName string) error {
	path := "/v2/rbac/" + permissionName
	_, err := c.DeleteJSON(ctx, path, nil)
	return err
}
