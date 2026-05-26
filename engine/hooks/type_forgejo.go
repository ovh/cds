package hooks

import "time"

// Payload https://codeberg.org/forgejo/forgejo/raw/branch/forgejo/modules/structs/hook.go
// Event https://codeberg.org/forgejo/forgejo/raw/branch/forgejo/modules/webhook/type.go
// Event Type https://codeberg.org/forgejo/forgejo/raw/branch/forgejo/services/webhook/shared/payloader.go

// ForgejoEventType represents the value of the X-Forgejo-Event-Type header.
type ForgejoEventType string

const (
	ForgejoEventTypeCreate                    ForgejoEventType = "create"
	ForgejoEventTypeDelete                    ForgejoEventType = "delete"
	ForgejoEventTypeFork                      ForgejoEventType = "fork"
	ForgejoEventTypePush                      ForgejoEventType = "push"
	ForgejoEventTypeIssues                    ForgejoEventType = "issues"
	ForgejoEventTypeIssueAssign               ForgejoEventType = "issue_assign"
	ForgejoEventTypeIssueLabel                ForgejoEventType = "issue_label"
	ForgejoEventTypeIssueMilestone            ForgejoEventType = "issue_milestone"
	ForgejoEventTypeIssueComment              ForgejoEventType = "issue_comment"
	ForgejoEventTypePullRequest               ForgejoEventType = "pull_request"
	ForgejoEventTypePullRequestAssign         ForgejoEventType = "pull_request_assign"
	ForgejoEventTypePullRequestLabel          ForgejoEventType = "pull_request_label"
	ForgejoEventTypePullRequestMilestone      ForgejoEventType = "pull_request_milestone"
	ForgejoEventTypePullRequestComment        ForgejoEventType = "pull_request_comment"
	ForgejoEventTypePullRequestReviewApproved ForgejoEventType = "pull_request_review_approved"
	ForgejoEventTypePullRequestReviewRejected ForgejoEventType = "pull_request_review_rejected"
	ForgejoEventTypePullRequestReviewComment  ForgejoEventType = "pull_request_review_comment"
	ForgejoEventTypePullRequestSync           ForgejoEventType = "pull_request_sync"
	ForgejoEventTypePullRequestReviewRequest  ForgejoEventType = "pull_request_review_request"
	ForgejoEventTypeWiki                      ForgejoEventType = "wiki"
	ForgejoEventTypeRepository                ForgejoEventType = "repository"
	ForgejoEventTypeRelease                   ForgejoEventType = "release"
	ForgejoEventTypePackage                   ForgejoEventType = "package"
	ForgejoEventTypeSchedule                  ForgejoEventType = "schedule"
	ForgejoEventTypeWorkflowDispatch          ForgejoEventType = "workflow_dispatch"
	ForgejoEventTypeActionRunFailure          ForgejoEventType = "action_run_failure"
	ForgejoEventTypeActionRunRecover          ForgejoEventType = "action_run_recover"
	ForgejoEventTypeActionRunSuccess          ForgejoEventType = "action_run_success"
)

// ForgejoEvent represents the value of the X-Forgejo-Event header (aggregated event name).
type ForgejoEvent string

const (
	ForgejoEventCreate              ForgejoEvent = "create"
	ForgejoEventDelete              ForgejoEvent = "delete"
	ForgejoEventFork                ForgejoEvent = "fork"
	ForgejoEventPush                ForgejoEvent = "push"
	ForgejoEventIssues              ForgejoEvent = "issues"
	ForgejoEventIssueComment        ForgejoEvent = "issue_comment"
	ForgejoEventPullRequest         ForgejoEvent = "pull_request"
	ForgejoEventPullRequestApproved ForgejoEvent = "pull_request_approved"
	ForgejoEventPullRequestRejected ForgejoEvent = "pull_request_rejected"
	ForgejoEventPullRequestComment  ForgejoEvent = "pull_request_comment"
	ForgejoEventWiki                ForgejoEvent = "wiki"
	ForgejoEventRepository          ForgejoEvent = "repository"
	ForgejoEventRelease             ForgejoEvent = "release"
	ForgejoEventPackage             ForgejoEvent = "package"
	ForgejoEventActionRunFailure    ForgejoEvent = "action_run_failure"
	ForgejoEventActionRunRecover    ForgejoEvent = "action_run_recover"
	ForgejoEventActionRunSuccess    ForgejoEvent = "action_run_success"
)

