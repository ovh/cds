package sanity

import (
	"sync"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// CheckApplication checks all application variables
func CheckApplication(db gorp.SqlExecutor, proj *sdk.Project, app *sdk.Application) error {
	warChan := make(chan []sdk.Warning)
	done := make(chan bool)

	if err := DeleteAllApplicationWarnings(db, app.ID); err != nil {
		return err
	}

	go func() {
		for {
			ws, ok := <-warChan
			if ws != nil {
				log.Debug("CheckApplication> Inserting warnings %v", ws)

				for _, w := range ws {
					w.Application.ID = app.ID
					if w.Pipeline.ID == 0 && w.Action.ID == 0 {
						if err := InsertApplicationWarning(db, proj.ID, app.ID, &w); err != nil {
							log.Warning("CheckApplication> Error inserting warnings %s", err)
						}
					}
				}
			}
			if !ok {
				done <- true
				return
			}
		}
	}()

	wg := &sync.WaitGroup{}
	for i := range app.Variable {
		wg.Add(1)
		go func(v *sdk.Variable) {
			ws, err := checkApplicationVariable(proj, app, v)
			if err != nil {
				log.Warning("CheckApplication> Error checking application %s/%s variable %s", proj.Name, app.Name, v.Name)
			} else {
				warChan <- ws
			}
			wg.Done()
		}(&app.Variable[i])
	}
	wg.Wait()
	close(warChan)
	<-done

	return nil
}

func checkApplicationVariable(project *sdk.Project, app *sdk.Application, variable *sdk.Variable) ([]sdk.Warning, error) {
	resChan := make(chan usedVariablesResponse)
	go loadUsedVariablesFromValue(variable.Value, resChan)
	res := <-resChan
	close(resChan)

	log.Debug("checkApplicationVariable> loadUsedVariablesFromValue => %v", res)

	warnings := []sdk.Warning{}

	warChan := make(chan []sdk.Warning, len(project.Environments))
	done := make(chan bool)

	//If application is using Environments variables, there must be at least one Environment
	var nbVars int
	for _, e := range project.Environments {
		nbVars += len(e.Variable)
	}
	if len(res.evars) > 0 && nbVars == 0 {
		w := sdk.Warning{
			ID: MissingEnvironment,
			MessageParam: map[string]string{
				"ApplicationName": app.Name,
			},
		}
		w.Application.ID = app.ID
		warnings = append(warnings, w)
	}

	//Compute badly formatted variables
	for _, v := range res.badvars {
		log.Warning("checkApplicationVariable> Badly formatted variable: '%s'\n", v)
		w := sdk.Warning{
			ID: InvalidVariableFormatUsedInApplication,
			MessageParam: map[string]string{
				"VarName":         v,
				"ApplicationName": app.Name,
			},
		}
		w.Application.ID = app.ID
		warnings = append(warnings, w)
	}

	//Read warnings channels for all Environments
	go func() {
		for {
			ws, ok := <-warChan
			warnings = append(warnings, ws...)
			if !ok {
				done <- true
				return
			}
		}
	}()

	//Checks variables on all Environments
	wg := &sync.WaitGroup{}
	for i := range project.Environments {
		wg.Add(1)
		go func(e *sdk.Environment) {
			wgv := &sync.WaitGroup{}
			for j := range res.evars {
				wgv.Add(1)
				go func(e *sdk.Environment, v string) {
					var found bool
					for _, envVar := range e.Variable {
						if envVar.Name == v {
							found = true
							break
						}
					}
					if !found {
						warChan <- []sdk.Warning{
							{
								ID: EnvironmentVariableUsedInApplicationDoesNotExist,
								MessageParam: map[string]string{
									"VarName":         v,
									"ApplicationName": app.Name,
								},
							},
						}
					}
					wgv.Done()
				}(e, res.evars[j])
			}
			wgv.Wait()
			wg.Done()
		}(&project.Environments[i])
	}
	wg.Wait()
	close(warChan)

	<-done

	return warnings, nil
}
