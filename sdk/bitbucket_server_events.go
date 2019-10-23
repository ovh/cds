package sdk

type BitbucketServerWebhookEvent struct {
	EventKey string                `json:"eventKey"`
	Date     string                `json:"date"`
	Actor    *BitbucketServerActor `json:"actor"`

	// PR event data
	PullRequest         *BitbucketServerPullRequest    `json:"pullRequest"`
	PreviousTitle       string                         `json:"previousTitle"`
	PreviousDescription interface{}                    `json:"previousDescription"`
	PreviousTarget      *BitbucketServerPreviousTarget `json:"previousTarget"`

	// Review event data
	Participant    *BitbucketServerParticipant `json:"participant"`
	PreviousStatus string                      `json:"previousStatus"`

	// Reviewer edited data
	AddedReviewers   []BitbucketServerActor `json:"addedReviewers"`
	RemovedReviewers []BitbucketServerActor `json:"removedReviewers"`

	// PR Comment event data
	Comment         *BitbucketServerComment `json:"comment"`
	PreviousComment string                  `json:"previousComment"`

	// PushEvent data
	Repository *BitbucketServerRepository `json:"repository"`
	Changes    []BitbucketServerChange    `json:"changes"`
}
