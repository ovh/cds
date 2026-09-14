package internal

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

func TestCheckRequirement(t *testing.T) {
	r := sdk.Requirement{
		Name:  "Go",
		Type:  sdk.BinaryRequirement,
		Value: "go",
	}

	ok, err := checkRequirement(nil, r)
	if err != nil {
		t.Fatalf("checkRequirement should not fail: %s", err)
	}
	if !ok {
		t.Fatalf("Requirement go should be here")
	}

	r.Value = "foo"
	ok, err = checkRequirement(nil, r)
	if err != nil {
		t.Fatalf("checkRequirement should not fail: %s", err)
	}
	if ok {
		t.Fatalf("Requirement foo should not be ok")
	}
}

// A worker started from a worker model v2 carries no model of its own, so the
// requirement is only checked on the shape of its path. The path is not a fixed count
// of segments: the ref it ends with can hold slashes, and so can a repository name.
func TestCheckModelRequirementV2(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "on the default ref", value: "MYPROJ/myvcs/my/repo/my-model"},
		{name: "on a ref", value: "MYPROJ/myvcs/my/repo/my-model@master"},
		{name: "on a ref containing a slash", value: "MYPROJ/myvcs/my/repo/my-model@feat/my-branch"},
		{name: "on a ref containing several slashes", value: "MYPROJ/myvcs/my/repo/my-model@users/me/feat/my-branch"},
		{name: "in a repository nested in subgroups", value: "MYPROJ/myvcs/my/group/repo/my-model@feat/my-branch"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &CurrentWorker{}
			ok, err := checkModelRequirement(w, sdk.Requirement{Type: sdk.ModelRequirement, Value: tt.value})
			require.NoError(t, err)
			require.True(t, ok)
		})
	}
}

func TestCheckModelRequirementV1(t *testing.T) {
	w := &CurrentWorker{}
	w.model = sdk.Model{ID: 1, Name: "my-model", Group: &sdk.Group{Name: "my-group"}}

	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{name: "the group and the name match", value: "my-group/my-model", expected: true},
		{name: "the name alone matches", value: "my-model", expected: true},
		{name: "options after the name are ignored", value: "my-group/my-model --privileged", expected: true},
		{name: "another group does not match", value: "another-group/my-model", expected: false},
		{name: "another name does not match", value: "my-group/another-model", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := checkModelRequirement(w, sdk.Requirement{Type: sdk.ModelRequirement, Value: tt.value})
			require.NoError(t, err)
			require.Equal(t, tt.expected, ok)
		})
	}
}

// A worker with no model cannot satisfy a model v1 requirement.
func TestCheckModelRequirementV1WithoutModel(t *testing.T) {
	w := &CurrentWorker{}
	ok, err := checkModelRequirement(w, sdk.Requirement{Type: sdk.ModelRequirement, Value: "my-group/my-model"})
	require.NoError(t, err)
	require.False(t, ok)
}

func TestCheckHostnameRequirement(t *testing.T) {
	h, err := os.Hostname()
	if err != nil {
		// Meh, no way to test it
		t.Skip()
		return
	}
	r := sdk.Requirement{
		Type:  sdk.HostnameRequirement,
		Value: h,
	}

	ok, err := checkRequirement(nil, r)
	if err != nil {
		t.Fatalf("checkRequirement should not fail: %s", err)
	}

	if !ok {
		t.Fatalf("Requirement should be ok")
	}

	r.Value = "fewfewf"
	ok, err = checkRequirement(nil, r)
	if err != nil {
		t.Fatalf("checkRequirement should not fail: %s", err)
	}

	if ok {
		t.Fatalf("Requirement should not be ok")
	}
}
