package gitlab

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/xanzy/go-gitlab"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (c *gitlabClient) GetHook(ctx context.Context, repo, id string) (sdk.VCSHook, error) {
	return sdk.VCSHook{}, fmt.Errorf("Not yet implemented")
}

func (c *gitlabClient) GetHookByID(ctx context.Context, repo, idS string) (*gitlab.ProjectHook, error) {
	id, err := strconv.Atoi(idS)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to parse hook id: %s", idS)
	}
	hook, _, err := c.client.Projects.GetProjectHook(repo, id)
	if err != nil {
		return hook, sdk.WrapError(err, "unable to get hook %d", id)
	}
	return hook, nil
}

//CreateHook enables the defaut HTTP POST Hook in Gitlab
func (c *gitlabClient) CreateHook(ctx context.Context, repo string, hook *sdk.VCSHook) error {
	var url string
	if !hook.Workflow {
		var err error
		url, err = buildGitlabURL(hook.URL)
		if err != nil {
			return err
		}
	} else {
		url = hook.URL
	}

	// if the hook already exists, do not recreate it
	hs, resp, err := c.client.Projects.ListProjectHooks(repo, nil)
	if err != nil {
		return sdk.WrapError(err, "cannot list gitlab project hooks for %s", repo)
	}
	if resp.StatusCode >= 400 {
		return sdk.WithStack(fmt.Errorf("cannot list project hooks. Http %d, Repo %s", resp.StatusCode, repo))
	}
	for i := range hs {
		if hs[i].URL == url {
			return nil
		}
	}

	var pushEvent, mergeRequestEvent, TagPushEvent, issueEvent, noteEvent, wikiPageEvent, pipelineEvent, jobEvent bool
	if len(hook.Events) == 0 {
		hook.Events = sdk.GitlabEventsDefault
	}

	for _, e := range hook.Events {
		switch gitlab.EventType(e) {
		case gitlab.EventTypePush:
			pushEvent = true
		case gitlab.EventTypeTagPush:
			TagPushEvent = true
		case gitlab.EventTypeIssue:
			issueEvent = true
		case gitlab.EventTypeNote:
			noteEvent = true
		case gitlab.EventTypeMergeRequest:
			mergeRequestEvent = true
		case gitlab.EventTypeWikiPage:
			wikiPageEvent = true
		case gitlab.EventTypePipeline:
			pipelineEvent = true
		case "Job Hook": // TODO update gitlab sdk
			jobEvent = true
		}
	}

	f := false
	opt := gitlab.AddProjectHookOptions{
		URL:                   &url,
		PushEvents:            &pushEvent,
		MergeRequestsEvents:   &mergeRequestEvent,
		TagPushEvents:         &TagPushEvent,
		IssuesEvents:          &issueEvent,
		WikiPageEvents:        &wikiPageEvent,
		NoteEvents:            &noteEvent,
		PipelineEvents:        &pipelineEvent,
		JobEvents:             &jobEvent,
		EnableSSLVerification: &f,
	}

	log.Debug("GitlabClient.CreateHook: %s %s\n", repo, *opt.URL)
	ph, resp, err := c.client.Projects.AddProjectHook(repo, &opt)
	if err != nil {
		return sdk.WrapError(err, "cannot create gitlab project hook with url: %s", url)
	}
	if resp.StatusCode >= 400 {
		return sdk.WithStack(fmt.Errorf("cannot create hook. Http %d, Repo %s, hook %+v", resp.StatusCode, repo, opt))
	}
	hook.ID = fmt.Sprintf("%d", ph.ID)
	return nil
}

//UpdateHook updates a gitlab webhook
func (c *gitlabClient) UpdateHook(ctx context.Context, repo string, hook *sdk.VCSHook) error {
	gitlabHook, err := c.GetHookByID(ctx, repo, hook.ID)
	if err != nil {
		return err
	}

	var pushEvent, mergeRequestEvent, TagPushEvent, issueEvent, noteEvent, wikiPageEvent, pipelineEvent, jobEvent bool
	if len(hook.Events) == 0 {
		hook.Events = sdk.GitlabEventsDefault
	}

	for _, e := range hook.Events {
		switch gitlab.EventType(e) {
		case gitlab.EventTypePush:
			pushEvent = true
		case gitlab.EventTypeTagPush:
			TagPushEvent = true
		case gitlab.EventTypeIssue:
			issueEvent = true
		case gitlab.EventTypeNote:
			noteEvent = true
		case gitlab.EventTypeMergeRequest:
			mergeRequestEvent = true
		case gitlab.EventTypeWikiPage:
			wikiPageEvent = true
		case gitlab.EventTypePipeline:
			pipelineEvent = true
		case "Job Hook": // TODO update gitlab sdk
			jobEvent = true
		}
	}

	opt := gitlab.EditProjectHookOptions{
		URL:                      &gitlabHook.URL,
		PushEvents:               &pushEvent,
		MergeRequestsEvents:      &mergeRequestEvent,
		TagPushEvents:            &TagPushEvent,
		IssuesEvents:             &issueEvent,
		WikiPageEvents:           &wikiPageEvent,
		NoteEvents:               &noteEvent,
		PipelineEvents:           &pipelineEvent,
		JobEvents:                &jobEvent,
		EnableSSLVerification:    &gitlabHook.EnableSSLVerification,
		ConfidentialIssuesEvents: &gitlabHook.ConfidentialIssuesEvents,
	}

	log.Debug("GitlabClient.UpdateHook: %s %s", repo, *opt.URL)
	_, resp, err := c.client.Projects.EditProjectHook(repo, gitlabHook.ID, &opt)
	if err != nil {
		return sdk.WrapError(err, "cannot update gitlab project hook %s", hook.ID)
	}
	if resp.StatusCode >= 400 {
		return sdk.WithStack(fmt.Errorf("cannot update hook. Http %d, Repo %s, hook %+v", resp.StatusCode, repo, opt))
	}
	return nil
}

//DeleteHook disables the defaut HTTP POST Hook in Gitlab
func (c *gitlabClient) DeleteHook(ctx context.Context, repo string, hook sdk.VCSHook) error {
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
			return sdk.WrapError(err, "buildGitlabURL")
		}

		hooks, _, err := c.client.Projects.ListProjectHooks(repo, nil)
		if err != nil {
			return sdk.WrapError(err, "ListProjectHooks")
		}

		log.Debug("GitlabClient.DeleteHook: Got '%s'", url)
		for _, h := range hooks {
			log.Debug("GitlabClient.DeleteHook: Found '%s'", h.URL)
			if h.URL == url {
				_, err = c.client.Projects.DeleteProjectHook(repo, h.ID)
				return sdk.WrapError(err, "DeleteProjectHook")
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
		return sdk.WrapError(sdk.ErrInvalidID, "GitlabClient.DeleteHook > Cannot delete gitlab hook %s on project %s. Get code: %d", hook.ID, repo, res.StatusCode)
	}
	return nil
}

func buildGitlabURL(givenURL string) (string, error) {
	u, err := url.Parse(givenURL)
	if err != nil {
		return "", sdk.WithStack(err)
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return "", sdk.WithStack(err)
	}

	url := fmt.Sprintf("%s://%s/%s?uid=%s", u.Scheme, u.Host, u.Path, q.Get("uid"))

	for k := range q {
		if k != "uid" && !strings.Contains(q.Get(k), "{") {
			url = fmt.Sprintf("%s&%s=%s", url, k, q.Get(k))
		}
	}

	return url, nil
}
