package cdsclient

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectNotificationCreate(ctx context.Context, pKey string, notif *sdk.ProjectNotification) error {
	path := fmt.Sprintf("/v2/project/%s/notification", pKey)
	_, err := c.PostJSON(ctx, path, notif, notif)
	return err
}
func (c *client) ProjectNotificationUpdate(ctx context.Context, pKey string, notif *sdk.ProjectNotification) error {
	path := fmt.Sprintf("/v2/project/%s/notification/%s", pKey, notif.Name)
	_, err := c.PutJSON(ctx, path, notif, notif)
	return err
}
func (c *client) ProjectNotificationDelete(ctx context.Context, pKey string, notifName string) error {
	path := fmt.Sprintf("/v2/project/%s/notification/%s", pKey, notifName)
	_, err := c.DeleteJSON(ctx, path, nil)
	return err
}
func (c *client) ProjectNotificationGet(ctx context.Context, pKey string, notifName string) (*sdk.ProjectNotification, error) {
	var notif sdk.ProjectNotification
	path := fmt.Sprintf("/v2/project/%s/notification/%s", pKey, notifName)
	_, err := c.GetJSON(ctx, path, &notif)
	return &notif, err
}

func (c *client) ProjectNotificationList(ctx context.Context, pKey string) ([]sdk.ProjectNotification, error) {
	var notifs []sdk.ProjectNotification
	path := fmt.Sprintf("/v2/project/%s/notification", pKey)
	_, err := c.GetJSON(ctx, path, &notifs)
	return notifs, err
}
