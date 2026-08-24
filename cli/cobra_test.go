package cli

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/sdk"
)

func TestListItem(t *testing.T) {
	keyProject := sdk.ProjectKey{
		Name:      "myKey",
		Type:      "ssh",
		Public:    "pubb",
		Private:   "privv",
		ProjectID: 1,
		Disabled:  true,
	}

	result := listItem(keyProject, nil, false, nil, false, map[string]string{})
	assert.Equal(t, 4, len(result))

	result = listItem(keyProject, nil, false, []string{"name"}, false, map[string]string{})
	assert.Equal(t, map[string]string{"name": "myKey"}, result)

	result = listItem(keyProject, nil, false, []string{"NAME"}, false, map[string]string{})
	assert.Equal(t, map[string]string{"name": "myKey"}, result)
}

// TestListItemUserDisabled checks that `cdsctl user list` exposes the disabled state:
// an admin cleaning up users has to be able to tell a neutralized account from a live one.
func TestListItemUserDisabled(t *testing.T) {
	disabled := listItem(sdk.AuthentifiedUser{
		Username: "jdoe",
		Fullname: "John Doe",
		Ring:     sdk.UserRingUser,
		Disabled: true,
	}, nil, false, nil, false, map[string]string{})
	assert.Equal(t, "true", disabled["disabled"])

	active := listItem(sdk.AuthentifiedUser{
		Username: "jdoe",
		Fullname: "John Doe",
		Ring:     sdk.UserRingUser,
	}, nil, false, nil, false, map[string]string{})
	assert.Equal(t, "false", active["disabled"])

	// The column can be selected explicitly with --fields
	filtered := listItem(sdk.AuthentifiedUser{
		Username: "jdoe",
		Disabled: true,
	}, nil, false, []string{"username", "disabled"}, false, map[string]string{})
	assert.Equal(t, map[string]string{"username": "jdoe", "disabled": "true"}, filtered)
}

// TestListItemWorkerModelEOL checks that `cdsctl worker model show` renders the end of life date,
// and above all that a model without one does not blow up: EOL is the only *time.Time on sdk.Model
// carrying a cli tag, so nil pointer rendering is exercised here for the first time.
func TestListItemWorkerModelEOL(t *testing.T) {
	eol := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	withEOL := listItem(sdk.Model{
		Name:         "myModel",
		Type:         sdk.Docker,
		IsDeprecated: true,
		EOL:          &eol,
	}, nil, false, []string{"name", "eol"}, false, map[string]string{})
	assert.Equal(t, map[string]string{"name": "myModel", "eol": "2026-12-31 00:00:00 +0000 UTC"}, withEOL)

	withoutEOL := listItem(sdk.Model{
		Name: "myModel",
		Type: sdk.Docker,
	}, nil, false, []string{"name", "eol"}, false, map[string]string{})
	assert.Equal(t, map[string]string{"name": "myModel", "eol": ""}, withoutEOL)
}
