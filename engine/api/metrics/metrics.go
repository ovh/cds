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
func Initialize(c context.Context, DBFunc func(context.Context) *gorp.DbMap, instance string) {
	labels := prometheus.Labels{"instance": instance}

	nbUsers := prometheus.NewSummary(prometheus.SummaryOpts{Name: "nb_users", Help: "metrics nb_users", ConstLabels: labels})
	nbApplications := prometheus.NewSummary(prometheus.SummaryOpts{Name: "nb_applications", Help: "metrics nb_applications", ConstLabels: labels})
	nbProjects := prometheus.NewSummary(prometheus.SummaryOpts{Name: "nb_projects", Help: "metrics nb_projects", ConstLabels: labels})
	nbGroups := prometheus.NewSummary(prometheus.SummaryOpts{Name: "nb_groups", Help: "metrics nb_groups", ConstLabels: labels})
	nbPipelines := prometheus.NewSummary(prometheus.SummaryOpts{Name: "nb_pipelines", Help: "metrics nb_pipelines", ConstLabels: labels})
	nbWorkflows := prometheus.NewSummary(prometheus.SummaryOpts{Name: "nb_workflows", Help: "metrics nb_workflows", ConstLabels: labels})
	nbArtifacts := prometheus.NewSummary(prometheus.SummaryOpts{Name: "nb_artifacts", Help: "metrics nb_artifacts", ConstLabels: labels})
	nbWorkerModels := prometheus.NewSummary(prometheus.SummaryOpts{Name: "nb_worker_models", Help: "metrics nb_worker_models", ConstLabels: labels})
	nbWorkflowRuns := prometheus.NewSummary(prometheus.SummaryOpts{Name: "nb_workflow_runs", Help: "metrics nb_workflow_runs", ConstLabels: labels})
	nbWorkflowNodeRuns := prometheus.NewSummary(prometheus.SummaryOpts{Name: "nb_workflow_node_runs", Help: "metrics nb_workflow_node_runs", ConstLabels: labels})
	nbWorkflowNodeRunJobs := prometheus.NewSummary(prometheus.SummaryOpts{Name: "nb_workflow_node_run_jobs", Help: "metrics nb_workflow_node_run_jobs", ConstLabels: labels})
	nbMaxWorkersBuilding := prometheus.NewSummary(prometheus.SummaryOpts{Name: "nb_max_workers_building", Help: "metrics nb_max_workers_building", ConstLabels: labels})

	nbOldPipelineBuilds := prometheus.NewSummary(prometheus.SummaryOpts{Name: "nb_old_pipeline_builds", Help: "metrics nb_old_pipeline_builds", ConstLabels: labels})
	nbOldPipelineBuildJobs := prometheus.NewSummary(prometheus.SummaryOpts{Name: "nb_old_pipeline_build_jobs", Help: "metrics nb_old_pipeline_build_jobs", ConstLabels: labels})

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
	registry.MustRegister(nbOldPipelineBuilds)
	registry.MustRegister(nbOldPipelineBuildJobs)
	registry.MustRegister(nbMaxWorkersBuilding)

	tick := time.NewTicker(30 * time.Second).C

	go func(c context.Context, DBFunc func(context.Context) *gorp.DbMap) {
		for {
			select {
			case <-c.Done():
				if c.Err() != nil {
					log.Error("Exiting metrics.Initialize: %v", c.Err())
					return
				}
			case <-tick:
				count(DBFunc(c), "SELECT COUNT(1) FROM \"user\"", nbUsers)
				count(DBFunc(c), "SELECT COUNT(1) FROM application", nbApplications)
				count(DBFunc(c), "SELECT COUNT(1) FROM project", nbProjects)
				count(DBFunc(c), "SELECT COUNT(1) FROM \"group\"", nbGroups)
				count(DBFunc(c), "SELECT COUNT(1) FROM pipeline", nbPipelines)
				count(DBFunc(c), "SELECT COUNT(1) FROM workflow", nbWorkflows)
				count(DBFunc(c), "SELECT COUNT(1) FROM artifact", nbArtifacts)
				count(DBFunc(c), "SELECT COUNT(1) FROM worker_model", nbWorkerModels)
				count(DBFunc(c), "SELECT MAX(id) FROM workflow_run", nbWorkflowRuns)
				count(DBFunc(c), "SELECT MAX(id) FROM workflow_node_run", nbWorkflowNodeRuns)
				count(DBFunc(c), "SELECT MAX(id) FROM workflow_node_run_job", nbWorkflowNodeRunJobs)
				count(DBFunc(c), "SELECT MAX(id) FROM pipeline_build", nbOldPipelineBuilds)
				count(DBFunc(c), "SELECT MAX(id) FROM pipeline_build_job", nbOldPipelineBuildJobs)
				count(DBFunc(c), "SELECT COUNT(1) FROM worker where status like 'Building' ", nbMaxWorkersBuilding)
			}
		}
	}(c, DBFunc)
}

func count(db *gorp.DbMap, query string, v prometheus.Summary) {
	if db == nil {
		return
	}
	var n sql.NullInt64
	if err := db.QueryRow(query).Scan(&n); err != nil {
		log.Warning("metrics>Errors while fetching count %s: %v", query, err)
		return
	}
	if n.Valid {
		v.Observe(float64(n.Int64))
	}

}

// GetGatherer returns CDS API gatherer
func GetGatherer() prometheus.Gatherer {
	return registry
}