// ForgejoPushPayload represents a Forgejo push webhook event payload.
type ForgejoPushPayload struct {
	Ref          string          `json:"ref"`
	Before       string          `json:"before"`
	After        string          `json:"after"`
	CompareURL   string          `json:"compare_url"`
	Commits      []ForgejoCommit `json:"commits"`
	TotalCommits int             `json:"total_commits"`
	HeadCommit   *ForgejoCommit  `json:"head_commit"`
	Repository   *ForgejoRepo    `json:"repository"`
	Pusher       *ForgejoUser    `json:"pusher"`
	Sender       *ForgejoUser    `json:"sender"`
}

type HookIssueAction string

const (
	// HookIssueOpened opened
	HookIssueOpened HookIssueAction = "opened"
	// HookIssueClosed closed
	HookIssueClosed HookIssueAction = "closed"
	// HookIssueReOpened reopened
	HookIssueReOpened HookIssueAction = "reopened"
	// HookIssueEdited edited
	HookIssueEdited HookIssueAction = "edited"
	// HookIssueAssigned assigned
	HookIssueAssigned HookIssueAction = "assigned"
	// HookIssueUnassigned unassigned
	HookIssueUnassigned HookIssueAction = "unassigned"
	// HookIssueLabelUpdated label_updated
	HookIssueLabelUpdated HookIssueAction = "label_updated"
	// HookIssueLabelCleared label_cleared
	HookIssueLabelCleared HookIssueAction = "label_cleared"
	// HookIssueSynchronized synchronized
	HookIssueSynchronized HookIssueAction = "synchronized"
	// HookIssueMilestoned is an issue action for when a milestone is set on an issue.
	HookIssueMilestoned HookIssueAction = "milestoned"
	// HookIssueDemilestoned is an issue action for when a milestone is cleared on an issue.
	HookIssueDemilestoned HookIssueAction = "demilestoned"
	// HookIssueReviewed is an issue action for when a pull request is reviewed
	HookIssueReviewed HookIssueAction = "reviewed"
	// HookIssueReviewRequested is an issue action for when a reviewer is requested for a pull request.
	HookIssueReviewRequested HookIssueAction = "review_requested"
	// HookIssueReviewRequestRemoved is an issue action for removing a review request to someone on a pull request.
	HookIssueReviewRequestRemoved HookIssueAction = "review_request_removed"
)

// ForgejoPullRequestPayload represents a Forgejo pull_request webhook event payload.
type ForgejoPullRequestPayload struct {
	Action            HookIssueAction `json:"action"`
	Number            int64           `json:"number"`
	Changes           *ForgejoChanges `json:"changes,omitempty"`
	PullRequest       *ForgejoPR      `json:"pull_request"`
	RequestedReviewer *ForgejoUser    `json:"requested_reviewer"`
	Repository        *ForgejoRepo    `json:"repository"`
	Sender            *ForgejoUser    `json:"sender"`
	CommitID          string          `json:"commit_id"`
	Review            *ForgejoReview  `json:"review,omitempty"`
	Label             *ForgejoLabel   `json:"label,omitempty"`
}

// HookIssueCommentAction defines hook issue comment action
type HookIssueCommentAction string

// all issue comment actions
const (
	HookIssueCommentCreated HookIssueCommentAction = "created"
	HookIssueCommentEdited  HookIssueCommentAction = "edited"
	HookIssueCommentDeleted HookIssueCommentAction = "deleted"
)

// ForgejoIssueCommentPayload represents a Forgejo issue_comment webhook event payload.
type ForgejoIssueCommentPayload struct {
	Action      HookIssueCommentAction `json:"action"`
	Issue       *ForgejoIssue          `json:"issue"`
	PullRequest *ForgejoPR             `json:"pull_request,omitempty"`
	Comment     *ForgejoComment        `json:"comment"`
	Changes     *ForgejoChanges        `json:"changes,omitempty"`
	Repository  *ForgejoRepo           `json:"repository"`
	Sender      *ForgejoUser           `json:"sender"`
	IsPull      bool                   `json:"is_pull"`
}

