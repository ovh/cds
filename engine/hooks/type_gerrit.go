package hooks

const (
	GerritChangeStatusNew       = "NEW"
	GerritChangeStatusMerged    = "MERGED"
	GerritChangeStatusAbandoned = "ABANDONED"

	GerritPatchSetKindRework                 = "REWORK"
	GerritPatchSetKindTrivialRebase          = "TRIVIAL_REBASE"
	GerritPatchSetKindMergeFirstParentUpdate = "MERGE_FIRST_PARENT_UPDATE"
	GerritPatchSetKindNoCodeChange           = "NO_CODE_CHANGE"
	GerritPatchSetKindNoChange               = "NO_CHANGE"

	GerritFileTypeAdded    = "ADDED"
	GerritFileTypeModified = "MODIFIED"
	GerritFileTypeDeleted  = "DELETED"
	GerritFileTypeRenamed  = "RENAMED"
	GerritFileTypeCopied   = "COPIED"
	GerritFileTypeRewrite  = "REWRITE"

	GerritSubmitRecordStatusOk        = "OK"
	GerritSubmitRecordStatusNotReady  = "NOT_READY"
	GerritSubmitRecordStatusRuleError = "RULE_ERROR"

	GerritLabelStatusOk         = "OK"
	GerritLabelStatusReject     = "REJECT"
	GerritLabelStatusNeed       = "NEED"
	GerritLabelStatusMay        = "MAY"
	GerritLabelStatusImpossible = "IMPOSSIBLE"

	GerritEmptyRef = "0000000000000000000000000000000000000000"

	GerritEventTypeAssignedChanged     = "assignee-changed"
	GerritEventTypeChangeAbandoned     = "change-abandoned"
	GerritEventTypeChangeDeleted       = "change-deleted"
	GerritEventTypeChangeMerged        = "change-merged"
	GerritEventTypeChangeRestored      = "change-restored"
	GerritEventTypeCommentAdded        = "comment-added"
	GerritEventTypeDroppedOutput       = "dropped-output"
	GerritEventTypeHashTagsChanged     = "hashtags-changed"
	GerritEventTypeProjectCreated      = "project-created"
	GerritEventTypePatchsetCreated     = "patchset-created"
	GerritEventTypeRefUpdated          = "ref-updated"
	GerritEventTypeReviewerAdded       = "reviewer-added"
	GerritEventTypeReviewerDelete      = "reviewer-deleted"
	GerritEventTypeTopicChanged        = "topic-changed"
	GerritEventTypeWIPStateChanged     = "wip-state-changed"
	GerritEventTypePrivateStateChanged = "private-state-changed"
	GerritEventTypeVoteDeleted         = "vote-deleted"
)

// GerritEvent rerpesents the events send by gerrit
// https://gerrit-review.googlesource.com/Documentation/cmd-stream-events.html
type GerritEvent struct {
	Type           string           `json:"type,omitempty"`
	Change         *GerritChange    `json:"change,omitempty"`
	Changer        *GerritAccount   `json:"changer,omitempty"`
	OldAssignee    string           `json:"oldAssignee,omitempty"`
	EventCreatedOn int64            `json:"eventCreatedOn,omitempty"`
	PatchSet       *GerritPatchSet  `json:"patchSet,omitempty"`
	Abandoner      *GerritAccount   `json:"abandoner,omitempty"`
	Reason         string           `json:"reason,omitempty"`
	Deleter        *GerritAccount   `json:"deleted,omitempty"`
	Submitter      *GerritAccount   `json:"submitter,omitempty"`
	NewRev         string           `json:"newRev,omitempty"`
	Restorer       string           `json:"restorer,omitempty"`
	Author         *GerritAccount   `json:"author,omitempty"`
	Approvals      []GerritApproval `json:"approvals,omitempty"`
	Comment        string           `json:"comment,omitempty"`
	Editor         *GerritAccount   `json:"editor,omitempty"`
	Added          []string         `json:"added,omitempty"`
	Removed        []string         `json:"removed,omitempty"`
	HashTags       []string         `json:"hashtags,omitempty"`
	ProjectName    string           `json:"projectName,omitempty"`
	ProjectHead    string           `json:"projectHead,omitempty"`
	Uploader       *GerritAccount   `json:"updaloed,omitempty"`
	RefUpdate      *GerritRefUpdate `json:"refUpdate,omitempty"`
	Reviewer       *GerritAccount   `json:"reviewer,omitempty"`
	Remover        *GerritAccount   `json:"remover,omitempty"`
	OldTopic       string           `json:"oldTopic,omitempty"`
}

