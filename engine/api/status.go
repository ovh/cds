package api

import (
	"bytes"
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
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// VersionHandler returns version of current uservice
func VersionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, sdk.VersionCurrent(), http.StatusOK)
	}
}

// Status returns status, implements interface service.Service
func (api *API) Status() sdk.MonitoringStatus {
	m := api.CommonMonitoring()

	m.Lines = append(m.Lines, getStatusLine(sdk.MonitoringStatusLine{Component: "Hostname", Value: event.GetHostname(), Status: sdk.MonitoringStatusOK}))
	m.Lines = append(m.Lines, getStatusLine(sdk.MonitoringStatusLine{Component: "CDSName", Value: event.GetCDSName(), Status: sdk.MonitoringStatusOK}))
	m.Lines = append(m.Lines, getStatusLine(api.Router.StatusPanic()))
	m.Lines = append(m.Lines, getStatusLine(event.Status()))
	m.Lines = append(m.Lines, getStatusLine(api.Cache.Status()))
	m.Lines = append(m.Lines, getStatusLine(sessionstore.Status))
	m.Lines = append(m.Lines, getStatusLine(objectstore.Status()))
	m.Lines = append(m.Lines, getStatusLine(mail.Status()))
	m.Lines = append(m.Lines, getStatusLine(api.DBConnectionFactory.Status()))
	m.Lines = append(m.Lines, getStatusLine(worker.Status(api.mustDB())))
	m.Lines = append(m.Lines, getStatusLine(migrate.Status(api.mustDB())))

	return m
}

func getStatusLine(s sdk.MonitoringStatusLine) sdk.MonitoringStatusLine {
	return s
}

func (api *API) statusHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var status = http.StatusOK
		if api.Router.panicked {
			status = http.StatusServiceUnavailable
		}

		srvs, err := services.All(api.mustDB())
		if err != nil {
			return err
		}

		mStatus := api.computeGlobalStatus(srvs)
		return service.WriteJSON(w, mStatus, status)
	}
}

func (api *API) smtpPingHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if deprecatedGetUser(ctx) == nil {
			return sdk.ErrForbidden
		}

		message := "mail sent"
		if err := mail.SendEmail("Ping", bytes.NewBufferString("Pong"), deprecatedGetUser(ctx).Email, false); err != nil {
			message = err.Error()
		}

		return service.WriteJSON(w, map[string]string{"message": message}, http.StatusOK)
	}
}

type computeGlobalNumbers struct {
	nbSrv       int
	nbOK        int
	nbAlerts    int
	nbWarn      int
	minInstance int
}

var (
	tagRange                tag.Key
	tagStatus               tag.Key
	tagServiceName          tag.Key
	tagService              tag.Key
	tagsService             []tag.Key
	tagsServiceAvailability []tag.Key
)

// computeGlobalStatus returns global status
func (api *API) computeGlobalStatus(srvs []sdk.Service) sdk.MonitoringStatus {
	mStatus := sdk.MonitoringStatus{}

	var version string
	versionOk := true
	linesGlobal := []sdk.MonitoringStatusLine{}

	resume := map[string]computeGlobalNumbers{
		services.TypeAPI:           {minInstance: api.Config.Status.API.MinInstance},
		services.TypeRepositories:  {minInstance: api.Config.Status.Repositories.MinInstance},
		services.TypeVCS:           {minInstance: api.Config.Status.VCS.MinInstance},
		services.TypeHooks:         {minInstance: api.Config.Status.Hooks.MinInstance},
		services.TypeHatchery:      {minInstance: api.Config.Status.Hatchery.MinInstance},
		services.TypeDBMigrate:     {minInstance: api.Config.Status.DBMigrate.MinInstance},
		services.TypeElasticsearch: {minInstance: api.Config.Status.ElasticSearch.MinInstance},
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

	for stype, r := range resume {
		if r.minInstance == 0 {
			continue
		}
		st := sdk.MonitoringStatusOK
		if r.nbSrv < r.minInstance {
			st = sdk.MonitoringStatusAlert
			nbg.nbAlerts++
		} else {
			nbg.nbOK++
		}
		percent := float64(r.nbSrv / r.minInstance)
		linesGlobal = append(linesGlobal, sdk.MonitoringStatusLine{
			Status:    st,
			Component: fmt.Sprintf("Availability/%s", stype),
			Value:     fmt.Sprintf("%f", percent),
			Type:      stype,
		})
	}

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
	} else if s.nbSrv < s.minInstance {
		r = sdk.MonitoringStatusAlert
	}
	return r
}

