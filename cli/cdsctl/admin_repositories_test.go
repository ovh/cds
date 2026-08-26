package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

func Test_adminRepositoriesLines(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	until := now.Add(time.Hour)
	lists := []sdk.RepositoriesAdminList{
		{
			Instance:   "repositories-b",
			ComputedAt: &now,
			Repositories: []sdk.RepositoriesAdminEntry{
				{ID: "idB1", URL: "git@example.net:cds/small.git", Size: 1024, Expired: true},
				{ID: "idB2", URL: "git@example.net:cds/big.git", Size: 3 * 1024 * 1024, ProtectedUntil: &until},
			},
		},
		{
			Instance:   "repositories-a",
			ComputedAt: &now,
			Repositories: []sdk.RepositoriesAdminEntry{
				{ID: "idA1", URL: "git@example.net:cds/small.git", Size: 1024, Expired: true},
			},
		},
		{
			Instance: "repositories-c", // sizes not measured yet
			Repositories: []sdk.RepositoriesAdminEntry{
				{ID: "idC1", URL: "git@example.net:cds/fresh.git", Expired: true},
			},
		},
	}

	lines := adminRepositoriesLines(lists)
	require.Len(t, lines, 4)

	require.Equal(t, adminRepositoryLine{
		Instance: "repositories-b", URL: "git@example.net:cds/big.git", Size: "3.0 MiB",
		ProtectedUntil: until.Local().Format(time.RFC3339), ID: "idB2",
	}, lines[0], "largest first")

	require.Equal(t, "repositories-a", lines[1].Instance, "same size: sorted by instance")
	require.Equal(t, "repositories-b", lines[2].Instance)
	require.Equal(t, "1.0 KiB", lines[1].Size)
	require.True(t, lines[1].Expired)
	require.Empty(t, lines[1].ProtectedUntil)

	require.Equal(t, "unknown", lines[3].Size, "instance without a size snapshot")
	require.Equal(t, "repositories-c", lines[3].Instance)
}

func Test_adminRepositoriesLines_empty(t *testing.T) {
	require.Empty(t, adminRepositoriesLines(nil))
}
