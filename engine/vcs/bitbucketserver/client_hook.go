package bitbucketserver

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
		return nil, sdk.WrapError(err, "unable to get hook config")
	}

	return resp.Values, nil
}

func (b *bitbucketClient) getHookByID(ctx context.Context, repo string, webHookID string) (WebHook, error) {
	var resp WebHook

	project, slug, err := getRepo(repo)
	if err != nil {
		return resp, sdk.WithStack(err)
	}

	getPath := fmt.Sprintf("/projects/%s/repos/%s/webhooks/%s", project, slug, webHookID)
	if err := b.do(ctx, "GET", "core", getPath, nil, nil, &resp, nil); err != nil {
		return resp, sdk.WrapError(err, "unable to get hook %s", webHookID)
	}

	return resp, nil
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
				ID:          fmt.Sprintf("%d", h.ID),
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

	if len(hook.Events) == 0 {
		hook.Events = []string{"repo:refs_changed"}
	}

	url := fmt.Sprintf("/projects/%s/repos/%s/webhooks", project, slug)
	request := WebHook{
		URL:           hook.URL,
		Events:        hook.Events,
		Active:        true,
		Name:          repo,
		Configuration: make(map[string]string),
	}

	values, err := json.Marshal(&request)
	if err != nil {
		return sdk.WithStack(err)
	}
	if err := b.do(ctx, "POST", "core", url, nil, values, &request, nil); err != nil {
		return sdk.WrapError(err, "unable to get enable webhook")
	}
	hook.ID = fmt.Sprintf("%d", request.ID)
	return nil
}

func (b *bitbucketClient) UpdateHook(ctx context.Context, repo string, hook *sdk.VCSHook) error {
	project, slug, err := getRepo(repo)
	if err != nil {
		return err
	}

	// Get hooks
	bitbucketHook, err := b.getHookByID(ctx, repo, hook.ID)
	if err != nil {
		return err
	}

	if len(hook.Events) == 0 {
		hook.Events = []string{"repo:refs_changed"}
	}

	bitbucketHook.Events = hook.Events

	url := fmt.Sprintf("/projects/%s/repos/%s/webhooks/%d", project, slug, bitbucketHook.ID)

	values, err := json.Marshal(&bitbucketHook)
	if err != nil {
		return err
	}
	if err := b.do(ctx, "PUT", "core", url, nil, values, &bitbucketHook, nil); err != nil {
		return sdk.WrapError(err, "unable to update webhook")
	}
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
			return sdk.WrapError(err, "unable to get enable webhook")
		}
	}
	return nil
}
