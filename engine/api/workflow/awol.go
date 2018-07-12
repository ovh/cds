package workflow

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func loadAwolJobRuns(db gorp.SqlExecutor) ([]sdk.WorkflowNodeJobRun, error) {
	query := `
	SELECT workflow_node_run_job.* 
	FROM workflow_node_run_job 
	WHERE worker_id IS NOT NULL 
	AND worker_id NOT IN (
		SELECT id from worker 
		where status <> 'Disabled'
	)
	UNION
	SELECT workflow_node_run_job.* 
	FROM workflow_node_run_job 
	JOIN workflow_node_run_job_logs ON workflow_node_run_job.id = workflow_node_run_job_logs.workflow_node_run_job_id
	WHERE workflow_node_run_job.status = 'Building'
    AND workflow_node_run_job_logs.last_modified < $1
	`

	t0 := time.Now().Add(-15 * time.Minute)
	jobRuns := []JobRun{}
	if _, err := db.Select(&jobRuns, query, t0); err != nil {
		return nil, err
	}

	jobs := make([]sdk.WorkflowNodeJobRun, len(jobRuns))
	for i := range jobRuns {
		if err := jobRuns[i].PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "workflow.LoadAwolJobRuns> Unable to load job runs (PostGet)")
		}
		jobs[i] = sdk.WorkflowNodeJobRun(jobRuns[i])
	}

	return jobs, nil
}

// RestartAwolJobs runs with a ticker within a context.
// it loads all workflow_node_run_job which are linked to a worker that doesn't exist anymore
// and the all workflow_node_run_job at status 'Building' but without any logs since more than 15 minutes
// each of those workflow_node_run_job is restart by RestartWorkflowNodeJob
func RestartAwolJobs(ctx context.Context, store cache.Store, dbFunc func() *gorp.DbMap) {
	ticker := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			db := dbFunc()
			jobs, err := loadAwolJobRuns(db)
			if err != nil {
				log.Error("RestartAwolJobs> unable to load jobs: %v", err)
			}
			log.Debug("RestartAwolJobs> %d jobs to restart", len(jobs))
			for _, j := range jobs {
				tx, err := db.Begin()
				if err != nil {
					log.Error("RestartAwolJobs> unable to start tx:%v", err)
					continue
				}
				if err := RestartWorkflowNodeJob(ctx, tx, j); err != nil {
					log.Error("RestartAwolJobs> unable to restart job %d:%v", j.ID, err)
					_ = tx.Rollback()
					continue
				}
				if err := tx.Commit(); err != nil {
					log.Error("RestartAwolJobs> unable to commit tx: %v", err)
					_ = tx.Rollback()
				}
				log.Debug("RestartAwolJobs> job %d restarted", j.ID)
			}
		}
	}
}
