package sdk

import "encoding/base64"

// Operation is the main business object use in repositories service
type Operation struct {
	UUID               string                   `json:"uuid"`
	VCSServer          string                   `json:"vcs_server,omitempty"`
	RepoFullName       string                   `json:"repo_fullname,omitempty"`
	URL                string                   `json:"url"`
	RepositoryStrategy RepositoryStrategy       `json:"strategy,omitempty"`
	Setup              OperationSetup           `json:"setup,omitempty"`
	LoadFiles          OperationLoadFiles       `json:"load_files,omitempty"`
	Status             OperationStatus          `json:"status"`
	Error              string                   `json:"error,omitempty"`
	RepositoryInfo     *OperationRepositoryInfo `json:"repository_info,omitempty"`
}

type OperationSetup struct {
	Checkout OperationCheckout `json:"checkout,omitempty"`
}

type OperationRepositoryInfo struct {
	Name          string `json:"name,omitempty"`
	FetchURL      string `json:"fetch_url,omitempty"`
	DefaultBranch string `json:"default_branch,omitempty"`
}

type OperationLoadFiles struct {
	Pattern string            `json:"pattern,omitempty"`
	Results map[string][]byte `json:"results,omitempty"`
}

type OperationCheckout struct {
	Branch string `json:"branch,omitempty"`
	Commit string `json:"commit,omitempty"`
}

type OperationStatus int

const (
	OperationStatusPending OperationStatus = iota
	OperationStatusProcessing
	OperationStatusDone
	OperationStatusError
)

type OperationRepo struct {
	Basedir            string
	URL                string
	RepositoryStrategy RepositoryStrategy
}

func (r OperationRepo) ID() string {
	return base64.StdEncoding.EncodeToString([]byte(r.URL))
}