func (api *API) initMetrics(ctx context.Context) error {
	label := fmt.Sprintf("cds/cds-api/%s/workflow_runs_started", api.Name)
	api.Metrics.WorkflowRunStarted = stats.Int64(label, "number of started workflow runs", stats.UnitDimensionless)

	label = fmt.Sprintf("cds/cds-api/%s/workflow_runs_failed", api.Name)
	api.Metrics.WorkflowRunFailed = stats.Int64(label, "number of failed workflow runs", stats.UnitDimensionless)

	log.Info("api> Metrics initialized")

	tagCDSInstance, _ := tag.NewKey("cds")
	tags := []tag.Key{tagCDSInstance}

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

	tagRange, _ = tag.NewKey("range")
	tagStatus, _ = tag.NewKey("status")
	tagServiceName, _ = tag.NewKey("name")
	tagService, _ = tag.NewKey("service")

	tagsRange := []tag.Key{tagCDSInstance, tagRange, tagStatus}
	tagsService = []tag.Key{tagCDSInstance, tagServiceName, tagService}
	tagsServiceAvailability = []tag.Key{tagCDSInstance, tagService}

	api.computeMetrics(ctx)

	err := observability.RegisterView(
		observability.NewViewLast("nb_users", api.Metrics.nbUsers, tags),
		observability.NewViewLast("nb_applications", api.Metrics.nbApplications, tags),
		observability.NewViewLast("nb_projects", api.Metrics.nbProjects, tags),
		observability.NewViewLast("nb_groups", api.Metrics.nbGroups, tags),
		observability.NewViewLast("nb_pipelines", api.Metrics.nbPipelines, tags),
		observability.NewViewLast("nb_workflows", api.Metrics.nbWorkflows, tags),
		observability.NewViewLast("nb_artifacts", api.Metrics.nbArtifacts, tags),
		observability.NewViewLast("nb_worker_models", api.Metrics.nbWorkerModels, tags),
		observability.NewViewLast("nb_workflow_runs", api.Metrics.nbWorkflowRuns, tags),
		observability.NewViewLast("nb_workflow_node_runs", api.Metrics.nbWorkflowNodeRuns, tags),
		observability.NewViewLast("nb_max_workers_building", api.Metrics.nbMaxWorkersBuilding, tags),
		observability.NewViewLast("queue", api.Metrics.queue, tagsRange),
		observability.NewViewCount("workflow_runs_started", api.Metrics.WorkflowRunStarted, tags),
		observability.NewViewCount("workflow_runs_failed", api.Metrics.WorkflowRunFailed, tags),
	)

	return err
}

func (api *API) computeMetrics(ctx context.Context) {
	sdk.GoRoutine(ctx, "api.computeMetrics", func(ctx context.Context) {
		tick := time.NewTicker(9 * time.Second).C
		for {
			select {
			case <-ctx.Done():
				if ctx.Err() != nil {
					log.Error("Exiting metrics.Initialize: %v", ctx.Err())
					return
				}
			case <-tick:
				api.countMetric(ctx, api.Metrics.nbUsers, "SELECT COUNT(1) FROM \"user\"")
				api.countMetric(ctx, api.Metrics.nbApplications, "SELECT COUNT(1) FROM application")
				api.countMetric(ctx, api.Metrics.nbProjects, "SELECT COUNT(1) FROM project")
				api.countMetric(ctx, api.Metrics.nbGroups, "SELECT COUNT(1) FROM \"group\"")
				api.countMetric(ctx, api.Metrics.nbPipelines, "SELECT COUNT(1) FROM pipeline")
				api.countMetric(ctx, api.Metrics.nbWorkflows, "SELECT COUNT(1) FROM workflow")
				api.countMetric(ctx, api.Metrics.nbArtifacts, "SELECT COUNT(1) FROM workflow_node_run_artifacts")
				api.countMetric(ctx, api.Metrics.nbWorkerModels, "SELECT COUNT(1) FROM worker_model")
				api.countMetric(ctx, api.Metrics.nbWorkflowRuns, "SELECT MAX(id) FROM workflow_run")
				api.countMetric(ctx, api.Metrics.nbWorkflowNodeRuns, "SELECT MAX(id) FROM workflow_node_run")
				api.countMetric(ctx, api.Metrics.nbMaxWorkersBuilding, "SELECT COUNT(1) FROM worker where status = 'Building'")

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
		// Example: Errors while fetching count SELECT MAX(id) FROM workflow_run: sql: Scan error on column index 0: converting driver.Value type <nil> ("<nil>") to a int64: invalid syntax
		// this error is displayed when there is no data in the current table
		// so that, record 0
		// this will avoid unuseful warn logs on a fresh CDS Installation
		if strings.Contains(query, "SELECT MAX") && strings.Contains(err.Error(), "converting driver.Value type <nil>") {
			n = 0
		} else {
			log.Warning("metrics>Errors while fetching count %s: %v", query, err)
		}
	}
	observability.Record(ctx, v, n)
}

func (api *API) countMetricRange(ctx context.Context, status string, timerange string, v *stats.Int64Measure, query string, args ...interface{}) {
	n, err := api.mustDB().SelectInt(query, args...)
	if err != nil {
		log.Warning("metrics>Errors while fetching count range %s: %v", query, err)
	}
	ctx, _ = tag.New(ctx, tag.Upsert(tagStatus, status), tag.Upsert(tagRange, timerange))
	observability.Record(ctx, v, n)
}

func (api *API) processStatusMetrics(ctx context.Context) {
	srvs, err := services.All(api.mustDB())
	if err != nil {
		log.Error("Error while getting services list: %v", err)
		return
	}
	mStatus := api.computeGlobalStatus(srvs)

	ignoreList := []string{"version", "hostname", "time", "uptime", "cdsname"}

	for _, line := range mStatus.Lines {
		idx := strings.Index(line.Component, "/")
		service := line.Component[0:idx]
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

		if service == "Availability" {
			number, err := strconv.ParseFloat(line.Value, 64)
			if err != nil {
				number = 0
				log.Warning("metrics>Errors while parsing float %s: %v", line.Value, err)
			}

			item = "Availability"
			ctx, _ = tag.New(ctx, tag.Upsert(tagService, line.Type))
			v, err := observability.FindAndRegisterViewLastFloat64(item, tagsServiceAvailability)
			if err != nil {
				log.Warning("metrics>Errors while FindAndRegisterViewLastFloat64 %s: %v", item, err)
				continue
			}
			observability.RecordFloat64(ctx, v.Measure, number)
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
		v, err := observability.FindAndRegisterViewLast(item, tagsService)
		if err != nil {
			log.Warning("metrics>Errors while FindAndRegisterViewLast %s: %v", item, err)
			continue
		}
		observability.Record(ctx, v.Measure, number)

	}
}
