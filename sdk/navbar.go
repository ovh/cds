package sdk

// NavbarProjectData is the sub struct returned by getNavbarHandler
type NavbarProjectData struct {
	Key             string `json:"key" db:"key"`
	Name            string `json:"name" db:"project_name"`
	Description     string `json:"description" db:"description"`
	ApplicationName string `json:"application_name,omitempty" db:"application_name"`
	WorkflowName    string `json:"workflow_name,omitempty" db:"workflow_name"`
	Type            string `json:"type,omitempty" db:"type"`
	Favorite        bool   `json:"favorite" db:"favorite"`
}
