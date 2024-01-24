package repositoriesmanager

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"
	gocache "github.com/patrickmn/go-cache"
	"github.com/rockbears/log"
	"github.com/xanzy/go-gitlab"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func (c *vcsClient) IsBitbucketCloud() bool {
	if c.vcsProject != nil {
		return c.vcsProject.Type == "bitbucketcloud"
	}
	return false
}

func (c *vcsClient) IsGerrit(ctx context.Context, db gorp.SqlExecutor) (bool, error) {
	if c.vcsProject != nil {
		return c.vcsProject.Type == "gerrit", nil
	}
	return false, nil
}

type vcsConsumer struct {
	name string
	proj *sdk.Project
	db   gorpmapper.SqlExecutorWithTx
}

type vcsClient struct {
	name       string
	token      string
	secret     string
	projectKey string
	created    int64 //Timestamp .Unix() of creation
	srvs       []sdk.Service
	cache      *gocache.Cache
	db         gorpmapper.SqlExecutorWithTx
	vcsProject *sdk.VCSProject
}

func (c *vcsClient) Cache() *gocache.Cache {
	if c.cache == nil {
		c.cache = gocache.New(5*time.Second, 60*time.Second)
	}
	return c.cache
}

type Options struct {
	Sync bool
}

func GetReposForProjectVCSServer(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, vcsServerName string, opts Options) ([]sdk.VCSRepo, error) {
	log.Debug(ctx, "GetReposForProjectVCSServer> Loading repo for %s", vcsServerName)

	client, err := AuthorizedClient(ctx, db, store, proj.Key, vcsServerName)
	if err != nil {
		return nil, sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrNoReposManagerClientAuth,
			"cannot get client got %s %s", proj.Key, vcsServerName))
	}

	cacheKey := cache.Key("reposmanager", "repos", proj.Key, vcsServerName)
	if opts.Sync {
		if err := store.Delete(cacheKey); err != nil {
			log.Error(ctx, "GetReposForProjectVCSServer> error on delete cache key %v: %s", cacheKey, err)
		}
	}

	var repos []sdk.VCSRepo
	find, err := store.Get(cacheKey, &repos)
	if err != nil {
		log.Error(ctx, "GetReposForProjectVCSServer> cannot get from cache %s: %v", cacheKey, err)
	}
	if !find || len(repos) == 0 {
		repos, err = client.Repos(ctx)
		if err != nil {
			return nil, sdk.NewErrorFrom(err, "cannot get repositories")
		}
		if err := store.SetWithTTL(cacheKey, repos, 0); err != nil {
			log.Error(ctx, "GetReposForProjectVCSServer> cannot SetWithTTL: %s: %v", cacheKey, err)
		}
	}

	return repos, nil
}

func (c *vcsConsumer) GetAuthorizedClient(ctx context.Context, token, secret string, created int64) (sdk.VCSAuthorizedClientService, error) {
	srvs, err := services.LoadAllByType(ctx, c.db, sdk.TypeVCS)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	return &vcsClient{
		name:       c.name,
		token:      token,
		projectKey: c.proj.Key,
		created:    created,
		secret:     secret,
		srvs:       srvs,
		cache:      gocache.New(5*time.Second, 60*time.Second),
		db:         c.db,
	}, nil
}

// AuthorizedClient returns an implementation of AuthorizedClient wrapping calls to vcs uService
func AuthorizedClient(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, projectKey string, vcsName string) (sdk.VCSAuthorizedClientService, error) {
	vcsProject, err := vcs.LoadVCSByProject(ctx, db, projectKey, vcsName, gorpmapping.GetOptions.WithDecryption)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	srvs, err := services.LoadAllByType(ctx, db, sdk.TypeVCS)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	vcs := &vcsClient{
		name:       vcsProject.Name,
		srvs:       srvs,
		db:         db,
		projectKey: projectKey,
		vcsProject: vcsProject,
	}

	return vcs, nil
}

