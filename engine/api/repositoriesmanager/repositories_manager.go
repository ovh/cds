package repositoriesmanager

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
)

type vcsClient struct {
	name   string
	token  string
	secret string
	srvs   []sdk.Service
}

// GetVCSServer returns sdk.ProjectVCSServer for a project
func GetVCSServer(p *sdk.Project, name string) *sdk.ProjectVCSServer {
	for _, v := range p.VCSServers {
		if v.Name == name {
			return &v
		}
	}

	return nil
}

//AuthorizedClient returns an implementation of AuthorizedClient wrapping calls to vcs uService
func AuthorizedClient(db gorp.SqlExecutor, store cache.Store, repo *sdk.ProjectVCSServer) (sdk.VCSAuthorizedClient, error) {
	if repo == nil {
		return nil, sdk.ErrUnauthorized
	}

	servicesDao := services.Querier(db, store)
	srvs, err := servicesDao.FindByType("vcs")
	if err != nil {
		return nil, err
	}

	return &vcsClient{
		name:   repo.Name,
		token:  repo.Data["token"],
		secret: repo.Data["secret"],
		srvs:   srvs,
	}, nil
}

func (c *vcsClient) doJSONRequest(method, path string, in interface{}, out interface{}) (int, error) {
	return services.DoJSONRequest(c.srvs, method, path, in, out, func(req *http.Request) {
		req.Header.Set("X-CDS-ACCESS-TOKEN", c.token)
		req.Header.Set("X-CDS-ACCESS-TOKEN-SECRET", c.token)
	})
}

func (c *vcsClient) Repos() ([]sdk.VCSRepo, error) {
	repos := []sdk.VCSRepo{}
	path := fmt.Sprintf("/vcs/%s/repos", c.name)
	if _, err := c.doJSONRequest("GET", path, nil, &repos); err != nil {
		return nil, err
	}
	return repos, nil
}

func (c *vcsClient) RepoByFullname(fullname string) (sdk.VCSRepo, error) {
	repo := sdk.VCSRepo{}
	path := fmt.Sprintf("/vcs/%s/repos/%s", c.name, fullname)
	if _, err := c.doJSONRequest("GET", path, nil, &repo); err != nil {
		return repo, err
	}
	return repo, nil
}

func (c *vcsClient) Branches(string) ([]sdk.VCSBranch, error) {
	return nil, nil
}
func (c *vcsClient) Branch(string, string) (*sdk.VCSBranch, error) {
	return nil, nil
}
func (c *vcsClient) Commits(repo, branch, since, until string) ([]sdk.VCSCommit, error) {
	return nil, nil
}
func (c *vcsClient) Commit(repo, hash string) (sdk.VCSCommit, error) {
	return sdk.VCSCommit{}, nil
}
func (c *vcsClient) PullRequests(string) ([]sdk.VCSPullRequest, error) {
	return nil, nil
}
func (c *vcsClient) CreateHook(repo string, hook sdk.VCSHook) error {
	return nil
}
func (c *vcsClient) GetHook(repo, url string) (sdk.VCSHook, error) {
	return sdk.VCSHook{}, nil
}
func (c *vcsClient) UpdateHook(repo, url string, hook sdk.VCSHook) error {
	return nil
}
func (c *vcsClient) DeleteHook(repo string, hook sdk.VCSHook) error {
	return nil
}
func (c *vcsClient) GetEvents(repo string, dateRef time.Time) ([]interface{}, time.Duration, error) {
	return nil, time.Duration(0), nil
}
func (c *vcsClient) PushEvents(string, []interface{}) ([]sdk.VCSPushEvent, error) {
	return nil, nil
}
func (c *vcsClient) CreateEvents(string, []interface{}) ([]sdk.VCSCreateEvent, error) {
	return nil, nil
}
func (c *vcsClient) DeleteEvents(string, []interface{}) ([]sdk.VCSDeleteEvent, error) {
	return nil, nil
}
func (c *vcsClient) PullRequestEvents(string, []interface{}) ([]sdk.VCSPullRequestEvent, error) {
	return nil, nil
}
func (c *vcsClient) SetStatus(event sdk.Event) error {
	return nil
}
func (c *vcsClient) Release(repo, tagName, releaseTitle, releaseDescription string) (*sdk.VCSRelease, error) {
	return nil, nil
}
func (c *vcsClient) UploadReleaseFile(repo string, release *sdk.VCSRelease, runArtifact sdk.WorkflowNodeRunArtifact, file *bytes.Buffer) error {
	return nil
}
