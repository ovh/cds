package api

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/engine/api/migrate"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
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

		srvs, err := services.LoadAll(ctx, api.mustDB(), services.LoadOptions.WithStatus)
		if err != nil {
			return err
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

// computeGlobalStatus returns global status
func (api *API) computeGlobalStatus(srvs []sdk.Service) sdk.MonitoringStatus {
	mStatus := sdk.MonitoringStatus{}

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
	api.Metrics.nbArtifacts = stats.Int64("cds/cds-api/nb_artifacts", "nb_artifacts", stats.UnitDimensionless)
	api.Metrics.nbWorkerModels = stats.Int64("cds/cds-api/nb_worker_models", "nb_worker_models", stats.UnitDimensionless)
	api.Metrics.nbWorkflowRuns = stats.Int64("cds/cds-api/nb_workflow_runs", "nb_workflow_runs", stats.UnitDimensionless)
	api.Metrics.nbWorkflowNodeRuns = stats.Int64("cds/cds-api/nb_workflow_node_runs", "nb_workflow_node_runs", stats.UnitDimensionless)
	api.Metrics.nbMaxWorkersBuilding = stats.Int64("cds/cds-api/nb_max_workers_building", "nb_max_workers_building", stats.UnitDimensionless)
	api.Metrics.queue = stats.Int64("cds/cds-api/queue", "queue", stats.UnitDimensionless)
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
		telemetry.NewViewLast("cds/nb_artifacts", api.Metrics.nbArtifacts, nil),
		telemetry.NewViewLast("cds/nb_worker_models", api.Metrics.nbWorkerModels, nil),
		telemetry.NewViewLast("cds/nb_workflow_runs", api.Metrics.nbWorkflowRuns, nil),
		telemetry.NewViewLast("cds/nb_workflow_node_runs", api.Metrics.nbWorkflowNodeRuns, nil),
		telemetry.NewViewLast("cds/nb_max_workers_building", api.Metrics.nbMaxWorkersBuilding, nil),
		telemetry.NewViewLast("cds/queue", api.Metrics.queue, tagsRange),
		telemetry.NewViewCount("cds/workflow_runs_started", api.Metrics.WorkflowRunStarted, tagsService),
		telemetry.NewViewCount("cds/workflow_runs_failed", api.Metrics.WorkflowRunFailed, tagsService),
		telemetry.NewViewLast("cds/workflow_runs_mark_to_delete", api.Metrics.WorkflowRunsMarkToDelete, tagsService),
		telemetry.NewViewCount("cds/workflow_runs_deleted", api.Metrics.WorkflowRunsDeleted, tagsService),
		telemetry.NewViewLast("cds/database_conn", api.Metrics.DatabaseConns, tagsService),
	)

	api.computeMetrics(ctx)

	return err
}

func (api *API) computeMetrics(ctx context.Context) {
	tags := telemetry.ContextGetTags(ctx, telemetry.TagServiceType, telemetry.TagServiceName)
	ctx, err := tag.New(ctx, tags...)
	if err != nil {
		log.Error(ctx, "api.computeMetrics> unable to tag observability context: %v", err)
	}

	api.GoRoutines.Run(ctx, "api.computeMetrics", func(ctx context.Context) {
		tick := time.NewTicker(9 * time.Second).C
		for {
			select {
			case <-ctx.Done():
				if ctx.Err() != nil {
					log.Error(ctx, "Exiting metrics.Initialize: %v", ctx.Err())
					return
				}
			case <-tick:
				api.countMetric(ctx, api.Metrics.nbUsers, "SELECT COUNT(1) FROM \"authentified_user\"")
				api.countMetric(ctx, api.Metrics.nbApplications, "SELECT COUNT(1) FROM application")
				api.countMetric(ctx, api.Metrics.nbProjects, "SELECT COUNT(1) FROM project")
				api.countMetric(ctx, api.Metrics.nbGroups, "SELECT COUNT(1) FROM \"group\"")
				api.countMetric(ctx, api.Metrics.nbPipelines, "SELECT COUNT(1) FROM pipeline")
				api.countMetric(ctx, api.Metrics.nbWorkflows, "SELECT COUNT(1) FROM workflow")
				api.countMetric(ctx, api.Metrics.nbArtifacts, "SELECT COUNT(1) FROM workflow_node_run_artifacts")
				api.countMetric(ctx, api.Metrics.nbWorkerModels, "SELECT COUNT(1) FROM worker_model")
				api.countMetric(ctx, api.Metrics.nbWorkflowRuns, "SELECT COALESCE(MAX(id), 0) FROM workflow_run")
				api.countMetric(ctx, api.Metrics.nbWorkflowNodeRuns, "SELECT COALESCE(MAX(id),0) FROM workflow_node_run")
				api.countMetric(ctx, api.Metrics.nbMaxWorkersBuilding, "SELECT COUNT(1) FROM worker where status = 'Building'")

				telemetry.Record(ctx, api.Metrics.DatabaseConns, int64(api.DBConnectionFactory.DB().Stats().OpenConnections))

				now := time.Now()
				now10s := now.Add(-10 * time.Second)
				now30s := now.Add(-30 * time.Second)
				now1min := now.Add(-1 * time.Minute)
				now2min := now.Add(-2 * time.Minute)
				now5min := now.Add(-5 * time.Minute)
				now10min := now.Add(-10 * time.Minute)

				queryBuilding := "SELECT COUNT(1) FROM workflow_node_run_job where status = 'Building'"
				query := "select COUNT(1) from workflow_node_run_job where queued > $1 and queued <= $2 and status = 'Waiting'"
				queryOld := "select COUNT(1) from workflow_node_run_job where queued < $1 and status = 'Waiting'"

				api.countMetricRange(ctx, "building", "all", api.Metrics.queue, queryBuilding)
				api.countMetricRange(ctx, "waiting", "10_less_10s", api.Metrics.queue, query, now10s, now)
				api.countMetricRange(ctx, "waiting", "20_more_10s_less_30s", api.Metrics.queue, query, now30s, now10s)
				api.countMetricRange(ctx, "waiting", "30_more_30s_less_1min", api.Metrics.queue, query, now1min, now30s)
				api.countMetricRange(ctx, "waiting", "40_more_1min_less_2min", api.Metrics.queue, query, now2min, now1min)
				api.countMetricRange(ctx, "waiting", "50_more_2min_less_5min", api.Metrics.queue, query, now5min, now2min)
				api.countMetricRange(ctx, "waiting", "60_more_5min_less_10min", api.Metrics.queue, query, now10min, now5min)
				api.countMetricRange(ctx, "waiting", "70_more_10min", api.Metrics.queue, queryOld, now10min)

				api.processStatusMetrics(ctx)
			}
		}
	})
}

func (api *API) countMetric(ctx context.Context, v *stats.Int64Measure, query string) {
	n, err := api.mustDB().SelectInt(query)
	if err != nil {
		log.Warning(ctx, "metrics>Errors while fetching count %s: %v", query, err)
	}
	telemetry.Record(ctx, v, n)
}

func (api *API) countMetricRange(ctx context.Context, status string, timerange string, v *stats.Int64Measure, query string, args ...interface{}) {
	n, err := api.mustDB().SelectInt(query, args...)
	if err != nil {
		log.Warning(ctx, "metrics>Errors while fetching count range %s: %v", query, err)
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
			log.Warning(ctx, "metrics>Errors while FindAndRegisterViewLast %s: %v", item, err)
			continue
		}
		telemetry.Record(ctx, v.Measure, number)
	}
}
