package sdk

import (
	"encoding/base64"
	"time"
)

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
	Error              *OperationError          `json:"error_details,omitempty"`
	DeprecatedError    string                   `json:"error,omitempty"`
	RepositoryInfo     *OperationRepositoryInfo `json:"repository_info,omitempty"`
	Date               *time.Time               `json:"date,omitempty"`
	User               struct {
		Username string `json:"username,omitempty"  db:"-" cli:"-"`
		Fullname string `json:"fullname,omitempty"  db:"-" cli:"-"`
		Email    string `json:"email,omitempty"  db:"-" cli:"-"`
	} `json:"user,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	NbRetries int    `json:"nb_retries,omitempty"`
}

type OperationError struct {
	ID         int    `json:"id"`
	Status     int    `json:"status,omitempty"`
	Message    string `json:"message"`
	StackTrace string `json:"stack_trace,omitempty"`
	From       string `json:"from,omitempty"`
}

func ToOperationError(err error) *OperationError {
	if err == nil {
		return nil
	}
	sdkError := ExtractHTTPError(err)
	return &OperationError{
		ID:         sdkError.ID,
		Status:     sdkError.Status,
		From:       sdkError.From,
		Message:    sdkError.Message,
		StackTrace: sdkError.StackTrace,
	}
}

func (opError *OperationError) ToError() error {
	if opError == nil {
		return nil
	}
	return &Error{
		ID:      opError.ID,
		Status:  opError.Status,
		Message: opError.Message,
		From:    opError.From,
	}
}

// OperationSetup is the setup for an operation basically its a checkout
type OperationSetup struct {
	Checkout OperationCheckout `json:"checkout,omitempty"`
	Push     OperationPush     `json:"push,omitempty"`
}

// OperationRepositoryInfo represents global information about the repository
type OperationRepositoryInfo struct {
	Name          string `json:"name,omitempty"`
	FetchURL      string `json:"fetch_url,omitempty"`
	DefaultBranch string `json:"default_branch,omitempty"`
}

// OperationLoadFiles represents files loading from a globbing pattern
type OperationLoadFiles struct {
	Pattern string            `json:"pattern,omitempty"`
	Results map[string][]byte `json:"results,omitempty"`
}

// OperationCheckout represents a smart git checkout
type OperationCheckout struct {
	Tag            string `json:"tag,omitempty"`
	Branch         string `json:"branch,omitempty"`
	Commit         string `json:"commit,omitempty"`
	CheckSignature bool   `json:"check_signature,omitempty"`
	Result         struct {
		SignKeyID      string `json:"sign_key_id"`
		CommitVerified bool   `json:"verified"`
		Msg            string `json:"msg"`
	} `json:"result"`
}

// OperationPush represents information about push operation
type OperationPush struct {
	FromBranch string `json:"from_branch,omitempty"`
	ToBranch   string `json:"to_branch,omitempty"`
	Message    string `json:"message,omitempty"`
	PRLink     string `json:"pr_link,omitempty"`
	Update     bool   `json:"update,omitempty"`
}

// OperationStatus is the status of an operation
type OperationStatus int

// There are the different OperationStatus values
const (
	OperationStatusPending OperationStatus = iota
	OperationStatusProcessing
	OperationStatusDone
	OperationStatusError
)

// OperationRepo is an operation
type OperationRepo struct {
	Basedir            string
	URL                string
	RepositoryStrategy RepositoryStrategy
}

// ID returns a generated ID for a Operation
func (r OperationRepo) ID() string {
	return base64.StdEncoding.EncodeToString([]byte(r.URL))
}
