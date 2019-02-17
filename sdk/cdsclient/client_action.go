package cdsclient

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func (c *client) Requirements() ([]sdk.Requirement, error) {
	var req []sdk.Requirement
	if _, err := c.GetJSON(context.Background(), "/action/requirement", &req); err != nil {
		return nil, err
	}
	return req, nil
}

func (c *client) ActionDelete(groupName, name string) error {
	path := fmt.Sprintf("/action/%s/%s", groupName, name)
	_, err := c.DeleteJSON(context.Background(), path, nil)
	return err
}

func (c *client) ActionGet(groupName, name string, mods ...RequestModifier) (*sdk.Action, error) {
	var a sdk.Action

	path := fmt.Sprintf("/action/%s/%s", groupName, name)
	if _, err := c.GetJSON(context.Background(), path, &a, mods...); err != nil {
		return nil, err
	}

	return &a, nil
}

func (c *client) ActionList() ([]sdk.Action, error) {
	actions := []sdk.Action{}
	if _, err := c.GetJSON(context.Background(), "/action", &actions); err != nil {
		return nil, err
	}
	return actions, nil
}

func (c *client) ActionImport(content io.Reader, format string) error {
	url := "/action/import"
	mods := []RequestModifier{}
	switch format {
	case "json":
		mods = []RequestModifier{
			func(r *http.Request) {
				r.Header.Set("Content-Type", "application/json")
			},
		}
	case "yaml", "yml":
		mods = []RequestModifier{
			func(r *http.Request) {
				r.Header.Set("Content-Type", "application/x-yaml")
			},
		}
	default:
		return exportentities.ErrUnsupportedFormat
	}

	_, _, code, err := c.Request(context.Background(), "POST", url, content, mods...)
	if err != nil {
		return err
	}

	if code > 400 {
		return fmt.Errorf("HTTP Code %d", code)
	}

	return nil
}

func (c *client) ActionExport(groupName, name string, format string) ([]byte, error) {
	path := fmt.Sprintf("/action/%s/%s/export?format=%s", groupName, name, format)
	body, _, _, err := c.Request(context.Background(), "GET", path, nil)
	if err != nil {
		return nil, err
	}
	return body, nil
}
