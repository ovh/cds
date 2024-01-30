package sdk

type BitbucketServerPullRequest struct {
	ID          int                    `json:"id"`
	Version     int                    `json:"version"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	State       string                 `json:"state"`
	Open        bool                   `json:"open"`
	Closed      bool                   `json:"closed"`
	CreatedDate int                    `json:"createdDate"`
	UpdatedDate int                    `json:"updatedDate"`
	FromRef     BitbucketServerRef     `json:"fromRef"`
	ToRef       BitbucketServerRef     `json:"toRef"`
	Locked      bool                   `json:"locked"`
	Author      *BitbucketServerAuthor `json:"author,omitempty"`
	Reviewers   []struct {
		User               BitbucketServerActor `json:"user"`
		LastReviewedCommit string               `json:"lastReviewedCommit"`
		Role               string               `json:"role"`
		Approved           bool                 `json:"approved"`
		Status             string               `json:"status"`
	} `json:"reviewers,omitempty"`
	Participants []BitbucketServerParticipant `json:"participants,omitempty"`
	Properties   PullRequestProperties        `json:"properties,omitempty"`
	Links        struct {
		Self []struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"links"`
}

type PullRequestProperties struct {
	MergeCommit MergeCommitProperty `json:"mergeCommit"`
}

type MergeCommitProperty struct {
	ID        string `json:"id"`
	DisplayID string `json:"displayId"`
}

type BitbucketServerParticipant struct {
	User               BitbucketServerActor `json:"user"`
	LastReviewedCommit string               `json:"lastReviewedCommit"`
	Role               string               `json:"role"`
	Approved           bool                 `json:"approved"`
	Status             string               `json:"status"`
}

type BitbucketServerRef struct {
	ID           string                    `json:"id"`
	DisplayID    string                    `json:"displayId"`
	LatestCommit string                    `json:"latestCommit"`
	Repository   BitbucketServerRepository `json:"repository"`
	Type         string                    `json:"type"`
}

type BitbucketServerAuthor struct {
	User     BitbucketServerActor `json:"user"`
	Role     string               `json:"role"`
	Approved bool                 `json:"approved"`
	Status   string               `json:"status"`
}

type BitbucketServerRepository struct {
	ID            int64                  `json:"id"`
	Slug          string                 `json:"slug"`
	Name          interface{}            `json:"name"`
	ScmID         string                 `json:"scmId"`
	State         string                 `json:"state"`
	StatusMessage string                 `json:"statusMessage"`
	Forkable      bool                   `json:"forkable"`
	Project       BitbucketServerProject `json:"project"`
	Public        bool                   `json:"public"`
}

type BitbucketServerProject struct {
	Key    string `json:"key"`
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Public bool   `json:"public"`
	Type   string `json:"type"`
}

type BitbucketServerActor struct {
	Name         string `json:"name"`
	EmailAddress string `json:"emailAddress"`
	ID           int    `json:"id"`
	DisplayName  string `json:"displayName"`
	Active       bool   `json:"active"`
	Slug         string `json:"slug"`
	Type         string `json:"type"`
}

type BitbucketServerChange struct {
	Ref      BitbucketServerRef `json:"ref"`
	RefID    string             `json:"refId"`
	FromHash string             `json:"fromHash"`
	ToHash   string             `json:"toHash"`
	Type     string             `json:"type"`
}

type BitbucketServerPreviousTarget struct {
	ID              string `json:"id"`
	DisplayID       string `json:"displayId"`
	Type            string `json:"type"`
	LatestCommit    string `json:"latestCommit"`
	LatestChangeset string `json:"latestChangeset"`
}

type BitbucketServerComment struct {
	Properties  BitbucketServerProperties `json:"properties"`
	ID          int64                     `json:"id"`
	Version     int64                     `json:"version"`
	Text        string                    `json:"text"`
	Author      BitbucketServerActor      `json:"author"`
	CreatedDate int64                     `json:"createdDate"`
	UpdatedDate int64                     `json:"updatedDate"`
	Comments    []string                  `json:"comments"`
	Tasks       []string                  `json:"tasks"`
}

type BitbucketServerProperties struct {
	RepositoryID int64 `json:"repositoryId"`
}
