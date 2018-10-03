package sdk

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

// RepositoryEvents group all repostiory events
type RepositoryEvents struct {
	PushEvents        []VCSPushEvent        `json:"push_events" db:"-"`
	CreateEvents      []VCSCreateEvent      `json:"create_events" db:"-"`
	DeleteEvents      []VCSDeleteEvent      `json:"delete_events" db:"-"`
	PullRequestEvents []VCSPullRequestEvent `json:"pullrequest_events" db:"-"`
}

//RepositoryPollerExecution is a polling execution
type RepositoryPollerExecution struct {
	ID                    int64            `json:"id" db:"id"`
	ApplicationID         int64            `json:"-" db:"application_id"`
	PipelineID            int64            `json:"-" db:"pipeline_id"`
	ExecutionPlannedDate  time.Time        `json:"execution_planned_date,omitempty" db:"execution_planned_date"`
	ExecutionDate         *time.Time       `json:"execution_date" db:"execution_date"`
	Executed              bool             `json:"executed" db:"executed"`
	PipelineBuildVersions map[string]int64 `json:"pipeline_build_version" db:"-"`
	Error                 string           `json:"error" db:"error"`
	RepositoryEvents
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

//VCSTag represents branches known by the repositories manager
type VCSTag struct {
	Tag     string    `json:"tag"`
	Sha     string    `json:"sha"` // Represent sha of tag
	Message string    `json:"message"`
	Tagger  VCSAuthor `json:"tagger"`
	Hash    string    `json:"hash"` // Represent hash of commit
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
	ID   int          `json:"id"`
	URL  string       `json:"url"`
	User VCSAuthor    `json:"user"`
	Head VCSPushEvent `json:"head"`
	Base VCSPushEvent `json:"base"`
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

// VCSHook represents a hook on a VCS repository
type VCSHook struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Events      []string `json:"events"`
	Method      string   `json:"method"`
	URL         string   `json:"url"`
	ContentType string   `json:"content_type"`
	Body        string   `json:"body"`
	Disable     bool     `json:"disable"`
	InsecureSSL bool     `json:"insecure_ssl"`
	Workflow    bool     `json:"workflow"`
}

// VCSCommitStatus represents a status on a VCS repository
type VCSCommitStatus struct {
	Ref        string    `json:"ref"`
	CreatedAt  time.Time `json:"created_at"`
	State      string    `json:"state"`
	Decription string    `json:"description"`
}
