package sdk

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

//RepositoriesManagerType lists the different repositories manager currently planned to be supported
type RepositoriesManagerType string

const (
	//Stash is valued to "STASH"
	Stash RepositoriesManagerType = "STASH"
	//Github is valued to "GITHUB"
	Github RepositoriesManagerType = "GITHUB"
)

//RepositoriesManager is the struct for every repositories manager.
//It can be stored in CDS DB in repositories_manager table
type RepositoriesManager struct {
	ID               int64                     `json:"id"`
	Consumer         RepositoriesManagerDriver `json:"-"`
	Type             RepositoriesManagerType   `json:"type"`
	Name             string                    `json:"name"`
	URL              string                    `json:"url"`
	HooksSupported   bool                      `json:"hooks_supported"`
	PollingSupported bool                      `json:"polling_supported"`
}

//RepositoryPoller is an alternative to hooks
type RepositoryPoller struct {
	Name         string      `json:"name"`
	Application  Application `json:"application"`
	Pipeline     Pipeline    `json:"pipeline"`
	Enabled      bool        `json:"enabled"`
	DateCreation time.Time   `json:"date_creation"`
}

//RepositoriesManagerDriver is the consumer interface
type RepositoriesManagerDriver interface {
	AuthorizeRedirect() (string, string, error)
	AuthorizeToken(string, string) (string, string, error)
	GetAuthorized(string, string) (RepositoriesManagerClient, error)
	Data() string
	HooksSupported() bool
	PollingSupported() bool
}

//GetReposManager calls API to get list of repositories manager
func GetReposManager() ([]RepositoriesManager, error) {
	var rms []RepositoriesManager
	uri := fmt.Sprintf("/repositories_manager")

	data, code, err := Request("GET", uri, nil)
	if err != nil {
		return rms, err
	}

	if code >= 300 {
		return rms, fmt.Errorf("HTTP %d", code)
	}

	if err := json.Unmarshal(data, &rms); err != nil {
		return rms, err
	}
	return rms, nil
}

//AddReposManager add a new repositories manager in CDS
func AddReposManager(args map[string]string) (*RepositoriesManager, error) {
	var rm RepositoriesManager
	uri := fmt.Sprintf("/repositories_manager/add")
	b, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	data, code, err := Request("POST", uri, b)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("HTTP %d", code)
	}

	if err := json.Unmarshal(data, &rm); err != nil {
		return nil, err
	}
	return &rm, nil

}

//ConnectReposManager add a new repositories manager in CDS for a project
//It returns accessToken and authorize URL
func ConnectReposManager(key, name string) (string, string, error) {
	uri := fmt.Sprintf("/project/%s/repositories_manager/%s/authorize", key, name)

	data, code, err := Request("POST", uri, nil)
	if err != nil {
		return "", "", err
	}
	if code >= 300 {
		return "", "", fmt.Errorf("HTTP %d", code)
	}

	var r map[string]interface{}
	if err := json.Unmarshal(data, &r); err != nil {
		return "", "", err
	}

	return r["request_token"].(string), r["url"].(string), nil
}

//ConnectReposManagerCallback returns accessToken and accessTokenSecret
func ConnectReposManagerCallback(key, name, requestToken, verifier string) (string, string, error) {
	uri := fmt.Sprintf("/project/%s/repositories_manager/%s/authorize/callback", key, name)
	tv := map[string]string{
		"request_token": requestToken,
		"verifier":      verifier,
	}
	b, _ := json.Marshal(tv)
	data, code, err := Request("POST", uri, b)
	if err != nil {
		return "", "", err
	}
	if code >= 300 {
		return "", "", fmt.Errorf("HTTP %d", code)
	}

	var r map[string]interface{}
	if err := json.Unmarshal(data, &r); err != nil {
		return "", "", err
	}

	return r["access_token"].(string), r["access_token_secret"].(string), nil
}

//DisconnectReposManager removes access token for the project
func DisconnectReposManager(key, name string) error {
	uri := fmt.Sprintf("/project/%s/repositories_manager/%s", key, name)
	_, code, err := Request("DELETE", uri, nil)
	if err != nil {
		return err
	}
	if code >= 300 {
		return fmt.Errorf("HTTP %d", code)
	}
	return nil
}

//GetProjectReposManager returns connected repository manager for a specific project
func GetProjectReposManager(k string) ([]RepositoriesManager, error) {
	var rms []RepositoriesManager
	uri := fmt.Sprintf("/project/%s/repositories_manager", k)
	data, code, err := Request("GET", uri, nil)
	if err != nil {
		return rms, err
	}

	if code >= 300 {
		return rms, fmt.Errorf("HTTP %d", code)
	}

	if err := json.Unmarshal(data, &rms); err != nil {
		return rms, err
	}
	return rms, nil
}

//GetProjectReposFromReposManager returns the repositories
func GetProjectReposFromReposManager(k, n string) ([]VCSRepo, error) {
	var repos []VCSRepo
	uri := fmt.Sprintf("/project/%s/repositories_manager/%s/repos", k, n)
	data, code, err := Request("GET", uri, nil)
	if err != nil {
		return repos, err
	}

	if code >= 300 {
		return repos, fmt.Errorf("HTTP %d", code)
	}

	if err := json.Unmarshal(data, &repos); err != nil {
		return repos, err
	}
	return repos, nil
}