// ForgejoCommit represents a commit in a Forgejo webhook payload.
type ForgejoCommit struct {
	ID           string                     `json:"id"`
	Message      string                     `json:"message"`
	URL          string                     `json:"url"`
	Author       *ForgejoCommitUser         `json:"author"`
	Committer    *ForgejoCommitUser         `json:"committer"`
	Verification *ForgejoCommitVerification `json:"verification"`
	Timestamp    time.Time                  `json:"timestamp"`
	Added        []string                   `json:"added"`
	Removed      []string                   `json:"removed"`
	Modified     []string                   `json:"modified"`
}

// ForgejoCommitVerification represents the GPG verification of a commit.
type ForgejoCommitVerification struct {
	Verified  bool               `json:"verified"`
	Reason    string             `json:"reason"`
	Signature string             `json:"signature"`
	Signer    *ForgejoCommitUser `json:"signer"`
	Payload   string             `json:"payload"`
}

// ForgejoCommitUser represents the author or committer of a commit.
type ForgejoCommitUser struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

// ForgejoUser represents a Forgejo user in a webhook payload.
type ForgejoUser struct {
	ID                int64     `json:"id"`
	Login             string    `json:"login"`
	LoginName         string    `json:"login_name"`
	FullName          string    `json:"full_name"`
	Email             string    `json:"email"`
	AvatarURL         string    `json:"avatar_url"`
	Language          string    `json:"language"`
	IsAdmin           bool      `json:"is_admin"`
	LastLogin         time.Time `json:"last_login"`
	Created           time.Time `json:"created"`
	Restricted        bool      `json:"restricted"`
	Active            bool      `json:"active"`
	ProhibitLogin     bool      `json:"prohibit_login"`
	Location          string    `json:"location"`
	Website           string    `json:"website"`
	Description       string    `json:"description"`
	Visibility        string    `json:"visibility"`
	FollowersCount    int       `json:"followers_count"`
	FollowingCount    int       `json:"following_count"`
	StarredReposCount int       `json:"starred_repos_count"`
	Username          string    `json:"username"`
}

// ForgejoRepo represents a Forgejo repository in a webhook payload.
type ForgejoRepo struct {
	ID              int64       `json:"id"`
	Owner           ForgejoUser `json:"owner"`
	Name            string      `json:"name"`
	FullName        string      `json:"full_name"`
	Description     string      `json:"description"`
	Empty           bool        `json:"empty"`
	Private         bool        `json:"private"`
	Fork            bool        `json:"fork"`
	Template        bool        `json:"template"`
	Mirror          bool        `json:"mirror"`
	Size            int         `json:"size"`
	Language        string      `json:"language"`
	LanguagesURL    string      `json:"languages_url"`
	HTMLURL         string      `json:"html_url"`
	URL             string      `json:"url"`
	Link            string      `json:"link"`
	SSHURL          string      `json:"ssh_url"`
	CloneURL        string      `json:"clone_url"`
	OriginalURL     string      `json:"original_url"`
	Website         string      `json:"website"`
	StarsCount      int         `json:"stars_count"`
	ForksCount      int         `json:"forks_count"`
	WatchersCount   int         `json:"watchers_count"`
	OpenIssuesCount int         `json:"open_issues_count"`
	OpenPrCounter   int         `json:"open_pr_counter"`
	ReleaseCounter  int         `json:"release_counter"`
	DefaultBranch   string      `json:"default_branch"`
	Archived        bool        `json:"archived"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
	Permissions     struct {
		Admin bool `json:"admin"`
		Push  bool `json:"push"`
		Pull  bool `json:"pull"`
	} `json:"permissions"`
	HasIssues       bool   `json:"has_issues"`
	HasWiki         bool   `json:"has_wiki"`
	HasPullRequests bool   `json:"has_pull_requests"`
	HasProjects     bool   `json:"has_projects"`
	HasReleases     bool   `json:"has_releases"`
	HasPackages     bool   `json:"has_packages"`
	HasActions      bool   `json:"has_actions"`
	AvatarURL       string `json:"avatar_url"`
	Internal        bool   `json:"internal"`
}

// ForgejoPR represents a pull request in a Forgejo webhook payload.
type ForgejoPR struct {
	ID                  int64             `json:"id"`
	URL                 string            `json:"url"`
	Number              int64             `json:"number"`
	User                *ForgejoUser      `json:"user"`
	Title               string            `json:"title"`
	Body                string            `json:"body"`
	Labels              []ForgejoLabel    `json:"labels"`
	Milestone           *ForgejoMilestone `json:"milestone"`
	Assignee            *ForgejoUser      `json:"assignee"`
	Assignees           []*ForgejoUser    `json:"assignees"`
	RequestedReviewers  []*ForgejoUser    `json:"requested_reviewers"`
	State               string            `json:"state"`
	IsLocked            bool              `json:"is_locked"`
	Comments            int               `json:"comments"`
	HTMLURL             string            `json:"html_url"`
	DiffURL             string            `json:"diff_url"`
	PatchURL            string            `json:"patch_url"`
	Mergeable           bool              `json:"mergeable"`
	Merged              bool              `json:"merged"`
	MergedAt            *time.Time        `json:"merged_at"`
	MergeCommitSha      string            `json:"merge_commit_sha"`
	MergedBy            *ForgejoUser      `json:"merged_by"`
	AllowMaintainerEdit bool              `json:"allow_maintainer_edit"`
	Base                *ForgejoPRRef     `json:"base"`
	Head                *ForgejoPRRef     `json:"head"`
	MergeBase           string            `json:"merge_base"`
	DueDate             *time.Time        `json:"due_date"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
	ClosedAt            *time.Time        `json:"closed_at"`
}

