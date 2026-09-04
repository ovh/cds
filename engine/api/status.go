package api

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/rockbears/log"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/engine/api/migrate"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

const (
	// queueMetricsInterval paces the queue metrics, which drive scheduling dashboards and alerting
	// and therefore have to stay close to real time.
	queueMetricsInterval = 9 * time.Second
	// inventoryMetricsInterval paces the counts of what CDS holds. They move slowly and each of them
	// is a full table scan, so they are refreshed at a rate that does not weigh on the database.
	inventoryMetricsInterval = 5 * time.Minute
	// inventoryMetricsQueryTimeout caps a single count. A count that cannot complete within it is
	// worth losing rather than holding a connection for the next tick to pile onto.
	inventoryMetricsQueryTimeout = 2 * time.Minute
	// inventoryMetricsCacheKey holds the counts one instance read for all of them. Reading them once
	// per instance multiplies over the replicas a full scan of the largest tables CDS has, for
	// numbers that cannot differ from one instance to the next.
	inventoryMetricsCacheKey = "api:metrics:inventory"
)

// Status returns status, implements interface service.Service
func (api *API) Status(ctx context.Context) *sdk.MonitoringStatus {
	m := api.NewMonitoringStatus()
	m.ServiceName = event.GetCDSName()

	m.AddLine(sdk.MonitoringStatusLine{Component: "Hostname", Value: event.GetHostname(), Status: sdk.MonitoringStatusOK})
	m.AddLine(sdk.MonitoringStatusLine{Component: "CDSName", Value: api.Name(), Status: sdk.MonitoringStatusOK})
	m.AddLine(api.Router.StatusPanic())
	m.AddLine(event.Status(ctx))
	m.AddLine(api.SharedStorage.Status(ctx))
	m.AddLine(mail.Status(ctx))
	m.AddLine(api.DBConnectionFactory.Status(ctx))
	m.AddLine(workermodel.Status(api.mustDB()))
	m.AddLine(migrate.Status(api.mustDB()))

	return m
}

func (api *API) statusHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var status = http.StatusOK
		if api.Router.panicked {
			status = http.StatusServiceUnavailable
		}

		// Always load services to ensure that database connection is ok.
		srvs, err := services.LoadAll(ctx, api.mustDB(), services.LoadOptions.WithStatus)
		if err != nil {
			return err
		}

		// If there is a valid session and user is maintainer, allows to get status details.
		currentConsumer := getUserConsumer(ctx)
		if currentConsumer == nil || !isMaintainer(ctx) {
			mStatus := api.computeGlobalPublicStatus()
			return service.WriteJSON(w, mStatus, status)
		}

		mStatus := api.computeGlobalStatus(srvs)
		return service.WriteJSON(w, mStatus, status)
	}
}

type computeGlobalNumbers struct {
	nbSrv    int
	nbOK     int
	nbAlerts int
	nbWarn   int
}

var (
	tagRange       tag.Key
	tagStatus      tag.Key
	tagServiceName tag.Key
	tagService     tag.Key
	tagsService    []tag.Key
)

// computeGlobalPublicStatus returns global public status
func (api *API) computeGlobalPublicStatus() sdk.MonitoringStatus {
	return sdk.MonitoringStatus{
		Lines: []sdk.MonitoringStatusLine{
			{
				Status:    sdk.MonitoringStatusOK,
				Component: "Global/Maintenance",
				Value:     fmt.Sprintf("%v", api.Maintenance),
			},
		},
	}
}

