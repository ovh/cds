package cdsclient

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ovh/cds/sdk"
)

type WorkerModelTemplateFilter struct {
	Branch string
}

func (c *client) WorkerModelTemplateList(ctx context.Context, projKey string, vcsIdentifier string, repoIdentifier string, filter *WorkerModelTemplateFilter) ([]sdk.WorkerModelTemplate, error) {
	var mods []RequestModifier
	if filter != nil {
		mods = []RequestModifier{
			func(req *http.Request) {
				q := req.URL.Query()
				if filter.Branch != "" {
					q.Add("branch", url.QueryEscape(filter.Branch))
				}
				req.URL.RawQuery = q.Encode()
			},
		}
	}
	var modelTmpls []sdk.WorkerModelTemplate
	uri := fmt.Sprintf("/v2/project/%s/vcs/%s/repository/%s/workermodel/template", projKey, url.PathEscape(vcsIdentifier), url.PathEscape(repoIdentifier))
	if _, err := c.GetJSON(ctx, uri, &modelTmpls, mods...); err != nil {
		return nil, err
	}
	return modelTmpls, nil
}
