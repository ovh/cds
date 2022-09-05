package cdsclient

import (
	"context"
	"io"

	"github.com/rockbears/yaml"

	"github.com/ovh/cds/sdk"
)

func (c *client) RBACImport(ctx context.Context, content io.Reader, mods ...RequestModifier) (sdk.RBAC, error) {
	var rbacRule sdk.RBAC

	body, err := io.ReadAll(content)
	if err != nil {
		return rbacRule, err
	}

	if err := yaml.Unmarshal(body, &rbacRule); err != nil {
		return rbacRule, err
	}

	path := "/v2/rbac/import"
	_, err = c.PostJSON(ctx, path, &rbacRule, &rbacRule, mods...)
	return rbacRule, err
}
