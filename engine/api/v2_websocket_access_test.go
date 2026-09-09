package api

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

func clientWithAccess(readAt time.Time) *websocketV2ClientData {
	return &websocketV2ClientData{
		AuthConsumer: sdk.AuthUserConsumer{
			AuthConsumerUser: sdk.AuthUserConsumerData{
				AuthentifiedUser:   &sdk.AuthentifiedUser{Ring: sdk.UserRingUser},
				AuthentifiedUserID: "user-1",
			},
		},
		access: &runJobAccess{
			projectKeys: sdk.StringSlice{"KEY"},
			regionNames: sdk.StringSlice{"default"},
			readAt:      readAt,
		},
	}
}

// The database given to the check is nil: reading the permissions again would panic on it, so these
// answers can only come from what the client already holds.
func TestEventPostCheck_JobEventAnsweredFromTheHeldAccess(t *testing.T) {
	c := clientWithAccess(time.Now())

	for _, tc := range []struct {
		name    string
		event   sdk.FullEventV2
		allowed bool
	}{
		{"a project it reads, on a region it runs on", sdk.FullEventV2{Type: sdk.EventRunJobBuilding, ProjectKey: "KEY", Region: "default"}, true},
		{"another project", sdk.FullEventV2{Type: sdk.EventRunJobBuilding, ProjectKey: "OTHER", Region: "default"}, false},
		{"another region", sdk.FullEventV2{Type: sdk.EventRunJobBuilding, ProjectKey: "KEY", Region: "gpu"}, false},
		{"no region on the event", sdk.FullEventV2{Type: sdk.EventRunJobEnded, ProjectKey: "KEY"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			allowed, err := c.eventPostCheck(context.TODO(), nil, nil, tc.event, nil)
			require.NoError(t, err)
			require.Equal(t, tc.allowed, allowed)
		})
	}
}

// Past the delay the permissions are read again rather than trusted. The nil database makes that
// read fail, which is what tells the two apart: an event that the held access would have allowed is
// refused instead of being answered from it.
func TestEventPostCheck_JobEventReadsTheAccessAgainOnceItAged(t *testing.T) {
	c := clientWithAccess(time.Now().Add(-runJobAccessTTL - time.Second))

	allowed, err := c.eventPostCheck(context.TODO(), nil, nil,
		sdk.FullEventV2{Type: sdk.EventRunJobBuilding, ProjectKey: "KEY", Region: "default"}, nil)
	require.Error(t, err)
	require.False(t, allowed)
}

// A maintainer sees every job event, and is not worth a lookup.
func TestEventPostCheck_JobEventOfAMaintainerNeedsNothing(t *testing.T) {
	c := clientWithAccess(time.Now().Add(-runJobAccessTTL - time.Second))
	c.AuthConsumer.AuthConsumerUser.AuthentifiedUser.Ring = sdk.UserRingMaintainer

	allowed, err := c.eventPostCheck(context.TODO(), nil, nil,
		sdk.FullEventV2{Type: sdk.EventRunJobBuilding, ProjectKey: "OTHER", Region: "gpu"}, nil)
	require.NoError(t, err)
	require.True(t, allowed)
}
