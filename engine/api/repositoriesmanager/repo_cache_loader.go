package repositoriesmanager

import (
	"database/sql"
	"sync"
	"time"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

//RepositoriesCacheLoader has to be launched as a goroutine. It will scan all repositories manager
//for all projects and start preloading repositories
func RepositoriesCacheLoader(delay int) {
	for {
		db := database.DB()
		if db != nil {
			var mayIWork string
			loaderKey := cache.Key("reposmanager", "loading")
			cache.Get(loaderKey, &mayIWork)
			if mayIWork == "" {
				cache.Set(loaderKey, "true")
				projects := []*sdk.Project{}

				var query string
				var err error
				var rows *sql.Rows

				query = `SELECT project.id, project.projectKey,project.name
			  FROM project
			  ORDER by project.name, project.projectkey ASC`
				rows, err = db.Query(query)

				if err != nil {
					log.Warning("RepositoriesCacheLoader> Cannot get projects: %s", err)
				}
				defer rows.Close()

				for rows.Next() {
					var id int64
					var key, name string
					rows.Scan(&id, &key, &name)
					p := sdk.NewProject(key)
					p.Name = name
					p.ID = id
					projects = append(projects, p)
				}
				wg := &sync.WaitGroup{}
				for _, proj := range projects {
					projectKey := proj.Key
					rms, err := LoadAllForProject(db, projectKey)
					if err != nil {
						log.Warning("RepositoriesCacheLoader> Cannot get repositories manager: %s", err)
					}

					for _, rm := range rms {
						rmName := rm.Name
						client, err := AuthorizedClient(db, projectKey, rmName)
						if err != nil {
							log.Warning("RepositoriesCacheLoader> Cannot get client %s: %s", rmName, err)
							continue
						}
						if client == nil {
							continue
						}
						go func(projectKey, rmName string) {
							wg.Add(1)
							cacheKey := cache.Key("reposmanager", "repos", projectKey, rmName)
							log.Info("RepositoriesCacheLoader> Loading repos for %s on %s", projectKey, rmName)
							repos, errr := client.Repos()
							if errr != nil {
								log.Warning("RepositoriesCacheLoader> Error on loading repos for %s on %s, err:%s", projectKey, rmName, errr)
								cache.SetWithTTL(cacheKey, &repos, 0)
							}
							wg.Done()
						}(projectKey, rmName)
						time.Sleep(120 * time.Second)
					}
				}
				wg.Wait()
				cache.Delete(loaderKey)
			}
		}
		time.Sleep(time.Duration(delay) * time.Second)
	}
}
