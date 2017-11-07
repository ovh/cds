package pipeline

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type awolPipelineBuildJob struct {
	pipelineBuildJobID int64
	pieplineBuildID    int64
}

// AWOLPipelineKiller will search in database for actions :
// - Having building status
// - Without any logs output in the last 15 minutes
func AWOLPipelineKiller(c context.Context, DBFunc func() *gorp.DbMap) {
	tick := time.NewTicker(1 * time.Minute).C
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting AWOLPipelineKiller: %v", c.Err())
			}
			return
		case <-tick:
			db := DBFunc()
			if db != nil {
				pbJobDatas, err := loadAWOLPipelineBuildJob(db)
				if err != nil {
					log.Warning("AWOLPipelineKiller> Cannot load awol building actions: %s\n", err)
				}

				for _, data := range pbJobDatas {
					err = killOrRestartAWOLPipelineBuildJob(db, data)
					if err != nil {
						log.Warning("AWOLPipelineKiller> Cannot kill or restart action build %d: %s\n", data.pipelineBuildJobID, err)
					}
				}
			}
		}
	}
}

func killOrRestartAWOLPipelineBuildJob(db *gorp.DbMap, pbJobData awolPipelineBuildJob) error {
	log.Warning("killOrRestartAWOLPipelineBuildJob> Killing pipeline_job_build %d\n", pbJobData.pipelineBuildJobID)

	tx, errb := db.Begin()
	if errb != nil {
		return sdk.WrapError(errb, "killOrRestartAWOLPipelineBuildJob> cannot begin transaction")
	}
	defer tx.Rollback()

	pbJob, errJob := GetPipelineBuildJob(tx, pbJobData.pipelineBuildJobID)
	if errJob != nil {
		return errJob
	}

	// replace in queue 3 times before failed
	if pbJob.Retry < 3 {
		pbJob.Status = sdk.StatusWaiting.String()
		pbJob.Retry++
		if err := UpdatePipelineBuildJob(tx, pbJob); err != nil {
			return err
		}

		for _, step := range pbJob.Job.StepStatus {
			if step.Status == sdk.StatusNeverBuilt.String() || step.Status == sdk.StatusSkipped.String() || step.Status == sdk.StatusDisabled.String() {
				continue
			}

			l, errL := LoadStepLogs(tx, pbJob.ID, int64(step.StepOrder))
			if errL != nil {
				return sdk.WrapError(errL, "killOrRestartAWOLPipelineBuildJob> error while load step logs")
			}

			l.Val += "\n\n------------- Worker timeout: job replaced in queue -------------\n\n"
			if err := UpdateLog(tx, l); err != nil {
				return sdk.WrapError(errL, "killOrRestartAWOLPipelineBuildJob> error while update step log")
			}
		}
	} else {
		pbJob.Job.Reason = "Killed (Reason: Timeout)\n"

		if err := UpdatePipelineBuildJobStatus(tx, pbJob, sdk.StatusFail); err != nil {
			return err
		}
	}

	query := `UPDATE worker SET status = $1, action_build_id = NULL WHERE action_build_id = $2`
	if _, err := tx.Exec(query, string(sdk.StatusDisabled), pbJobData.pipelineBuildJobID); err != nil {
		return sdk.WrapError(err, "killOrRestartAWOLPipelineBuildJob> error while execute query. pbJobData.pipelineBuildJobID:%d", pbJobData.pipelineBuildJobID)
	}

	return tx.Commit()
}

func loadAWOLPipelineBuildJob(db gorp.SqlExecutor) ([]awolPipelineBuildJob, error) {
	query := `
		SELECT pipeline_build_job.id, pipeline_build_job.pipeline_build_id FROM pipeline_build_job
		LEFT OUTER JOIN pipeline_build_log ON pipeline_build_log.pipeline_build_job_id = pipeline_build_job.id
		WHERE status = 'Building'
		AND pipeline_build_job.start < NOW() - INTERVAL '15 minutes'
		GROUP BY pipeline_build_job.id, pipeline_build_job.pipeline_build_id
		HAVING MAX(pipeline_build_log.last_modified) < NOW() - INTERVAL '15 minutes' OR MAX(pipeline_build_log.last_modified) IS NULL
		`
	var datas []awolPipelineBuildJob
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var d awolPipelineBuildJob
		err := rows.Scan(&d.pipelineBuildJobID, &d.pieplineBuildID)
		if err != nil {
			return nil, err
		}
		datas = append(datas, d)
	}

	return datas, nil
}