// computeGlobalStatus returns global status
func (api *API) computeGlobalStatus(srvs []sdk.Service) sdk.MonitoringStatus {
	mStatus := sdk.MonitoringStatus{Now: time.Now()}

	var version string
	versionOk := true
	linesGlobal := []sdk.MonitoringStatusLine{}

	resume := map[string]computeGlobalNumbers{
		sdk.TypeAPI:           {},
		sdk.TypeCDN:           {},
		sdk.TypeRepositories:  {},
		sdk.TypeVCS:           {},
		sdk.TypeHooks:         {},
		sdk.TypeHatchery:      {},
		sdk.TypeDBMigrate:     {},
		sdk.TypeElasticsearch: {},
	}
	var nbg computeGlobalNumbers
	for _, s := range srvs {
		var nbOK, nbWarn, nbAlert int
		for i := range s.MonitoringStatus.Lines {
			l := s.MonitoringStatus.Lines[i]
			mStatus.Lines = append(mStatus.Lines, l)

			switch l.Status {
			case sdk.MonitoringStatusOK:
				nbOK++
			case sdk.MonitoringStatusWarn:
				nbWarn++
			default:
				nbAlert++
			}

			// services should have same version
			if strings.Contains(l.Component, "Version") {
				if version == "" {
					version = l.Value
				} else if version != l.Value && versionOk {
					versionOk = false
					linesGlobal = append(linesGlobal, sdk.MonitoringStatusLine{
						Status:    sdk.MonitoringStatusWarn,
						Component: "Global/Version",
						Value:     fmt.Sprintf("%s vs %s", version, l.Value),
					})
				}
			}
		}

		t := resume[s.Type]
		t.nbOK += nbOK
		t.nbWarn += nbWarn
		t.nbAlerts += nbAlert
		t.nbSrv++
		resume[s.Type] = t

		nbg.nbOK += nbOK
		nbg.nbWarn += nbWarn
		nbg.nbAlerts += nbAlert
		nbg.nbSrv++
	}

	if versionOk {
		linesGlobal = append(linesGlobal, sdk.MonitoringStatusLine{
			Status:    sdk.MonitoringStatusOK,
			Component: "Global/Version",
			Value:     version,
		})
	}

	linesGlobal = append(linesGlobal, sdk.MonitoringStatusLine{
		Status:    sdk.MonitoringStatusOK,
		Component: "Global/Maintenance",
		Value:     fmt.Sprintf("%v", api.Maintenance),
	})

	for stype, r := range resume {
		linesGlobal = append(linesGlobal, sdk.MonitoringStatusLine{
			Status:    api.computeGlobalStatusByNumbers(r),
			Component: fmt.Sprintf("Global/%s", stype),
			Value:     fmt.Sprintf("%d", r.nbSrv),
		})
	}

	linesGlobal = append(linesGlobal, sdk.MonitoringStatusLine{
		Status:    api.computeGlobalStatusByNumbers(nbg),
		Component: "Global/Status",
		Value:     fmt.Sprintf("%d services", len(srvs)),
	})

	sort.Slice(linesGlobal, func(i, j int) bool {
		return linesGlobal[i].Component < linesGlobal[j].Component
	})

	mStatus.Lines = append(linesGlobal, mStatus.Lines...)
	return mStatus
}

func (api *API) computeGlobalStatusByNumbers(s computeGlobalNumbers) string {
	r := sdk.MonitoringStatusOK
	if s.nbAlerts > 0 {
		r = sdk.MonitoringStatusAlert
	} else if s.nbWarn > 0 {
		r = sdk.MonitoringStatusWarn
	}
	return r
}

