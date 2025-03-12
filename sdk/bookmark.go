package sdk

type BookmarkType string

const (
	ProjectBookmarkType        BookmarkType = "project"
	WorkflowBookmarkType       BookmarkType = "workflow"
	WorkflowLegacyBookmarkType BookmarkType = "workflow-legacy"
)

type Bookmark struct {
	Type  BookmarkType `json:"type"`
	ID    string       `json:"id"`
	Label string       `json:"label"`
}
