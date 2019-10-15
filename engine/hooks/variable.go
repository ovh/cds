package hooks

const (
	PR_ID              = "git.pr.id"
	PR_TITLE           = "git.pr.title"
	PR_STATE           = "git.pr.state"
	PR_PREVIOUS_TITLE  = "git.pr.previous.title"
	PR_PREVIOUS_BRANCH = "git.pr.previous.branch"
	PR_PREVIOUS_HASH   = "git.pr.previous.has"

	GIT_AUTHOR            = "git.author"
	GIT_AUTHOR_EMAIL      = "git.author.email"
	GIT_BRANCH            = "git.branch"
	GIT_BRANCH_BEFORE     = "git.branch.before"
	GIT_TAG               = "git.tag"
	GIT_HASH_BEFORE       = "git.hash.before"
	GIT_HASH              = "git.hash"
	GIT_HASH_SHORT        = "git.hash.short"
	GIT_REPOSITORY        = "git.repository"
	GIT_REPOSITORY_BEFORE = "git.repository.before"
	GIT_EVENT             = "git.hook"

	CDS_TRIGGERED_BY_USERNAME = "cds.triggered_by.username"
	CDS_TRIGGERED_BY_FULLNAME = "cds.triggered_by.fullname"
	CDS_TRIGGERED_BY_EMAIL    = "cds.triggered_by.email"

	PAYLOAD = "payload"
)