func (api *API) initMetrics(ctx context.Context) error {
	log.Info(ctx, "Metrics initialized for %s/%s", api.Type(), api.Name())

	// TODO refactor all the metrics name to replace "cds-api" by "api.Type()"
	api.Metrics.nbUsers = stats.Int64("cds/cds-api/nb_users", "number of users", stats.UnitDimensionless)
	api.Metrics.nbApplications = stats.Int64("cds/cds-api/nb_applications", "nb_applications", stats.UnitDimensionless)
	api.Metrics.nbProjects = stats.Int64("cds/cds-api/nb_projects", "nb_projects", stats.UnitDimensionless)
	api.Metrics.nbGroups = stats.Int64("cds/cds-api/nb_groups", "nb_groups", stats.UnitDimensionless)
	api.Metrics.nbPipelines = stats.Int64("cds/cds-api/nb_pipelines", "nb_pipelines", stats.UnitDimensionless)
	api.Metrics.nbWorkflows = stats.Int64("cds/cds-api/nb_workflows", "nb_workflows", stats.UnitDimensionless)
	api.Metrics.nbWorkflowsAsCodeV2 = stats.Int64("cds/cds-api/nb_workflows_as_code_v2", "nb_workflows_as_code_v2", stats.UnitDimensionless)
	api.Metrics.nbArtifacts = stats.Int64("cds/cds-api/nb_artifacts", "nb_artifacts", stats.UnitDimensionless)
	api.Metrics.nbWorkerModels = stats.Int64("cds/cds-api/nb_worker_models", "nb_worker_models", stats.UnitDimensionless)
	api.Metrics.nbWorkflowRuns = stats.Int64("cds/cds-api/nb_workflow_runs", "nb_workflow_runs", stats.UnitDimensionless)
	api.Metrics.nbWorkflowNodeRuns = stats.Int64("cds/cds-api/nb_workflow_node_runs", "nb_workflow_node_runs", stats.UnitDimensionless)
	api.Metrics.nbMaxWorkersBuilding = stats.Int64("cds/cds-api/nb_max_workers_building", "nb_max_workers_building", stats.UnitDimensionless)
	api.Metrics.queue = stats.Int64("cds/cds-api/queue", "queue", stats.UnitDimensionless)
	api.Metrics.v2Queue = stats.Int64("cds/cds-api/v2_queue", "v2_queue", stats.UnitDimensionless)
	api.Metrics.WorkflowRunsMarkToDelete = stats.Int64(
		fmt.Sprintf("cds/cds-api/%s/workflow_runs_mark_to_delete", api.Name()),
		"number of workflow runs mark to delete",
		stats.UnitDimensionless)
	api.Metrics.WorkflowRunsDeleted = stats.Int64(
		fmt.Sprintf("cds/cds-api/%s/workflow_runs_deleted", api.Name()),
		"number of workflow runs deleted",
		stats.UnitDimensionless)
	api.Metrics.WorkflowRunStarted = stats.Int64(
		fmt.Sprintf("cds/cds-api/%s/workflow_runs_started", api.Name()),
		"number of started workflow runs",
		stats.UnitDimensionless)
	api.Metrics.WorkflowRunFailed = stats.Int64(
		fmt.Sprintf("cds/cds-api/%s/workflow_runs_failed", api.Name()),
		"number of failed workflow runs",
		stats.UnitDimensionless)
	api.Metrics.DatabaseConns = stats.Int64(
		fmt.Sprintf("cds/cds-api/%s/database_conn", api.Name()),
		"number database connections",
		stats.UnitDimensionless)
	api.Metrics.RunResultSynchronized = stats.Int64(
		fmt.Sprintf("cds/cds-api/%s/run_results_synchronized", api.Name()),
		"number of synchronized run results",
		stats.UnitDimensionless)
	api.Metrics.RunResultToSynchronized = stats.Int64(
		fmt.Sprintf("cds/cds-api/%s/run_results_to_synchronized", api.Name()),
		"number of non synchronized run results",
		stats.UnitDimensionless)
	api.Metrics.RunResultSynchronizedError = stats.Int64(
		fmt.Sprintf("cds/cds-api/%s/run_results_synchronized_error", api.Name()),
		"number of synchronized run results with error",
		stats.UnitDimensionless)

	tagRange, _ = tag.NewKey("range")
	tagStatus, _ = tag.NewKey("status")

	tagServiceType := telemetry.MustNewKey(telemetry.TagServiceType)
	tagServiceName := telemetry.MustNewKey(telemetry.TagServiceName)
	tagsRange := []tag.Key{tagRange, tagStatus}
	tagsService = []tag.Key{tagServiceName, tagServiceType}

	err := telemetry.RegisterView(ctx,
		telemetry.NewViewLast("cds/nb_users", api.Metrics.nbUsers, nil),
		telemetry.NewViewLast("cds/nb_applications", api.Metrics.nbApplications, nil),
		telemetry.NewViewLast("cds/nb_projects", api.Metrics.nbProjects, nil),
		telemetry.NewViewLast("cds/nb_groups", api.Metrics.nbGroups, nil),
		telemetry.NewViewLast("cds/nb_pipelines", api.Metrics.nbPipelines, nil),
		telemetry.NewViewLast("cds/nb_workflows", api.Metrics.nbWorkflows, nil),
		telemetry.NewViewLast("cds/nb_workflows_as_code_v2", api.Metrics.nbWorkflowsAsCodeV2, nil),
		telemetry.NewViewLast("cds/nb_artifacts", api.Metrics.nbArtifacts, nil),
		telemetry.NewViewLast("cds/nb_worker_models", api.Metrics.nbWorkerModels, nil),
		telemetry.NewViewLast("cds/nb_workflow_runs", api.Metrics.nbWorkflowRuns, nil),
		telemetry.NewViewLast("cds/nb_workflow_node_runs", api.Metrics.nbWorkflowNodeRuns, nil),
		telemetry.NewViewLast("cds/nb_max_workers_building", api.Metrics.nbMaxWorkersBuilding, nil),
		telemetry.NewViewLast("cds/queue", api.Metrics.queue, tagsRange),
		telemetry.NewViewLast("cds/v2_queue", api.Metrics.v2Queue, tagsRange),
		telemetry.NewViewCount("cds/workflow_runs_started", api.Metrics.WorkflowRunStarted, tagsService),
		telemetry.NewViewCount("cds/workflow_runs_failed", api.Metrics.WorkflowRunFailed, tagsService),
		telemetry.NewViewLast("cds/workflow_runs_mark_to_delete", api.Metrics.WorkflowRunsMarkToDelete, tagsService),
		telemetry.NewViewCount("cds/workflow_runs_deleted", api.Metrics.WorkflowRunsDeleted, tagsService),
		telemetry.NewViewLast("cds/database_conn", api.Metrics.DatabaseConns, tagsService),
		telemetry.NewViewLast("cds/run_results_synchronized", api.Metrics.RunResultSynchronized, tagsService),
		telemetry.NewViewLast("cds/run_results_to_synchronized", api.Metrics.RunResultToSynchronized, tagsService),
		telemetry.NewViewLast("cds/run_results_to_synchronized_error", api.Metrics.RunResultSynchronizedError, tagsService),
	)

	// The pool of the database describes itself: the connections in use, and above all how long the
	// callers waited for one. cds/database_conn only counts the connections that are open, which says
	// that the pool is full but nothing of the queue behind it, and a queue is what a saturated pool
	// is read from.
	if db := api.DBConnectionFactory.DB(); db != nil {
		if err := telemetry.RegisterCollector(ctx, collectors.NewDBStatsCollector(db, api.DBConnectionFactory.DBName)); err != nil {
			log.Error(ctx, "unable to expose the metrics of the database pool: %v", err)
		}
	}

	api.computeMetrics(ctx)

	return err
}

