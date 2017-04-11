package main

import (
	"fmt"
	"net/http"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getStats(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	var st sdk.Stats
	var err error

	st.History, err = initHistory(db)
	if err != nil {
		log.Warning("getStats> cannot initialize history: %s\n", err)
		return err

	}

	for i := range st.History {
		n, err := getNewUsers(db, i+1, i)
		if err != nil {
			log.Warning("getStats> cannot getNewUsers: %s\n", err)
			return err

		}
		st.History[i].NewUsers = n

		// Number of users back then
		n, err = getNewUsers(db, 540, i)
		if err != nil {
			log.Warning("getStats> cannot getPeriodTotalUsers: %s\n", err)
			return err

		}
		st.History[i].Users = n

		n, err = getNewProjects(db, i+1, i)
		if err != nil {
			log.Warning("getStats> cannot getNewProjects: %s\n", err)
			return err

		}
		st.History[i].NewProjects = n

		n, err = getNewProjects(db, 540, i)
		if err != nil {
			log.Warning("getStats> cannot getPeriodTotalUsers: %s\n", err)
			return err

		}
		st.History[i].Projects = n

		n, err = getNewApplications(db, i+1, i)
		if err != nil {
			log.Warning("getStats> cannot getNewApplications: %s\n", err)
			return err

		}
		st.History[i].NewApplications = n

		n, err = getNewApplications(db, 540, i)
		if err != nil {
			log.Warning("getStats> cannot getNewApplications: %s\n", err)
			return err

		}
		st.History[i].Applications = n

		n, err = getNewPipelines(db, i+1, i)
		if err != nil {
			log.Warning("getStats> cannot getNewPipelines: %s\n", err)
			return err

		}
		st.History[i].NewPipelines = n

		st.History[i].Pipelines.Build, st.History[i].Pipelines.Testing, st.History[i].Pipelines.Deploy, err = getPeriodTotalPipelinesByType(db, i)
		if err != nil {
			log.Warning("getStats> cannot getPeriodTotalPipelinesByType: %s\n", err)
			return err

		}
	}

	return WriteJSON(w, r, st, http.StatusOK)
}

func getNewPipelines(db *gorp.DbMap, fromWeek, toWeek int) (int64, error) {
	query := `SELECT COUNT(id) FROM "pipeline" WHERE created > NOW() - INTERVAL '%d weeks' AND created < NOW() - INTERVAL '%d weeks'`
	var n int64

	err := db.QueryRow(fmt.Sprintf(query, fromWeek, toWeek)).Scan(&n)
	if err != nil {
		return 0, err
	}

	return n, nil
}

func getNewApplications(db *gorp.DbMap, fromWeek, toWeek int) (int64, error) {
	query := `SELECT COUNT(id) FROM "application" WHERE created > NOW() - INTERVAL '%d weeks' AND created < NOW() - INTERVAL '%d weeks'`
	var n int64

	err := db.QueryRow(fmt.Sprintf(query, fromWeek, toWeek)).Scan(&n)
	if err != nil {
		return 0, err
	}

	return n, nil
}

func getNewProjects(db *gorp.DbMap, fromWeek, toWeek int) (int64, error) {
	query := `SELECT COUNT(id) FROM "project" WHERE created > NOW() - INTERVAL '%d weeks' AND created < NOW() - INTERVAL '%d weeks'`
	var n int64

	err := db.QueryRow(fmt.Sprintf(query, fromWeek, toWeek)).Scan(&n)
	if err != nil {
		return 0, err
	}

	return n, nil
}

func getNewUsers(db *gorp.DbMap, fromWeek, toWeek int) (int64, error) {
	query := `SELECT COUNT(username) FROM "user" WHERE created > NOW() - INTERVAL '%d weeks' AND created < NOW() - INTERVAL '%d weeks'`
	var n int64

	err := db.QueryRow(fmt.Sprintf(query, fromWeek, toWeek)).Scan(&n)
	if err != nil {
		return 0, err
	}

	return n, nil
}

func getPeriodTotalPipelinesByType(db *gorp.DbMap, toWeek int) (build, test, deploy int64, err error) {
	query := `SELECT COUNT(id) FROM pipeline WHERE created < NOW() - INTERVAL '%d weeks' AND type = $1`

	err = db.QueryRow(fmt.Sprintf(query, toWeek), string(sdk.BuildPipeline)).Scan(&build)
	if err != nil {
		return
	}

	err = db.QueryRow(fmt.Sprintf(query, toWeek), string(sdk.TestingPipeline)).Scan(&test)
	if err != nil {
		return
	}

	err = db.QueryRow(fmt.Sprintf(query, toWeek), string(sdk.DeploymentPipeline)).Scan(&deploy)
	if err != nil {
		return
	}

	return
}

func initHistory(db *gorp.DbMap) ([]sdk.Week, error) {
	var sts []sdk.Week
	var st sdk.Week

	query := `
	SELECT MIN(day), MAX(day), SUM(build) as b, SUM(unit_test) as ut, SUM(testing) as testing, SUM(deployment) as deployment, MAX(max_building_worker) as workers, MAX(max_building_pipeline) as building_pi
	FROM stats
	WHERE day > NOW() - INTERVAL '%d weeks' AND day < NOW() - INTERVAL '%d weeks'
	`

	err := db.QueryRow(fmt.Sprintf(query, 1, 0)).Scan(&st.From, &st.To, &st.RunnedPipelines.Build, &st.UnitTests, &st.RunnedPipelines.Testing, &st.RunnedPipelines.Deploy, &st.MaxBuildingWorkers, &st.MaxBuildingPipelines)
	if err != nil {
		return nil, err
	}
	st.Builds = st.RunnedPipelines.Build + st.RunnedPipelines.Testing + st.RunnedPipelines.Deploy
	sts = append(sts, st)
	err = db.QueryRow(fmt.Sprintf(query, 2, 1)).Scan(&st.From, &st.To, &st.RunnedPipelines.Build, &st.UnitTests, &st.RunnedPipelines.Testing, &st.RunnedPipelines.Deploy, &st.MaxBuildingWorkers, &st.MaxBuildingPipelines)
	if err != nil {
		return nil, err
	}
	st.Builds = st.RunnedPipelines.Build + st.RunnedPipelines.Testing + st.RunnedPipelines.Deploy
	sts = append(sts, st)

	return sts, nil
}