func (c *vcsClient) setAuthHeader(ctx context.Context, req *http.Request) {
	log.Debug(ctx, "requesting vcs via vcs project")
	req.Header.Set(sdk.HeaderXVCSType, base64.StdEncoding.EncodeToString([]byte(c.vcsProject.Type)))
	req.Header.Set(sdk.HeaderXVCSURL, base64.StdEncoding.EncodeToString([]byte(c.vcsProject.URL)))
	req.Header.Set(sdk.HeaderXVCSUsername, base64.StdEncoding.EncodeToString([]byte(c.vcsProject.Auth.Username)))
	req.Header.Set(sdk.HeaderXVCSToken, base64.StdEncoding.EncodeToString([]byte(c.vcsProject.Auth.Token)))
	req.Header.Set(sdk.HeaderXVCSURLApi, base64.StdEncoding.EncodeToString([]byte(c.vcsProject.Options.URLAPI)))

	req.Header.Set(sdk.HeaderXVCSSSHUsername, base64.StdEncoding.EncodeToString([]byte(c.vcsProject.Auth.SSHUsername)))
	req.Header.Set(sdk.HeaderXVCSSSHPort, base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(c.vcsProject.Auth.SSHPort))))
	req.Header.Set(sdk.HeaderXVCSSSHPrivateKey, base64.StdEncoding.EncodeToString([]byte(c.vcsProject.Auth.SSHPrivateKey)))
}

func (c *vcsClient) doStreamRequest(ctx context.Context, method, path string, in interface{}) (io.Reader, http.Header, error) {
	reader, headers, code, err := services.NewClient(c.db, c.srvs).StreamRequest(ctx, method, path, in, func(req *http.Request) {
		c.setAuthHeader(ctx, req)
	})
	if code >= 400 {
		log.Warn(ctx, "repositories manager %s HTTP %s %s error %d", c.name, method, path, code)
		switch code {
		case http.StatusUnauthorized:
			err = sdk.NewError(sdk.ErrNoReposManagerClientAuth, err)
		case http.StatusBadRequest:
			err = sdk.NewError(sdk.ErrWrongRequest, err)
		case http.StatusNotFound:
			err = sdk.NewError(sdk.ErrNotFound, err)
		case http.StatusForbidden:
			err = sdk.NewError(sdk.ErrForbidden, err)
		default:
			err = sdk.NewError(sdk.ErrUnknownError, err)
		}
	}

	return reader, headers, sdk.WithStack(err)
}

func (c *vcsClient) doJSONRequest(ctx context.Context, method, path string, in interface{}, out interface{}) (int, error) {
	_, code, err := services.NewClient(c.db, c.srvs).DoJSONRequest(ctx, method, path, in, out, func(req *http.Request) {
		c.setAuthHeader(ctx, req)
	})

	if code >= 400 {
		log.Warn(ctx, "repositories manager %s HTTP %s %s error %d", c.name, method, path, code)
		switch code {
		case http.StatusUnauthorized:
			err = sdk.NewError(sdk.ErrNoReposManagerClientAuth, err)
		case http.StatusBadRequest:
			err = sdk.NewError(sdk.ErrWrongRequest, err)
		case http.StatusNotFound:
			err = sdk.NewError(sdk.ErrNotFound, err)
		case http.StatusForbidden:
			err = sdk.NewError(sdk.ErrForbidden, err)
		default:
			err = sdk.NewError(sdk.ErrUnknownError, err)
		}
	}

	return code, sdk.WithStack(err)
}

func (c *vcsClient) postBinary(ctx context.Context, path string, fileLength int, r io.Reader, out interface{}) (int, error) {
	return services.PostBinary(ctx, c.srvs, path, r, out, func(req *http.Request) {
		c.setAuthHeader(ctx, req)
		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("Content-Length", strconv.Itoa(fileLength))
	})
}

func (c *vcsClient) Repos(ctx context.Context) ([]sdk.VCSRepo, error) {
	items, has := c.Cache().Get("/repos")
	if has {
		return items.([]sdk.VCSRepo), nil
	}

	repos := []sdk.VCSRepo{}
	path := fmt.Sprintf("/vcs/%s/repos", c.name)
	if _, err := c.doJSONRequest(ctx, "GET", path, nil, &repos); err != nil {
		return nil, sdk.NewErrorFrom(err, "unable to get repositories from %s", c.name)
	}

	c.Cache().SetDefault("/repos", repos)

	return repos, nil
}

