package forgejo

import "time"

// CommitStatusState holds the state of a CommitStatus.
// It can be "pending", "success", "error", "failure", and "warning".
type CommitStatusState string

const (
	StatusPending CommitStatusState = "pending"
	StatusSuccess CommitStatusState = "success"
	StatusError   CommitStatusState = "error"
	StatusFailure CommitStatusState = "failure"
	StatusWarning CommitStatusState = "warning"
)

// StateType is the state of an issue / PR (open, closed).
type StateType string

// ReviewStateType is the state of a review.
type ReviewStateType string

// --- User ---

// User represents a Forgejo user.
type User struct {
	ID        int64     `json:"id"`
	UserName  string    `json:"login"`
	LoginName string    `json:"login_name"`
	SourceID  int64     `json:"source_id"`
	FullName  string    `json:"full_name"`
	Email     string    `json:"email"`
	AvatarURL string    `json:"avatar_url"`
	HTMLURL   string    `json:"html_url"`
	Language  string    `json:"language"`
	IsAdmin   bool      `json:"is_admin"`
	LastLogin time.Time `json:"last_login"`
	Created   time.Time `json:"created"`
	IsActive  bool      `json:"active"`
}

// --- Commit types ---

// CommitMeta contains meta information of a commit in terms of API.
type CommitMeta struct {
	URL     string    `json:"url"`
	SHA     string    `json:"sha"`
	Created time.Time `json:"created"`
}

// CommitUser contains information of a user in the context of a commit.
type CommitUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Date  string `json:"date"`
}

// PayloadCommitVerification represents the GPG verification of a commit.
type PayloadCommitVerification struct {
	Verified  bool         `json:"verified"`
	Reason    string       `json:"reason"`
	Signature string       `json:"signature"`
	Payload   string       `json:"payload"`
	Signer    *PayloadUser `json:"signer"`
}

// RepoCommit contains information of a commit in the context of a repository.
type RepoCommit struct {
	URL          string                     `json:"url"`
	Author       *CommitUser                `json:"author"`
	Committer    *CommitUser                `json:"committer"`
	Message      string                     `json:"message"`
	Tree         *CommitMeta                `json:"tree"`
	Verification *PayloadCommitVerification `json:"verification"`
}

// CommitAffectedFiles stores information about files affected by the commit.
type CommitAffectedFiles struct {
	Filename string `json:"filename"`
	Status   string `json:"status"`
}

// CommitStats is statistics for a RepoCommit.
type CommitStats struct {
	Total     int64 `json:"total"`
	Additions int64 `json:"additions"`
	Deletions int64 `json:"deletions"`
}

// Commit contains information generated from a Git commit.
type Commit struct {
	SHA        string                 `json:"sha"`
	URL        string                 `json:"url"`
	HTMLURL    string                 `json:"html_url"`
	Created    time.Time              `json:"created"`
	RepoCommit *RepoCommit            `json:"commit"`
	Author     *User                  `json:"author"`
	Committer  *User                  `json:"committer"`
	Parents    []*CommitMeta          `json:"parents"`
	Files      []*CommitAffectedFiles `json:"files"`
	Stats      *CommitStats           `json:"stats"`
}

// --- Branch types ---

// PayloadUser represents the author or committer of a commit.
type PayloadUser struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	UserName string `json:"username"`
}

// PayloadCommit represents a commit (used in Branch, webhook payloads, etc.).
type PayloadCommit struct {
	ID           string                     `json:"id"`
	Message      string                     `json:"message"`
	URL          string                     `json:"url"`
	Author       *PayloadUser               `json:"author"`
	Committer    *PayloadUser               `json:"committer"`
	Verification *PayloadCommitVerification `json:"verification"`
	Timestamp    time.Time                  `json:"timestamp"`
	Added        []string                   `json:"added"`
	Removed      []string                   `json:"removed"`
	Modified     []string                   `json:"modified"`
}

// Branch represents a repository branch.
type Branch struct {
	Name                          string         `json:"name"`
	Commit                        *PayloadCommit `json:"commit"`
	Protected                     bool           `json:"protected"`
	RequiredApprovals             int64          `json:"required_approvals"`
	EnableStatusCheck             bool           `json:"enable_status_check"`
	StatusCheckContexts           []string       `json:"status_check_contexts"`
	UserCanPush                   bool           `json:"user_can_push"`
	UserCanMerge                  bool           `json:"user_can_merge"`
	EffectiveBranchProtectionName string         `json:"effective_branch_protection_name"`
}

// --- Repository ---

