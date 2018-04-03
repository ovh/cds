package sdk

// NavbarProjectData is the sub struct returned by getNavbarHandler
type NavbarProjectData struct {
	Key             string `json:"key"`
	Name            string `json:"name"`
	ApplicationName string `json:"application_name,omitempty"`
	WorkflowName    string `json:"workflow_name,omitempty"`
	Type            string `json:"type,omitempty"`
	Favorite        bool   `json:"favorite"`
}