func (api *API) computeMetrics(ctx context.Context) {
	tags := telemetry.ContextGetTags(ctx, telemetry.TagServiceType, telemetry.TagServiceName)
	ctx, err := tag.New(ctx, tags...)
	if err != nil {
		log.Error(ctx, "api.computeMetrics> unable to tag observability context: %v", err)
	}

	api.computeQueueMetrics(ctx)
	api.computeInventoryMetrics(ctx)
}

// computeQueueMetrics refreshes what a scheduling dashboard reads: the queues, which are small hot
// tables filtered on an indexed status, and the state of the process itself.
func (api *API) computeQueueMetrics(ctx context.Context) {
	api.GoRoutines.RunWithRestart(ctx, "api.computeQueueMetrics", func(ctx context.Context) {
		tick := time.NewTicker(queueMetricsInterval).C
		for {
			select {
			case <-ctx.Done():
				if ctx.Err() != nil {
					log.Error(ctx, "Exiting metrics.Initialize: %v", ctx.Err())
					return
				}
			case <-tick:
				telemetry.Record(ctx, api.Metrics.DatabaseConns, int64(api.DBConnectionFactory.DB().Stats().OpenConnections))

				// Queue common
				now := time.Now()
				now10s, now30s, now1min, now2min, now5min, now10min := now.Add(-10*time.Second), now.Add(-30*time.Second), now.Add(-1*time.Minute), now.Add(-2*time.Minute), now.Add(-5*time.Minute), now.Add(-10*time.Minute)

				// V1 queue metrics
				queryBuilding := "SELECT COUNT(1) FROM workflow_node_run_job WHERE status = 'Building'"
				queryWaiting := "SELECT COUNT(1) FROM workflow_node_run_job WHERE status = 'Waiting'"
				queryWaitingInterval := "SELECT COUNT(1) FROM workflow_node_run_job WHERE queued > $1 AND queued <= $2 AND status = 'Waiting'"
				queryWaitingOlder := "SELECT COUNT(1) FROM workflow_node_run_job WHERE queued < $1 AND status = 'Waiting'"
				api.countMetricRange(ctx, "building", "all", api.Metrics.queue, queryBuilding)
				api.countMetricRange(ctx, "waiting", "all", api.Metrics.queue, queryWaiting)
				api.countMetricRange(ctx, "waiting", "10_less_10s", api.Metrics.queue, queryWaitingInterval, now10s, now)
				api.countMetricRange(ctx, "waiting", "20_more_10s_less_30s", api.Metrics.queue, queryWaitingInterval, now30s, now10s)
				api.countMetricRange(ctx, "waiting", "30_more_30s_less_1min", api.Metrics.queue, queryWaitingInterval, now1min, now30s)
				api.countMetricRange(ctx, "waiting", "40_more_1min_less_2min", api.Metrics.queue, queryWaitingInterval, now2min, now1min)
				api.countMetricRange(ctx, "waiting", "50_more_2min_less_5min", api.Metrics.queue, queryWaitingInterval, now5min, now2min)
				api.countMetricRange(ctx, "waiting", "60_more_5min_less_10min", api.Metrics.queue, queryWaitingInterval, now10min, now5min)
				api.countMetricRange(ctx, "waiting", "70_more_10min", api.Metrics.queue, queryWaitingOlder, now10min)

				// V2 queue metrics
				queryV2Building := "SELECT COUNT(1) FROM v2_workflow_run_job WHERE status = 'Building'"
				queryV2Scheduling := "SELECT COUNT(1) FROM v2_workflow_run_job WHERE status = 'Scheduling'"
				queryV2SchedulingInterval := "SELECT COUNT(1) FROM v2_workflow_run_job WHERE scheduled > $1 AND scheduled <= $2 AND status = 'Scheduling'"
				queryV2SchedulingOlder := "SELECT COUNT(1) FROM v2_workflow_run_job WHERE scheduled < $1 AND status = 'Scheduling'"
				queryV2Waiting := "SELECT COUNT(1) FROM v2_workflow_run_job WHERE status = 'Waiting'"
				queryV2WaitingInterval := "SELECT COUNT(1) FROM v2_workflow_run_job WHERE queued > $1 AND queued <= $2 AND status = 'Waiting'"
				queryV2WaitingOlder := "SELECT COUNT(1) FROM v2_workflow_run_job WHERE queued < $1 AND status = 'Waiting'"
				api.countMetricRange(ctx, "v2_building", "all", api.Metrics.v2Queue, queryV2Building)
				api.countMetricRange(ctx, "v2_scheduling", "all", api.Metrics.v2Queue, queryV2Scheduling)
				api.countMetricRange(ctx, "v2_scheduling", "10_less_10s", api.Metrics.v2Queue, queryV2SchedulingInterval, now10s, now)
				api.countMetricRange(ctx, "v2_scheduling", "20_more_10s_less_30s", api.Metrics.v2Queue, queryV2SchedulingInterval, now30s, now10s)
				api.countMetricRange(ctx, "v2_scheduling", "30_more_30s_less_1min", api.Metrics.v2Queue, queryV2SchedulingInterval, now1min, now30s)
				api.countMetricRange(ctx, "v2_scheduling", "40_more_1min_less_2min", api.Metrics.v2Queue, queryV2SchedulingInterval, now2min, now1min)
				api.countMetricRange(ctx, "v2_scheduling", "50_more_2min_less_5min", api.Metrics.v2Queue, queryV2SchedulingInterval, now5min, now2min)
				api.countMetricRange(ctx, "v2_scheduling", "60_more_5min_less_10min", api.Metrics.v2Queue, queryV2SchedulingInterval, now10min, now5min)
				api.countMetricRange(ctx, "v2_scheduling", "70_more_10min", api.Metrics.v2Queue, queryV2SchedulingOlder, now10min)
				api.countMetricRange(ctx, "v2_waiting", "all", api.Metrics.v2Queue, queryV2Waiting)
				api.countMetricRange(ctx, "v2_waiting", "10_less_10s", api.Metrics.v2Queue, queryV2WaitingInterval, now10s, now)
				api.countMetricRange(ctx, "v2_waiting", "20_more_10s_less_30s", api.Metrics.v2Queue, queryV2WaitingInterval, now30s, now10s)
				api.countMetricRange(ctx, "v2_waiting", "30_more_30s_less_1min", api.Metrics.v2Queue, queryV2WaitingInterval, now1min, now30s)
				api.countMetricRange(ctx, "v2_waiting", "40_more_1min_less_2min", api.Metrics.v2Queue, queryV2WaitingInterval, now2min, now1min)
				api.countMetricRange(ctx, "v2_waiting", "50_more_2min_less_5min", api.Metrics.v2Queue, queryV2WaitingInterval, now5min, now2min)
				api.countMetricRange(ctx, "v2_waiting", "60_more_5min_less_10min", api.Metrics.v2Queue, queryV2WaitingInterval, now10min, now5min)
				api.countMetricRange(ctx, "v2_waiting", "70_more_10min", api.Metrics.v2Queue, queryV2WaitingOlder, now10min)
				api.processStatusMetrics(ctx)
			}
		}
	})
}

