package sanity

import (
	"regexp"
	"sync"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//checkGitVariables needs full loaded project, pipeline
func checkGitVariables(db gorp.SqlExecutor, store cache.Store, vars []string, p *sdk.Project, pip *sdk.Pipeline, a *sdk.Action) []sdk.Warning {
	var warnings []sdk.Warning

	var errapps error
	p.Applications, errapps = application.LoadAll(db, store, p.Key, nil, application.LoadOptions.WithPipelines, application.LoadOptions.WithVariables)
	if errapps != nil {
		log.Warning("checkGitVariables> Unable to load applications for project %s : %s", p.Key, errapps)
		return nil
	}

	var foundGitURLVar bool
	for _, v := range vars {
		if v == "url" {
			foundGitURLVar = true
			break
		}
	}

	//Usage of git.url should be considered only for linked application
	//Usage of git.url should be considered with keys
	if foundGitURLVar {
		//Check key at project level
		var hasKey bool
		for _, v := range p.Variable {
			if v.Type == sdk.KeyVariable {
				hasKey = true
				break
			}
		}
		//Check key at application level
		if !hasKey {
		loopApp:
			for _, a := range p.Applications {
				ok, _ := application.IsAttached(db, p.ID, a.ID, pip.Name)
				if ok {
					for _, v := range a.Variable {
						if v.Type == sdk.KeyVariable {
							hasKey = true
							break loopApp
						}
					}
				}
			}
		}

		if !hasKey {
			w := sdk.Warning{
				ID: GitURLWithoutKey,
				MessageParam: map[string]string{
					"ActionName":   a.Name,
					"PipelineName": pip.Name,
					"ProjectKey":   p.Key,
				},
			}
			w.Action.ID = a.ID
			warnings = append(warnings, w)
		}

		if len(p.VCSServers) == 0 {
			w := sdk.Warning{
				ID: GitURLWithoutLinkedRepository,
				MessageParam: map[string]string{
					"ActionName":   a.Name,
					"PipelineName": pip.Name,
					"ProjectKey":   p.Key,
				},
			}
			w.Action.ID = a.ID
			warnings = append(warnings, w)
		} else {
			for _, app := range p.Applications {
				ok, _ := application.IsAttached(db, p.ID, a.ID, pip.Name)
				if ok {
					if app.VCSServer == "" || app.RepositoryFullname == "" {
						w := sdk.Warning{
							ID: GitURLWithoutLinkedRepository,
							MessageParam: map[string]string{
								"ActionName":   a.Name,
								"PipelineName": pip.Name,
								"ProjectKey":   p.Key,
							},
						}
						w.Action.ID = a.ID
						warnings = append(warnings, w)
					}
				}
			}
		}
	}

	return warnings
}

// loadUsedVariables browse all parameters of an action and returns
// - project variables used in this actions and all its children
// - application variables used in this actions and all its children
// - environment variables used in this actions and all its children
// - git variables used in this actions and all its children
// - mal formated variables used in this actions and all its children

var (
	projectVarReg = regexp.MustCompile(`{{\.cds\.proj\.(.*?)}}`)
	appVarReg     = regexp.MustCompile(`{{\.cds\.app\.(.*?)}}`)
	envVarReg     = regexp.MustCompile(`{{\.cds\.env\.(.*?)}}`)
	gitVarReg     = regexp.MustCompile(`{{\.git\.(.*?)}}`)
	badVarReg     = regexp.MustCompile(`({{cds.*}})|({{ .*}})|({{.* }})`)
)

type usedVariablesResponse struct {
	pvars, avars, evars, gitvars, badvars []string
}

func loadUsedVariablesFromValue(value string, responseChan chan<- usedVariablesResponse) {
	log.Debug("loadUsedVariablesFromValue> checking \"%s\"", value)
	if value == "" {
		responseChan <- usedVariablesResponse{}
		return
	}

	var pvars, avars, evars, gitvars, badvars []string

	pmap := make(map[string]uint8)
	amap := make(map[string]uint8)
	emap := make(map[string]uint8)
	gitmap := make(map[string]uint8)
	badmap := make(map[string]uint8)

	wg := &sync.WaitGroup{}
	wg.Add(5)
	go func() {
		allloc := projectVarReg.FindAllIndex([]byte(value), -1)
		for _, loc := range allloc {
			match := value[loc[0]+len("{{.cds.proj.") : loc[1]-2]
			pmap[match] = 1
		}
		wg.Done()
	}()

	go func() {
		allloc := appVarReg.FindAllIndex([]byte(value), -1)
		for _, loc := range allloc {
			match := value[loc[0]+len("{{.cds.app.") : loc[1]-2]
			amap[match] = 1
		}
		wg.Done()
	}()

	go func() {
		allloc := envVarReg.FindAllIndex([]byte(value), -1)
		for _, loc := range allloc {
			match := value[loc[0]+len("{{.cds.env.") : loc[1]-2]
			emap[match] = 1
		}
		wg.Done()
	}()

	go func() {
		allloc := gitVarReg.FindAllIndex([]byte(value), -1)
		for _, loc := range allloc {
			match := value[loc[0]+len("{{.git.") : loc[1]-2]
			gitmap[match] = 1
		}
		wg.Done()
	}()

	go func() {
		allloc := badVarReg.FindAllIndex([]byte(value), -1)
		for _, loc := range allloc {
			match := value[loc[0]:loc[1]]
			badmap[match] = 1
		}
		wg.Done()
	}()

	wg.Wait()

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
	for key := range gitmap {
		gitvars = append(gitvars, key)
	}

	responseChan <- usedVariablesResponse{
		pvars:   pvars,
		avars:   avars,
		evars:   evars,
		badvars: badvars,
		gitvars: gitvars,
	}

	log.Debug("loadUsedVariablesFromValue> value \"%s\" checked", value)
}

func loadUsedVariables(a *sdk.Action) ([]string, []string, []string, []string, []string) {
	var pvars, avars, evars, gitvars, badvars []string

	respChan := make(chan usedVariablesResponse, len(a.Parameters))
	done := make(chan bool)
	pmap := make(map[string]int)
	amap := make(map[string]int)
	emap := make(map[string]int)
	gitmap := make(map[string]int)
	badmap := make(map[string]int)

	go func() {
		for {
			r, ok := <-respChan
			for _, v := range r.pvars {
				pmap[v] = 1
			}
			for _, v := range r.avars {
				amap[v] = 1
			}
			for _, v := range r.evars {
				emap[v] = 1
			}
			for _, v := range r.gitvars {
				gitmap[v] = 1
			}
			for _, v := range r.badvars {
				badmap[v] = 1
			}
			if !ok {
				done <- true
				return
			}
		}
	}()

	wg := &sync.WaitGroup{}
	for i := range a.Parameters {
		wg.Add(1)
		go func(p *sdk.Parameter) {
			loadUsedVariablesFromValue(p.Value, respChan)
			wg.Done()
		}(&a.Parameters[i])
	}
	wg.Wait()
	close(respChan)

	<-done

	childRespChan := make(chan usedVariablesResponse, len(a.Actions))
	childDone := make(chan bool)
	go func() {
		for {
			r, ok := <-childRespChan
			for _, v := range r.pvars {
				pmap[v] = 1
			}
			for _, v := range r.avars {
				amap[v] = 1
			}
			for _, v := range r.evars {
				emap[v] = 1
			}
			for _, v := range r.gitvars {
				gitmap[v] = 1
			}
			for _, v := range r.badvars {
				badmap[v] = 1
			}
			if !ok {
				childDone <- true
				return
			}
		}
	}()

	wgc := &sync.WaitGroup{}
	for i := range a.Actions {
		wgc.Add(1)
		go func(child *sdk.Action) {
			cpvars, cavars, cevars, cgitvars, cbadvars := loadUsedVariables(child)
			childRespChan <- usedVariablesResponse{
				pvars:   cpvars,
				avars:   cavars,
				evars:   cevars,
				gitvars: cgitvars,
				badvars: cbadvars,
			}
			wgc.Done()
		}(&a.Actions[i])
	}
	wgc.Wait()
	close(childRespChan)

	<-childDone

	wge := &sync.WaitGroup{}
	wge.Add(5)
	go func() {
		for key := range pmap {
			pvars = append(pvars, key)
		}
		wge.Done()
	}()
	go func() {
		for key := range amap {
			avars = append(avars, key)
		}
		wge.Done()
	}()
	go func() {
		for key := range emap {
			evars = append(evars, key)
		}
		wge.Done()
	}()
	go func() {
		for key := range badmap {
			badvars = append(badvars, key)
		}
		wge.Done()
	}()
	go func() {
		for key := range gitmap {
			gitvars = append(gitvars, key)
		}
		wge.Done()
	}()
	wge.Wait()

	return pvars, avars, evars, gitvars, badvars
}

// For each project variable used, check it's present in project variables
func checkProjectVariables(db gorp.SqlExecutor, vars []string, p *sdk.Project, pip *sdk.Pipeline, a *sdk.Action) ([]sdk.Warning, error) {
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
			log.Warning("Variable %s was not found in project %s !", m, p.Key)
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
func checkApplicationVariables(db gorp.SqlExecutor, store cache.Store, vars []string, project *sdk.Project, pip *sdk.Pipeline, a *sdk.Action) ([]sdk.Warning, error) {
	var warnings []sdk.Warning

	// Load all application where pipeline is attached
	apps, err := application.LoadByPipeline(db, store, pip.ID, nil)
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
func checkEnvironmentVariables(db gorp.SqlExecutor, vars []string, project *sdk.Project, pip *sdk.Pipeline, a *sdk.Action) ([]sdk.Warning, error) {
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
