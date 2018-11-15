package observability

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Metrics contains metrics values
type Metrics struct {
	nbUsers              *stats.Int64Measure
	nbApplications       *stats.Int64Measure
	nbProjects           *stats.Int64Measure
	nbGroups             *stats.Int64Measure
	nbPipelines          *stats.Int64Measure
	nbWorkflows          *stats.Int64Measure
	nbArtifacts          *stats.Int64Measure
	nbWorkerModels       *stats.Int64Measure
	nbWorkflowRuns       *stats.Int64Measure
	nbWorkflowNodeRuns   *stats.Int64Measure
	nbMaxWorkersBuilding *stats.Int64Measure

	queue  *stats.Int64Measure
	status *stats.Int64Measure
}

var (
	tagRange     tag.Key
	tagStatus    tag.Key
	tagComponent tag.Key
)

// Initialize initializes metrics series
func Initialize(ctx context.Context, DBFunc func() *gorp.DbMap, instances MinInstances) error {
	minInstances = instances

	m := &Metrics{}
	m.nbUsers = stats.Int64("cds/cds-api/nb_users", "number of users", stats.UnitDimensionless)
	m.nbApplications = stats.Int64("cds/cds-api/nb_applications", "nb_applications", stats.UnitDimensionless)
	m.nbProjects = stats.Int64("cds/cds-api/nb_projects", "nb_projects", stats.UnitDimensionless)
	m.nbGroups = stats.Int64("cds/cds-api/nb_groups", "nb_groups", stats.UnitDimensionless)
	m.nbPipelines = stats.Int64("cds/cds-api/nb_pipelines", "nb_pipelines", stats.UnitDimensionless)
	m.nbWorkflows = stats.Int64("cds/cds-api/nb_workflows", "nb_workflows", stats.UnitDimensionless)
	m.nbArtifacts = stats.Int64("cds/cds-api/nb_artifacts", "nb_artifacts", stats.UnitDimensionless)
	m.nbWorkerModels = stats.Int64("cds/cds-api/nb_worker_models", "nb_worker_models", stats.UnitDimensionless)
	m.nbWorkflowRuns = stats.Int64("cds/cds-api/nb_workflow_runs", "nb_workflow_runs", stats.UnitDimensionless)
	m.nbWorkflowNodeRuns = stats.Int64("cds/cds-api/nb_workflow_node_runs", "nb_workflow_node_runs", stats.UnitDimensionless)
	m.nbMaxWorkersBuilding = stats.Int64("cds/cds-api/nb_max_workers_building", "nb_max_workers_building", stats.UnitDimensionless)

	m.queue = stats.Int64("cds/cds-api/queue", "queue", stats.UnitDimensionless)
	m.status = stats.Int64("cds/cds-api/status", "status", stats.UnitDimensionless)

	tagInstance, _ := tag.NewKey("instance")
	tagRange, _ = tag.NewKey("range")
	tagStatus, _ = tag.NewKey("status")
	tagComponent, _ = tag.NewKey("component")

	tags := []tag.Key{tagInstance}
	tagsRange := []tag.Key{tagInstance, tagRange, tagStatus}
	tagsComponent := []tag.Key{tagInstance, tagComponent, tagStatus}

	m.compute(ctx, DBFunc)

	err := RegisterView(
		newView("nb_users", m.nbUsers, tags),
		newView("nb_applications", m.nbApplications, tags),
		newView("nb_projects", m.nbProjects, tags),
		newView("nb_groups", m.nbGroups, tags),
		newView("nb_pipelines", m.nbPipelines, tags),
		newView("nb_workflows", m.nbWorkflows, tags),
		newView("nb_artifacts", m.nbArtifacts, tags),
		newView("nb_worker_models", m.nbWorkerModels, tags),
		newView("nb_workflow_runs", m.nbWorkflowRuns, tags),
		newView("nb_workflow_node_runs", m.nbWorkflowNodeRuns, tags),
		newView("nb_max_workers_building", m.nbMaxWorkersBuilding, tags),
		newView("queue", m.queue, tagsRange),
		newView("status", m.status, tagsComponent),
	)
	return err
}

func newView(name string, s *stats.Int64Measure, tags []tag.Key) *view.View {
	return &view.View{
		Name:        name,
		Description: s.Description(),
		Measure:     s,
		Aggregation: view.LastValue(),
		TagKeys:     tags,
	}
}

