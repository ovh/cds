package telemetry

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/stretchr/testify/require"
)

type testService struct{}

func (testService) Name() string { return "test" }
func (testService) Type() string { return "api" }

// unreachableConnector stands for a pool without a database: the statistics of a pool are held by
// the pool itself, so they can be read from one that never connected.
type unreachableConnector struct{}

func (unreachableConnector) Connect(context.Context) (driver.Conn, error) {
	return nil, errors.New("no database")
}
func (unreachableConnector) Driver() driver.Driver { return nil }

func scrape(t *testing.T, ctx context.Context) string {
	t.Helper()
	w := httptest.NewRecorder()
	StatsExporter(ctx).ServeHTTP(w, httptest.NewRequest("GET", "/mon/metrics", nil))
	require.Equal(t, 200, w.Code)
	return w.Body.String()
}

// The metrics of a service say what it did; what it costs is described by the collectors of the
// runtime, which report nothing through a view and have to be registered on the registry the
// exporter serves.
func TestInit_ExposesTheRuntimeOfTheProcess(t *testing.T) {
	ctx, err := Init(context.Background(), Configuration{}, testService{})
	require.NoError(t, err)

	body := scrape(t, ctx)
	require.Contains(t, body, "go_goroutines", "the goroutine count is what a fan-out or a leak is read from")
	require.Contains(t, body, "go_memstats_heap_inuse_bytes")
	require.Contains(t, body, "process_resident_memory_bytes")
}

// A pool reports the connections in use and the time its callers spent waiting for one, which is
// what a saturated pool is read from and what counting the open connections cannot tell.
func TestRegisterCollector_ExposesADatabasePool(t *testing.T) {
	ctx, err := Init(context.Background(), Configuration{}, testService{})
	require.NoError(t, err)

	// Never connected: the collector reads the statistics of the pool, not the database behind it.
	db := sql.OpenDB(unreachableConnector{})
	defer db.Close()

	require.NoError(t, RegisterCollector(ctx, collectors.NewDBStatsCollector(db, "cds")))

	body := scrape(t, ctx)
	require.Contains(t, body, "go_sql_in_use_connections")
	require.Contains(t, body, "go_sql_wait_count_total")
	require.Contains(t, body, "go_sql_wait_duration_seconds_total")
	require.Contains(t, body, `db_name="cds"`)
}

func TestRegisterCollector_WithoutAnExporter(t *testing.T) {
	require.Error(t, RegisterCollector(context.Background(), collectors.NewGoCollector()))
}
