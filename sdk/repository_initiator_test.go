package sdk

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProjectRepositoryData_InitiatorUsername(t *testing.T) {
	require.Equal(t, "legacy", ProjectRepositoryData{DeprecatedCDSUserName: "legacy"}.InitiatorUsername())
	require.Equal(t, "alice", ProjectRepositoryData{
		DeprecatedCDSUserName: "legacy",
		Initiator:             &V2Initiator{UserID: "u1", User: &V2InitiatorUser{Username: "alice"}},
	}.InitiatorUsername())
	require.Equal(t, "github/octocat", ProjectRepositoryData{
		Initiator: &V2Initiator{VCS: "github", VCSUsername: "octocat"},
	}.InitiatorUsername())
	require.Empty(t, ProjectRepositoryData{Initiator: &V2Initiator{}}.InitiatorUsername())

	// A user id without snapshot must not be dereferenced
	idOnly := ProjectRepositoryData{DeprecatedCDSUserName: "legacy", Initiator: &V2Initiator{UserID: "u1"}}
	require.Equal(t, "legacy", idOnly.InitiatorUsername())
	idOnly.DeprecatedCDSUserName = ""
	require.Empty(t, idOnly.InitiatorUsername())
}
