package repositoriesmanager

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/go-gorp/gorp"
	gocache "github.com/patrickmn/go-cache"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/services"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//LoadAll Load all RepositoriesManager from the database
func LoadAll(ctx context.Context, db *gorp.DbMap, store cache.Store) ([]string, error) {
	srvs, err := services.FindByType(db, services.TypeVCS)
	if err != nil {
		return nil, sdk.WrapError(err, "repositoriesmanager.LoadAll> Unable to load services")
	}

	vcsServers := map[string]interface{}{}
	if _, err := services.DoJSONRequest(ctx, srvs, "GET", "/vcs", nil, &vcsServers); err != nil {
		return nil, err
	}
	servers := []string{}
	for k := range vcsServers {
		servers = append(servers, k)
	}
	return servers, nil
}

type vcsConsumer struct {
	name   string
	proj   *sdk.Project
	dbFunc func() *gorp.DbMap
	cache  cache.Store
}

type vcsClient struct {
	name   string
	token  string
	secret string
	srvs   []sdk.Service
	cache  *gocache.Cache
}

func (c *vcsClient) Cache() *gocache.Cache {
	if c.cache == nil {
		c.cache = gocache.New(5*time.Second, 60*time.Second)
	}
	return c.cache
}

// GetProjectVCSServer returns sdk.ProjectVCSServer for a project
func GetProjectVCSServer(p *sdk.Project, name string) *sdk.ProjectVCSServer {
	if name == "" {
		return nil
	}
	for _, v := range p.VCSServers {
		if v.Name == name {
			return &v
		}
	}

	return nil
}

// NewVCSServerConsumer returns a sdk.VCSServer wrapping vcs uservices calls
func NewVCSServerConsumer(dbFunc func() *gorp.DbMap, store cache.Store, name string) (sdk.VCSServer, error) {
	return &vcsConsumer{name: name, dbFunc: dbFunc}, nil
}

func (c *vcsConsumer) AuthorizeRedirect(ctx context.Context) (string, string, error) {
	srv, err := services.FindByType(c.dbFunc(), services.TypeVCS)
	if err != nil {
		return "", "", err
	}

	res := map[string]string{}
	path := fmt.Sprintf("/vcs/%s/authorize", c.name)
	log.Info("Performing request on %s", path)
	if _, err := services.DoJSONRequest(ctx, srv, "GET", path, nil, &res); err != nil {
		return "", "", sdk.WrapError(err, "repositoriesmanager.AuthorizeRedirect> ")
	}

	return res["token"], res["url"], nil
}

func (c *vcsConsumer) AuthorizeToken(ctx context.Context, token string, secret string) (string, string, error) {
	srv, err := services.FindByType(c.dbFunc(), services.TypeVCS)
	if err != nil {
		return "", "", err
	}

	body := map[string]string{
		"token":  token,
		"secret": secret,
	}

	res := map[string]string{}
	path := fmt.Sprintf("/vcs/%s/authorize", c.name)
	if _, err := services.DoJSONRequest(ctx, srv, "POST", path, body, &res); err != nil {
		return "", "", err
	}

	return res["token"], res["secret"], nil
}

func (c *vcsConsumer) GetAuthorizedClient(ctx context.Context, token string, secret string) (sdk.VCSAuthorizedClient, error) {
	s := GetProjectVCSServer(c.proj, c.name)
	if s == nil {
		return nil, sdk.ErrNoReposManagerClientAuth
	}

	srvs, err := services.FindByType(c.dbFunc(), services.TypeVCS)
	if err != nil {
		return nil, err
	}

	return &vcsClient{
		name:   c.name,
		token:  token,
		secret: secret,
		srvs:   srvs,
		cache:  gocache.New(5*time.Second, 60*time.Second),
	}, nil
}

var local = localAuthorizedClientCache{
	cache: make(map[uint64]sdk.VCSAuthorizedClient),
}

type localAuthorizedClientCache struct {
	mutex sync.RWMutex
	cache map[uint64]sdk.VCSAuthorizedClient
}

func (c *localAuthorizedClientCache) Set(repo *sdk.ProjectVCSServer, vcs sdk.VCSAuthorizedClient) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	hash := repo.Hash()
	if hash == 0 {
		return
	}
	c.cache[hash] = vcs
}

