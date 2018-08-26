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

	nbUsers := prometheus.NewCounter(prometheus.CounterOpts{Name: "nb_users", Help: "metrics nb_users", ConstLabels: labels})
	nbApplications := prometheus.NewCounter(prometheus.CounterOpts{Name: "nb_applications", Help: "metrics nb_applications", ConstLabels: labels})
	nbProjects := prometheus.NewCounter(prometheus.CounterOpts{Name: "nb_projects", Help: "metrics nb_projects", ConstLabels: labels})
	nbGroups := prometheus.NewCounter(prometheus.CounterOpts{Name: "nb_groups", Help: "metrics nb_groups", ConstLabels: labels})
	nbPipelines := prometheus.NewCounter(prometheus.CounterOpts{Name: "nb_pipelines", Help: "metrics nb_pipelines", ConstLabels: labels})
	nbWorkflows := prometheus.NewCounter(prometheus.CounterOpts{Name: "nb_workflows", Help: "metrics nb_workflows", ConstLabels: labels})
	nbArtifacts := prometheus.NewCounter(prometheus.CounterOpts{Name: "nb_artifacts", Help: "metrics nb_artifacts", ConstLabels: labels})
	nbWorkerModels := prometheus.NewCounter(prometheus.CounterOpts{Name: "nb_worker_models", Help: "metrics nb_worker_models", ConstLabels: labels})
	nbWorkflowRuns := prometheus.NewCounter(prometheus.CounterOpts{Name: "nb_workflow_runs", Help: "metrics nb_workflow_runs", ConstLabels: labels})
	nbWorkflowNodeRuns := prometheus.NewCounter(prometheus.CounterOpts{Name: "nb_workflow_node_runs", Help: "metrics nb_workflow_node_runs", ConstLabels: labels})
	nbWorkflowNodeRunJobs := prometheus.NewCounter(prometheus.CounterOpts{Name: "nb_workflow_node_run_jobs", Help: "metrics nb_workflow_node_run_jobs", ConstLabels: labels})
	nbMaxWorkersBuilding := prometheus.NewCounter(prometheus.CounterOpts{Name: "nb_max_workers_building", Help: "metrics nb_max_workers_building", ConstLabels: labels})
	nbNodeRunJobBuilding := prometheus.NewCounter(prometheus.CounterOpts{Name: "queue", Help: "metrics queue building", ConstLabels: prometheus.Labels{"instance": instance, "status": "building"}})
	nbNodeRunJobWaiting := prometheus.NewCounter(prometheus.CounterOpts{Name: "queue", Help: "metrics queue waiting", ConstLabels: prometheus.Labels{"instance": instance, "status": "waiting"}})

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
	registry.MustRegister(nbWorkflowNodeRunJobs)
	registry.MustRegister(nbMaxWorkersBuilding)
	registry.MustRegister(nbNodeRunJobBuilding)
	registry.MustRegister(nbNodeRunJobWaiting)

	tick := time.NewTicker(30 * time.Second).C

	go func(c context.Context, DBFunc func() *gorp.DbMap) {
		for {
			select {
			case <-c.Done():
				if c.Err() != nil {
					log.Error("Exiting metrics.Initialize: %v", c.Err())
					return
				}
			case <-tick:
				count(DBFunc(), "SELECT COUNT(1) FROM \"user\"", nbUsers)
				count(DBFunc(), "SELECT COUNT(1) FROM application", nbApplications)
				count(DBFunc(), "SELECT COUNT(1) FROM project", nbProjects)
				count(DBFunc(), "SELECT COUNT(1) FROM \"group\"", nbGroups)
				count(DBFunc(), "SELECT COUNT(1) FROM pipeline", nbPipelines)
				count(DBFunc(), "SELECT COUNT(1) FROM workflow", nbWorkflows)
				count(DBFunc(), "SELECT COUNT(1) FROM artifact", nbArtifacts)
				count(DBFunc(), "SELECT COUNT(1) FROM worker_model", nbWorkerModels)
				count(DBFunc(), "SELECT MAX(id) FROM workflow_run", nbWorkflowRuns)
				count(DBFunc(), "SELECT MAX(id) FROM workflow_node_run", nbWorkflowNodeRuns)
				count(DBFunc(), "SELECT MAX(id) FROM workflow_node_run_job", nbWorkflowNodeRunJobs)
				count(DBFunc(), "SELECT COUNT(1) FROM worker where status like 'Building' ", nbMaxWorkersBuilding)
				count(DBFunc(), "SELECT COUNT(1) FROM workflow_node_run_job where status like 'Building' ", nbNodeRunJobBuilding)
				count(DBFunc(), "SELECT COUNT(1) FROM workflow_node_run_job where status like 'Waiting' ", nbNodeRunJobBuilding)
			}
		}
	}(c, DBFunc)
}

func count(db *gorp.DbMap, query string, v prometheus.Counter) {
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

// GetGatherer returns CDS API gatherer
func GetGatherer() prometheus.Gatherer {
	return registry
}
