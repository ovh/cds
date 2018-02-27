package cdsclient

import (
	"fmt"
	"io"
	"net/http"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func (c *client) Requirements() ([]sdk.Requirement, error) {
	var req []sdk.Requirement
	if _, err := c.GetJSON("/action/requirement", &req); err != nil {
		return nil, err
	}
	return req, nil
}

func (c *client) ActionDelete(actionName string) error {
	_, err := c.DeleteJSON("/action/"+actionName, nil)
	return err
}

func (c *client) ActionGet(actionName string, mods ...RequestModifier) (*sdk.Action, error) {
	action := &sdk.Action{}
	if _, err := c.GetJSON("/action/"+actionName, action, mods...); err != nil {
		return nil, err
	}
	return action, nil
}

func (c *client) ActionList() ([]sdk.Action, error) {
	actions := []sdk.Action{}
	if _, err := c.GetJSON("/action", &actions); err != nil {
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

	_, _, code, err := c.Request("POST", url, content, mods...)
	if err != nil {
		return err
	}

	if code > 400 {
		return fmt.Errorf("HTTP Code %d", code)
	}

	return nil
}

func (c *client) ActionExport(name string, format string) ([]byte, error) {
	path := fmt.Sprintf("/action/%s/export?format=%s", name, format)
	body, _, _, err := c.Request("GET", path, nil)
	if err != nil {
		return nil, err
	}
	return body, nil
}
