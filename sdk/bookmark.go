package sdk

// Bookmark represents the type for a bookmark with his icon and description
type Bookmark struct {
	Icon        string `json:"icon" db:"icon"`
	Description string `json:"description" db:"description"`
	NavbarProjectData
}
