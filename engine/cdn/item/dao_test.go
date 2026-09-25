package item_test

import (
	"context"
	"github.com/ovh/cds/sdk/cdn"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/cdn/item"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

func TestLoadItem(t *testing.T) {
	m := gorpmapper.New()
	item.InitDBMapping(m)

	db, _ := test.SetupPGWithMapper(t, m, sdk.TypeCDN)
	cdntest.ClearItem(t, context.TODO(), m, db)

	apiRef := sdk.NewCDNLogApiRef(cdn.Signature{
		ProjectKey: sdk.RandomString(10),
	})
	hashRef, err := apiRef.ToHash()
	require.NoError(t, err)

	i := sdk.CDNItem{
		APIRef:     apiRef,
		APIRefHash: hashRef,
		Type:       sdk.CDNTypeItemStepLog,
	}
	require.NoError(t, item.Insert(context.TODO(), m, db, &i))
	t.Cleanup(func() { _ = item.DeleteByID(db, i.ID) })

	res, err := item.LoadByID(context.TODO(), m, db, i.ID)
	require.NoError(t, err)
	require.Equal(t, i.ID, res.ID)
	require.Equal(t, i.Type, res.Type)
	_, has := res.GetCDNLogApiRef()
	require.True(t, has)

	_, no := res.APIRef.(*sdk.CDNRunResultAPIRef)
	require.False(t, no)
}

// TestCountLogItemsByJobIdentifiers pins what the coverage of a job means: every log item of the
// job counts, whichever run version it belongs to and whether or not it is on its way out, and
// nothing that is not a log counts. A job the CDN holds nothing for has to be absent rather than
// zero, because that absence is the whole signal the caller is after.
func TestCountLogItemsByJobIdentifiers(t *testing.T) {
	m := gorpmapper.New()
	item.InitDBMapping(m)

	db, _ := test.SetupPGWithMapper(t, m, sdk.TypeCDN)
	cdntest.ClearItem(t, context.TODO(), m, db)

	projectKey := sdk.RandomString(10)
	runJobID := sdk.UUID()
	nodeRunJobID := int64(4242)

	insert := func(apiRef sdk.CDNApiRef, itemType sdk.CDNItemType, toDelete bool) {
		hash, err := apiRef.ToHash()
		require.NoError(t, err)
		i := sdk.CDNItem{APIRef: apiRef, APIRefHash: hash, Type: itemType}
		require.NoError(t, item.Insert(context.TODO(), m, db, &i))
		t.Cleanup(func() { _ = item.DeleteByID(db, i.ID) })
		if toDelete {
			i.ToDelete = true
			require.NoError(t, item.Update(context.TODO(), m, db, &i))
		}
	}

	// Two steps of one v2 job.
	for _, stepOrder := range []int64{0, 1} {
		insert(sdk.NewCDNLogApiRefV2(cdn.Signature{
			ProjectKey: projectKey,
			RunJobID:   runJobID,
			Worker:     &cdn.SignatureWorker{StepOrder: stepOrder, StepName: sdk.RandomString(5)},
		}), sdk.CDNTypeItemJobStepLog, false)
	}

	// A run result of the same v2 job: it carries the job identifier but it is not a log.
	insert(sdk.NewCDNRunResultApiRefV2(cdn.Signature{
		ProjectKey: projectKey,
		RunJobID:   runJobID,
		Worker: &cdn.SignatureWorker{
			RunResultID:   sdk.UUID(),
			RunResultName: sdk.RandomString(5),
			RunResultType: "generic",
		},
	}), sdk.CDNTypeItemRunResultV2, false)

	// One step of a v1 job, already flagged for deletion: it existed, so the job is covered.
	insert(sdk.NewCDNLogApiRef(cdn.Signature{
		ProjectKey: projectKey,
		JobID:      nodeRunJobID,
		Worker:     &cdn.SignatureWorker{StepOrder: 0, StepName: sdk.RandomString(5)},
	}), sdk.CDNTypeItemStepLog, true)

	// A service of the first v2 job, and a second v2 job that only has the log of its service: its
	// steps lost theirs, which the step count has to show and the total cannot.
	serviceOnlyJobID := sdk.UUID()
	for _, id := range []string{runJobID, serviceOnlyJobID} {
		insert(sdk.NewCDNLogApiRefV2(cdn.Signature{
			ProjectKey:      projectKey,
			RunJobID:        id,
			HatcheryService: &cdn.SignatureHatcheryService{ServiceName: sdk.RandomString(5)},
		}), sdk.CDNTypeItemServiceLogV2, false)
	}

	unknownJobID := sdk.UUID()
	counts, stepCounts, err := item.CountLogItemsByJobIdentifiers(db, []string{runJobID, "4242", serviceOnlyJobID, unknownJobID})
	require.NoError(t, err)

	require.Equal(t, int64(3), counts[runJobID], "the run result must not be counted as a log")
	require.Equal(t, int64(1), counts["4242"], "an item flagged to_delete still proves the logs were stored")
	_, found := counts[unknownJobID]
	require.False(t, found, "a job with no item at all must be absent from the result")

	require.Equal(t, int64(2), stepCounts[runJobID], "the log of a service is not the log of a step")
	require.Equal(t, int64(1), stepCounts["4242"])
	nb, found := stepCounts[serviceOnlyJobID]
	require.True(t, found, "a job with log items but none for its steps must be present with a zero")
	require.Equal(t, int64(0), nb)
	_, found = stepCounts[unknownJobID]
	require.False(t, found)

	empty, emptySteps, err := item.CountLogItemsByJobIdentifiers(db, nil)
	require.NoError(t, err)
	require.Empty(t, empty)
	require.NotNil(t, emptySteps, "an empty answer still says the step log items were counted")
}
