package sdk

// NavbarData is the struct returned by getNavbarHandler
type NavbarData struct {
	Projects []NavbarProjectData `json:"projects"`
}

// NavbarProjectData is the sub struct returned by getNavbarHandler
type NavbarProjectData struct {
	Key              string   `json:"key"`
	Name             string   `json:"name"`
	ApplicationNames []string `json:"application_names,omitempty"`
	WorkflowNames    []string `json:"workflow_names,omitempty"`
}