// ForgejoPRRef represents a pull request branch reference (base or head).
type ForgejoPRRef struct {
	Label  string       `json:"label"`
	Ref    string       `json:"ref"`
	Sha    string       `json:"sha"`
	RepoID int64        `json:"repo_id"`
	Repo   *ForgejoRepo `json:"repo"`
}

// ForgejoIssue represents an issue in a Forgejo webhook payload.
type ForgejoIssue struct {
	ID               int64             `json:"id"`
	URL              string            `json:"url"`
	HTMLURL          string            `json:"html_url"`
	Number           int64             `json:"number"`
	User             *ForgejoUser      `json:"user"`
	OriginalAuthor   string            `json:"original_author"`
	OriginalAuthorID int64             `json:"original_author_id"`
	Title            string            `json:"title"`
	Body             string            `json:"body"`
	Ref              string            `json:"ref"`
	Labels           []ForgejoLabel    `json:"labels"`
	Milestone        *ForgejoMilestone `json:"milestone"`
	Assignee         *ForgejoUser      `json:"assignee"`
	Assignees        []*ForgejoUser    `json:"assignees"`
	State            string            `json:"state"`
	IsLocked         bool              `json:"is_locked"`
	Comments         int               `json:"comments"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	ClosedAt         *time.Time        `json:"closed_at"`
	DueDate          *time.Time        `json:"due_date"`
	PullRequest      *ForgejoPR        `json:"pull_request,omitempty"`
}

// ForgejoComment represents a comment in a Forgejo webhook payload.
type ForgejoComment struct {
	ID               int64        `json:"id"`
	HTMLURL          string       `json:"html_url"`
	PRURL            string       `json:"pull_request_url"`
	IssueURL         string       `json:"issue_url"`
	User             *ForgejoUser `json:"user"`
	OriginalAuthor   string       `json:"original_author"`
	OriginalAuthorID int64        `json:"original_author_id"`
	Body             string       `json:"body"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

// ForgejoLabel represents a label in a Forgejo webhook payload.
type ForgejoLabel struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

// ForgejoMilestone represents a milestone in a Forgejo webhook payload.
type ForgejoMilestone struct {
	ID           int64      `json:"id"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	State        string     `json:"state"`
	OpenIssues   int        `json:"open_issues"`
	ClosedIssues int        `json:"closed_issues"`
	DueOn        *time.Time `json:"due_on"`
	ClosedAt     *time.Time `json:"closed_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// ForgejoReview represents a review payload in a Forgejo webhook event.
type ForgejoReview struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// ForgejoChanges represents changes in a Forgejo webhook payload (for edit events).
type ForgejoChanges struct {
	Title *ForgejoChangesFrom `json:"title,omitempty"`
	Body  *ForgejoChangesFrom `json:"body,omitempty"`
	Ref   *ForgejoChangesFrom `json:"ref,omitempty"`
}

// ForgejoChangesFrom represents the previous value of a changed field.
type ForgejoChangesFrom struct {
	From string `json:"from"`
}
