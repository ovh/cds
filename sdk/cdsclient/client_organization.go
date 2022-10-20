package cdsclient

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (c *client) OrganizationAdd(ctx context.Context, orga sdk.Organization) error {
	if _, err := c.PostJSON(ctx, "/v2/organization", &orga, nil); err != nil {
		return err
	}
	return nil
}

func (c *client) OrganizationGet(ctx context.Context, organizationIdentifier string) (sdk.Organization, error) {
	var orga sdk.Organization
	if _, err := c.GetJSON(ctx, "/v2/organization/"+organizationIdentifier, &orga, nil); err != nil {
		return orga, err
	}
	return orga, nil
}

func (c *client) OrganizationList(ctx context.Context) ([]sdk.Organization, error) {
	var orgs []sdk.Organization
	if _, err := c.GetJSON(ctx, "/v2/organization", &orgs, nil); err != nil {
		return nil, err
	}
	return orgs, nil
}

func (c *client) OrganizationDelete(ctx context.Context, organizationIdentifier string) error {
	if _, err := c.DeleteJSON(ctx, "/v2/organization/"+organizationIdentifier, nil, nil); err != nil {
		return err
	}
	return nil
}
