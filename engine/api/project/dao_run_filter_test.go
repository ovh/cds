package project_test

import (
	"context"
	"testing"

	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_InsertRunFilter(t *testing.T) {
	db, cache := test.SetupPG(t)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	filter := &sdk.ProjectRunFilter{
		ProjectKey: proj.Key,
		Name:       "My Filter",
		Value:      "status:Success branch:main",
		Sort:       "started:desc",
		Order:      0,
	}

	require.NoError(t, project.InsertRunFilter(context.TODO(), db, filter))
	assert.NotEmpty(t, filter.ID)
	assert.NotEmpty(t, filter.LastModified)
	assert.Equal(t, int64(0), filter.Order)
}

func Test_InsertRunFilter_DuplicateName(t *testing.T) {
	db, cache := test.SetupPG(t)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	filter1 := &sdk.ProjectRunFilter{
		ProjectKey: proj.Key,
		Name:       "My Filter",
		Value:      "status:Success",
		Sort:       "started:desc",
		Order:      0,
	}

	require.NoError(t, project.InsertRunFilter(context.TODO(), db, filter1))

	filter2 := &sdk.ProjectRunFilter{
		ProjectKey: proj.Key,
		Name:       "My Filter",
		Value:      "status:Fail",
		Sort:       "started:asc",
		Order:      1,
	}

	err := project.InsertRunFilter(context.TODO(), db, filter2)
	require.Error(t, err)
	assert.True(t, sdk.ErrorIs(err, sdk.ErrConflictData))
}

func Test_LoadRunFiltersByProjectKey(t *testing.T) {
	db, cache := test.SetupPG(t)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	filter1 := &sdk.ProjectRunFilter{
		ProjectKey: proj.Key,
		Name:       "Filter B",
		Value:      "status:Success",
		Sort:       "started:desc",
		Order:      1,
	}
	filter2 := &sdk.ProjectRunFilter{
		ProjectKey: proj.Key,
		Name:       "Filter A",
		Value:      "status:Fail",
		Sort:       "started:asc",
		Order:      0,
	}
	filter3 := &sdk.ProjectRunFilter{
		ProjectKey: proj.Key,
		Name:       "Filter C",
		Value:      "branch:main",
		Sort:       "last_modified:desc",
		Order:      2,
	}

	require.NoError(t, project.InsertRunFilter(context.TODO(), db, filter1))
	require.NoError(t, project.InsertRunFilter(context.TODO(), db, filter2))
	require.NoError(t, project.InsertRunFilter(context.TODO(), db, filter3))

	filters, err := project.LoadRunFiltersByProjectKey(context.TODO(), db, proj.Key)
	require.NoError(t, err)
	require.Len(t, filters, 3)

	// Should be sorted by order ASC, then name ASC
	assert.Equal(t, "Filter A", filters[0].Name)
	assert.Equal(t, int64(0), filters[0].Order)
	assert.Equal(t, "Filter B", filters[1].Name)
	assert.Equal(t, int64(1), filters[1].Order)
	assert.Equal(t, "Filter C", filters[2].Name)
	assert.Equal(t, int64(2), filters[2].Order)
}

func Test_LoadRunFiltersByProjectKey_AlphabeticalFallback(t *testing.T) {
	db, cache := test.SetupPG(t)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	// Create filters with same order
	filter1 := &sdk.ProjectRunFilter{
		ProjectKey: proj.Key,
		Name:       "Zebra",
		Value:      "status:Success",
		Sort:       "started:desc",
		Order:      0,
	}
	filter2 := &sdk.ProjectRunFilter{
		ProjectKey: proj.Key,
		Name:       "Apple",
		Value:      "status:Fail",
		Sort:       "started:asc",
		Order:      0,
	}
	filter3 := &sdk.ProjectRunFilter{
		ProjectKey: proj.Key,
		Name:       "Banana",
		Value:      "branch:main",
		Sort:       "last_modified:desc",
		Order:      0,
	}

	require.NoError(t, project.InsertRunFilter(context.TODO(), db, filter1))
	require.NoError(t, project.InsertRunFilter(context.TODO(), db, filter2))
	require.NoError(t, project.InsertRunFilter(context.TODO(), db, filter3))

	filters, err := project.LoadRunFiltersByProjectKey(context.TODO(), db, proj.Key)
	require.NoError(t, err)
	require.Len(t, filters, 3)

	// Should be sorted alphabetically when order is the same
	assert.Equal(t, "Apple", filters[0].Name)
	assert.Equal(t, "Banana", filters[1].Name)
	assert.Equal(t, "Zebra", filters[2].Name)
}

func Test_LoadRunFilterByNameAndProjectKey(t *testing.T) {
	db, cache := test.SetupPG(t)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	filter := &sdk.ProjectRunFilter{
		ProjectKey: proj.Key,
		Name:       "My Filter",
		Value:      "status:Success branch:main",
		Sort:       "started:desc",
		Order:      0,
	}

	require.NoError(t, project.InsertRunFilter(context.TODO(), db, filter))

	loaded, err := project.LoadRunFilterByNameAndProjectKey(context.TODO(), db, proj.Key, "My Filter")
	require.NoError(t, err)
	assert.Equal(t, filter.ID, loaded.ID)
	assert.Equal(t, filter.Name, loaded.Name)
	assert.Equal(t, filter.Value, loaded.Value)
	assert.Equal(t, filter.Sort, loaded.Sort)
	assert.Equal(t, filter.Order, loaded.Order)
}

func Test_LoadRunFilterByNameAndProjectKey_NotFound(t *testing.T) {
	db, cache := test.SetupPG(t)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	_, err := project.LoadRunFilterByNameAndProjectKey(context.TODO(), db, proj.Key, "NonExistent")
	require.Error(t, err)
	assert.True(t, sdk.ErrorIs(err, sdk.ErrNotFound))
}

func Test_UpdateRunFilterOrder(t *testing.T) {
	db, cache := test.SetupPG(t)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	filter := &sdk.ProjectRunFilter{
		ProjectKey: proj.Key,
		Name:       "My Filter",
		Value:      "status:Success",
		Sort:       "started:desc",
		Order:      0,
	}

	require.NoError(t, project.InsertRunFilter(context.TODO(), db, filter))

	require.NoError(t, project.UpdateRunFilterOrder(context.TODO(), db, proj.Key, "My Filter", 5))

	loaded, err := project.LoadRunFilterByNameAndProjectKey(context.TODO(), db, proj.Key, "My Filter")
	require.NoError(t, err)
	assert.Equal(t, int64(5), loaded.Order)
}

func Test_DeleteRunFilter(t *testing.T) {
	db, cache := test.SetupPG(t)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	filter := &sdk.ProjectRunFilter{
		ProjectKey: proj.Key,
		Name:       "My Filter",
		Value:      "status:Success",
		Sort:       "started:desc",
		Order:      0,
	}

	require.NoError(t, project.InsertRunFilter(context.TODO(), db, filter))

	require.NoError(t, project.DeleteRunFilter(db, proj.Key, filter.ID))

	_, err := project.LoadRunFilterByNameAndProjectKey(context.TODO(), db, proj.Key, "My Filter")
	require.Error(t, err)
	assert.True(t, sdk.ErrorIs(err, sdk.ErrNotFound))
}

func Test_RecomputeRunFilterOrder(t *testing.T) {
	db, cache := test.SetupPG(t)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	// Create filters with order 0, 1, 2, 3
	filter0 := &sdk.ProjectRunFilter{ProjectKey: proj.Key, Name: "Filter 0", Value: "status:Success", Sort: "started:desc", Order: 0}
	filter1 := &sdk.ProjectRunFilter{ProjectKey: proj.Key, Name: "Filter 1", Value: "status:Fail", Sort: "started:desc", Order: 1}
	filter2 := &sdk.ProjectRunFilter{ProjectKey: proj.Key, Name: "Filter 2", Value: "branch:main", Sort: "started:desc", Order: 2}
	filter3 := &sdk.ProjectRunFilter{ProjectKey: proj.Key, Name: "Filter 3", Value: "branch:dev", Sort: "started:desc", Order: 3}

	require.NoError(t, project.InsertRunFilter(context.TODO(), db, filter0))
	require.NoError(t, project.InsertRunFilter(context.TODO(), db, filter1))
	require.NoError(t, project.InsertRunFilter(context.TODO(), db, filter2))
	require.NoError(t, project.InsertRunFilter(context.TODO(), db, filter3))

	// Delete filter with order 1
	require.NoError(t, project.DeleteRunFilter(db, proj.Key, filter1.ID))

	// Recompute orders - should compact to 0, 1, 2
	require.NoError(t, project.RecomputeRunFilterOrder(context.TODO(), db, proj.Key))

	filters, err := project.LoadRunFiltersByProjectKey(context.TODO(), db, proj.Key)
	require.NoError(t, err)
	require.Len(t, filters, 3)

	// Orders should be compacted: 0, 1, 2
	assert.Equal(t, "Filter 0", filters[0].Name)
	assert.Equal(t, int64(0), filters[0].Order)
	assert.Equal(t, "Filter 2", filters[1].Name)
	assert.Equal(t, int64(1), filters[1].Order)
	assert.Equal(t, "Filter 3", filters[2].Name)
	assert.Equal(t, int64(2), filters[2].Order)
}

func Test_RunFilter_UTF8Support(t *testing.T) {
	db, cache := test.SetupPG(t)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	// Test with emojis and special characters
	filter := &sdk.ProjectRunFilter{
		ProjectKey: proj.Key,
		Name:       "My Filter 🚀 with émojis",
		Value:      "status:Success branch:féature/日本語",
		Sort:       "started:desc",
		Order:      0,
	}

	require.NoError(t, project.InsertRunFilter(context.TODO(), db, filter))

	loaded, err := project.LoadRunFilterByNameAndProjectKey(context.TODO(), db, proj.Key, "My Filter 🚀 with émojis")
	require.NoError(t, err)
	assert.Equal(t, filter.Name, loaded.Name)
	assert.Equal(t, filter.Value, loaded.Value)
}
