package gitlab

import (
	"fmt"
	"net/url"
	"strings"

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

	url, err := buildGitlabURL(hook)
	if err != nil {
		return err
	}
	log.Warning(">>>%s", url)
	opt := gitlab.AddProjectHookOptions{
		URL:                   &url,
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

	url, err := buildGitlabURL(hook)
	if err != nil {
		return err
	}

	hooks, _, err := c.client.Projects.ListProjectHooks(repo, nil)
	if err != nil {
		return err
	}

	log.Debug("GitlabClient.DeleteHook: Got '%s'", url)
	log.Debug("GitlabClient.DeleteHook: Want '%s'", url)
	for _, h := range hooks {
		log.Debug("GitlabClient.DeleteHook: Found '%s'", h.URL)
		if h.URL == url {
			_, err = c.client.Projects.DeleteProjectHook(repo, h.ID)
			return err
		}
	}

	return fmt.Errorf("not found")
}

func buildGitlabURL(h sdk.VCSHook) (string, error) {

	u, err := url.Parse(h.URL)
	if err != nil {
		return "", err
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s://%s/%s?uid=%s", u.Scheme, u.Host, u.Path, h.UUID)

	for k, _ := range q {
		if k != "uid" && !strings.Contains(q.Get(k), "{") {
			url = fmt.Sprintf("%s&%s=%s", url, k, q.Get(k))
		}
	}

	return url, nil
}
