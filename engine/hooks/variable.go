package hooks

const (
	PR_ID              = "git.pr.id"
	PR_TITLE           = "git.pr.title"
	PR_STATE           = "git.pr.state"
	PR_FROM_BRANCH     = "git.pr.from.branch"
	PR_FROM_COMMIT     = "git.pr.from.hash"
	PR_TO_BRANCH       = "git.pr.to.branch"
	PR_TO_COMMIT       = "git.pr.to.hash"
	PR_PREVIOUS_TITLE  = "git.pr.previous.title"
	PR_PREVIOUS_BRANCH = "git.pr.previous.branch"
	PR_PREVIOUS_HASH   = "git.pr.previous.has"

	GIT_AUTHOR          = "git.author"
	GIT_AUTHOR_EMAIL    = "git.author.email"
	GIT_BRANCH          = "git.branch"
	GIT_TAG             = "git.tag"
	GIT_HASH_BEFORE     = "git.hash.before"
	GIT_HASH            = "git.hash"
	GIT_HASH_SHORT      = "git.hash.short"
	GIT_REPOSITORY      = "git.repository"
	GIT_FROM_REPOSITORY = "git.from.repository"

	CDS_TRIGGERED_BY_USERNAME = "cds.triggered_by.username"
	CDS_TRIGGERED_BY_FULLNAME = "cds.triggered_by.fullname"
	CDS_TRIGGERED_BY_EMAIL    = "cds.triggered_by.email"
)
