package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

//RepositoryPoller is an alternative to hooks
type RepositoryPoller struct {
	Name          string                     `json:"name" db:"name"`
	ApplicationID int64                      `json:"-" db:"application_id"`
	PipelineID    int64                      `json:"-" db:"pipeline_id"`
	Application   Application                `json:"application" db:"-"`
	Pipeline      Pipeline                   `json:"pipeline" db:"-"`
	Enabled       bool                       `json:"enabled" db:"enabled"`
	DateCreation  time.Time                  `json:"date_creation" db:"date_creation"`
	NextExecution *RepositoryPollerExecution `json:"next_execution" db:"-"`
}

//RepositoryPollerExecution is a polling execution
type RepositoryPollerExecution struct {
	ID                    int64                 `json:"id" db:"id"`
	ApplicationID         int64                 `json:"-" db:"application_id"`
	PipelineID            int64                 `json:"-" db:"pipeline_id"`
	ExecutionPlannedDate  time.Time             `json:"execution_planned_date,omitempty" db:"execution_planned_date"`
	ExecutionDate         *time.Time            `json:"execution_date" db:"execution_date"`
	Executed              bool                  `json:"executed" db:"executed"`
	PipelineBuildVersions map[string]int64      `json:"pipeline_build_version" db:"-"`
	PushEvents            []VCSPushEvent        `json:"push_events" db:"-"`
	CreateEvents          []VCSCreateEvent      `json:"create_events" db:"-"`
	DeleteEvents          []VCSDeleteEvent      `json:"delete_events" db:"-"`
	PullRequestEvents     []VCSPullRequestEvent `json:"pullrequest_events" db:"-"`
	Error                 string                `json:"error" db:"error"`
}

//GetReposManager calls API to get list of repositories manager
func GetReposManager() ([]string, error) {
	var rms = []string{}
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
func GetProjectReposManager(k string) ([]ProjectVCSServer, error) {
	var rms []ProjectVCSServer
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
func DeleteHookOnRepositoriesManager(projectKey, appName string, hookID int64) error {
	uri := fmt.Sprintf("/project/%s/application/%s/repositories_manager/hook/%d", projectKey, appName, hookID)
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
	Branch(string, string) (*VCSBranch, error)

	//Commits
	Commits(repo, branch, since, until string) ([]VCSCommit, error)
	Commit(repo, hash string) (VCSCommit, error)

	// PullRequests
	PullRequests(string) ([]VCSPullRequest, error)

	//Hooks
	CreateHook(repo, url string) error
	DeleteHook(repo, url string) error

	//Events
	GetEvents(repo string, dateRef time.Time) ([]interface{}, time.Duration, error)
	PushEvents(string, []interface{}) ([]VCSPushEvent, error)
	CreateEvents(string, []interface{}) ([]VCSCreateEvent, error)
	DeleteEvents(string, []interface{}) ([]VCSDeleteEvent, error)
	PullRequestEvents(string, []interface{}) ([]VCSPullRequestEvent, error)

	// Set build status on repository
	SetStatus(event Event) error

	// Release
	Release(repo, tagName, releaseTitle, releaseDescription string) (*VCSRelease, error)
	UploadReleaseFile(repo string, release *VCSRelease, runArtifact WorkflowNodeRunArtifact, file *bytes.Buffer) error
}

// VCSRelease represents data about release on github, etc..
type VCSRelease struct {
	ID        int64  `json:"id"`
	UploadURL string `json:"upload_url"`
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

//VCSRemote represents remotes known by the repositories manager
type VCSRemote struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

//VCSBranch represents branches known by the repositories manager
type VCSBranch struct {
	ID           string   `json:"id"`
	DisplayID    string   `json:"display_id"`
	LatestCommit string   `json:"latest_commit"`
	Default      bool     `json:"default"`
	Parents      []string `json:"parents"`
}

//VCSPullRequest represents a pull request
type VCSPullRequest struct {
	URL    string       `json:"url"`
	User   VCSAuthor    `json:"user"`
	Head   VCSPushEvent `json:"head"`
	Base   VCSPushEvent `json:"base"`
	Branch VCSBranch    `json:"branch"`
}

//VCSPushEvent represents a push events for polling
type VCSPushEvent struct {
	Repo     string    `json:"repo"`
	Branch   VCSBranch `json:"branch"`
	Commit   VCSCommit `json:"commit"`
	CloneURL string    `json:"clone_url"`
}

//VCSCreateEvent represents a push events for polling
type VCSCreateEvent VCSPushEvent

//VCSDeleteEvent represents a push events for polling
type VCSDeleteEvent struct {
	Branch VCSBranch `json:"branch"`
}

//VCSPullRequestEvent represents a push events for polling
type VCSPullRequestEvent struct {
	Action string       `json:"action"` // opened | closed
	URL    string       `json:"url"`
	Repo   string       `json:"repo"`
	User   VCSAuthor    `json:"user"`
	Head   VCSPushEvent `json:"head"`
	Base   VCSPushEvent `json:"base"`
	Branch VCSBranch    `json:"branch"`
}

type VCSHook struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Disable     bool     `json:"disable"`
	Events      []string `json:"events"`
	Method      string   `json:"method"`
	URL         string   `json:"url"`
	ContentType string   `json:"content_type"`
	Body        string   `json:"body"`
	InsecureSSL bool     `json:"insecure_ssl"`
	UUID        string   `json:"uuid"`
}
