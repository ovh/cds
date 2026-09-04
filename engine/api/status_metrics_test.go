package api

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"

	"github.com/ovh/cds/engine/cache"
)

// sharedCountsCache stands for the cache of a cluster where another instance has already read the
// counts. Only what refreshInventoryMetrics is allowed to use is implemented: anything else it
// reached for would panic on the embedded nil.
type sharedCountsCache struct {
	cache.Store
	locked bool
	found  bool
	shared map[string]int64
	stored map[string]int64
}

func (c *sharedCountsCache) Lock(string, time.Duration, int, int) (bool, error) {
	return c.locked, nil
}

func (c *sharedCountsCache) Get(_ string, value interface{}) (bool, error) {
	if m, ok := value.(*map[string]int64); ok {
		*m = c.shared
	}
	return c.found, nil
}

func (c *sharedCountsCache) SetWithDuration(_ string, value interface{}, _ time.Duration) error {
	c.stored, _ = value.(map[string]int64)
	return nil
}

// The counts describe the database, so an instance that did not read them reports what the instance
// that did has shared. The API here holds no database at all: reaching for one would be reading them
// again, which is what this is about.
func TestRefreshInventoryMetrics_RecordsWhatAnotherInstanceRead(t *testing.T) {
	api := &API{}
	api.Metrics.nbProjects = stats.Int64("test/nb_projects_shared", "", stats.UnitDimensionless)

	v := &view.View{
		Name:        "test/nb_projects_shared",
		Measure:     api.Metrics.nbProjects,
		Aggregation: view.LastValue(),
	}
	require.NoError(t, view.Register(v))
	defer view.Unregister(v)

	api.Cache = &sharedCountsCache{found: true, shared: map[string]int64{"projects": 42}}

	api.refreshInventoryMetrics(context.Background())

	rows, err := view.RetrieveData(v.Name)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.EqualValues(t, 42, rows[0].Data.(*view.LastValueData).Value)
}

// The key a count is shared under is what pairs it with its measure from one instance to the next,
// and from one version to the next while they run side by side. Two counts sharing a key would
// report one another's value.
func TestInventoryCounts_KeysAreDistinct(t *testing.T) {
	counts := (&API{}).inventoryCounts()
	require.NotEmpty(t, counts)

	seen := make(map[string]bool, len(counts))
	for _, c := range counts {
		require.NotEmpty(t, c.key)
		require.NotEmpty(t, c.query, "the count %q has no query", c.key)
		require.False(t, seen[c.key], "the key %q is used by two counts", c.key)
		seen[c.key] = true
	}
}
