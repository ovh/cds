package workflow

import (
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

func getNodeJobRunRequirements(db gorp.SqlExecutor, j sdk.Job, run *sdk.WorkflowNodeRun) ([]sdk.Requirement, *sdk.MultiError) {
	requirements := []sdk.Requirement{}
	tmp := map[string]string{}
	errm := &sdk.MultiError{}

	for _, v := range run.BuildParameters {
		tmp[v.Name] = v.Value
	}

	for _, v := range j.Action.Requirements {
		name, errName := sdk.Interpolate(v.Name, tmp)
		if errName != nil {
			errm.Append(errName)
			continue
		}
		value, errValue := sdk.Interpolate(v.Value, tmp)
		if errValue != nil {
			errm.Append(errValue)
			continue
		}
		sdk.AddRequirement(&requirements, name, v.Type, value)
	}

	if errm.IsEmpty() {
		return requirements, nil
	}
	return requirements, errm
}

func prepareRequirementsToNodeJobRunParameters(reqs []sdk.Requirement) []sdk.Parameter {
	params := []sdk.Parameter{}
	for _, r := range reqs {
		if r.Type == sdk.ServiceRequirement {
			k := fmt.Sprintf("cds.requirement.%s.%s", strings.ToLower(r.Type), strings.ToLower(r.Name))
			values := strings.Split(r.Value, " ")
			if len(values) > 1 {
				sdk.AddParameter(&params, k+".image", sdk.StringParameter, values[0])
				sdk.AddParameter(&params, k+".options", sdk.StringParameter, strings.Join(values[1:], " "))
			}
		}
		k := fmt.Sprintf("cds.requirement.%s.%s", strings.ToLower(r.Type), strings.ToLower(r.Name))
		sdk.AddParameter(&params, k, sdk.StringParameter, r.Value)
	}
	return params
}