// GerritChange represents a gerrit change
// https://gerrit-review.googlesource.com/Documentation/json.html#change
type GerritChange struct {
	Project         string               `json:"project,omitempty"` // repository name
	Branch          string               `json:"branch,omitempty"`  // master
	Topic           string               `json:"topic,omitempty"`
	ID              string               `json:"id,omitempty"`
	Subject         string               `json:"subject,omitempty"`
	Owner           *GerritAccount       `json:"owner,omitempty"`
	URL             string               `json:"url,omitempty"`
	CommitMessage   string               `json:"commitMessage,omitempty"`
	HashTags        []string             `json:"hashtags,omitempty"`
	CreatedOn       int64                `json:"createdOn,omitempty"`
	LastUpdated     int64                `json:"lastUpdated,omitempty"`
	Open            bool                 `json:"open,omitempty"`
	Status          string               `json:"status,omitempty"`
	Private         bool                 `json:"private,omitempty"`
	Wip             bool                 `json:"wip,omitempty"`
	Comments        []GerritMessage      `json:"comments,omitempty"`
	TrackingIDs     []GerritTrackingID   `json:"trackingIds,omitempty"`
	CurrentPatchSet *GerritPatchSet      `json:"currentPatchSet,omitempty"`
	PatchSets       []GerritPatchSet     `json:"patchSets,omitempty"`
	DependsOn       []GerritDependency   `json:"dependsOn,omitempty"`
	NeededBy        []GerritDependency   `json:"neededBy,omitempty"`
	SubmitRecords   []GerritSubmitRecord `json:"submitRecord,omitempty"`
	AllReviewers    []GerritAccount      `json:"allReviewers,omitempty"`
}

// GerritAccount https://gerrit-review.googlesource.com/Documentation/json.html#account
type GerritAccount struct {
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
	Username string `json:"username,omitempty"`
}

// GerritMessage https://gerrit-review.googlesource.com/Documentation/json.html#message
type GerritMessage struct {
	Timestamp int64          `json:"timestamp,omitempty"`
	Reviewer  *GerritAccount `json:"reviewer,omitempty"`
	Message   string         `json:"message,omitempty"`
}

// GerritTrackingID https://gerrit-review.googlesource.com/Documentation/json.html#trackingid
type GerritTrackingID struct {
	System string `json:"system,omitempty"`
	ID     string `json:"id,omitempty"`
}

// GerritPatchSet https://gerrit-review.googlesource.com/Documentation/json.html#patchSet
type GerritPatchSet struct {
	Number         int64                   `json:"number,omitempty"`
	Revision       string                  `json:"revision,omitempty"`
	Parents        []string                `json:"parents,omitempty"`
	Ref            string                  `json:"ref,omitempty"`
	Uploader       *GerritAccount          `json:"uploader,omitempty"`
	Author         *GerritAccount          `json:"author,omitempty"`
	CreatedOn      int                     `json:"createdOn,omitempty"`
	IsDraft        bool                    `json:"isDraft,omitempty"`
	Kind           string                  `json:"kind,omitempty"`
	Approvals      []GerritApproval        `json:"approvals,omitempty"`
	Comments       []GerritPatchSetComment `json:"comments,omitempty"`
	Files          []GerritFile            `json:"files,omitempty"`
	SizeInsertions int                     `json:"sizeInsertions,omitempty"`
	SizeDeletions  int                     `json:"sizeDeletions,omitempty"`
}

// GerritApproval https://gerrit-review.googlesource.com/Documentation/json.html#approval
type GerritApproval struct {
	Type        string         `json:"type,omitempty"`
	Description string         `json:"description,omitempty"`
	Value       string         `json:"value,omitempty"`
	OldValue    string         `json:"oldValue,omitempty"`
	GrantedOn   int            `json:"grantedOn,omitempty"`
	By          *GerritAccount `json:"by,omitempty"`
}

// GerritPatchSetComment https://gerrit-review.googlesource.com/Documentation/json.html#patchsetcomment
type GerritPatchSetComment struct {
	File     string         `json:"file,omitempty"`
	Line     int            `json:"line,omitempty"`
	Reviewer *GerritAccount `json:"reviewer,omitempty"`
	Message  string         `json:"message,omitempty"`
}

// GerritFile https://gerrit-review.googlesource.com/Documentation/json.html#file
type GerritFile struct {
	File       string `json:"file,omitempty"`
	FileOld    string `json:"fileOld,omitempty"`
	Type       string `json:"type,omitempty"`
	Insertions int    `json:"insertions,omitempty"`
	Deletions  int    `json:"deletions,omitempty"`
}

// GerritDependency https://gerrit-review.googlesource.com/Documentation/json.html#dependency
type GerritDependency struct {
	ID                string `json:"id,omitempty"`
	Number            string `json:"number,omitempty"`
	Revision          string `json:"revision,omitempty"`
	Ref               string `json:"ref,omitempty"`
	IsCurrentPatchSet bool   `json:"isCurrentPatchSet,omitempty"`
}

// GerritSubmitRecord https://gerrit-review.googlesource.com/Documentation/json.html#submitRecord
type GerritSubmitRecord struct {
	Status       string              `json:"status,omitempty"`
	Labels       []GerritLabel       `json:"labels,omitempty"`
	Requirements []GerritRequirement `json:"requirements,omitempty"`
}

// GerritLabel https://gerrit-review.googlesource.com/Documentation/json.html#label
type GerritLabel struct {
	Label  string         `json:"label,omitempty"`
	Status string         `json:"status,omitempty"`
	By     *GerritAccount `json:"by,omitempty"`
}

// GerritRequirement https://gerrit-review.googlesource.com/Documentation/json.html#requirement
type GerritRequirement struct {
	FallbackText string `json:"fallbackText,omitempty"`
	Type         string `json:"type,omitempty"`
	Data         string `json:"data,omitempty"`
}

// GerritRefUpdate https://gerrit-review.googlesource.com/Documentation/json.html#refUpdate
type GerritRefUpdate struct {
	OldRev  string `json:"oldRev,omitempty"`
	NewRev  string `json:"newRev,omitempty"`
	RefName string `json:"refName,omitempty"`
	Project string `json:"project,omitempty"`
}
