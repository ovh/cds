package repogitlab

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/xanzy/go-gitlab"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// GitlabClient implements RepositoriesManagerClient interface
type GitlabClient struct {
	client *gitlab.Client
}

func NewGitlabClient(URL, token string) (*GitlabClient, error) {
	c := GitlabClient{}

	url := URL + "/api/v4"
	log.Debug("New gitlab client, ursing URL'%s' and token '%s'\n", url, token)
	c.client = gitlab.NewOAuthClient(nil, token)
	c.client.SetBaseURL(url)

	return &c, nil
}

//Repos returns the list of accessible repositories
func (c *GitlabClient) Repos() ([]sdk.VCSRepo, error) {

	var repos []sdk.VCSRepo

	pp := 1000
	opts := &gitlab.ListProjectsOptions{}
	opts.PerPage = pp

	projects, resp, err := c.client.Projects.ListProjects(opts)
	if err != nil {
		return nil, err
	}

	for _, p := range projects {
		r := sdk.VCSRepo{
			ID:           fmt.Sprintf("%d", p.ID),
			Name:         p.NameWithNamespace,
			Slug:         p.PathWithNamespace,
			Fullname:     p.PathWithNamespace,
			URL:          p.WebURL,
			HTTPCloneURL: p.HTTPURLToRepo,
			SSHCloneURL:  p.SSHURLToRepo,
		}
		repos = append(repos, r)
	}

	for resp.NextPage != 0 {
		opts.Page = resp.NextPage

		projects, resp, err = c.client.Projects.ListProjects(opts)
		if err != nil {
			return nil, err
		}

		for _, p := range projects {
			r := sdk.VCSRepo{
				ID:           fmt.Sprintf("%d", p.ID),
				Name:         p.NameWithNamespace,
				Slug:         p.PathWithNamespace,
				Fullname:     p.PathWithNamespace,
				URL:          p.WebURL,
				HTTPCloneURL: p.HTTPURLToRepo,
				SSHCloneURL:  p.SSHURLToRepo,
			}
			repos = append(repos, r)
		}
	}

	return repos, nil
}

//RepoByFullname returns the repo from its fullname
func (c *GitlabClient) RepoByFullname(fullname string) (sdk.VCSRepo, error) {
	repo := sdk.VCSRepo{}

	p, _, err := c.client.Projects.GetProject(fullname)
	if err != nil {
		return repo, err
	}

	repo.ID = fmt.Sprintf("%d", p.ID)
	repo.Name = p.NameWithNamespace
	repo.Slug = p.Name
	repo.Fullname = p.Path
	repo.URL = p.WebURL
	repo.HTTPCloneURL = p.HTTPURLToRepo
	repo.SSHCloneURL = p.SSHURLToRepo

	return repo, nil
}

func (c *GitlabClient) PullRequests(string) ([]sdk.VCSPullRequest, error) {
	return []sdk.VCSPullRequest{}, nil
}

//Branches retrieves the branches
func (c *GitlabClient) Branches(fullname string) ([]sdk.VCSBranch, error) {

	branches, _, err := c.client.Branches.ListBranches(fullname, nil)
	if err != nil {
		return nil, err
	}

	var brs []sdk.VCSBranch
	for _, b := range branches {
		br := sdk.VCSBranch{
			ID:           b.Name,
			DisplayID:    b.Name,
			LatestCommit: b.Commit.ID,
			Default:      false,
			Parents:      nil,
		}
		brs = append(brs, br)
	}

	return brs, nil
}

//Branch retrieves the branch
func (c *GitlabClient) Branch(fullname, branchName string) (*sdk.VCSBranch, error) {

	b, _, err := c.client.Branches.GetBranch(fullname, branchName)
	if err != nil {
		return nil, err
	}

	br := &sdk.VCSBranch{
		ID:           b.Name,
		DisplayID:    b.Name,
		LatestCommit: b.Commit.ID,
		Default:      false,
		Parents:      nil,
	}

	return br, nil
}

//Commits returns commit data from a given starting commit, between two commits
//The commits may be identified by branch or tag name or by hash.
func (c *GitlabClient) Commits(repo, branch, since, until string) ([]sdk.VCSCommit, error) {
	// Gitlab commit listing only allow 'since' and 'until' parameter as dates
	// Need to fetch commit date, then use it to filter

	opt := &gitlab.ListCommitsOptions{
		RefName: &branch,
	}

	commit, err := c.Commit(repo, since)
	if err == nil {
		opt.Since = time.Unix(commit.Timestamp, 0)
	}

	commit, err = c.Commit(repo, until)
	if err == nil {
		opt.Since = time.Unix(commit.Timestamp, 0)
	}

	commits, _, err := c.client.Commits.ListCommits(repo, opt)
	if err != nil {
		return nil, err
	}

	var vcscommits []sdk.VCSCommit
	for _, c := range commits {
		vcsc := sdk.VCSCommit{
			Hash: c.ID,
			Author: sdk.VCSAuthor{
				Name:        c.AuthorName,
				DisplayName: c.AuthorName,
				Email:       c.AuthorEmail,
			},
			Timestamp: c.CommittedDate.Unix(),
			Message:   c.Message,
		}

		vcscommits = append(vcscommits, vcsc)
	}

	return vcscommits, nil
}

