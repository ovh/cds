package bitbucketcloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (client *bitbucketcloudClient) CreateHook(ctx context.Context, repo string, hook *sdk.VCSHook) error {
	url := "/repos/" + repo + "/hooks"
	if client.proxyURL != "" {
		lastIndexSlash := strings.LastIndex(hook.URL, "/")
		if client.proxyURL[len(client.proxyURL)-1] == '/' {
			lastIndexSlash++
		}
		hook.URL = client.proxyURL + hook.URL[lastIndexSlash:]
	}

	r := WebhookCreate{
		Name:   "web",
		Active: true,
		Events: []string{"push"},
		Config: WebHookConfig{
			URL:         hook.URL,
			ContentType: "json",
		},
	}
	b, err := json.Marshal(r)
	if err != nil {
		return sdk.WrapError(err, "Cannot marshal body %+v", r)
	}
	res, err := client.post(url, "application/json", bytes.NewBuffer(b), nil)
	if err != nil {
		return sdk.WrapError(err, "bitbucketcloud.CreateHook")
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
		err := fmt.Errorf("Unable to create webhook on bitbucketcloud. Status code : %d - Body: %s. ", res.StatusCode, body)
		return sdk.WrapError(err, "bitbucketcloud.CreateHook. Data : %s", b)
	}

	if err := json.Unmarshal(body, &r); err != nil {
		return sdk.WrapError(err, "Cannot unmarshal response")
	}
	hook.ID = fmt.Sprintf("%d", r.ID)
	return nil
}

func (client *bitbucketcloudClient) getHooks(ctx context.Context, fullname string) ([]Webhook, error) {
	var webhooks = []Webhook{}
	var nextPage = "/repos/" + fullname + "/hooks"
	cacheKey := cache.Key("vcs", "bitbucketcloud", "hooks", client.OAuthToken, "/repos/"+fullname+"/hooks")

	for nextPage != "" {
		status, body, headers, err := client.get(nextPage)
		if err != nil {
			log.Warning("bitbucketcloudClient.PullRequests> Error %s", err)
			return nil, sdk.WithStack(err)
		}
		if status >= 400 {
			return nil, sdk.NewError(sdk.ErrUnknownError, errorAPI(body))
		}
		nextHooks := []Webhook{}

		//bitbucketcloud may return 304 status because we are using conditional request with ETag based headers
		if status == http.StatusNotModified {
			//If repos aren't updated, lets get them from cache
			if !client.Cache.Get(cacheKey, &webhooks) {
				log.Error("Unable to get getHooks (%s) from the cache", strings.ReplaceAll(cacheKey, client.OAuthToken, ""))
				continue
			}
			break
		} else {
			if err := json.Unmarshal(body, &nextHooks); err != nil {
				log.Warning("bitbucketcloudClient.getHooks> Unable to parse bitbucketcloud hooks: %s", err)
				return nil, err
			}
		}
		webhooks = append(webhooks, nextHooks...)
		nextPage = getNextPage(headers)
	}

	//Put the body on cache for one hour and one minute
	client.Cache.SetWithTTL(cacheKey, webhooks, 61*60)
	return webhooks, nil
}

func (client *bitbucketcloudClient) GetHook(ctx context.Context, fullname, webhookURL string) (sdk.VCSHook, error) {
	hooks, err := client.getHooks(ctx, fullname)
	if err != nil {
		return sdk.VCSHook{}, sdk.WithStack(err)
	}

	for _, h := range hooks {
		log.Info("hooks: %s (expecting: %s)", h.Config.URL, webhookURL)
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
func (client *bitbucketcloudClient) UpdateHook(ctx context.Context, repo, id string, hook sdk.VCSHook) error {
	return fmt.Errorf("Not yet implemented")
}
func (client *bitbucketcloudClient) DeleteHook(ctx context.Context, repo string, hook sdk.VCSHook) error {
	return client.delete("/repos/" + repo + "/hooks/" + hook.ID)
}
