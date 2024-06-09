package gitlab

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/rockbears/log"
	"github.com/xanzy/go-gitlab"

	"github.com/ovh/cds/sdk"
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

// CreateHook enables the default HTTP POST Hook in Gitlab
func (c *gitlabClient) CreateHook(ctx context.Context, repo string, hook *sdk.VCSHook) error {
	url := c.buildUrlWithProxy(hook.URL)

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

	log.Debug(ctx, "GitlabClient.CreateHook: %s %s\n", repo, *opt.URL)
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

// UpdateHook updates a gitlab webhook
func (c *gitlabClient) UpdateHook(ctx context.Context, repo string, hook *sdk.VCSHook) error {
	gitlabHook, err := c.GetHookByID(ctx, repo, hook.ID)
	if err != nil {
		return err
	}
	url := c.buildUrlWithProxy(hook.URL)

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
		URL:                      &url,
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

	log.Debug(ctx, "GitlabClient.UpdateHook: %s %s", repo, *opt.URL)
	_, resp, err := c.client.Projects.EditProjectHook(repo, gitlabHook.ID, &opt)
	if err != nil {
		return sdk.WrapError(err, "cannot update gitlab project hook %s", hook.ID)
	}
	if resp.StatusCode >= 400 {
		return sdk.WithStack(fmt.Errorf("cannot update hook. Http %d, Repo %s, hook %+v", resp.StatusCode, repo, opt))
	}
	return nil
}

// DeleteHook disables the default HTTP POST Hook in Gitlab
func (c *gitlabClient) DeleteHook(ctx context.Context, repo string, hook sdk.VCSHook) error {
	c.buildUrlWithProxy(hook.URL)

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

func (c *gitlabClient) buildUrlWithProxy(hookURL string) string {
	if c.proxyURL != "" {
		lastIndexSlash := strings.LastIndex(hookURL, "/")
		if c.proxyURL[len(c.proxyURL)-1] == '/' {
			lastIndexSlash++
		}
		return c.proxyURL + hookURL[lastIndexSlash:]
	}

	return hookURL
}