func (c *localAuthorizedClientCache) Get(repo *sdk.ProjectVCSServer) (sdk.VCSAuthorizedClient, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	vcs, ok := c.cache[repo.Hash()]
	return vcs, ok
}

//AuthorizedClient returns an implementation of AuthorizedClient wrapping calls to vcs uService
func AuthorizedClient(ctx context.Context, db gorp.SqlExecutor, store cache.Store, repo *sdk.ProjectVCSServer) (sdk.VCSAuthorizedClient, error) {
	if repo == nil {
		return nil, sdk.ErrUnauthorized
	}

	vcs, has := local.Get(repo)
	if has {
		return vcs, nil
	}

	srvs, err := services.FindByType(db, services.TypeVCS)
	if err != nil {
		return nil, err
	}

	vcs = &vcsClient{
		name:   repo.Name,
		token:  repo.Data["token"],
		secret: repo.Data["secret"],
		srvs:   srvs,
	}
	local.Set(repo, vcs)
	return vcs, nil
}

func (c *vcsClient) doJSONRequest(ctx context.Context, method, path string, in interface{}, out interface{}) (int, error) {
	code, err := services.DoJSONRequest(ctx, c.srvs, method, path, in, out, func(req *http.Request) {
		req.Header.Set("X-CDS-ACCESS-TOKEN", base64.StdEncoding.EncodeToString([]byte(c.token)))
		req.Header.Set("X-CDS-ACCESS-TOKEN-SECRET", base64.StdEncoding.EncodeToString([]byte(c.secret)))
	})

	if code >= 400 {
		switch code {
		case http.StatusUnauthorized:
			err = sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "%s", err)
		case http.StatusBadRequest:
			err = sdk.WrapError(sdk.ErrWrongRequest, "%s", err)
		case http.StatusNotFound:
			err = sdk.WrapError(sdk.ErrNotFound, "%s", err)
		case http.StatusForbidden:
			err = sdk.WrapError(sdk.ErrForbidden, "%s", err)
		default:
			err = sdk.WrapError(sdk.ErrUnknownError, "%s", err)
		}
	}

	return code, err
}