// Repository represents a repository.
type Repository struct {
	ID                            int64       `json:"id"`
	Owner                         *User       `json:"owner"`
	Name                          string      `json:"name"`
	FullName                      string      `json:"full_name"`
	Description                   string      `json:"description"`
	Empty                         bool        `json:"empty"`
	Private                       bool        `json:"private"`
	Fork                          bool        `json:"fork"`
	Template                      bool        `json:"template"`
	Parent                        *Repository `json:"parent"`
	Mirror                        bool        `json:"mirror"`
	Size                          int64       `json:"size"`
	Language                      string      `json:"language"`
	LanguagesURL                  string      `json:"languages_url"`
	HTMLURL                       string      `json:"html_url"`
	URL                           string      `json:"url"`
	Link                          string      `json:"link"`
	SSHURL                        string      `json:"ssh_url"`
	CloneURL                      string      `json:"clone_url"`
	OriginalURL                   string      `json:"original_url"`
	Website                       string      `json:"website"`
	Stars                         int64       `json:"stars_count"`
	Forks                         int64       `json:"forks_count"`
	Watchers                      int64       `json:"watchers_count"`
	OpenIssues                    int64       `json:"open_issues_count"`
	OpenPulls                     int64       `json:"open_pr_counter"`
	Releases                      int64       `json:"release_counter"`
	DefaultBranch                 string      `json:"default_branch"`
	Archived                      bool        `json:"archived"`
	ArchivedAt                    time.Time   `json:"archived_at"`
	Created                       time.Time   `json:"created_at"`
	Updated                       time.Time   `json:"updated_at"`
	HasIssues                     bool        `json:"has_issues"`
	HasWiki                       bool        `json:"has_wiki"`
	HasPullRequests               bool        `json:"has_pull_requests"`
	HasProjects                   bool        `json:"has_projects"`
	HasReleases                   bool        `json:"has_releases"`
	HasPackages                   bool        `json:"has_packages"`
	HasActions                    bool        `json:"has_actions"`
	IgnoreWhitespaceConflicts     bool        `json:"ignore_whitespace_conflicts"`
	AllowMerge                    bool        `json:"allow_merge_commits"`
	AllowRebase                   bool        `json:"allow_rebase"`
	AllowRebaseMerge              bool        `json:"allow_rebase_explicit"`
	AllowRebaseUpdate             bool        `json:"allow_rebase_update"`
	AllowSquash                   bool        `json:"allow_squash_merge"`
	AllowFastForwardOnly          bool        `json:"allow_fast_forward_only_merge"`
	DefaultDeleteBranchAfterMerge bool        `json:"default_delete_branch_after_merge"`
	DefaultMergeStyle             string      `json:"default_merge_style"`
	DefaultUpdateStyle            string      `json:"default_update_style"`
	DefaultAllowMaintainerEdit    bool        `json:"default_allow_maintainer_edit"`
	AvatarURL                     string      `json:"avatar_url"`
	Internal                      bool        `json:"internal"`
	MirrorInterval                string      `json:"mirror_interval"`
	MirrorUpdated                 time.Time   `json:"mirror_updated"`
	ObjectFormatName              string      `json:"object_format_name"`
	WikiBranch                    string      `json:"wiki_branch"`
	GloballyEditableWiki          bool        `json:"globally_editable_wiki"`
}

// --- Tag ---

// Tag represents a repository tag.
type Tag struct {
	Name       string      `json:"name"`
	Message    string      `json:"message"`
	ID         string      `json:"id"`
	Commit     *CommitMeta `json:"commit"`
	ZipballURL string      `json:"zipball_url"`
	TarballURL string      `json:"tarball_url"`
}

// AnnotatedTag represents a git annotated tag object.
type AnnotatedTag struct {
	Tag          string                     `json:"tag"`
	SHA          string                     `json:"sha"`
	URL          string                     `json:"url"`
	Message      string                     `json:"message"`
	Tagger       *CommitUser                `json:"tagger"`
	Object       *AnnotatedTagObject        `json:"object"`
	Verification *PayloadCommitVerification `json:"verification"`
}

// AnnotatedTagObject represents the object targeted by an annotated tag.
type AnnotatedTagObject struct {
	SHA  string `json:"sha"`
	Type string `json:"type"`
	URL  string `json:"url"`
}

// --- Pull Request ---

// PRBranchInfo information about a branch in the context of a PR.
type PRBranchInfo struct {
	Name       string      `json:"label"`
	Ref        string      `json:"ref"`
	Sha        string      `json:"sha"`
	RepoID     int64       `json:"repo_id"`
	Repository *Repository `json:"repo"`
}

