package pipeline

import (
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

type awolPipelineBuildJob struct {
	pipelineBuildJobID int64
	pieplineBuildID    int64
}

// AWOLPipelineKiller will search in database for actions :
// - Having building status
// - Without any logs ouput in the last 15 minutes
func AWOLPipelineKiller() {
	// If this goroutine exits, then it's a crash
	defer log.Fatalf("Goroutine of pipeline.AWOLPipelineKiller exited - Exit CDS Engine")

	for {
		time.Sleep(1 * time.Minute)
		db := database.DBMap(database.DB())

		if db != nil {
			pbJobDatas, err := loadAWOLActionBuild(db)
			if err != nil {
				log.Warning("AWOLPipelineKiller> Cannot load awol building actions: %s\n", err)
			}

			for _, data := range pbJobDatas {
				err = killAWOLAction(db, data)
				if err != nil {
					log.Warning("AWOLPipelineKiller> Cannot kill action build %d: %s\n", data.pipelineBuildJobID, err)
					time.Sleep(1 * time.Second) // Do not spam an unavailable database
				}
			}
		}
	}
}

func killAWOLAction(db *gorp.DbMap, pbJobData awolPipelineBuildJob) error {
	log.Warning("killAWOLAction> Killing pipeline_job_build %d\n", pbJobData.pipelineBuildJobID)

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	InsertLog(tx, pbJobData.pipelineBuildJobID, "SYSTEM", "Killed (Reason: Timeout)\n", pbJobData.pieplineBuildID)
	pbJob, errJob := GetPipelineBuildJob(tx, pbJobData.pipelineBuildJobID)
	if errJob != nil {
		return errJob
	}

	if err := UpdatePipelineBuildJobStatus(tx, pbJob, sdk.StatusFail); err != nil {
		return err
	}

	query := `UPDATE worker SET status = $1, action_build_id = NULL WHERE action_build_id = $2`
	_, err = tx.Exec(query, string(sdk.StatusDisabled), pbJobData.pipelineBuildJobID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// SELECT action_build.id
// JOIN WITH build_log even if there is no log !
// WHERE action_build status is building, obviously
// WHERE action_build was started at least 15 minutes ago
// WHERE LAAAAAAAAAAAAAAST logs are older than 15 minutes OR no logs at all
func loadAWOLActionBuild(db gorp.SqlExecutor) ([]awolPipelineBuildJob, error) {
	query := `
		SELECT pipeline_build_job.id, pipeline_build_job.pipeline_build_id FROM pipeline_build_job
		LEFT OUTER JOIN build_log ON build_log.action_build_id = pipeline_build_job.id
		WHERE status = 'Building'
		AND pipeline_build_job.start < NOW() - INTERVAL '15 minutes'
		GROUP BY pipeline_build_job.id, pipeline_build_job.pipeline_build_id
		HAVING MAX(build_log.timestamp) < NOW() - INTERVAL '15 minutes' OR MAX(build_log.timestamp) IS NULL
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