func (c *vcsClient) postMultipart(ctx context.Context, path string, fileContent []byte, out interface{}) (int, error) {
	return services.PostMultipart(ctx, c.srvs, "POST", path, fileContent, out, func(req *http.Request) {
		req.Header.Set("X-CDS-ACCESS-TOKEN", base64.StdEncoding.EncodeToString([]byte(c.token)))
		req.Header.Set("X-CDS-ACCESS-TOKEN-SECRET", base64.StdEncoding.EncodeToString([]byte(c.secret)))
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
		return nil, err
	}

	c.Cache().SetDefault("/repos", repos)

	return repos, nil
}

func (c *vcsClient) RepoByFullname(ctx context.Context, fullname string) (sdk.VCSRepo, error) {
	var end func()
	ctx, end = observability.Span(ctx, "repositories.RepoByFullname")
	defer end()

	items, has := c.Cache().Get("/repos/" + fullname)
	if has {
		return items.(sdk.VCSRepo), nil
	}

	repo := sdk.VCSRepo{}
	path := fmt.Sprintf("/vcs/%s/repos/%s", c.name, fullname)
	if _, err := c.doJSONRequest(ctx, "GET", path, nil, &repo); err != nil {
		return repo, err
	}

	c.Cache().SetDefault("/repos/"+fullname, repo)

	return repo, nil
}

func (c *vcsClient) Branches(ctx context.Context, fullname string) ([]sdk.VCSBranch, error) {
	items, has := c.Cache().Get("/branches/" + fullname)
	if has {
		return items.([]sdk.VCSBranch), nil
	}

	branches := []sdk.VCSBranch{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/branches", c.name, fullname)
	if _, err := c.doJSONRequest(ctx, "GET", path, nil, &branches); err != nil {
		return nil, err
	}

	c.Cache().SetDefault("/branches/"+fullname, branches)

	return branches, nil
}

func (c *vcsClient) Branch(ctx context.Context, fullname string, branchName string) (*sdk.VCSBranch, error) {
	branch := sdk.VCSBranch{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/branches/?branch=%s", c.name, fullname, url.QueryEscape(branchName))
	if _, err := c.doJSONRequest(ctx, "GET", path, nil, &branch); err != nil {
		return nil, err
	}
	return &branch, nil
}

func (c *vcsClient) Commits(ctx context.Context, fullname, branch, since, until string) ([]sdk.VCSCommit, error) {
	commits := []sdk.VCSCommit{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/branches/commits?branch=%s&since=%s&until=%s", c.name, fullname, url.QueryEscape(branch), url.QueryEscape(since), url.QueryEscape(until))
	if code, err := c.doJSONRequest(ctx, "GET", path, nil, &commits); err != nil {
		if code != http.StatusNotFound {
			return nil, err
		}
	}
	return commits, nil
}

func (c *vcsClient) Commit(ctx context.Context, fullname, hash string) (sdk.VCSCommit, error) {
	commit := sdk.VCSCommit{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/commits/%s", c.name, fullname, hash)
	if code, err := c.doJSONRequest(ctx, "GET", path, nil, &commit); err != nil {
		if code != http.StatusNotFound {
			return commit, err
		}
	}
	return commit, nil
}

func (c *vcsClient) PullRequests(ctx context.Context, fullname string) ([]sdk.VCSPullRequest, error) {
	prs := []sdk.VCSPullRequest{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/pullrequests", c.name, fullname)
	if _, err := c.doJSONRequest(ctx, "GET", path, nil, &prs); err != nil {
		return nil, err
	}
	return prs, nil
}

func (c *vcsClient) PullRequestComment(ctx context.Context, fullname string, id int, body string) error {
	path := fmt.Sprintf("/vcs/%s/repos/%s/pullrequests/%d/comments", c.name, fullname, id)
	if _, err := c.doJSONRequest(ctx, "POST", path, body, nil); err != nil {
		return err
	}
	return nil
}

func (c *vcsClient) CreateHook(ctx context.Context, fullname string, hook *sdk.VCSHook) error {
	path := fmt.Sprintf("/vcs/%s/repos/%s/hooks", c.name, fullname)
	_, err := c.doJSONRequest(ctx, "POST", path, hook, hook)
	return err
}

func (c *vcsClient) GetHook(ctx context.Context, fullname, u string) (sdk.VCSHook, error) {
	path := fmt.Sprintf("/vcs/%s/repos/%s/hooks?url=%s", c.name, fullname, url.QueryEscape(u))
	hook := &sdk.VCSHook{}
	_, err := c.doJSONRequest(ctx, "GET", path, nil, hook)
	return *hook, err
}

func (c *vcsClient) UpdateHook(ctx context.Context, fullname, url string, hook sdk.VCSHook) error {
	return nil
}

func (c *vcsClient) DeleteHook(ctx context.Context, fullname string, hook sdk.VCSHook) error {
	// If we are not able to remove anything, just ignore
	if hook.URL == "" && hook.ID == "" {
		return nil
	}
	path := fmt.Sprintf("/vcs/%s/repos/%s/hooks?url=%s&id=%s", c.name, fullname, url.QueryEscape(hook.URL), hook.ID)
	_, err := c.doJSONRequest(ctx, "DELETE", path, nil, nil)
	return err
}

func (c *vcsClient) GetEvents(ctx context.Context, fullname string, dateRef time.Time) ([]interface{}, time.Duration, error) {
	res := struct {
		Events []interface{} `json:"events"`
		Delay  time.Duration `json:"delay"`
	}{}

	path := fmt.Sprintf("/vcs/%s/repos/%s/events?since=%d", c.name, fullname, dateRef.Unix())
	if _, err := c.doJSONRequest(ctx, "GET", path, nil, &res); err != nil {
		return nil, time.Duration(0), err
	}

	return res.Events, res.Delay, nil
}

func (c *vcsClient) PushEvents(ctx context.Context, fullname string, evts []interface{}) ([]sdk.VCSPushEvent, error) {
	events := []sdk.VCSPushEvent{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/events?filter=push", c.name, fullname)
	if _, err := c.doJSONRequest(ctx, "POST", path, evts, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (c *vcsClient) CreateEvents(ctx context.Context, fullname string, evts []interface{}) ([]sdk.VCSCreateEvent, error) {
	events := []sdk.VCSCreateEvent{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/events?filter=create", c.name, fullname)
	if _, err := c.doJSONRequest(ctx, "POST", path, evts, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (c *vcsClient) DeleteEvents(ctx context.Context, fullname string, evts []interface{}) ([]sdk.VCSDeleteEvent, error) {
	events := []sdk.VCSDeleteEvent{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/events?filter=delete", c.name, fullname)
	if _, err := c.doJSONRequest(ctx, "POST", path, evts, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (c *vcsClient) PullRequestEvents(ctx context.Context, fullname string, evts []interface{}) ([]sdk.VCSPullRequestEvent, error) {
	events := []sdk.VCSPullRequestEvent{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/events?filter=pullrequests", c.name, fullname)
	if _, err := c.doJSONRequest(ctx, "POST", path, evts, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (c *vcsClient) SetStatus(ctx context.Context, event sdk.Event) error {
	path := fmt.Sprintf("/vcs/%s/status", c.name)
	_, err := c.doJSONRequest(ctx, "POST", path, event, nil)
	return err
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
		return nil, err
	}
	return &release, nil
}

func (c *vcsClient) UploadReleaseFile(ctx context.Context, fullname string, releaseName, uploadURL string, artifactName string, r io.ReadCloser) error {
	path := fmt.Sprintf("/vcs/%s/repos/%s/releases/%s/artifacts/%s", c.name, fullname, releaseName, artifactName)
	defer r.Close()

	fileContent, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	if _, err := c.postMultipart(ctx, path, fileContent, nil); err != nil {
		return err
	}
	return nil
}

func (c *vcsClient) ListForks(ctx context.Context, repo string) ([]sdk.VCSRepo, error) {
	forks := []sdk.VCSRepo{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/forks", c.name, repo)
	if _, err := c.doJSONRequest(ctx, "GET", path, nil, &forks); err != nil {
		return nil, err
	}
	return forks, nil
}

func (c *vcsClient) ListStatuses(ctx context.Context, repo string, ref string) ([]sdk.VCSCommitStatus, error) {
	statuses := []sdk.VCSCommitStatus{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/commits/%s/statuses", c.name, repo, ref)
	if _, err := c.doJSONRequest(ctx, "GET", path, nil, &statuses); err != nil {
		return nil, err
	}
	return statuses, nil
}

func (c *vcsClient) GrantReadPermission(ctx context.Context, repo string) error {
	path := fmt.Sprintf("/vcs/%s/repos/%s/grant", c.name, repo)
	if _, err := c.doJSONRequest(ctx, "POST", path, nil, nil); err != nil {
		return err
	}
	return nil
}

// WebhooksInfos is a set of info about webhooks
type WebhooksInfos struct {
	WebhooksSupported bool   `json:"webhooks_supported"`
	WebhooksDisabled  bool   `json:"webhooks_disabled"`
	Icon              string `json:"webhooks_icon"`
}

// GetWebhooksInfos returns webhooks_supported, webhooks_disabled, webhooks_creation_supported, webhooks_creation_disabled for a vcs server
func GetWebhooksInfos(ctx context.Context, c sdk.VCSAuthorizedClient) (WebhooksInfos, error) {
	client, ok := c.(*vcsClient)
	if !ok {
		return WebhooksInfos{}, fmt.Errorf("Polling infos cast error")
	}
	res := WebhooksInfos{}
	path := fmt.Sprintf("/vcs/%s/webhooks", client.name)
	if _, err := client.doJSONRequest(ctx, "GET", path, nil, &res); err != nil {
		return WebhooksInfos{}, err
	}
	return res, nil
}

// PollingInfos is a set of info about polling functions
type PollingInfos struct {
	PollingSupported bool `json:"polling_supported"`
	PollingDisabled  bool `json:"polling_disabled"`
}

// GetPollingInfos returns polling_supported and polling_disabled for a vcs server
func GetPollingInfos(ctx context.Context, c sdk.VCSAuthorizedClient, prj sdk.Project) (PollingInfos, error) {
	client, ok := c.(*vcsClient)
	if !ok {
		return PollingInfos{}, fmt.Errorf("Polling infos cast error")
	}
	res := PollingInfos{}
	path := fmt.Sprintf("/vcs/%s/polling", client.name)
	if _, err := client.doJSONRequest(ctx, "GET", path, nil, &res); err != nil {
		return PollingInfos{}, sdk.WrapError(err, "GetPollingInfos> project %s", prj.Key)
	}
	return res, nil
}
