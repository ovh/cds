package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ovh/cds/sdk"
)

func (b *bitbucketClient) getHooks(ctx context.Context, repo string) ([]WebHook, error) {
	project, slug, err := getRepo(repo)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	var resp GetWebHooksResponse
	getPath := fmt.Sprintf("/projects/%s/repos/%s/webhooks", project, slug)
	if err := b.do(ctx, "GET", "core", getPath, nil, nil, &resp, nil); err != nil {
		return nil, sdk.WrapError(err, "Unable to get hook config")
	}

	return resp.Values, nil
}

func (b *bitbucketClient) GetHook(ctx context.Context, repo, url string) (sdk.VCSHook, error) {
	whooks, err := b.getHooks(ctx, repo)
	if err != nil {
		return sdk.VCSHook{}, err
	}

	for _, h := range whooks {
		if h.URL == url {
			return sdk.VCSHook{
				Disable:     h.Active,
				Events:      h.Events,
				ID:          fmt.Sprintf("%s", h.ID),
				InsecureSSL: false,
				Method:      http.MethodPost,
				Name:        h.Name,
				URL:         h.URL,
				Body:        "",
			}, nil
		}
	}

	return sdk.VCSHook{}, sdk.ErrHookNotFound
}

func (b *bitbucketClient) CreateHook(ctx context.Context, repo string, hook *sdk.VCSHook) error {
	project, slug, err := getRepo(repo)
	if err != nil {
		return err
	}

	// Get hooks
	hooks, err := b.getHooks(ctx, repo)
	if err != nil {
		return err
	}

	for _, h := range hooks {
		if h.URL == hook.URL {
			return nil
		}
	}

	url := fmt.Sprintf("/projects/%s/repos/%s/webhooks", project, slug)
	request := WebHook{
		URL:           hook.URL,
		Events:        []string{"repo:refs_changed"},
		Active:        true,
		Name:          repo,
		Configuration: make(map[string]string),
	}

	values, err := json.Marshal(&request)
	if err != nil {
		return err
	}
	if err := b.do(ctx, "POST", "core", url, nil, values, &request, nil); err != nil {
		return sdk.WrapError(err, "Unable to get enable webhook")
	}
	hook.ID = fmt.Sprintf("%d", request.ID)
	return nil
}

func (b *bitbucketClient) DeleteHook(ctx context.Context, repo string, hook sdk.VCSHook) error {
	project, slug, err := getRepo(repo)
	if err != nil {
		return sdk.WithStack(err)
	}

	url := fmt.Sprintf("/projects/%s/repos/%s/webhooks/%s", project, slug, hook.ID)
	if err := b.do(ctx, "DELETE", "core", url, nil, nil, nil, nil); err != nil {
		if !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return sdk.WrapError(err, "Unable to get enable webhook")
		}
	}
	return nil
}