// computeInventoryMetrics counts what CDS holds. None of those counts can be answered from an index:
// they are sequential scans of the largest tables of the instance, so they run on their own slow
// ticker rather than with the queues. Refreshing them every few seconds kept a connection of every
// API replica permanently busy scanning, and evicted from the shared buffers the pages every other
// query needs.
func (api *API) computeInventoryMetrics(ctx context.Context) {
	api.GoRoutines.RunWithRestart(ctx, "api.computeInventoryMetrics", func(ctx context.Context) {
		tick := time.NewTicker(inventoryMetricsInterval).C
		for {
			select {
			case <-ctx.Done():
				if ctx.Err() != nil {
					log.Error(ctx, "Exiting metrics.Initialize: %v", ctx.Err())
					return
				}
			case <-tick:
				api.refreshInventoryMetrics(ctx)
			}
		}
	})
}

// inventoryCount is one of the counts of what CDS holds, with the measure it feeds. The key is what
// the value is shared under, so it stays the same while a mix of versions runs.
type inventoryCount struct {
	key     string
	measure *stats.Int64Measure
	query   string
}

func (api *API) inventoryCounts() []inventoryCount {
	return []inventoryCount{
		// Common
		{"users", api.Metrics.nbUsers, `SELECT COUNT(1) FROM "authentified_user"`},
		{"projects", api.Metrics.nbProjects, "SELECT COUNT(1) FROM project"},
		{"groups", api.Metrics.nbGroups, `SELECT COUNT(1) FROM "group"`},

		// V1
		{"applications", api.Metrics.nbApplications, "SELECT COUNT(1) FROM application"},
		{"pipelines", api.Metrics.nbPipelines, "SELECT COUNT(1) FROM pipeline"},
		{"workflows", api.Metrics.nbWorkflows, "SELECT COUNT(1) FROM workflow"},
		{"artifacts", api.Metrics.nbArtifacts, "SELECT COUNT(1) FROM workflow_node_run_artifacts"},
		{"worker_models", api.Metrics.nbWorkerModels, "SELECT COUNT(1) FROM worker_model"},
		{"workflow_runs", api.Metrics.nbWorkflowRuns, "SELECT COUNT(1) FROM workflow_run"},
		{"workflow_node_runs", api.Metrics.nbWorkflowNodeRuns, "SELECT COUNT(1) FROM workflow_node_run"},
		{"max_workers_building", api.Metrics.nbMaxWorkersBuilding, "SELECT COUNT(1) FROM worker where status = 'Building'"},
		{"run_results_synchronized", api.Metrics.RunResultSynchronized, "SELECT COUNT(1) FROM workflow_run_result where sync is NOT NULL"},
		{"run_results_to_synchronize", api.Metrics.RunResultToSynchronized, "SELECT COUNT(1) FROM workflow_run_result where sync is NULL"},
		{"run_results_synchronized_error", api.Metrics.RunResultSynchronizedError, "SELECT COUNT(1) FROM workflow_run_result where sync ? 'error'"},

		// V2
		{"workflows_as_code_v2", api.Metrics.nbWorkflowsAsCodeV2, "select count(distinct(project_repository_id,name)) from entity where type = 'Workflow'"},
	}
}

