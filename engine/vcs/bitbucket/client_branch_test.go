package bitbucket

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/test"

	"github.com/stretchr/testify/assert"
)

func TestBranches(t *testing.T) {
	client := getAuthorizedClient(t)
	branches, err := client.Branches(context.Background(), "CDS/images")
	test.NoError(t, err)
	assert.NotEmpty(t, branches)
	t.Logf("branches: %+v", branches)
}

func TestBranch(t *testing.T) {
	client := getAuthorizedClient(t)
	branch, err := client.Branch(context.Background(), "CDS/images", "master")
	test.NoError(t, err)
	assert.NotNil(t, branch)
	t.Logf("branch: %+v", branch)
}
