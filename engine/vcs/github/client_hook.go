package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/ovh/cds/sdk"
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
		return sdk.WrapError(err, "github.CreateHook > Cannot marshal body %+v", r)
	}
	res, err := g.post(url, "application/json", bytes.NewBuffer(b), nil)
	if err != nil {
		return sdk.WrapError(err, "github.CreateHook")
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return sdk.WrapError(err, "github.CreateHook> ReadAll")
	}
	if res.StatusCode != 201 {
		if strings.Contains(string(body), "Hook already exists on this repository") {
			return nil
		}
		err := fmt.Errorf("Unable to create webhook on github. Status code : %d - Body: %s. ", res.StatusCode, body)
		return sdk.WrapError(err, "github.CreateHook. Data : %s", b)
	}

	if err := json.Unmarshal(body, &r); err != nil {
		return sdk.WrapError(err, "github.CreateHook> Cannot unmarshal response")
	}
	hook.ID = fmt.Sprintf("%d", r.ID)
	return nil
}
func (g *githubClient) GetHook(ctx context.Context, repo, id string) (sdk.VCSHook, error) {
	return sdk.VCSHook{}, fmt.Errorf("Not yet implemented")
}
func (g *githubClient) UpdateHook(ctx context.Context, repo, id string, hook sdk.VCSHook) error {
	return fmt.Errorf("Not yet implemented")
}
func (g *githubClient) DeleteHook(ctx context.Context, repo string, hook sdk.VCSHook) error {
	return g.delete("/repos/" + repo + "/hooks/" + hook.ID)
}