// refreshInventoryMetrics records the counts of what CDS holds. They describe the database, so they
// are the same read from any instance, and each of them is a full table scan: one instance reads
// them and shares them, the others record what it read.
//
// Every instance records them rather than only the one that read them: a measure only exists where
// it was recorded, so leaving the others silent would spread one number over as many series as there
// are instances, each holding whatever it last saw. Aggregating those is only right for a count that
// never goes down, and several of these do.
func (api *API) refreshInventoryMetrics(ctx context.Context) {
	counts := api.inventoryCounts()

	// Held for the interval and not released: the instance that reads them is whichever one ticks
	// first once the last read has aged out.
	locked, err := api.Cache.Lock(cache.Key(inventoryMetricsCacheKey, "lock"), inventoryMetricsInterval, 0, 1)
	if err != nil {
		log.Warn(ctx, "metrics> unable to take the inventory counts lock: %v", err)
	}

	if !locked {
		var shared map[string]int64
		found, err := api.Cache.Get(inventoryMetricsCacheKey, &shared)
		if err != nil {
			log.Warn(ctx, "metrics> unable to read the shared inventory counts: %v", err)
		}
		if found {
			for _, c := range counts {
				if n, ok := shared[c.key]; ok {
					telemetry.Record(ctx, c.measure, n)
				}
			}
			return
		}
		// Nothing has been shared yet, which is the first pass of a cluster starting: reading them
		// here costs a duplicated pass, reporting nothing costs the metrics until the next tick.
	}

	shared := make(map[string]int64, len(counts))
	for _, c := range counts {
		n, err := api.count(ctx, c.query)
		if err != nil {
			// Reporting nothing leaves the last count in place. Reporting the zero of a count that
			// did not happen reads as the disappearance of everything it counts.
			log.Warn(ctx, "metrics> unable to count %s: %v", c.key, err)
			continue
		}
		shared[c.key] = n
		telemetry.Record(ctx, c.measure, n)
	}

	// Kept longer than the interval so that an instance ticking while no read is in progress still
	// finds them.
	if err := api.Cache.SetWithDuration(inventoryMetricsCacheKey, shared, 3*inventoryMetricsInterval); err != nil {
		log.Warn(ctx, "metrics> unable to share the inventory counts: %v", err)
	}
}

