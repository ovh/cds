package cdsclient

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ovh/cds/sdk"
)

func (c *client) TemplateGet(groupName, templateSlug string) (*sdk.WorkflowTemplate, error) {
	url := fmt.Sprintf("/template/%s/%s", groupName, templateSlug)

	var wt sdk.WorkflowTemplate
	if _, err := c.GetJSON(context.Background(), url, &wt); err != nil {
		return nil, err
	}

	return &wt, nil
}

func (c *client) TemplateGetByID(id int64) (*sdk.WorkflowTemplate, error) {
	url := fmt.Sprintf("/template/%d", id)

	var wt sdk.WorkflowTemplate
	if _, err := c.GetJSON(context.Background(), url, &wt); err != nil {
		return nil, err
	}

	return &wt, nil
}

func (c *client) TemplateGetAll() ([]sdk.WorkflowTemplate, error) {
	url := "/template"

	var wts []sdk.WorkflowTemplate
	if _, err := c.GetJSON(context.Background(), url, &wts); err != nil {
		return nil, err
	}

	return wts, nil
}

func (c *client) TemplateApply(groupName, templateSlug string, req sdk.WorkflowTemplateRequest) (*tar.Reader, error) {
	url := fmt.Sprintf("/template/%s/%s/apply", groupName, templateSlug)

	bs, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	body, _, _, err := c.Request(context.Background(), "POST", url, bytes.NewReader(bs))
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(body)
	tr := tar.NewReader(r)
	return tr, nil
}

func (c *client) TemplateBulk(groupName, templateSlug string, req sdk.WorkflowTemplateBulk) (*sdk.WorkflowTemplateBulk, error) {
	url := fmt.Sprintf("/template/%s/%s/bulk", groupName, templateSlug)

	var res sdk.WorkflowTemplateBulk
	_, err := c.PostJSON(context.Background(), url, req, &res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func (c *client) TemplateGetBulk(groupName, templateSlug string, id int64) (*sdk.WorkflowTemplateBulk, error) {
	url := fmt.Sprintf("/template/%s/%s/bulk/%d", groupName, templateSlug, id)

	var res sdk.WorkflowTemplateBulk
	_, err := c.GetJSON(context.Background(), url, &res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func (c *client) TemplatePull(groupName, templateSlug string) (*tar.Reader, error) {
	url := fmt.Sprintf("/template/%s/%s/pull", groupName, templateSlug)

	body, _, _, err := c.Request(context.Background(), "POST", url, nil)
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(body)
	tr := tar.NewReader(r)
	return tr, nil
}

func (c *client) TemplatePush(tarContent io.Reader) ([]string, *tar.Reader, error) {
	url := "/template/push"

	btes, headers, code, err := c.Request(context.Background(), "POST", url, tarContent, func(r *http.Request) {
		r.Header.Set("Content-Type", "application/tar")
	})
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

	tGroupName := headers.Get(sdk.ResponseTemplateGroupNameHeader)
	tSlug := headers.Get(sdk.ResponseTemplateSlugHeader)
	if tGroupName == "" || tSlug == "" {
		return messages, nil, nil
	}
	tarReader, err := c.TemplatePull(tGroupName, tSlug)
	if err != nil {
		return nil, nil, err
	}

	return messages, tarReader, nil
}

func (c *client) TemplateGetInstances(groupName, templateSlug string) ([]sdk.WorkflowTemplateInstance, error) {
	url := fmt.Sprintf("/template/%s/%s/instance", groupName, templateSlug)

	var wtis []sdk.WorkflowTemplateInstance
	if _, err := c.GetJSON(context.Background(), url, &wtis); err != nil {
		return nil, err
	}

	return wtis, nil
}
