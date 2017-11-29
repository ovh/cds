package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (g *githubClient) CreateHook(repo string, hook *sdk.VCSHook) error {
	url := "/repos/" + repo + "/hooks"

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
	res, err := g.post(url, "application/json", bytes.NewBuffer(b), false)
	if err != nil {
		log.Warning("github.CreateHook> Error %s", err)
		return err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode != 201 {
		err := fmt.Errorf("Unable to create webhook on github. Status code : %d - Body: %s", res.StatusCode, body)
		log.Warning("github.CreateHook> %s", err)
		log.Warning("github.CreateHook> Sent data %s", b)
		return err
	}

	var webhook Webhook
	if err := json.Unmarshal(body, &webhook); err != nil {
		log.Warning("github.CreateHook> Cannot unmarshal response")
		return err
	}
	hook.ID = fmt.Sprintf("%d", webhook.ID)
	return nil
}
func (g *githubClient) GetHook(repo, id string) (sdk.VCSHook, error) {
	return sdk.VCSHook{}, fmt.Errorf("Not yet implemented")
}
func (g *githubClient) UpdateHook(repo, id string, hook sdk.VCSHook) error {
	return fmt.Errorf("Not yet implemented")
}
func (g *githubClient) DeleteHook(repo string, hook sdk.VCSHook) error {
	return g.delete("/repos/" + repo + "/hooks/" + hook.ID)
}