func (api *API) count(ctx context.Context, query string) (int64, error) {
	// Bounded so that a scan that outgrew the instance releases its connection instead of holding one
	// of the few the pool has until it completes.
	ctxQuery, cancel := context.WithTimeout(ctx, inventoryMetricsQueryTimeout)
	defer cancel()

	n, err := api.mustDBWithCtx(ctxQuery).SelectInt(query)
	return n, sdk.WithStack(err)
}

func (api *API) countMetricRange(ctx context.Context, status string, timerange string, v *stats.Int64Measure, query string, args ...interface{}) {
	n, err := api.mustDB().SelectInt(query, args...)
	if err != nil {
		log.Warn(ctx, "metrics>Errors while fetching count range %s: %v", query, err)
	}
	ctx, _ = tag.New(ctx, tag.Upsert(tagStatus, status), tag.Upsert(tagRange, timerange))
	telemetry.Record(ctx, v, n)
}

func (api *API) processStatusMetrics(ctx context.Context) {
	srvs, err := services.LoadAll(ctx, api.mustDB())
	if err != nil {
		log.Error(ctx, "Error while getting services list: %v", err)
		return
	}
	mStatus := api.computeGlobalStatus(srvs)

	ignoreList := []string{"version", "hostname", "time", "uptime", "cdsname"}

	for _, line := range mStatus.Lines {
		idx := strings.Index(line.Component, "/")

		var service string
		if idx >= 0 {
			service = line.Component[0:idx]
		}

		item := strings.ToLower(line.Component[idx+1:])

		if service == "Global" {
			// Global is an aggregation of status, useful only for cdsctl ui
			// we avoid to push them, with metrics pushed, aggregation have be done
			// with metrics tools (grafana, etc...)
			continue
		}

		// ignore some status line
		var found bool
		for _, v := range ignoreList {
			if v == item {
				found = true
				break
			}
		}
		if found {
			continue
		}

		// take the value if it's an integer for metrics
		// if it's not an integer, AL -> 0, OK -> 1
		number, err := strconv.ParseInt(line.Value, 10, 64)
		if err != nil {
			number = 1
			if line.Status == sdk.MonitoringStatusAlert {
				number = 0
			}
		}

		ctx, _ = tag.New(ctx, tag.Upsert(tagServiceName, service), tag.Upsert(tagService, line.Type))
		v, err := telemetry.FindAndRegisterViewLast(item, tagsService)
		if err != nil {
			log.Warn(ctx, "metrics>Errors while FindAndRegisterViewLast %s: %v", item, err)
			continue
		}
		telemetry.Record(ctx, v.Measure, number)
	}
}
