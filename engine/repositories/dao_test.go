package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

func Test_daoLastAccessPerInstance(t *testing.T) {
	s, err := newTestService(t)
	require.NoError(t, err)

	daoA := dao{store: s.Cache, hostname: "instance-a"}
	daoB := dao{store: s.Cache, hostname: "instance-b"}

	repoID := sdk.UUID()
	t.Cleanup(func() {
		_ = s.Cache.Delete(daoA.lastAccessKey(repoID))
	})

	daoA.saveLastAccess(repoID, time.Now().Add(time.Minute), 60)

	_, expired := daoA.isExpired(context.TODO(), repoID)
	require.False(t, expired, "repository must not be expired for the instance that accessed it")

	_, expired = daoB.isExpired(context.TODO(), repoID)
	require.True(t, expired, "a last access from another instance must not protect this instance's clone")
}
