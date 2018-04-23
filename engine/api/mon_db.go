package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rubenv/sql-migrate"

	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getMonDBStatusMigrateHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		records, err := migrate.GetMigrationRecords(api.mustDB(ctx).Db, "postgres")
		if err != nil {
			return sdk.WrapError(err, "DBStatusHandler> Cannot GetMigrationRecords")
		}
		m := []sdk.MonDBMigrate{}
		for _, r := range records {
			m = append(m, sdk.MonDBMigrate{ID: r.Id, AppliedAt: r.AppliedAt})
		}
		return WriteJSON(w, m, http.StatusOK)
	}
}

func (api *API) getMonDBTimesDBHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		hostname, _ := os.Hostname()
		o := sdk.MonDBTimes{
			Now:      time.Now(),
			Version:  sdk.VERSION,
			Hostname: hostname,
		}

		o.ProjectLoadAll = api.getMonDBTimesDBProjectLoadHandler(ctx)
		o.ProjectLoadAllWithApps = api.getMonDBTimesDBProjectLoadWithAppsHandler(ctx)
		o.ProjectLoadAllRaw = api.getMonDBTimesDBProjectLoadAllRawHandler(ctx)
		o.ProjectCount = api.getMonDBTimesDBProjectCountHandler(ctx)
		o.QueueWorkflow = api.getMonDBTimesDBQueueWorkflow(ctx, r)

		log.Info("getMonDBTimesDBHandler> elapsed %s", elapsed("getMonDBTimesDBHandler", o.Now))
		return WriteJSON(w, o, http.StatusOK)
	}
}

func (api *API) getMonDBTimesDBProjectLoadHandler(ctx context.Context) string {
	s1 := time.Now()
	if _, err := project.LoadAll(api.mustDB(ctx), api.Cache, getUser(ctx)); err != nil {
		return fmt.Sprintf("ERR getMonDBTimesDBProjectLoadHandler:%s", err)
	}
	return elapsed("getMonDBTimesDBProjectLoadHandler", s1)
}

func (api *API) getMonDBTimesDBProjectLoadWithAppsHandler(ctx context.Context) string {
	s1 := time.Now()
	if _, err := project.LoadAll(api.mustDB(ctx), api.Cache, getUser(ctx), project.LoadOptions.WithApplications); err != nil {
		return fmt.Sprintf("ERR getMonDBTimesDBProjectLoadWithAppsHandler:%s", err)
	}
	return elapsed("getMonDBTimesDBProjectLoadWithAppsHandler", s1)
}

func (api *API) getMonDBTimesDBProjectCountHandler(ctx context.Context) string {
	s1 := time.Now()
	query := `SELECT COUNT(id) FROM "project"`
	var n int64
	if err := api.mustDB(ctx).QueryRow(query).Scan(&n); err != nil {
		return fmt.Sprintf("ERR getMonDBTimesDBProjectCountHandler:%s", err)
	}
	return elapsed("getMonDBTimesDBProjectCountHandler", s1)
}

func (api *API) getMonDBTimesDBProjectLoadAllRawHandler(ctx context.Context) string {
	s1 := time.Now()
	query := `SELECT name FROM "project"`

	rows, errq := api.mustDB(ctx).Query(query)
	if errq != nil {
		return fmt.Sprintf("ERR getMonDBTimesDBProjectLoadAllRawHandler:%s", errq)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Sprintf("ERR scan getMonDBTimesDBProjectLoadAllRawHandler:%s", err)
		}
	}
	return elapsed("getMonDBTimesDBProjectLoadAllRawHandler", s1)
}

func (api *API) getMonDBTimesDBQueueWorkflow(ctx context.Context, r *http.Request) string {
	groupsID := []int64{}
	for _, g := range getUser(ctx).Groups {
		groupsID = append(groupsID, g.ID)
	}
	s1 := time.Now()
	permissions := permission.PermissionReadExecute
	if !isHatcheryOrWorker(r) {
		permissions = permission.PermissionRead
	}

	if _, err := workflow.LoadNodeJobRunQueue(api.mustDB(ctx), api.Cache, permissions, groupsID, nil, nil); err != nil {
		return fmt.Sprintf("getMonDBTimesDBQueueWorkflow> Unable to load queue:: %s", err)
	}
	return elapsed("getMonDBTimesDBQueueWorkflow", s1)
}

func elapsed(what string, start time.Time) string {
	t := fmt.Sprintf("%v", time.Since(start))
	log.Info("%s: %v", what, t)
	return t
}
