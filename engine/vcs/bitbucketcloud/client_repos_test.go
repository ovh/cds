package bitbucketcloud

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToVCSRepoParent(t *testing.T) {
	var fork, origin Repository
	require.NoError(t, json.Unmarshal([]byte(`{"full_name":"khopa/cds","parent":{"full_name":"ovh/cds"}}`), &fork))
	require.NoError(t, json.Unmarshal([]byte(`{"full_name":"ovh/cds"}`), &origin))

	client := &bitbucketcloudClient{}
	require.Equal(t, "ovh/cds", client.ToVCSRepo(fork).Parent)
	require.Empty(t, client.ToVCSRepo(origin).Parent)
}
