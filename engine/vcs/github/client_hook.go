package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (g *githubClient) CreateHook(ctx context.Context, repo string, hook *sdk.VCSHook) error {
	url := "/repos/" + repo + "/hooks"
	if g.proxyURL != "" {
		lastIndexSlash := strings.LastIndex(hook.URL, "/")
		if g.proxyURL[len(g.proxyURL)-1] == '/' {
			lastIndexSlash++
		}
		hook.URL = g.proxyURL + hook.URL[lastIndexSlash:]
	}
	if len(hook.Events) == 0 {
		hook.Events = sdk.GitHubEventsDefault
	}

	r := WebhookCreate{
		Name:   "web",
		Active: true,
		Events: hook.Events,
		Config: WebHookConfig{
			URL:         hook.URL,
			ContentType: "json",
		},
	}
	b, err := json.Marshal(r)
	if err != nil {
		return sdk.WrapError(err, "Cannot marshal body %+v", r)
	}
	res, err := g.post(url, "application/json", bytes.NewBuffer(b), nil)
	if err != nil {
		return sdk.WrapError(err, "github.CreateHook")
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return sdk.WrapError(err, "ReadAll")
	}
	if res.StatusCode != 201 {
		if strings.Contains(string(body), "Hook already exists on this repository") {
			return nil
		}
		err := fmt.Errorf("Unable to create webhook on github. Status code : %d - Body: %s. ", res.StatusCode, body)
		return sdk.WrapError(err, "github.CreateHook. Data : %s", b)
	}

	if err := json.Unmarshal(body, &r); err != nil {
		return sdk.WrapError(err, "Cannot unmarshal response")
	}
	hook.ID = fmt.Sprintf("%d", r.ID)
	return nil
}

func (g *githubClient) UpdateHook(ctx context.Context, repo string, hook *sdk.VCSHook) error {
	githubWebHook, err := g.getHookByID(ctx, repo, hook.ID)
	if err != nil {
		return err
	}

	url := "/repos/" + repo + "/hooks/" + hook.ID
	if g.proxyURL != "" {
		lastIndexSlash := strings.LastIndex(hook.URL, "/")
		if g.proxyURL[len(g.proxyURL)-1] == '/' {
			lastIndexSlash++
		}
		hook.URL = g.proxyURL + hook.URL[lastIndexSlash:]
	}
	if len(hook.Events) == 0 {
		hook.Events = sdk.GitHubEventsDefault
	}

	githubWebHook.Events = hook.Events
	b, err := json.Marshal(githubWebHook)
	if err != nil {
		return sdk.WrapError(err, "Cannot marshal body %+v", githubWebHook)
	}
	res, err := g.patch(url, "application/json", bytes.NewBuffer(b), nil)
	if err != nil {
		return sdk.WrapError(err, "github.UpdateHook")
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return sdk.WrapError(err, "ReadAll")
	}
	if res.StatusCode != 200 {
		err := fmt.Errorf("Unable to update webhook on github. Status code : %d - Body: %s. ", res.StatusCode, body)
		return sdk.WrapError(err, "github.Update. Data : %s", b)
	}
	return nil
}