//AttachApplicationToReposistoriesManager attachs the application to the repo identified by its fullname in the reposManager
func AttachApplicationToReposistoriesManager(projectKey, appName, reposManager, repoFullname string) error {
	uri := fmt.Sprintf("/project/%s/repositories_manager/%s/application/%s/attach?fullname=%s", projectKey, reposManager, appName, url.QueryEscape(repoFullname))
	_, code, err := Request("POST", uri, nil)
	if err != nil {
		return err
	}
	if code >= 300 {
		return fmt.Errorf("HTTP %d", code)
	}
	return nil
}

//DetachApplicationToReposistoriesManager attachs the application from any reposManager
func DetachApplicationToReposistoriesManager(projectKey, appName, reposManager string) error {
	uri := fmt.Sprintf("/project/%s/repositories_manager/%s/application/%s/detach", projectKey, reposManager, appName)
	_, code, err := Request("POST", uri, nil)
	if err != nil {
		return err
	}
	if code >= 300 {
		return fmt.Errorf("HTTP %d", code)
	}
	return nil
}

//AddHookOnRepositoriesManager create hook on stash
func AddHookOnRepositoriesManager(projectKey, appName, reposManager, repoFullname, pipelineName string) error {
	uri := fmt.Sprintf("/project/%s/application/%s/repositories_manager/%s/hook", projectKey, appName, reposManager)
	data := map[string]string{
		"repository_fullname": repoFullname,
		"pipeline_name":       pipelineName,
	}
	b, _ := json.Marshal(data)
	_, code, err := Request("POST", uri, b)
	if err != nil {
		return err
	}
	if code >= 300 {
		return fmt.Errorf("HTTP %d", code)
	}
	return nil
}

//DeleteHookOnRepositoriesManager delete hook on stash
func DeleteHookOnRepositoriesManager(projectKey, appName, reposManager string, hookID int64) error {
	uri := fmt.Sprintf("/project/%s/application/%s/repositories_manager/%s/hook/%d", projectKey, appName, reposManager, hookID)
	_, code, err := Request("DELETE", uri, nil)
	if err != nil {
		return err
	}
	if code >= 300 {
		return fmt.Errorf("HTTP %d", code)
	}
	return nil
}

//AddApplicationFromReposManager create the application from the repofullname
func AddApplicationFromReposManager(projectkey, rmname, repoFullname string) error {
	data := map[string]string{
		"repository_fullname": repoFullname,
	}
	b, _ := json.Marshal(data)
	uri := fmt.Sprintf("/project/%s/repositories_manager/%s/application", projectkey, rmname)
	_, code, err := Request("POST", uri, b)
	if err != nil {
		return err
	}

	if code >= 300 {
		return fmt.Errorf("HTTP %d", code)
	}

	return nil
}

//RepositoriesManagerClient is the client interface
type RepositoriesManagerClient interface {
	//Repos
	Repos() ([]VCSRepo, error)
	RepoByFullname(fullname string) (VCSRepo, error)

	//Branches
	Branches(string) ([]VCSBranch, error)
	Branch(string, string) (VCSBranch, error)

	//Commits
	Commits(repo, branch, since, until string) ([]VCSCommit, error)
	Commit(repo, hash string) (VCSCommit, error)

	//Hooks
	CreateHook(repo, url string) error
	DeleteHook(repo, url string) error

	//Events
	PushEvents(repo string, dateRef time.Time) ([]VCSPushEvent, time.Duration, error)

	// Set build status on repository
	SetStatus(event Event) error
}

//VCSRepo represents data about repository even on stash, or github, etc...
type VCSRepo struct {
	ID           string `json:"id"`
	Name         string `json:"name"`     //On Github: Name = Slug
	Slug         string `json:"slug"`     //On Github: Slug = Name
	Fullname     string `json:"fullname"` //On Stash : projectkey/slug, on Github : owner/slug
	URL          string `json:"url"`      //Web URL
	HTTPCloneURL string `json:"http_url"` //Git clone URL  "https://<baseURL>/scm/PRJ/my-repo.git"
	SSHCloneURL  string `json:"ssh_url"`  //Git clone URL  "ssh://git@<baseURL>/PRJ/my-repo.git"
}

//VCSAuthor represents the auhor for every commit
type VCSAuthor struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Email       string `json:"emailAddress"`
	Avatar      string `json:"avatar"`
}

//VCSCommit represents the commit in the repository
type VCSCommit struct {
	Hash      string    `json:"id"`
	Author    VCSAuthor `json:"author"`
	Timestamp int64     `json:"authorTimestamp"`
	Message   string    `json:"message"`
	URL       string    `json:"url"`
}

//VCSBranch reprensents branches known by the repositories manager
type VCSBranch struct {
	ID           string `json:"id"`
	DisplayID    string `json:"display_id"`
	LatestCommit string `json:"latest_commit"`
	Default      bool   `json:"default"`
}

//VCSPushEvent represents a push events for polling
type VCSPushEvent struct {
	Branch VCSBranch `json:"branch"`
	Commit VCSCommit `json:"commit"`
}
