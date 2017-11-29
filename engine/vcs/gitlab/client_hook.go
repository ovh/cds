package gitlab

import (
	"fmt"

	"github.com/xanzy/go-gitlab"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (g *gitlabClient) GetHook(repo, id string) (sdk.VCSHook, error) {
	return sdk.VCSHook{}, fmt.Errorf("Not yet implemented")
}
func (g *gitlabClient) UpdateHook(repo, id string, hook sdk.VCSHook) error {
	return fmt.Errorf("Not yet implemented")
}

//CreateHook enables the defaut HTTP POST Hook in Gitlab
func (c *gitlabClient) CreateHook(repo string, hook sdk.VCSHook) error {
	t := true
	f := false

	log.Warning(">>%s", hook.URL)

	opt := gitlab.AddProjectHookOptions{
		URL:                   &hook.URL,
		PushEvents:            &t,
		MergeRequestsEvents:   &f,
		TagPushEvents:         &f,
		EnableSSLVerification: &f,
	}

	log.Debug("GitlabClient.CreateHook: %s %s\n", repo, *opt.URL)
	if _, _, err := c.client.Projects.AddProjectHook(repo, &opt); err != nil {
		return err
	}

	return nil
}

//DeleteHook disables the defaut HTTP POST Hook in Gitlab
func (c *gitlabClient) DeleteHook(repo string, hook sdk.VCSHook) error {

	hooks, _, err := c.client.Projects.ListProjectHooks(repo, nil)
	if err != nil {
		return err
	}

	log.Debug("GitlabClient.DeleteHook: Got '%s'", hook.URL)
	log.Debug("GitlabClient.DeleteHook: Want '%s'", hook.URL)
	for _, h := range hooks {
		log.Debug("GitlabClient.DeleteHook: Found '%s'", h.URL)
		if h.URL == hook.URL {
			_, err = c.client.Projects.DeleteProjectHook(repo, h.ID)
			return err
		}
	}

	return fmt.Errorf("not found")
}
