package sdk

// BitbucketServerPushEvent represents payload send by bitbucket server on a push event
type BitbucketServerPushEvent struct {
	EventKey   string                    `json:"eventKey"`
	Date       string                    `json:"date"`
	Actor      BitbucketServerActor      `json:"actor"`
	Repository BitbucketServerRepository `json:"repository"`
	Changes    []BitbucketServerChange   `json:"changes"`
}

type BitbucketServerPROpenedEvent struct {
	EventKey    string                     `json:"eventKey"`
	Date        string                     `json:"date"`
	Actor       BitbucketServerActor       `json:"actor"`
	PullRequest BitbucketServerPullRequest `json:"pullRequest"`
}

type BitbucketServerPRModifiedEvent struct {
	EventKey            string                        `json:"eventKey"`
	Date                string                        `json:"date"`
	Actor               BitbucketServerActor          `json:"actor"`
	PullRequest         BitbucketServerPullRequest    `json:"pullRequest"`
	PreviousTitle       string                        `json:"previousTitle"`
	PreviousDescription interface{}                   `json:"previousDescription"`
	PreviousTarget      BitbucketServerPreviousTarget `json:"previousTarget"`
}

type BitbucketServerPRDeclinedEvent struct {
	EventKey    string                     `json:"eventKey"`
	Date        string                     `json:"date"`
	Actor       BitbucketServerActor       `json:"actor"`
	PullRequest BitbucketServerPullRequest `json:"pullRequest"`
}

type BitbucketServerPRDeletedEvent struct {
	EventKey    string                     `json:"eventKey"`
	Date        string                     `json:"date"`
	Actor       BitbucketServerActor       `json:"actor"`
	PullRequest BitbucketServerPullRequest `json:"pullRequest"`
}

type BitbucketServerPRMergedEvent struct {
	EventKey    string                     `json:"eventKey"`
	Date        string                     `json:"date"`
	Actor       BitbucketServerActor       `json:"actor"`
	PullRequest BitbucketServerPullRequest `json:"pullRequest"`
}

type BitbucketServerPRCommentAddedEvent struct {
	EventKey    string                     `json:"eventKey"`
	Date        string                     `json:"date"`
	Actor       BitbucketServerActor       `json:"actor"`
	PullRequest BitbucketServerPullRequest `json:"pullRequest"`
	Comment     BitbucketServerComment     `json:"comment"`
}

type BitbucketServerPRCommentEditedEvent struct {
	EventKey        string                     `json:"eventKey"`
	Date            string                     `json:"date"`
	Actor           BitbucketServerActor       `json:"actor"`
	PullRequest     BitbucketServerPullRequest `json:"pullRequest"`
	Comment         BitbucketServerComment     `json:"comment"`
	PreviousComment string                     `json:"previousComment"`
}

type BitbucketServerPRCommentDeletedEvent struct {
	EventKey    string                     `json:"eventKey"`
	Date        string                     `json:"date"`
	Actor       BitbucketServerActor       `json:"actor"`
	PullRequest BitbucketServerPullRequest `json:"pullRequest"`
	Comment     BitbucketServerComment     `json:"comment"`
}

type BitbucketServerPRReviewerApproved struct {
	EventKey       string                     `json:"eventKey"`
	Date           string                     `json:"date"`
	Actor          BitbucketServerActor       `json:"actor"`
	PullRequest    BitbucketServerPullRequest `json:"pullRequest"`
	Participant    BitbucketServerParticipant `json:"participant"`
	PreviousStatus string                     `json:"previousStatus"`
}

type BitbucketServerPRReviewerUpdated struct {
	EventKey         string                     `json:"eventKey"`
	Date             string                     `json:"date"`
	Actor            BitbucketServerActor       `json:"actor"`
	PullRequest      BitbucketServerPullRequest `json:"pullRequest"`
	AddedReviewers   BitbucketServerActor       `json:"addedReviewers"`
	RemovedReviewers BitbucketServerActor       `json:"removedReviewers"`
}

type BitbucketServerPRReviewerUnapproved struct {
	EventKey       string                     `json:"eventKey"`
	Date           string                     `json:"date"`
	Actor          BitbucketServerActor       `json:"actor"`
	PullRequest    BitbucketServerPullRequest `json:"pullRequest"`
	Participant    BitbucketServerParticipant `json:"participant"`
	PreviousStatus string                     `json:"previousStatus"`
}

type BitbucketServerPRReviewerNeedsWorks struct {
	EventKey       string                     `json:"eventKey"`
	Date           string                     `json:"date"`
	Actor          BitbucketServerActor       `json:"actor"`
	PullRequest    BitbucketServerPullRequest `json:"pullRequest"`
	Participant    BitbucketServerParticipant `json:"participant"`
	PreviousStatus string                     `json:"previousStatus"`
}

/*
"repo:refs_changed",
"repo:modified",
"repo:forked",
"repo:comment:added",
"repo:comment:edited",
"repo:comment:deleted",



*/
