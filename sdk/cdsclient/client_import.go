package cdsclient

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/sdk/exportentities"
)

func (c *client) PipelineImport(projectKey string, content io.Reader, format string, force bool) ([]string, error) {
	var url string
	url = fmt.Sprintf("/project/%s/import/pipeline?format=%s", projectKey, format)

	if force {
		url += "&forceUpdate=true"
	}

	btes, _, code, errReq := c.Request("POST", url, content)
	if errReq != nil {
		return nil, errReq
	}

	if code >= 400 {
		return nil, fmt.Errorf("HTTP Status code %d", code)
	}

	var msgs []string
	if err := json.Unmarshal(btes, &msgs); err != nil {
		return []string{string(btes)}, errReq
	}

	return msgs, nil
}

func (c *client) ApplicationImport(projectKey string, content io.Reader, format string, force bool) ([]string, error) {
	var url string
	url = fmt.Sprintf("/project/%s/import/application", projectKey)
	if force {
		url += "?force=true"
	}

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
		return nil, exportentities.ErrUnsupportedFormat
	}

	btes, _, code, err := c.Request("POST", url, content, mods...)
	if err != nil {
		return nil, err
	}

	if code >= 400 {
		return nil, fmt.Errorf("HTTP Status code %d", code)
	}

	messages := []string{}
	if code > 400 {
		if err := json.Unmarshal(btes, &messages); err != nil {
			return nil, sdk.WrapError(err, "HTTP Code %d", code)
		}
		return messages, fmt.Errorf("HTTP Code %d", code)
	}

	if err := json.Unmarshal(btes, &messages); err != nil {
		return nil, err
	}

	return messages, nil
}

func (c *client) EnvironmentImport(projectKey string, content io.Reader, format string, force bool) ([]string, error) {
	var url string
	url = fmt.Sprintf("/project/%s/import/environment", projectKey)
	if force {
		url += "?force=true"
	}

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
		return nil, exportentities.ErrUnsupportedFormat
	}

	btes, _, code, err := c.Request("POST", url, content, mods...)
	if err != nil {
		return nil, err
	}

	if code >= 400 {
		return nil, fmt.Errorf("HTTP Status code %d", code)
	}

	messages := []string{}
	if err := json.Unmarshal(btes, &messages); err != nil {
		return nil, err
	}

	return messages, nil
}

func (c *client) WorkflowImport(projectKey string, content io.Reader, format string, force bool) ([]string, error) {
	var url string
	url = fmt.Sprintf("/project/%s/import/workflows", projectKey)
	if force {
		url += "?force=true"
	}

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
		return nil, exportentities.ErrUnsupportedFormat
	}

	btes, _, code, err := c.Request("POST", url, content, mods...)
	if err != nil {
		return nil, err
	}

	if code >= 400 {
		return nil, fmt.Errorf("HTTP Status code %d", code)
	}

	messages := []string{}
	if err := json.Unmarshal(btes, &messages); err != nil {
		return nil, err
	}

	return messages, nil
}

func (c *client) WorkflowPush(projectKey string, tarContent io.Reader, mods ...RequestModifier) ([]string, *tar.Reader, error) {
	url := fmt.Sprintf("/project/%s/push/workflows", projectKey)

	mods = append(mods,
		func(r *http.Request) {
			r.Header.Set("Content-Type", "application/tar")
		})

	btes, headers, code, err := c.Request("POST", url, tarContent, mods...)
	if err != nil {
		return nil, nil, err
	}

	if code >= 400 {
		return nil, nil, fmt.Errorf("HTTP Status code %d", code)
	}

	messages := []string{}
	if err := json.Unmarshal(btes, &messages); err != nil {
		return nil, nil, err
	}

	wName := headers.Get(sdk.ResponseWorkflowNameHeader)
	if wName == "" {
		return messages, nil, nil
	}
	tarReader, err := c.WorkflowPull(projectKey, wName, false)
	if err != nil {
		return nil, nil, err
	}

	return messages, tarReader, nil
}