func (g *githubClient) getHooks(ctx context.Context, fullname string) ([]Webhook, error) {
	var webhooks = []Webhook{}
	cacheKey := cache.Key("vcs", "github", "hooks", g.OAuthToken, "/repos/"+fullname+"/hooks")
	opts := []getArgFunc{withETag}

	var nextPage = "/repos/" + fullname + "/hooks"
	for nextPage != "" {
		if ctx.Err() != nil {
			break
		}

		status, body, headers, err := g.get(ctx, nextPage, opts...)
		if err != nil {
			log.Warning(ctx, "githubClient.PullRequests> Error %s", err)
			return nil, sdk.WithStack(err)
		}
		if status >= 400 {
			return nil, sdk.NewError(sdk.ErrUnknownError, errorAPI(body))
		}
		opts[0] = withETag
		nextHooks := []Webhook{}

		//Github may return 304 status because we are using conditional request with ETag based headers
		if status == http.StatusNotModified {
			//If repos aren't updated, lets get them from cache
			find, err := g.Cache.Get(cacheKey, &webhooks)
			if err != nil {
				log.Error(ctx, "cannot get from cache %s: %v", cacheKey, err)
			}
			if !find {
				opts[0] = withoutETag
				log.Error(ctx, "Unable to get getHooks (%s) from the cache", strings.ReplaceAll(cacheKey, g.OAuthToken, ""))
				continue
			}
			break
		} else {
			if err := json.Unmarshal(body, &nextHooks); err != nil {
				log.Warning(ctx, "githubClient.getHooks> Unable to parse github hooks: %s", err)
				return nil, err
			}
		}
		webhooks = append(webhooks, nextHooks...)
		nextPage = getNextPage(headers)
	}

	//Put the body on cache for one hour and one minute
	if err := g.Cache.SetWithTTL(cacheKey, webhooks, 61*60); err != nil {
		log.Error(ctx, "cannot SetWithTTL: %s: %v", cacheKey, err)
	}
	return webhooks, nil
}

func (g *githubClient) GetHook(ctx context.Context, fullname, webhookURL string) (sdk.VCSHook, error) {
	hooks, err := g.getHooks(ctx, fullname)
	if err != nil {
		return sdk.VCSHook{}, sdk.WithStack(err)
	}

	for _, h := range hooks {
		log.Info(ctx, "hooks: %s (expecting: %s)", h.Config.URL, webhookURL)
		if h.Config.URL == webhookURL {
			return sdk.VCSHook{
				Name:        h.Name,
				Events:      h.Events,
				URL:         h.Config.URL,
				ContentType: h.Config.ContentType,
				ID:          strconv.Itoa(h.ID),
			}, nil
		}
	}

	return sdk.VCSHook{}, sdk.WithStack(sdk.ErrNotFound)
}

func (g *githubClient) getHookByID(ctx context.Context, fullname, id string) (Webhook, error) {
	var webhook Webhook
	url := "/repos/" + fullname + "/hooks/" + id
	cacheKey := cache.Key("vcs", "github", "hooks", id, g.OAuthToken, "/repos/"+fullname+"/hooks/"+id)
	opts := []getArgFunc{withETag}

	status, body, _, err := g.get(ctx, url, opts...)
	if err != nil {
		log.Warning(ctx, "githubClient.PullRequests> Error %v", err)
		return webhook, sdk.WithStack(err)
	}
	if status >= 400 {
		return webhook, sdk.NewError(sdk.ErrUnknownError, errorAPI(body))
	}
	opts[0] = withETag

	//Github may return 304 status because we are using conditional request with ETag based headers
	if status == http.StatusNotModified {
		//If repos aren't updated, lets get them from cache
		find, err := g.Cache.Get(cacheKey, &webhook)
		if err != nil {
			log.Error(ctx, "cannot get from cache %s: %v", cacheKey, err)
		}
		if !find {
			return webhook, sdk.WithStack(fmt.Errorf("unable to get getHooks (%s) from the cache", strings.ReplaceAll(cacheKey, g.OAuthToken, "")))
		}
	} else {
		if err := json.Unmarshal(body, &webhook); err != nil {
			log.Warning(ctx, "githubClient.getHookByID> Unable to parse github hook: %v", err)
			return webhook, err
		}
	}

	//Put the body on cache for one hour and one minute
	if err := g.Cache.SetWithTTL(cacheKey, webhook, 61*60); err != nil {
		log.Error(ctx, "cannot SetWithTTL: %s: %v", cacheKey, err)
	}
	return webhook, nil
}

func (g *githubClient) DeleteHook(ctx context.Context, repo string, hook sdk.VCSHook) error {
	return g.delete("/repos/" + repo + "/hooks/" + hook.ID)
}