func (c *vcsClient) RepoByFullname(ctx context.Context, fullname string) (sdk.VCSRepo, error) {
	var end func()
	ctx, end = telemetry.Span(ctx, "repositories.RepoByFullname")
	defer end()

	items, has := c.Cache().Get("/repos/" + fullname)
	if has {
		return items.(sdk.VCSRepo), nil
	}

	repo := sdk.VCSRepo{}
	path := fmt.Sprintf("/vcs/%s/repos/%s", c.name, fullname)
	if _, err := c.doJSONRequest(ctx, "GET", path, nil, &repo); err != nil {
		return repo, sdk.NewErrorFrom(err, "unable to get repo %s from %s", fullname, c.name)
	}

	c.Cache().SetDefault("/repos/"+fullname, repo)

	return repo, nil
}

func (c *vcsClient) Tag(ctx context.Context, fullname string, tagName string) (sdk.VCSTag, error) {
	items, has := c.Cache().Get(fmt.Sprintf("/tags/%s/%s", tagName, fullname))
	if has {
		return items.(sdk.VCSTag), nil
	}
	var tag sdk.VCSTag
	path := fmt.Sprintf("/vcs/%s/repos/%s/tags/%s", c.name, fullname, tagName)
	if _, err := c.doJSONRequest(ctx, "GET", path, nil, &tag); err != nil {
		return tag, sdk.NewErrorFrom(err, "unable to get tag %s on repository %s from %s", tagName, fullname, c.name)
	}

	c.Cache().SetDefault(fmt.Sprintf("/tags/%s/%s", tagName, fullname), tag)

	return tag, nil
}

