package metrics

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk/log"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	registry = prometheus.NewRegistry()
)

// Initialize initializes metrics
func Initialize(c context.Context, DBFunc func() *gorp.DbMap, instance string) {
	labels := prometheus.Labels{"instance": instance}

	nbUsers := prometheus.NewGauge(prometheus.GaugeOpts{Name: "nb_users", Help: "metrics nb_users", ConstLabels: labels})
	nbApplications := prometheus.NewGauge(prometheus.GaugeOpts{Name: "nb_applications", Help: "metrics nb_applications", ConstLabels: labels})
	nbProjects := prometheus.NewGauge(prometheus.GaugeOpts{Name: "nb_projects", Help: "metrics nb_projects", ConstLabels: labels})
	nbGroups := prometheus.NewGauge(prometheus.GaugeOpts{Name: "nb_groups", Help: "metrics nb_groups", ConstLabels: labels})
	nbPipelines := prometheus.NewGauge(prometheus.GaugeOpts{Name: "nb_pipelines", Help: "metrics nb_pipelines", ConstLabels: labels})
	nbWorkflows := prometheus.NewGauge(prometheus.GaugeOpts{Name: "nb_workflows", Help: "metrics nb_workflows", ConstLabels: labels})
	nbArtifacts := prometheus.NewGauge(prometheus.GaugeOpts{Name: "nb_artifacts", Help: "metrics nb_artifacts", ConstLabels: labels})
	nbWorkerModels := prometheus.NewGauge(prometheus.GaugeOpts{Name: "nb_worker_models", Help: "metrics nb_worker_models", ConstLabels: labels})
	nbWorkflowRuns := prometheus.NewGauge(prometheus.GaugeOpts{Name: "nb_workflow_runs", Help: "metrics nb_workflow_runs", ConstLabels: labels})
	nbWorkflowNodeRuns := prometheus.NewGauge(prometheus.GaugeOpts{Name: "nb_workflow_node_runs", Help: "metrics nb_workflow_node_runs", ConstLabels: labels})
	nbMaxWorkersBuilding := prometheus.NewGauge(prometheus.GaugeOpts{Name: "nb_max_workers_building", Help: "metrics nb_max_workers_building", ConstLabels: labels})
	nbJobs := prometheus.NewGauge(prometheus.GaugeOpts{Name: "nb_jobs", Help: "nb_jobs", ConstLabels: labels})
	queue := prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "queue", Help: "metrics queue", ConstLabels: prometheus.Labels{"instance": instance}}, []string{"status", "range"})

	registry.MustRegister(nbUsers)
	registry.MustRegister(nbApplications)
	registry.MustRegister(nbProjects)
	registry.MustRegister(nbGroups)
	registry.MustRegister(nbPipelines)
	registry.MustRegister(nbWorkflows)
	registry.MustRegister(nbArtifacts)
	registry.MustRegister(nbWorkerModels)
	registry.MustRegister(nbWorkflowRuns)
	registry.MustRegister(nbWorkflowNodeRuns)
	registry.MustRegister(nbMaxWorkersBuilding)
	registry.MustRegister(nbJobs)
	registry.MustRegister(queue)

	tick := time.NewTicker(9 * time.Second).C

	go func(c context.Context, DBFunc func() *gorp.DbMap) {
		for {
			select {
			case <-c.Done():
				if c.Err() != nil {
					log.Error("Exiting metrics.Initialize: %v", c.Err())
					return
				}
			case <-tick:
				count(DBFunc(), nbUsers, "SELECT COUNT(1) FROM \"user\"")
				count(DBFunc(), nbApplications, "SELECT COUNT(1) FROM application")
				count(DBFunc(), nbProjects, "SELECT COUNT(1) FROM project")
				count(DBFunc(), nbGroups, "SELECT COUNT(1) FROM \"group\"")
				count(DBFunc(), nbPipelines, "SELECT COUNT(1) FROM pipeline")
				count(DBFunc(), nbWorkflows, "SELECT COUNT(1) FROM workflow")
				count(DBFunc(), nbArtifacts, "SELECT COUNT(1) FROM artifact")
				count(DBFunc(), nbWorkerModels, "SELECT COUNT(1) FROM worker_model")
				count(DBFunc(), nbWorkflowRuns, "SELECT MAX(id) FROM workflow_run")
				count(DBFunc(), nbWorkflowNodeRuns, "SELECT MAX(id) FROM workflow_node_run")
				count(DBFunc(), nbMaxWorkersBuilding, "SELECT COUNT(1) FROM worker where status = 'Building'")
				count(DBFunc(), nbJobs, "SELECT COUNT(1) FROM (SELECT distinct(workflow_node_run_job_id) from workflow_node_run_job_info group by workflow_node_run_job_id) AS temp")

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

				countGauge(DBFunc(), *queue, "building", "all", queryBuilding)
				countGauge(DBFunc(), *queue, "waiting", "10_less_10s", query, now10s, now)
				countGauge(DBFunc(), *queue, "waiting", "20_more_10s_less_30s", query, now30s, now10s)
				countGauge(DBFunc(), *queue, "waiting", "30_more_30s_less_1min", query, now1min, now30s)
				countGauge(DBFunc(), *queue, "waiting", "40_more_1min_less_2min", query, now2min, now1min)
				countGauge(DBFunc(), *queue, "waiting", "50_more_2min_less_5min", query, now5min, now2min)
				countGauge(DBFunc(), *queue, "waiting", "60_more_5min_less_10min", query, now10min, now5min)
				countGauge(DBFunc(), *queue, "waiting", "70_more_10min", queryOld, now10min)
			}
		}
	}(c, DBFunc)
}

func count(db *gorp.DbMap, v prometheus.Gauge, query string) {
	if db == nil {
		return
	}
	var n sql.NullInt64
	if err := db.QueryRow(query).Scan(&n); err != nil {
		log.Warning("metrics>Errors while fetching count %s: %v", query, err)
		return
	}
	if n.Valid {
		v.Set(float64(n.Int64))
	}
}

func countGauge(db *gorp.DbMap, v prometheus.GaugeVec, status, timerange string, query string, args ...interface{}) {
	if db == nil {
		return
	}
	var n sql.NullInt64
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		log.Warning("metrics>Errors while fetching count %s: %v", query, err)
		return
	}
	if n.Valid {
		v.WithLabelValues(status, timerange).Set(float64(n.Int64))
	}
}

// GetGatherer returns CDS API gatherer
func GetGatherer() prometheus.Gatherer {
	return registry
}
