package repositoriesmanager

import (
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//LoadAll Load all RepositoriesManager from the database
func LoadAll(db *gorp.DbMap, store cache.Store) ([]string, error) {
	serviceDAO := services.NewRepository(func() *gorp.DbMap { return db }, store)
	srvs, err := serviceDAO.FindByType("vcs")
	if err != nil {
		return nil, sdk.WrapError(err, "repositoriesmanager.LoadAll> Unable to load services")
	}

	vcsServers := map[string]interface{}{}
	if _, err := services.DoJSONRequest(srvs, "GET", "/vcs", nil, &vcsServers); err != nil {
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
}

// GetProjectVCSServer returns sdk.ProjectVCSServer for a project
func GetProjectVCSServer(p *sdk.Project, name string) *sdk.ProjectVCSServer {
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

func (c *vcsConsumer) AuthorizeRedirect() (string, string, error) {
	srvDAO := services.Querier(c.dbFunc(), c.cache)
	srv, err := srvDAO.FindByType("vcs")
	if err != nil {
		return "", "", err
	}

	res := map[string]string{}
	path := fmt.Sprintf("/vcs/%s/authorize", c.name)
	log.Info("Performing request on %s", path)
	if _, err := services.DoJSONRequest(srv, "GET", path, nil, &res); err != nil {
		return "", "", sdk.WrapError(err, "repositoriesmanager.AuthorizeRedirect> ")
	}

	return res["token"], res["url"], nil
}

func (c *vcsConsumer) AuthorizeToken(token string, secret string) (string, string, error) {
	srvDAO := services.Querier(c.dbFunc(), c.cache)
	srv, err := srvDAO.FindByType("vcs")
	if err != nil {
		return "", "", err
	}

	body := map[string]string{
		"token":  token,
		"secret": secret,
	}

	res := map[string]string{}
	path := fmt.Sprintf("/vcs/%s/authorize", c.name)
	if _, err := services.DoJSONRequest(srv, "POST", path, body, &res); err != nil {
		return "", "", err
	}

	return res["token"], res["secret"], nil
}

func (c *vcsConsumer) GetAuthorizedClient(token string, secret string) (sdk.VCSAuthorizedClient, error) {
	s := GetProjectVCSServer(c.proj, c.name)
	if s == nil {
		return nil, sdk.ErrNoReposManagerClientAuth
	}

	servicesDao := services.Querier(c.dbFunc(), c.cache)
	srvs, err := servicesDao.FindByType("vcs")
	if err != nil {
		return nil, err
	}

	return &vcsClient{
		name:   c.name,
		token:  token,
		secret: secret,
		srvs:   srvs,
	}, nil
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
		req.Header.Set("X-CDS-ACCESS-TOKEN", base64.StdEncoding.EncodeToString([]byte(c.token)))
		req.Header.Set("X-CDS-ACCESS-TOKEN-SECRET", base64.StdEncoding.EncodeToString([]byte(c.secret)))
	})
}

func (c *vcsClient) postMultipart(path string, fileContent []byte, out interface{}) (int, error) {
	return services.PostMultipart(c.srvs, "POST", path, fileContent, out, func(req *http.Request) {
		req.Header.Set("X-CDS-ACCESS-TOKEN", base64.StdEncoding.EncodeToString([]byte(c.token)))
		req.Header.Set("X-CDS-ACCESS-TOKEN-SECRET", base64.StdEncoding.EncodeToString([]byte(c.secret)))
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

func (c *vcsClient) Branches(fullname string) ([]sdk.VCSBranch, error) {
	branches := []sdk.VCSBranch{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/branches", c.name, fullname)
	if _, err := c.doJSONRequest("GET", path, nil, &branches); err != nil {
		return nil, err
	}
	return branches, nil
}

func (c *vcsClient) Branch(fullname string, branchName string) (*sdk.VCSBranch, error) {
	branch := sdk.VCSBranch{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/branches/?branch=%s", c.name, fullname, url.QueryEscape(branchName))
	if _, err := c.doJSONRequest("GET", path, nil, &branch); err != nil {
		return nil, err
	}
	return &branch, nil
}

func (c *vcsClient) Commits(fullname, branch, since, until string) ([]sdk.VCSCommit, error) {
	commits := []sdk.VCSCommit{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/branches/commits?branch=%s&since=%s&until=%s", c.name, fullname, url.QueryEscape(branch), url.QueryEscape(since), url.QueryEscape(until))
	if _, err := c.doJSONRequest("GET", path, nil, &commits); err != nil {
		return nil, err
	}
	return commits, nil
}

func (c *vcsClient) Commit(fullname, hash string) (sdk.VCSCommit, error) {
	commit := sdk.VCSCommit{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/commits/%s", c.name, fullname, hash)
	if _, err := c.doJSONRequest("GET", path, nil, &commit); err != nil {
		return commit, err
	}
	return commit, nil
}

func (c *vcsClient) PullRequests(fullname string) ([]sdk.VCSPullRequest, error) {
	prs := []sdk.VCSPullRequest{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/pullrequests", c.name, fullname)
	if _, err := c.doJSONRequest("GET", path, nil, &prs); err != nil {
		return nil, err
	}
	return prs, nil
}

func (c *vcsClient) CreateHook(fullname string, hook sdk.VCSHook) error {
	path := fmt.Sprintf("/vcs/%s/repos/%s/hooks/", c.name, fullname)
	_, err := c.doJSONRequest("POST", path, &hook, nil)
	return err
}

func (c *vcsClient) GetHook(fullname, u string) (sdk.VCSHook, error) {
	path := fmt.Sprintf("/vcs/%s/repos/%s/hooks/?url=%s", c.name, fullname, url.QueryEscape(u))
	hook := &sdk.VCSHook{}
	_, err := c.doJSONRequest("GET", path, nil, hook)
	return *hook, err
}

func (c *vcsClient) UpdateHook(fullname, url string, hook sdk.VCSHook) error {
	return nil
}

func (c *vcsClient) DeleteHook(fullname string, hook sdk.VCSHook) error {
	path := fmt.Sprintf("/vcs/%s/repos/%s/hooks/?url=%s", c.name, fullname, url.QueryEscape(hook.URL))
	_, err := c.doJSONRequest("DELETE", path, nil, nil)
	return err
}

func (c *vcsClient) GetEvents(fullname string, dateRef time.Time) ([]interface{}, time.Duration, error) {
	res := struct {
		Events []interface{} `json:"events"`
		Delay  time.Duration `json:"delay"`
	}{}

	path := fmt.Sprintf("/vcs/%s/repos/%s/events?since=%d", c.name, fullname, dateRef.Unix())
	if _, err := c.doJSONRequest("GET", path, nil, &res); err != nil {
		return nil, time.Duration(0), err
	}

	return res.Events, res.Delay, nil
}

func (c *vcsClient) PushEvents(fullname string, evts []interface{}) ([]sdk.VCSPushEvent, error) {
	events := []sdk.VCSPushEvent{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/events?filter=push", c.name, fullname)
	if _, err := c.doJSONRequest("POST", path, evts, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (c *vcsClient) CreateEvents(fullname string, evts []interface{}) ([]sdk.VCSCreateEvent, error) {
	events := []sdk.VCSCreateEvent{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/events?filter=create", c.name, fullname)
	if _, err := c.doJSONRequest("POST", path, evts, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (c *vcsClient) DeleteEvents(fullname string, evts []interface{}) ([]sdk.VCSDeleteEvent, error) {
	events := []sdk.VCSDeleteEvent{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/events?filter=delete", c.name, fullname)
	if _, err := c.doJSONRequest("POST", path, evts, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (c *vcsClient) PullRequestEvents(fullname string, evts []interface{}) ([]sdk.VCSPullRequestEvent, error) {
	events := []sdk.VCSPullRequestEvent{}
	path := fmt.Sprintf("/vcs/%s/repos/%s/events?filter=pullrequests", c.name, fullname)
	if _, err := c.doJSONRequest("POST", path, evts, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (c *vcsClient) SetStatus(event sdk.Event) error {
	path := fmt.Sprintf("/vcs/%s/status", c.name)
	_, err := c.doJSONRequest("POST", path, event, nil)
	return err
}

func (c *vcsClient) Release(fullname, tagName, releaseTitle, releaseDescription string) (*sdk.VCSRelease, error) {
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
	_, err := c.doJSONRequest("POST", path, &res, &release)
	if err != nil {
		return nil, err
	}
	return &release, nil
}

func (c *vcsClient) UploadReleaseFile(fullname string, releaseName, uploadURL string, artifactName string, r io.ReadCloser) error {
	path := fmt.Sprintf("/vcs/%s/repos/%s/releases/%s/artifacts/%s", c.name, fullname, releaseName, artifactName)
	defer r.Close()

	fileContent, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	if _, err := c.postMultipart(path, fileContent, nil); err != nil {
		return err
	}
	return nil
}

// WebhooksInfos is a set of info about webhooks
type WebhooksInfos struct {
	WebhooksSupported         bool `json:"webhooks_supported"`
	WebhooksDisabled          bool `json:"webhooks_disabled"`
	WebhooksCreationSupported bool `json:"webhooks_creation_supported"`
	WebhooksCreationDisabled  bool `json:"webhooks_creation_disabled"`
}

// GetWebhooksInfos returns webhooks_supported, webhooks_disabled, webhooks_creation_supported, webhooks_creation_disabled for a vcs server
func GetWebhooksInfos(c sdk.VCSAuthorizedClient) (WebhooksInfos, error) {
	client, ok := c.(*vcsClient)
	if !ok {
		return WebhooksInfos{}, fmt.Errorf("Polling infos cast error")
	}
	res := WebhooksInfos{}
	path := fmt.Sprintf("/vcs/%s/webhooks", client.name)
	if _, err := client.doJSONRequest("GET", path, nil, &res); err != nil {
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
func GetPollingInfos(c sdk.VCSAuthorizedClient) (PollingInfos, error) {
	client, ok := c.(*vcsClient)
	if !ok {
		return PollingInfos{}, fmt.Errorf("Polling infos cast error")
	}
	res := PollingInfos{}
	path := fmt.Sprintf("/vcs/%s/polling", client.name)
	if _, err := client.doJSONRequest("GET", path, nil, &res); err != nil {
		return PollingInfos{}, err
	}
	return res, nil
}