func (m *Metrics) compute(ctx context.Context, DBFunc func() *gorp.DbMap) {
	sdk.GoRoutine(ctx, "observability.compute", func(ctx context.Context) {
		tick := time.NewTicker(9 * time.Second).C
		for {
			select {
			case <-ctx.Done():
				if ctx.Err() != nil {
					log.Error("Exiting metrics.Initialize: %v", ctx.Err())
					return
				}
			case <-tick:
				count(ctx, DBFunc(), m.nbUsers, "SELECT COUNT(1) FROM \"user\"")
				count(ctx, DBFunc(), m.nbApplications, "SELECT COUNT(1) FROM application")
				count(ctx, DBFunc(), m.nbProjects, "SELECT COUNT(1) FROM project")
				count(ctx, DBFunc(), m.nbGroups, "SELECT COUNT(1) FROM \"group\"")
				count(ctx, DBFunc(), m.nbPipelines, "SELECT COUNT(1) FROM pipeline")
				count(ctx, DBFunc(), m.nbWorkflows, "SELECT COUNT(1) FROM workflow")
				count(ctx, DBFunc(), m.nbArtifacts, "SELECT COUNT(1) FROM artifact")
				count(ctx, DBFunc(), m.nbWorkerModels, "SELECT COUNT(1) FROM worker_model")
				count(ctx, DBFunc(), m.nbWorkflowRuns, "SELECT MAX(id) FROM workflow_run")
				count(ctx, DBFunc(), m.nbWorkflowNodeRuns, "SELECT MAX(id) FROM workflow_node_run")
				count(ctx, DBFunc(), m.nbMaxWorkersBuilding, "SELECT COUNT(1) FROM worker where status = 'Building'")

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

				countRange(ctx, DBFunc(), "building", "all", m.queue, queryBuilding)
				countRange(ctx, DBFunc(), "waiting", "10_less_10s", m.queue, query, now10s, now)
				countRange(ctx, DBFunc(), "waiting", "20_more_10s_less_30s", m.queue, query, now30s, now10s)
				countRange(ctx, DBFunc(), "waiting", "30_more_30s_less_1min", m.queue, query, now1min, now30s)
				countRange(ctx, DBFunc(), "waiting", "40_more_1min_less_2min", m.queue, query, now2min, now1min)
				countRange(ctx, DBFunc(), "waiting", "50_more_2min_less_5min", m.queue, query, now5min, now2min)
				countRange(ctx, DBFunc(), "waiting", "60_more_5min_less_10min", m.queue, query, now10min, now5min)
				countRange(ctx, DBFunc(), "waiting", "70_more_10min", m.queue, queryOld, now10min)

				processStatusMetrics(ctx, DBFunc, m.status)
			}
		}
	})
}

func count(ctx context.Context, db *gorp.DbMap, v *stats.Int64Measure, query string) {
	if db == nil {
		return
	}
	var n sql.NullInt64
	if err := db.QueryRow(query).Scan(&n); err != nil {
		log.Warning("metrics>Errors while fetching count %s: %v", query, err)
		return
	}
	if n.Valid {
		Record(ctx, v, n.Int64)
	}
}

func countRange(ctx context.Context, db *gorp.DbMap, status string, timerange string, v *stats.Int64Measure, query string, args ...interface{}) {
	if db == nil {
		return
	}
	var n sql.NullInt64
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		log.Warning("metrics>Errors while fetching count %s: %v", query, err)
		return
	}
	if n.Valid {
		ctx, _ = tag.New(ctx, tag.Upsert(tagStatus, status), tag.Upsert(tagRange, timerange))
		Record(ctx, v, n.Int64)
	}
}

func processStatusMetrics(ctx context.Context, DBFunc func() *gorp.DbMap, v *stats.Int64Measure) {
	srvs, err := services.All(DBFunc())
	if err != nil {
		log.Error("Error while getting services list: %v", err)
		return
	}
	mStatus := ComputeGlobalStatus(srvs)
	apis := make(map[string]int)
	for _, line := range mStatus.Lines {
		number, err := strconv.ParseInt(line.Value, 10, 64)
		if err != nil {
			number = 1
		}
		name := line.Component

		// rename api_foobar to api_0, api_1, etc...
		// this will avoid creating series with custom name
		if line.Type == services.TypeAPI {
			idx := strings.Index(line.Component, "/")
			if _, ok := apis[line.Component[0:idx]]; !ok {
				apis[line.Component[0:idx]] = len(apis)
			}
			name = fmt.Sprintf("%s_%d%s", services.TypeAPI, apis[line.Component[0:idx]], line.Component[idx:])
		}
		ctx, _ = tag.New(ctx, tag.Upsert(tagStatus, line.Status), tag.Upsert(tagComponent, name))
		Record(ctx, v, number)
	}
}
