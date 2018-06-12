package gitlab

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/xanzy/go-gitlab"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (c *gitlabClient) GetHook(repo, id string) (sdk.VCSHook, error) {
	return sdk.VCSHook{}, fmt.Errorf("Not yet implemented")
}
func (c *gitlabClient) UpdateHook(repo, id string, hook sdk.VCSHook) error {
	return fmt.Errorf("Not yet implemented")
}

//CreateHook enables the defaut HTTP POST Hook in Gitlab
func (c *gitlabClient) CreateHook(repo string, hook *sdk.VCSHook) error {
	t := true
	f := false

	var url string
	if !hook.Workflow {
		var err error
		url, err = buildGitlabURL(hook.URL)
		if err != nil {
			return sdk.WrapError(err, "GitlabClient.CreateHook> buildGitlabURL")
		}
	} else {
		url = hook.URL
	}

	opt := gitlab.AddProjectHookOptions{
		URL:                   &url,
		PushEvents:            &t,
		MergeRequestsEvents:   &f,
		TagPushEvents:         &f,
		EnableSSLVerification: &f,
	}

	log.Debug("GitlabClient.CreateHook: %s %s\n", repo, *opt.URL)
	ph, resp, err := c.client.Projects.AddProjectHook(repo, &opt)
	if err != nil {
		return sdk.WrapError(err, "GitlabClient.CreateHook> AddProjectHook")
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("GitlabClient.CreateHook> Cannot create hook. Http %d, Repo %s, hook %+v", resp.StatusCode, repo, opt)
	}
	hook.ID = fmt.Sprintf("%d", ph.ID)
	return nil
}

//DeleteHook disables the defaut HTTP POST Hook in Gitlab
func (c *gitlabClient) DeleteHook(repo string, hook sdk.VCSHook) error {
	if !hook.Workflow {
		if c.proxyURL != "" {
			lastIndexSlash := strings.LastIndex(hook.URL, "/")
			if c.proxyURL[len(c.proxyURL)-1] == '/' {
				lastIndexSlash++
			}
			hook.URL = c.proxyURL + hook.URL[lastIndexSlash:]
		}

		var url string
		var err error
		url, err = buildGitlabURL(hook.URL)
		if err != nil {
			return sdk.WrapError(err, "GitlabClient.DeleteHook> buildGitlabURL")
		}

		hooks, _, err := c.client.Projects.ListProjectHooks(repo, nil)
		if err != nil {
			return sdk.WrapError(err, "GitlabClient.DeleteHook> ListProjectHooks")
		}

		log.Debug("GitlabClient.DeleteHook: Got '%s'", url)
		for _, h := range hooks {
			log.Debug("GitlabClient.DeleteHook: Found '%s'", h.URL)
			if h.URL == url {
				_, err = c.client.Projects.DeleteProjectHook(repo, h.ID)
				return sdk.WrapError(err, "GitlabClient.DeleteHook> DeleteProjectHook")
			}
		}
		return fmt.Errorf("GitlabClient.DeleteHook> not found")
	}
	hookID, errI := strconv.Atoi(hook.ID)
	if errI != nil {
		return sdk.WrapError(sdk.ErrInvalidID, "GitlabClient.DeleteHook > Wrong gitlab webhook ID: %s", hook.ID)
	}
	res, err := c.client.Projects.DeleteProjectHook(repo, hookID)
	if err != nil && res.StatusCode != 404 {
		return sdk.WrapError(sdk.ErrInvalidID, "GitlabClient.DeleteHook > Cannot delete gitlab hook %s on project %s. Get code: %s", hook.ID, repo, res.StatusCode)
	}
	return nil
}

func buildGitlabURL(givenURL string) (string, error) {
	u, err := url.Parse(givenURL)
	if err != nil {
		return "", err
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s://%s/%s?uid=%s", u.Scheme, u.Host, u.Path, q.Get("uid"))

	for k := range q {
		if k != "uid" && !strings.Contains(q.Get(k), "{") {
			url = fmt.Sprintf("%s&%s=%s", url, k, q.Get(k))
		}
	}

	return url, nil
}
