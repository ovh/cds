package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/sdk"
)

func TestListItem(t *testing.T) {
	keyProject := sdk.ProjectKey{
		Key: sdk.Key{
			Name:    "myKey",
			Type:    "ssh",
			Public:  "pubb",
			Private: "privv",
		},
		ProjectID: 1,
	}

	myResult := listItem(keyProject, nil, false, nil, false, map[string]string{})
	assert.Equal(t, len(myResult), 3)
}
