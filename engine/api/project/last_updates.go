package project

import (
	"database/sql"
	"sync"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

//LastUpdates returns projects and application last update
func LastUpdates(db gorp.SqlExecutor, user *sdk.User, since time.Time) ([]sdk.ProjectLastUpdates, error) {
	query := `
		SELECT 	project.projectkey, project.last_modified, apps.name, apps.last_modified, pipelines.name, pipelines.last_modified
		FROM 	project
		JOIN    project_group ON project_group.project_id = project.id
		JOIN    group_user ON project_group.group_id = group_user.group_id
		LEFT OUTER JOIN (
			SELECT 	application.project_id, application.name, application.last_modified
			FROM 	application, application_group, group_user
			WHERE   application.id = application_group.application_id
			AND 	application_group.group_id = group_user.group_id
			AND 	group_user.user_id = $1
			AND		application.last_modified >= $2
		) apps ON apps.project_id = project.id
		LEFT OUTER JOIN (
			SELECT 	pipeline.project_id, pipeline.name, pipeline.last_modified
			FROM 	pipeline, pipeline_group, group_user
			WHERE   pipeline.id = pipeline_group.pipeline_id
			AND 	pipeline_group.group_id = group_user.group_id
			AND 	group_user.user_id = $1
			AND		pipeline.last_modified >= $2
		) pipelines ON pipelines.project_id = project.id
		WHERE 	group_user.user_id = $1
		AND 	project.last_modified >= $2
		ORDER	by project.projectkey asc
	`
	rows, err := db.Query(query, user.ID, since)
	if err != nil {
		return []sdk.ProjectLastUpdates{}, err
	}
	defer rows.Close()

	res := []sdk.ProjectLastUpdates{}

	mapRes := map[string]*sdk.ProjectLastUpdates{}

	chanProj := make(chan struct {
		key          string
		lastModified time.Time
	})
	chanApp := make(chan struct {
		key          string
		name         string
		lastModified time.Time
	})
	chanPip := make(chan struct {
		key          string
		name         string
		lastModified time.Time
	})

	wg := &sync.WaitGroup{}
	quit := make(chan int)
	go func() {
		for {
			select {
			case <-quit:
				return
			case proj := <-chanProj:
				r := mapRes[proj.key]
				if r == nil {
					mapRes[proj.key] = &sdk.ProjectLastUpdates{
						Key:          proj.key,
						LastModified: proj.lastModified.Unix(),
					}
					r = mapRes[proj.key]
				}
				r.LastModified = proj.lastModified.Unix()
				wg.Done()
			case app := <-chanApp:
				r := mapRes[app.key]
				if r == nil {
					mapRes[app.key] = &sdk.ProjectLastUpdates{
						Key: app.key,
					}
					r = mapRes[app.key]
				}
				var appExists bool
				for _, a := range r.Applications {
					if app.name == a.Name {
						appExists = true
						break
					}
				}
				if !appExists {
					r.Applications = append(r.Applications, struct {
						Name         string `json:"name"`
						LastModified int64  `json:"last_modified"`
					}{
						Name:         app.name,
						LastModified: app.lastModified.Unix(),
					})
				}
				wg.Done()
			case pip := <-chanPip:
				r := mapRes[pip.key]
				if r == nil {
					mapRes[pip.key] = &sdk.ProjectLastUpdates{
						Key: pip.key,
					}
					r = mapRes[pip.key]
				}
				var pipExists bool
				for _, p := range r.Pipelines {
					if pip.name == p.Name {
						pipExists = true
						break
					}
				}
				if !pipExists {
					r.Pipelines = append(r.Pipelines, struct {
						Name         string `json:"name"`
						LastModified int64  `json:"last_modified"`
					}{
						Name:         pip.name,
						LastModified: pip.lastModified.Unix(),
					})
				}
				wg.Done()
			}
		}
	}()

	for rows.Next() {
		var projectKey, appName, pipName sql.NullString
		var projectLastModified, appLastModified, pipLastModified pq.NullTime
		err := rows.Scan(&projectKey, &projectLastModified, &appName, &appLastModified, &pipName, &pipLastModified)
		if err != nil {
			log.Warning("LastUpdates> Error scanning values: %s", err)
			continue
		}

		wg.Add(1)
		chanProj <- struct {
			key          string
			lastModified time.Time
		}{
			projectKey.String,
			projectLastModified.Time,
		}

		if appName.Valid && appLastModified.Valid {
			wg.Add(1)
			chanApp <- struct {
				key          string
				name         string
				lastModified time.Time
			}{
				projectKey.String,
				appName.String,
				appLastModified.Time,
			}
		}

		if pipName.Valid && pipLastModified.Valid {
			wg.Add(1)
			chanPip <- struct {
				key          string
				name         string
				lastModified time.Time
			}{
				projectKey.String,
				pipName.String,
				pipLastModified.Time,
			}
		}
	}

	wg.Wait()
	quit <- 1
	close(chanProj)
	close(chanApp)
	close(chanPip)
	for _, v := range mapRes {
		res = append(res, *v)
	}

	return res, nil
}
