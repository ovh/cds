package github

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToVCSRepoParent(t *testing.T) {
	var fork, origin Repository
	require.NoError(t, json.Unmarshal([]byte(`{"id":2,"full_name":"khopa/cds","fork":true,"parent":{"id":1,"full_name":"ovh/cds"}}`), &fork))
	require.NoError(t, json.Unmarshal([]byte(`{"id":1,"full_name":"ovh/cds"}`), &origin))

	client := &githubClient{}
	require.Equal(t, "ovh/cds", client.ToVCSRepo(fork).Parent)
	require.Empty(t, client.ToVCSRepo(origin).Parent)
}
