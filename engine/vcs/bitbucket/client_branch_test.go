package bitbucket

import (
	"testing"

	"github.com/ovh/cds/engine/api/test"

	"github.com/stretchr/testify/assert"
)

func TestBranches(t *testing.T) {
	client := getAuthorizedClient(t)
	branches, err := client.Branches("CDS/images")
	test.NoError(t, err)
	assert.NotEmpty(t, branches)
	t.Logf("branches: %+v", branches)
}

func TestBranch(t *testing.T) {
	client := getAuthorizedClient(t)
	branch, err := client.Branch("CDS/images", "master")
	test.NoError(t, err)
	assert.NotNil(t, branch)
	t.Logf("branch: %+v", branch)
}
