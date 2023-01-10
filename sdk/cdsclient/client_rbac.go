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
