package cdsclient

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (c *client) TemplateGenerateWorkflowFromFile(ctx context.Context, req sdk.V2WorkflowTemplateGenerateRequest) (*sdk.V2WorkflowTemplateGenerateResponse, error) {
	var resp sdk.V2WorkflowTemplateGenerateResponse
	if _, err := c.PostJSON(context.Background(), "/v2/template/workflow/generate", req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