// PullRequest represents a pull request.
type PullRequest struct {
	ID                  int64         `json:"id"`
	URL                 string        `json:"url"`
	Index               int64         `json:"number"`
	Poster              *User         `json:"user"`
	Title               string        `json:"title"`
	Body                string        `json:"body"`
	State               StateType     `json:"state"`
	IsLocked            bool          `json:"is_locked"`
	Comments            int64         `json:"comments"`
	HTMLURL             string        `json:"html_url"`
	DiffURL             string        `json:"diff_url"`
	PatchURL            string        `json:"patch_url"`
	Mergeable           bool          `json:"mergeable"`
	HasMerged           bool          `json:"merged"`
	Merged              *time.Time    `json:"merged_at"`
	MergedCommitID      *string       `json:"merge_commit_sha"`
	MergedBy            *User         `json:"merged_by"`
	AllowMaintainerEdit bool          `json:"allow_maintainer_edit"`
	Base                *PRBranchInfo `json:"base"`
	Head                *PRBranchInfo `json:"head"`
	MergeBase           string        `json:"merge_base"`
	Deadline            *time.Time    `json:"due_date"`
	Created             *time.Time    `json:"created_at"`
	Updated             *time.Time    `json:"updated_at"`
	Closed              *time.Time    `json:"closed_at"`
	Additions           int64         `json:"additions"`
	Deletions           int64         `json:"deletions"`
	ChangedFiles        int64         `json:"changed_files"`
	Draft               bool          `json:"draft"`
	ReviewComments      int64         `json:"review_comments"`
	PinOrder            int64         `json:"pin_order"`
	Flow                int64         `json:"flow"`
}

// --- Pull Review ---

// PullReview represents a pull request review.
type PullReview struct {
	ID                int64           `json:"id"`
	Reviewer          *User           `json:"user"`
	State             ReviewStateType `json:"state"`
	Body              string          `json:"body"`
	CommitID          string          `json:"commit_id"`
	Stale             bool            `json:"stale"`
	Official          bool            `json:"official"`
	Dismissed         bool            `json:"dismissed"`
	CodeCommentsCount int             `json:"comments_count"`
	Submitted         time.Time       `json:"submitted_at"`
	Updated           time.Time       `json:"updated_at"`
	HTMLURL           string          `json:"html_url"`
	HTMLPullURL       string          `json:"pull_request_url"`
}

// CreatePullReviewOptions are options to create a pull review.
type CreatePullReviewOptions struct {
	State    ReviewStateType           `json:"event"`
	Body     string                    `json:"body"`
	CommitID string                    `json:"commit_id"`
	Comments []CreatePullReviewComment `json:"comments"`
}

// CreatePullReviewComment represents a review comment for creation API.
type CreatePullReviewComment struct {
	Path       string `json:"path"`
	Body       string `json:"body"`
	OldLineNum int64  `json:"old_position"`
	NewLineNum int64  `json:"new_position"`
}

// --- Status ---

// Status holds a single Status of a single Commit.
type Status struct {
	ID          int64             `json:"id"`
	State       CommitStatusState `json:"status"`
	TargetURL   string            `json:"target_url"`
	Description string            `json:"description"`
	URL         string            `json:"url"`
	Context     string            `json:"context"`
	Creator     *User             `json:"creator"`
	Created     time.Time         `json:"created_at"`
	Updated     time.Time         `json:"updated_at"`
}

// CreateStatusOption holds the information needed to create a new CommitStatus for a Commit.
type CreateStatusOption struct {
	State       CommitStatusState `json:"state"`
	TargetURL   string            `json:"target_url"`
	Description string            `json:"description"`
	Context     string            `json:"context"`
}

// --- Contents ---

// FileLinksResponse contains the links for a repo's file.
type FileLinksResponse struct {
	Self    *string `json:"self"`
	GitURL  *string `json:"git"`
	HTMLURL *string `json:"html"`
}

// ContentsResponse contains information about a repo's entry's (dir, file, symlink, submodule) metadata and content.
type ContentsResponse struct {
	Name string `json:"name"`
	Path string `json:"path"`
	SHA  string `json:"sha"`
	Type string `json:"type"` // "file", "dir", "symlink", or "submodule"
	Size int64  `json:"size"`
	// `encoding` is populated when `type` is `file`, otherwise null
	Encoding *string `json:"encoding"`
	// `content` is populated when `type` is `file`, otherwise null
	Content *string `json:"content"`
	// `target` is populated when `type` is `symlink`, otherwise null
	Target          *string            `json:"target"`
	URL             *string            `json:"url"`
	HTMLURL         *string            `json:"html_url"`
	GitURL          *string            `json:"git_url"`
	DownloadURL     *string            `json:"download_url"`
	SubmoduleGitURL *string            `json:"submodule_git_url"`
	Links           *FileLinksResponse `json:"_links"`
	LastCommitSHA   string             `json:"last_commit_sha"`
	LastCommitWhen  time.Time          `json:"last_commit_when"`
}

// --- Pagination / List Options ---

// ListOptions base options for paginated requests.
type ListOptions struct {
	Page     int `json:"page"`
	PageSize int `json:"limit"`
}