func (c *vcsClient) Tags(ctx context.Context, fullname string) ([]sdk.VCSTag, error) {
	items, has := c.Cache().Get("/tags/" + fullname)
	if has {
		return items.([]sdk.VCSTag), nil
	}

	tags := []sdk.VCSTag{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/tags", c.name, fullname)
	if _, err := c.doJSONRequest(ctx, "GET", path, nil, &tags); err != nil {
		return nil, sdk.NewErrorFrom(err, "unable to get tags on repository %s from %s", fullname, c.name)
	}

	c.Cache().SetDefault("/tags/"+fullname, tags)

	return tags, nil
}

func (c *vcsClient) Branches(ctx context.Context, fullname string, filters sdk.VCSBranchesFilter) ([]sdk.VCSBranch, error) {
	items, has := c.Cache().Get("/branches/" + fullname)
	if has {
		return items.([]sdk.VCSBranch), nil
	}

	branches := []sdk.VCSBranch{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/branches?limit=%d", c.name, fullname, filters.Limit)
	if _, err := c.doJSONRequest(ctx, "GET", path, nil, &branches); err != nil {
		return nil, sdk.NewErrorFrom(err, "unable to find branches on repository %s from %s", fullname, c.name)
	}

	c.Cache().SetDefault("/branches/"+fullname, branches)

	return branches, nil
}

func (c *vcsClient) SearchPullRequest(ctx context.Context, repoFullName, commit, state string) (*sdk.VCSPullRequest, error) {
	var pr sdk.VCSPullRequest
	path := fmt.Sprintf("/vcs/%s/repos/%s/search/pullrequest?commit=%s&state=%s", c.name, repoFullName, commit, state)
	if _, err := c.doJSONRequest(ctx, "GET", path, nil, &pr); err != nil {
		return nil, sdk.NewErrorFrom(err, "unable to find pullrequest with commit %s and state %s on %s", commit, state, c.name)
	}
	return &pr, nil
}

func (c *vcsClient) Branch(ctx context.Context, fullname string, filters sdk.VCSBranchFilters) (*sdk.VCSBranch, error) {
	branch := sdk.VCSBranch{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/branches/?branch=%s&default=%v", c.name, fullname, url.QueryEscape(filters.BranchName), filters.Default)
	if _, err := c.doJSONRequest(ctx, "GET", path, nil, &branch); err != nil {
		return nil, sdk.NewErrorFrom(err, "unable to find branch %q (default: %t) on repository %s from %s", filters.BranchName, filters.Default, fullname, c.name)
	}
	return &branch, nil
}

// DefaultBranch get default branch from given repository
func DefaultBranch(ctx context.Context, c sdk.VCSAuthorizedClientCommon, fullname string) (sdk.VCSBranch, error) {
	branch, err := c.Branch(ctx, fullname, sdk.VCSBranchFilters{Default: true})
	if err != nil {
		return sdk.VCSBranch{}, err
	}
	return *branch, nil
}

func (c *vcsClient) Commits(ctx context.Context, fullname, branch, since, until string) ([]sdk.VCSCommit, error) {
	commits := []sdk.VCSCommit{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/branches/commits?branch=%s&since=%s&until=%s", c.name, fullname, url.QueryEscape(branch), url.QueryEscape(since), url.QueryEscape(until))
	if code, err := c.doJSONRequest(ctx, "GET", path, nil, &commits); err != nil {
		if code != http.StatusNotFound {
			return nil, sdk.NewErrorFrom(err, "unable to find commits on repository %s from %s", fullname, c.name)
		}
	}
	return commits, nil
}

func (c *vcsClient) CommitsBetweenRefs(ctx context.Context, fullname, base, head string) ([]sdk.VCSCommit, error) {
	var commits []sdk.VCSCommit
	path := fmt.Sprintf("/vcs/%s/repos/%s/commits?base=%s&head=%s", c.name, fullname, url.QueryEscape(base), url.QueryEscape(head))
	if code, err := c.doJSONRequest(ctx, "GET", path, nil, &commits); err != nil {
		if code != http.StatusNotFound {
			return nil, sdk.NewErrorFrom(err, "unable to find commits on repository %s from %s", fullname, c.name)
		}
	}
	return commits, nil
}

func (c *vcsClient) Commit(ctx context.Context, fullname, hash string) (sdk.VCSCommit, error) {
	commit := sdk.VCSCommit{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/commits/%s", c.name, fullname, hash)
	if code, err := c.doJSONRequest(ctx, "GET", path, nil, &commit); err != nil {
		if code != http.StatusNotFound {
			return commit, sdk.NewErrorFrom(err, "unable to find commit %s on repository %s from %s", hash, fullname, c.name)
		}
	}
	return commit, nil
}

func (c *vcsClient) PullRequest(ctx context.Context, fullname string, ID string) (sdk.VCSPullRequest, error) {
	pr := sdk.VCSPullRequest{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/pullrequests/%s", c.name, fullname, url.PathEscape(ID))
	if code, err := c.doJSONRequest(ctx, "GET", path, nil, &pr); err != nil {
		if code != http.StatusNotFound {
			return pr, sdk.NewErrorFrom(err, "unable to find pullrequest %s on repository %s from %s", ID, fullname, c.name)
		}
		return pr, sdk.WithStack(sdk.ErrNotFound)
	}
	return pr, nil
}

func (c *vcsClient) PullRequests(ctx context.Context, fullname string, mods ...sdk.VCSRequestModifier) ([]sdk.VCSPullRequest, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("/vcs/%s/repos/%s/pullrequests", c.name, fullname), nil)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	for _, m := range mods {
		m(req)
	}
	prs := []sdk.VCSPullRequest{}
	if _, err := c.doJSONRequest(ctx, "GET", req.URL.String(), nil, &prs); err != nil {
		return nil, sdk.NewErrorFrom(err, "unable to find pullrequests on repository %s from %s", fullname, c.name)
	}
	return prs, nil
}

func (c *vcsClient) PullRequestComment(ctx context.Context, fullname string, body sdk.VCSPullRequestCommentRequest) error {
	path := fmt.Sprintf("/vcs/%s/repos/%s/pullrequests/comments", c.name, fullname)
	if _, err := c.doJSONRequest(ctx, "POST", path, body, nil); err != nil {
		return sdk.NewErrorFrom(err, "unable to post pullrequest comments on repository %s from %s", fullname, c.name)
	}
	return nil
}

func (c *vcsClient) PullRequestCreate(ctx context.Context, fullname string, pr sdk.VCSPullRequest) (sdk.VCSPullRequest, error) {
	path := fmt.Sprintf("/vcs/%s/repos/%s/pullrequests", c.name, fullname)
	if _, err := c.doJSONRequest(ctx, "POST", path, pr, &pr); err != nil {
		return pr, sdk.NewErrorFrom(err, "unable to create pullrequest on repository %s from %s", fullname, c.name)
	}
	return pr, nil
}

func (c *vcsClient) CreateHook(ctx context.Context, fullname string, hook *sdk.VCSHook) error {
	path := fmt.Sprintf("/vcs/%s/repos/%s/hooks", c.name, fullname)
	if _, err := c.doJSONRequest(ctx, "POST", path, hook, hook); err != nil {
		return sdk.NewErrorFrom(err, "unable to create hook on repository %s from %s", fullname, c.name)
	}
	return nil
}

func (c *vcsClient) UpdateHook(ctx context.Context, fullname string, hook *sdk.VCSHook) error {
	path := fmt.Sprintf("/vcs/%s/repos/%s/hooks", c.name, fullname)
	if _, err := c.doJSONRequest(ctx, "PUT", path, hook, hook); err != nil {
		return sdk.NewErrorFrom(err, "unable to update hook %s on repository %s from %s", hook.ID, fullname, c.name)
	}
	return nil
}

func (c *vcsClient) GetHook(ctx context.Context, fullname, u string) (sdk.VCSHook, error) {
	path := fmt.Sprintf("/vcs/%s/repos/%s/hooks?url=%s", c.name, fullname, url.QueryEscape(u))
	hook := &sdk.VCSHook{}
	_, err := c.doJSONRequest(ctx, "GET", path, nil, hook)
	return *hook, sdk.NewErrorFrom(err, "unable to get hook %s on repository %s from %s", u, fullname, c.name)
}

func (c *vcsClient) DeleteHook(ctx context.Context, fullname string, hook sdk.VCSHook) error {
	// If we are not able to remove anything, just ignore
	if hook.URL == "" && hook.ID == "" {
		return nil
	}
	path := fmt.Sprintf("/vcs/%s/repos/%s/hooks?url=%s&id=%s", c.name, fullname, url.QueryEscape(hook.URL), hook.ID)
	_, err := c.doJSONRequest(ctx, "DELETE", path, nil, nil)
	return sdk.NewErrorFrom(err, "unable to delete hook on repository %s from %s", fullname, c.name)
}

func (c *vcsClient) GetEvents(ctx context.Context, fullname string, dateRef time.Time) ([]interface{}, time.Duration, error) {
	res := struct {
		Events []interface{} `json:"events"`
		Delay  time.Duration `json:"delay"`
	}{}

	path := fmt.Sprintf("/vcs/%s/repos/%s/events?since=%d", c.name, fullname, dateRef.Unix())
	if _, err := c.doJSONRequest(ctx, "GET", path, nil, &res); err != nil {
		return nil, time.Duration(0), sdk.WrapError(err, "unable to get events on repository %s from %s", fullname, c.name)
	}

	return res.Events, res.Delay, nil
}

func (c *vcsClient) PushEvents(ctx context.Context, fullname string, evts []interface{}) ([]sdk.VCSPushEvent, error) {
	events := []sdk.VCSPushEvent{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/events?filter=push", c.name, fullname)
	if _, err := c.doJSONRequest(ctx, "POST", path, evts, &events); err != nil {
		return nil, sdk.NewErrorFrom(err, "unable to filter push events on repository %s from %s", fullname, c.name)
	}
	return events, nil
}

func (c *vcsClient) CreateEvents(ctx context.Context, fullname string, evts []interface{}) ([]sdk.VCSCreateEvent, error) {
	events := []sdk.VCSCreateEvent{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/events?filter=create", c.name, fullname)
	if _, err := c.doJSONRequest(ctx, "POST", path, evts, &events); err != nil {
		return nil, sdk.NewErrorFrom(err, "unable to filter create events on repository %s from %s", fullname, c.name)
	}
	return events, nil
}

func (c *vcsClient) DeleteEvents(ctx context.Context, fullname string, evts []interface{}) ([]sdk.VCSDeleteEvent, error) {
	events := []sdk.VCSDeleteEvent{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/events?filter=delete", c.name, fullname)
	if _, err := c.doJSONRequest(ctx, "POST", path, evts, &events); err != nil {
		return nil, sdk.NewErrorFrom(err, "unable to filter delete events on repository %s from %s", fullname, c.name)
	}
	return events, nil
}

func (c *vcsClient) PullRequestEvents(ctx context.Context, fullname string, evts []interface{}) ([]sdk.VCSPullRequestEvent, error) {
	events := []sdk.VCSPullRequestEvent{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/events?filter=pullrequests", c.name, fullname)
	if _, err := c.doJSONRequest(ctx, "POST", path, evts, &events); err != nil {
		return nil, sdk.NewErrorFrom(err, "unable to filter pull request events on repository %s from %s", fullname, c.name)
	}
	return events, nil
}

func (c *vcsClient) IsDisableStatusDetails(ctx context.Context) bool {
	if c.vcsProject != nil {
		return c.vcsProject.Options.DisableStatusDetails
	}
	return true
}

func (c *vcsClient) SetDisableStatusDetails(disableStatusDetails bool) {
	// nothing here for the implementation of this func into api
}

func (c *vcsClient) SetStatus(ctx context.Context, event sdk.Event) error {
	if c.vcsProject != nil && c.vcsProject.Options.DisableStatus {
		return nil
	}

	path := fmt.Sprintf("/vcs/%s/status", c.name)
	if c.vcsProject != nil && c.vcsProject.Options.DisableStatusDetails {
		path += "?disableStatusDetails=true"
	}

	_, err := c.doJSONRequest(ctx, "POST", path, event, nil)
	return sdk.NewErrorFrom(err, "unable to set status on %s (workflow: %s, application: %s)", event.WorkflowName, event.ApplicationName, c.name)
}

func (c *vcsClient) Release(ctx context.Context, fullname, tagName, releaseTitle, releaseDescription string) (*sdk.VCSRelease, error) {
	res := struct {
		Tag         string `json:"tag"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}{
		Tag:         tagName,
		Title:       releaseTitle,
		Description: releaseDescription,
	}

	release := sdk.VCSRelease{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/releases", c.name, fullname)
	_, err := c.doJSONRequest(ctx, "POST", path, &res, &release)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	return &release, nil
}

func (c *vcsClient) UploadReleaseFile(ctx context.Context, fullname string, releaseName, uploadURL string, artifactName string, r io.Reader, fileLength int) error {
	path := fmt.Sprintf("/vcs/%s/repos/%s/releases/%s/artifacts/%s?upload_url=%s", c.name, fullname, releaseName, artifactName, url.QueryEscape(uploadURL))
	if _, err := c.postBinary(ctx, path, fileLength, r, nil); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

func (c *vcsClient) ListForks(ctx context.Context, repo string) ([]sdk.VCSRepo, error) {
	forks := []sdk.VCSRepo{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/forks", c.name, repo)
	if _, err := c.doJSONRequest(ctx, "GET", path, nil, &forks); err != nil {
		return nil, sdk.WithStack(err)
	}
	return forks, nil
}

func (c *vcsClient) ListStatuses(ctx context.Context, repo string, ref string) ([]sdk.VCSCommitStatus, error) {
	statuses := []sdk.VCSCommitStatus{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/commits/%s/statuses", c.name, repo, ref)
	if _, err := c.doJSONRequest(ctx, "GET", path, nil, &statuses); err != nil {
		return nil, sdk.WithStack(err)
	}
	return statuses, nil
}

func (c *vcsClient) GetAccessToken(_ context.Context) string {
	return ""
}

func (c *vcsClient) ListContent(ctx context.Context, repo string, commit, dir string) ([]sdk.VCSContent, error) {
	path := fmt.Sprintf("/vcs/%s/repos/%s/contents/%s?commit=%s", c.name, repo, url.PathEscape(dir), commit)
	var contents []sdk.VCSContent
	if _, err := c.doJSONRequest(ctx, "GET", path, nil, &contents); err != nil {
		return nil, sdk.WithStack(err)
	}
	return contents, nil
}

func (c *vcsClient) GetContent(ctx context.Context, repo string, commit, dir string) (sdk.VCSContent, error) {
	path := fmt.Sprintf("/vcs/%s/repos/%s/content/%s?commit=%s", c.name, repo, url.PathEscape(dir), commit)
	var content sdk.VCSContent
	if _, err := c.doJSONRequest(ctx, "GET", path, nil, &content); err != nil {
		return content, sdk.WithStack(err)
	}
	return content, nil
}

func (c *vcsClient) GetArchive(ctx context.Context, repo string, dir string, format string, commit string) (io.Reader, http.Header, error) {
	archiveRequest := sdk.VCSArchiveRequest{
		Path:   dir,
		Commit: commit,
		Format: format,
	}
	urlPath := fmt.Sprintf("/vcs/%s/repos/%s/archive", c.name, repo)
	return c.doStreamRequest(ctx, "POST", urlPath, archiveRequest)
}

// WebhooksInfos is a set of info about webhooks
type WebhooksInfos struct {
	WebhooksSupported  bool     `json:"webhooks_supported"`
	WebhooksDisabled   bool     `json:"webhooks_disabled"`
	GerritHookDisabled bool     `json:"gerrithook_disabled"`
	Icon               string   `json:"webhooks_icon"`
	Events             []string `json:"events"`
}

// GetWebhooksInfos returns webhooks_supported, webhooks_disabled, webhooks_creation_supported, webhooks_creation_disabled for a vcs server
func GetWebhooksInfos(ctx context.Context, c sdk.VCSAuthorizedClientService) (WebhooksInfos, error) {
	client, ok := c.(*vcsClient)
	if !ok {
		return WebhooksInfos{}, errors.New("cast error")
	}

	log.Debug(ctx, "getting GetWebhooksInfos for type: %s", client.vcsProject.Type)
	res := WebhooksInfos{}
	switch {
	case client.vcsProject.Type == sdk.VCSTypeBitbucketServer:
		res.WebhooksSupported = true
		res.Icon = sdk.BitbucketIcon
		// https://confluence.atlassian.com/bitbucketserver/event-payload-938025882.html
		res.Events = sdk.BitbucketEvents
		res.WebhooksDisabled = client.vcsProject.Options.DisableWebhooks
	case client.vcsProject.Type == sdk.VCSTypeBitbucketCloud:
		res.WebhooksSupported = true
		res.Icon = sdk.BitbucketIcon
		// https://developer.atlassian.com/bitbucket/api/2/reference/resource/hook_events/%7Bsubject_type%7D
		res.Events = sdk.BitbucketCloudEvents
		res.WebhooksDisabled = client.vcsProject.Options.DisableWebhooks
	case client.vcsProject.Type == sdk.VCSTypeGithub:
		res.WebhooksSupported = true
		res.Icon = sdk.GitHubIcon
		// https://developer.github.com/v3/activity/events/types/
		res.Events = sdk.GitHubEvents
		res.WebhooksDisabled = client.vcsProject.Options.DisableWebhooks
	case client.vcsProject.Type == sdk.VCSTypeGitlab:
		res.WebhooksSupported = true
		res.Icon = sdk.GitlabIcon
		res.WebhooksDisabled = client.vcsProject.Options.DisableWebhooks
		// https://docs.gitlab.com/ee/user/project/integrations/webhooks.html
		res.Events = []string{
			string(gitlab.EventTypePush),
			string(gitlab.EventTypeTagPush),
			string(gitlab.EventTypeIssue),
			string(gitlab.EventTypeNote),
			string(gitlab.EventTypeMergeRequest),
			string(gitlab.EventTypeWikiPage),
			string(gitlab.EventTypePipeline),
			"Job Hook", // TODO update gitlab sdk
		}
	case client.vcsProject.Type == sdk.VCSTypeGerrit:
		res.WebhooksSupported = false
		res.Icon = sdk.GerritIcon
		// https://git.eclipse.org/r/Documentation/cmd-stream-events.html#events
		res.Events = sdk.GerritEvents
	}

	return res, nil
}

// PollingInfos is a set of info about polling functions
type PollingInfos struct {
	PollingSupported bool `json:"polling_supported"`
	PollingDisabled  bool `json:"polling_disabled"`
}

// GetPollingInfos returns polling_supported and polling_disabled for a vcs server
func GetPollingInfos(ctx context.Context, c sdk.VCSAuthorizedClientService, prj sdk.Project) (PollingInfos, error) {
	client, ok := c.(*vcsClient)
	if !ok {
		return PollingInfos{}, errors.New("cast error")
	}

	res := PollingInfos{}
	res.PollingDisabled = client.vcsProject.Options.DisablePolling

	switch {
	case client.vcsProject.Type == "bitbucketserver":
		res.PollingSupported = false
	case client.vcsProject.Type == "bitbucketcloud":
		res.PollingSupported = false
	case client.vcsProject.Type == "github":
		res.PollingSupported = true
	case client.vcsProject.Type == "gitlab":
		res.PollingSupported = false
	}

	return res, nil

}