//Commit retrieves a specific according to a hash
func (c *GitlabClient) Commit(repo, hash string) (sdk.VCSCommit, error) {
	commit := sdk.VCSCommit{}

	gc, _, err := c.client.Commits.GetCommit(repo, hash)
	if err != nil {
		return commit, err
	}

	commit.Hash = hash
	commit.Author = sdk.VCSAuthor{
		Name:        gc.AuthorName,
		DisplayName: gc.AuthorName,
		Email:       gc.AuthorEmail,
	}
	commit.Timestamp = gc.CommittedDate.Unix()
	commit.Message = gc.Message

	return commit, nil
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

	for k, _ := range q {
		if k != "uid" && !strings.Contains(q.Get(k), "{") {
			url = fmt.Sprintf("%s&%s=%s", url, k, q.Get(k))
		}
	}

	return url, nil
}

//CreateHook enables the defaut HTTP POST Hook in Gitlab
func (c *GitlabClient) CreateHook(repo, givenURL string) error {
	t := true
	f := false

	url, err := buildGitlabURL(givenURL)
	if err != nil {
		return err
	}

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
func (c *GitlabClient) DeleteHook(repo, givenURL string) error {

	url, err := buildGitlabURL(givenURL)
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

func getGitlabStateFromStatus(s sdk.Status) gitlab.BuildState {
	switch s {
	case sdk.StatusWaiting:
		return gitlab.Pending
	case sdk.StatusChecking:
		return gitlab.Pending
	case sdk.StatusBuilding:
		return gitlab.Running
	case sdk.StatusSuccess:
		return gitlab.Success
	case sdk.StatusFail:
		return gitlab.Failed
	case sdk.StatusDisabled:
		return gitlab.Canceled
	case sdk.StatusNeverBuilt:
		return gitlab.Canceled
	case sdk.StatusUnknown:
		return gitlab.Failed
	case sdk.StatusSkipped:
		return gitlab.Canceled
	}

	return gitlab.Failed
}

//SetStatus set build status on Gitlab
func (c *GitlabClient) SetStatus(event sdk.Event) error {
	var eventpb sdk.EventPipelineBuild
	if event.EventType != fmt.Sprintf("%T", sdk.EventPipelineBuild{}) {
		return nil
	}

	if err := mapstructure.Decode(event.Payload, &eventpb); err != nil {
		return err
	}

	log.Debug("Process event:%+v", event)

	cdsProject := eventpb.ProjectKey
	cdsApplication := eventpb.ApplicationName
	cdsPipelineName := eventpb.PipelineName
	cdsBuildNumber := eventpb.BuildNumber
	cdsEnvironmentName := eventpb.EnvironmentName

	key := fmt.Sprintf("%s-%s-%s",
		cdsProject,
		cdsApplication,
		cdsPipelineName,
	)

	url := fmt.Sprintf("%s/project/%s/application/%s/pipeline/%s/build/%d?envName=%s",
		uiURL,
		cdsProject,
		cdsApplication,
		cdsPipelineName,
		cdsBuildNumber,
		url.QueryEscape(cdsEnvironmentName),
	)

	desc := fmt.Sprintf("Build #%d %s", eventpb.BuildNumber, key)

	cds := "CDS"
	opt := &gitlab.SetCommitStatusOptions{
		Name:        &cds,
		Context:     &cds,
		State:       getGitlabStateFromStatus(eventpb.Status),
		Ref:         &eventpb.BranchName,
		TargetURL:   &url,
		Description: &desc,
	}

	if _, _, err := c.client.Commits.SetCommitStatus(eventpb.RepositoryFullname, eventpb.Hash, opt); err != nil {
		return err
	}

	return nil
}

//GetEvents is not implemented
func (c *GitlabClient) GetEvents(repo string, dateRef time.Time) ([]interface{}, time.Duration, error) {
	return nil, 0.0, fmt.Errorf("Not implemented on Gitlab")
}

//PushEvents is not implemented
func (c *GitlabClient) PushEvents(string, []interface{}) ([]sdk.VCSPushEvent, error) {
	return nil, fmt.Errorf("Not implemented on Gitlab")
}

//CreateEvents is not implemented
func (c *GitlabClient) CreateEvents(string, []interface{}) ([]sdk.VCSCreateEvent, error) {
	return nil, fmt.Errorf("Not implemented on Gitlab")
}

//DeleteEvents is not implemented
func (c *GitlabClient) DeleteEvents(string, []interface{}) ([]sdk.VCSDeleteEvent, error) {
	return nil, fmt.Errorf("Not implemented on Gitlab")
}

//PullRequestEvents is not implemented
func (c *GitlabClient) PullRequestEvents(string, []interface{}) ([]sdk.VCSPullRequestEvent, error) {
	return nil, fmt.Errorf("Not implemented on Gitlab")
}

// Release on gitlab
// TODO: https://docs.gitlab.com/ee/api/tags.html#create-a-new-release
func (c *GitlabClient) Release(repo string, tagName string, title string, releaseNote string) (*sdk.VCSRelease, error) {
	return nil, fmt.Errorf("not implemented")
}

// UploadReleaseFile upload a release file project
func (c *GitlabClient) UploadReleaseFile(repo string, release *sdk.VCSRelease, runArtifact sdk.WorkflowNodeRunArtifact, buf *bytes.Buffer) error {
	return fmt.Errorf("not implemented")
}
