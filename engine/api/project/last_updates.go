package project

import (
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
)

func getLastModified(store cache.Store, k string) (int64, string) {
	m := &sdk.LastModification{}
	if store.Get(k, m) {
		return m.LastModified, m.Username
	}
	return 0, ""
}

//LastUpdates returns projects and application last update
func LastUpdates(db gorp.SqlExecutor, store cache.Store, user *sdk.User, since time.Time) ([]sdk.ProjectLastUpdates, error) {
	res := []sdk.ProjectLastUpdates{}

	mapRes := map[string]*sdk.ProjectLastUpdates{}

	for _, g := range user.Groups {
		for _, pg := range g.ProjectGroups {
			t, s := getLastModified(store, cache.Key("lastModified", pg.Project.Key))
			if s != "" && t != 0 && t > since.Unix() {
				mapRes[pg.Project.Key] = &sdk.ProjectLastUpdates{
					LastModification: sdk.LastModification{
						Name:         pg.Project.Key,
						LastModified: t,
						Username:     s,
					},
					Applications: []sdk.LastModification{},
					Pipelines:    []sdk.LastModification{},
					Environments: []sdk.LastModification{},
					Workflows:    []sdk.LastModification{},
				}
			}
		}

		for _, ag := range g.ApplicationGroups {
			t, s := getLastModified(store, cache.Key("lastModified", ag.Application.ProjectKey, "application", ag.Application.Name))
			if s != "" && t != 0 && t > since.Unix() {
				proj := mapRes[ag.Application.ProjectKey]
				if proj != nil {
					proj.Applications = append(proj.Applications, sdk.LastModification{
						Name:         ag.Application.Name,
						LastModified: t,
						Username:     s,
					})
				}
			}
		}

		for _, pg := range g.PipelineGroups {
			t, s := getLastModified(store, cache.Key("lastModified", pg.Pipeline.ProjectKey, "pipeline", pg.Pipeline.Name))
			if s != "" && t != 0 && t > since.Unix() {
				proj := mapRes[pg.Pipeline.ProjectKey]
				if proj != nil {
					proj.Pipelines = append(proj.Pipelines, sdk.LastModification{
						Name:         pg.Pipeline.Name,
						LastModified: t,
						Username:     s,
					})
				}
			}
		}

		for _, eg := range g.EnvironmentGroups {
			t, s := getLastModified(store, cache.Key("lastModified", eg.Environment.ProjectKey, "environment", eg.Environment.Name))
			if s != "" && t != 0 && t > since.Unix() {
				proj := mapRes[eg.Environment.ProjectKey]
				if proj != nil {
					proj.Environments = append(proj.Environments, sdk.LastModification{
						Name:         eg.Environment.Name,
						LastModified: t,
						Username:     s,
					})
				}
			}
		}

	}

	//TODO workflows

	for _, v := range mapRes {
		res = append(res, *v)
	}

	return res, nil
}
