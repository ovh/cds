package bitbucketcloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (client *bitbucketcloudClient) CreateHook(ctx context.Context, repo string, hook *sdk.VCSHook) error {
	url := fmt.Sprintf("/repositories/%s/hooks", repo)
	if client.proxyURL != "" {
		lastIndexSlash := strings.LastIndex(hook.URL, "/")
		if client.proxyURL[len(client.proxyURL)-1] == '/' {
			lastIndexSlash++
		}
		hook.URL = client.proxyURL + hook.URL[lastIndexSlash:]
	}

	r := WebhookCreate{
		Description: "CDS webhook - " + hook.Name,
		Active:      true,
		Events:      []string{"repo:push"},
		URL:         hook.URL,
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
		err := fmt.Errorf("Unable to create webhook on bitbucketcloud. Status code : %d - Body: %s. ", res.StatusCode, body)
		return sdk.WrapError(err, "bitbucketcloud.CreateHook. Data : %s", b)
	}

	var webhook Webhook
	if err := json.Unmarshal(body, &webhook); err != nil {
		return sdk.WrapError(err, "Cannot unmarshal response")
	}
	hook.ID = webhook.UUID
	return nil
}

func (client *bitbucketcloudClient) getHooks(ctx context.Context, fullname string) ([]Webhook, error) {
	var webhooks []Webhook
	path := fmt.Sprintf("/repositories/%s/hooks", fullname)
	params := url.Values{}
	params.Set("pagelen", "100")
	nextPage := 1
	for {
		if nextPage != 1 {
			params.Set("page", fmt.Sprintf("%d", nextPage))
		}

		var response Webhooks
		if err := client.do(ctx, "GET", "core", path, params, nil, &response); err != nil {
			return nil, sdk.WrapError(err, "Unable to get repos")
		}
		if cap(webhooks) == 0 {
			webhooks = make([]Webhook, 0, response.Size)
		}

		webhooks = append(webhooks, response.Values...)

		if response.Next == "" {
			break
		} else {
			nextPage++
		}
	}
	return webhooks, nil
}

func (client *bitbucketcloudClient) GetHook(ctx context.Context, fullname, webhookURL string) (sdk.VCSHook, error) {
	var hook sdk.VCSHook
	hooks, err := client.getHooks(ctx, fullname)
	if err != nil {
		return hook, sdk.WithStack(err)
	}

	for _, h := range hooks {
		log.Info("hooks: %s (expecting: %s)", h.URL, webhookURL)
		if h.URL == webhookURL {
			return sdk.VCSHook{
				Name:   h.Description,
				Events: h.Events,
				URL:    h.URL,
				ID:     h.UUID,
			}, nil
		}
	}

	return hook, sdk.WithStack(sdk.ErrNotFound)
}
func (client *bitbucketcloudClient) UpdateHook(ctx context.Context, repo, id string, hook sdk.VCSHook) error {
	url := fmt.Sprintf("/repositories/%s/hooks", repo)
	if client.proxyURL != "" {
		lastIndexSlash := strings.LastIndex(hook.URL, "/")
		if client.proxyURL[len(client.proxyURL)-1] == '/' {
			lastIndexSlash++
		}
		hook.URL = client.proxyURL + hook.URL[lastIndexSlash:]
	}

	r := WebhookCreate{
		Description: "CDS webhook - " + hook.Name,
		Active:      true,
		Events:      []string{"repo:push"},
		URL:         hook.URL,
	}
	b, err := json.Marshal(r)
	if err != nil {
		return sdk.WrapError(err, "Cannot marshal body %+v", r)
	}
	res, err := client.put(url, "application/json", bytes.NewBuffer(b), nil)
	if err != nil {
		return sdk.WrapError(err, "bitbucketcloud.UpdateHook")
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return sdk.WrapError(err, "ReadAll")
	}
	if res.StatusCode != 200 {
		err := fmt.Errorf("Unable to update webhook on bitbucketcloud. Status code : %d - Body: %s. ", res.StatusCode, body)
		return sdk.WrapError(err, "bitbucketcloud.CreateHook. Data : %s", b)
	}

	return nil
}

func (client *bitbucketcloudClient) DeleteHook(ctx context.Context, repo string, hook sdk.VCSHook) error {
	return client.delete(fmt.Sprintf("/repositories/%s/hooks/%s", repo, hook.ID))
}
