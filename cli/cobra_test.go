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

	result := listItem(keyProject, nil, false, nil, false, map[string]string{})
	assert.Equal(t, 3, len(result))

	result = listItem(keyProject, nil, false, []string{"name"}, false, map[string]string{})
	assert.Equal(t, map[string]string{"name": "myKey"}, result)

	result = listItem(keyProject, nil, false, []string{"NAME"}, false, map[string]string{})
	assert.Equal(t, map[string]string{"name": "myKey"}, result)
}
