package v3

import (
	"testing"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/acceptance/tools"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
)

// CreateProject will create a project with a random name.
// It takes an optional createOpts parameter since creating a project
// has so many options. An error will be returned if the project was
// unable to be created.
func CreateProject(t *testing.T, client *gophercloud.ServiceClient, c *projects.CreateOpts) (*projects.Project, error) {
	name := tools.RandomString("ACPTTEST", 8)
	t.Logf("Attempting to create project: %s", name)

	var createOpts projects.CreateOpts
	if c != nil {
		createOpts = *c
	} else {
		createOpts = projects.CreateOpts{}
	}

	createOpts.Name = name

	project, err := projects.Create(client, createOpts).Extract()
	if err != nil {
		return project, err
	}

	t.Logf("Successfully created project %s with ID %s", name, project.ID)

	return project, nil
}

// DeleteProject will delete a project by ID. A fatal error will occur if
// the project ID failed to be deleted. This works best when using it as
// a deferred function.
func DeleteProject(t *testing.T, client *gophercloud.ServiceClient, projectID string) {
	err := projects.Delete(client, projectID).ExtractErr()
	if err != nil {
		t.Fatalf("Unable to delete project %s: %v", projectID, err)
	}

	t.Logf("Deleted project: %s", projectID)
}
