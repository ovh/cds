package action

import (
	"strings"

	"github.com/ovh/cds/sdk"
)

// ProcessActionBuildVariables create and process the full set of build variables from
// - Project variables not secret
// - Application variables not secret
// - Environment variables not secret
// - Pipeline parameters
// - Action definition in pipeline
// - ActionBuild variables (global ones + trigger parameters)
func ProcessActionBuildVariables(projectVariables []sdk.Variable, appVariables []sdk.Variable, envVariables []sdk.Variable, pipelineParameters []sdk.Parameter, actionBuildArguments []sdk.Parameter, stage *sdk.Stage, action sdk.Action) []sdk.Parameter {
	abv := make(map[string]sdk.Parameter)
	final := []sdk.Parameter{}
	project := "cds.proj"
	app := "cds.app"
	env := "cds.env"
	pipeline := "cds.pip"

	// Do not add secrets nor keys
	for _, t := range projectVariables {
		if sdk.NeedPlaceholder(t.Type) {
			continue
		}

		t.Name = project + "." + t.Name
		abv[t.Name] = sdk.Parameter{Name: t.Name, Type: t.Type, Value: t.Value}
	}

	for _, t := range appVariables {
		if sdk.NeedPlaceholder(t.Type) {
			continue
		}

		t.Name = app + "." + t.Name
		abv[t.Name] = sdk.Parameter{Name: t.Name, Type: t.Type, Value: t.Value}
	}

	for _, t := range envVariables {
		if sdk.NeedPlaceholder(t.Type) {
			continue
		}

		t.Name = env + "." + t.Name
		abv[t.Name] = sdk.Parameter{Name: t.Name, Type: t.Type, Value: t.Value}
	}

	for _, t := range pipelineParameters {
		t.Name = pipeline + "." + t.Name
		abv[t.Name] = t
	}

	for _, a := range actionBuildArguments {
		abv[a.Name] = a
	}

	abv["cds.stage"] = sdk.Parameter{Name: "cds.stage", Type: sdk.StringParameter, Value: stage.Name}
	abv["cds.job"] = sdk.Parameter{Name: "cds.job", Type: sdk.StringParameter, Value: action.Name}

	// Until there is no replace (or loop escape trigger), replace variables
	var loopEscape int
	var replaced bool
	for loopEscape < 10 {
		replaced = false
		// For each build action variable
		for i := range abv {
			// Replace possible variable with its value
			for _, v := range abv {
				newValue := abv[i]
				newValue.Value = strings.Replace(newValue.Value, "{{."+v.Name+"}}", v.Value, -1)
				if abv[i].Value != newValue.Value {
					replaced = true
					abv[i] = newValue
				}
			}
		}
		if replaced == false {
			break
		}
		loopEscape++
	}

	for _, p := range abv {
		final = append(final, p)
	}

	return final
}
