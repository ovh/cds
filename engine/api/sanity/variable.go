package sanity

import (
	"regexp"

	"stash.ovh.net/cds/engine/api/database"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func loadUsedVariables(tx gorp.SqlExecutor, a *sdk.Action) ([]string, []string, []string, []string, error) {
	projectVarReg := regexp.MustCompile(`{{\.cds\.proj\.(.*?)}}`)
	appVarReg := regexp.MustCompile(`{{\.cds\.app\.(.*?)}}`)
	envVarReg := regexp.MustCompile(`{{\.cds\.env\.(.*?)}}`)
	badVarReg := regexp.MustCompile(`{{cds.*}}`)
	var pvars, avars, evars, badvars []string
	var pmap, amap, emap, badmap map[string]int
	pmap = make(map[string]int)
	amap = make(map[string]int)
	emap = make(map[string]int)
	badmap = make(map[string]int)

	for _, p := range a.Parameters {
		allloc := projectVarReg.FindAllIndex([]byte(p.Value), -1)
		for _, loc := range allloc {
			match := p.Value[loc[0]+len("{{.cds.proj.") : loc[1]-2]
			pmap[match] = 1
		}
		allloc = appVarReg.FindAllIndex([]byte(p.Value), -1)
		for _, loc := range allloc {
			match := p.Value[loc[0]+len("{{.cds.app.") : loc[1]-2]
			amap[match] = 1
		}
		allloc = envVarReg.FindAllIndex([]byte(p.Value), -1)
		for _, loc := range allloc {
			match := p.Value[loc[0]+len("{{.cds.env.") : loc[1]-2]
			emap[match] = 1
		}
		allloc = badVarReg.FindAllIndex([]byte(p.Value), -1)
		for _, loc := range allloc {
			match := p.Value[loc[0]:loc[1]]
			badmap[match] = 1
		}
	}

	for _, child := range a.Actions {
		cpvars, cavars, cevars, cbadvars, err := loadUsedVariables(tx, &child)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		for _, v := range cpvars {
			pmap[v] = 1
		}
		for _, v := range cavars {
			amap[v] = 1
		}
		for _, v := range cevars {
			emap[v] = 1
		}
		for _, v := range cbadvars {
			badmap[v] = 1
		}
	}

	for key := range pmap {
		pvars = append(pvars, key)
	}
	for key := range amap {
		avars = append(avars, key)
	}
	for key := range emap {
		evars = append(evars, key)
	}
	for key := range badmap {
		badvars = append(badvars, key)
	}

	return pvars, avars, evars, badvars, nil
}

// For each project variable used, check it's present in project variables
func checkProjectVariables(db database.Querier, vars []string, p *sdk.Project, pip *sdk.Pipeline, a *sdk.Action) ([]sdk.Warning, error) {
	var warnings []sdk.Warning

	var err error
	p.Variable, err = project.GetAllVariableInProject(db, p.ID)
	if err != nil {
		return nil, err
	}

	for _, m := range vars {
		found := false
		for _, v := range p.Variable {
			if m == v.Name {
				found = true
				break
			}
		}
		if !found {
			log.Warning("Variable %s was not found in project %s !\n", m, p.Key)
			w := sdk.Warning{
				ID: ProjectVariableDoesNotExist,
				MessageParam: map[string]string{
					"ActionName":   a.Name,
					"PipelineName": pip.Name,
					"ProjectKey":   p.Key,
					"VarName":      m,
				},
			}
			w.Action.ID = a.ID
			warnings = append(warnings, w)
		}
	}

	return warnings, nil
}

// For each application variable used, check it's present in application where pipeline is used
func checkApplicationVariables(db database.Querier, vars []string, project *sdk.Project, pip *sdk.Pipeline, a *sdk.Action) ([]sdk.Warning, error) {
	var warnings []sdk.Warning

	// Load all application where pipeline is attached
	apps, err := application.LoadApplicationByPipeline(db, pip.ID)
	if err != nil {
		return nil, err
	}

	// For all apps, load variables and check all used vars
	for _, app := range apps {
		avars, err := application.GetAllVariableByID(db, app.ID)
		if err != nil {
			return nil, err
		}

		// Check all used variables with application variables
		for _, m := range vars {
			found := false
			for _, av := range avars {
				if av.Name == m {
					found = true
					break
				}
			}

			if !found {
				log.Warning("Variable %s was not found in application %s/%s !\n", m, project.Key, app.Name)
				w := sdk.Warning{
					ID: ApplicationVariableDoesNotExist,
					MessageParam: map[string]string{
						"ActionName":   a.Name,
						"PipelineName": pip.Name,
						"ProjectKey":   project.Key,
						"VarName":      m,
						"AppName":      app.Name,
					},
				}
				w.Action.ID = a.ID
				w.Application.ID = app.ID
				w.Pipeline.ID = pip.ID
				w.Project.ID = project.ID
				warnings = append(warnings, w)
			}
		}
	}

	return warnings, nil
}

// For each environment variable used:
// Add a warning for each variable if pipeline type is BuildPipeline
// Add a warning for each variable used but not presend in environment variables
func checkEnvironmentVariables(db database.Querier, vars []string, project *sdk.Project, pip *sdk.Pipeline, a *sdk.Action) ([]sdk.Warning, error) {
	var warnings []sdk.Warning

	// If it's a build pipeline, it cannot use environment variables at all
	if pip.Type == sdk.BuildPipeline {
		for _, v := range vars {
			w := sdk.Warning{
				ID: CannotUseEnvironmentVariable,
				MessageParam: map[string]string{
					"ActionName":   a.Name,
					"PipelineName": pip.Name,
					"ProjectKey":   project.Key,
					"VarName":      v,
				},
			}
			w.Action.ID = a.ID
			warnings = append(warnings, w)
		}
		return warnings, nil
	}

	// Load all project environment and check them
	envs, err := environment.LoadEnvironments(db, project.Key, true, &sdk.User{Admin: true})
	if err != nil {
		return nil, err
	}

	// Then check all used vars for each environment
	for _, v := range vars {
		foundInAll := true
		for _, env := range envs {
			found := false
			for _, ev := range env.Variable {
				if v == ev.Name {
					found = true
				}
			}
			if !found {
				foundInAll = false
				break
			}
		}

		// If not found in all environments, add warning
		if !foundInAll {
			w := sdk.Warning{
				ID: EnvironmentVariableDoesNotExist,
				MessageParam: map[string]string{
					"ActionName":   a.Name,
					"PipelineName": pip.Name,
					"ProjectKey":   project.Key,
					"VarName":      v,
				},
			}
			w.Action.ID = a.ID
			warnings = append(warnings, w)
		}
	}

	return warnings, nil
}
