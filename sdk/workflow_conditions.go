package sdk

import (
	"regexp"
	"strings"
)

// Workflow conditions operator
const (
	WorkflowConditionsOperatorEquals             = "eq"
	WorkflowConditionsOperatorNotEquals          = "ne"
	WorkflowConditionsOperatorLessThan           = "lt"
	WorkflowConditionsOperatorLessOrEqualThan    = "le"
	WorkflowConditionsOperatorGreaterThan        = "gt"
	WorkflowConditionsOperatorGreaterOrEqualThan = "ge"
	WorkflowConditionsOperatorRegex              = "regex"
)

// Workflow conditions operator
var (
	WorkflowConditionsOperators = []string{
		WorkflowConditionsOperatorEquals,
		WorkflowConditionsOperatorNotEquals,
		WorkflowConditionsOperatorLessThan,
		WorkflowConditionsOperatorLessOrEqualThan,
		WorkflowConditionsOperatorGreaterThan,
		WorkflowConditionsOperatorGreaterOrEqualThan,
		WorkflowConditionsOperatorRegex,
	}
)

//WorkflowCheckConditions checks conditions given a list of parameters
func WorkflowCheckConditions(conditions []WorkflowTriggerCondition, params []Parameter) (bool, error) {
	mapParams := ParametersToMap(params)
	for k, v := range mapParams {
		var err error
		mapParams[k], err = Interpolate(v, mapParams)
		if err != nil {
			return false, err
		}
	}

	var conditionsOK = true
	for _, cond := range conditions {
		var err error
		cond.Value, err = Interpolate(cond.Value, mapParams)
		if err != nil {
			return false, err
		}

		switch cond.Operator {
		case WorkflowConditionsOperatorEquals:
			conditionsOK = conditionsOK && cond.Value == mapParams[cond.Variable]

		case WorkflowConditionsOperatorNotEquals:
			conditionsOK = conditionsOK && cond.Value != mapParams[cond.Variable]

		case WorkflowConditionsOperatorLessThan:
			conditionsOK = conditionsOK && strings.Compare(mapParams[cond.Variable], cond.Value) < 0

		case WorkflowConditionsOperatorLessOrEqualThan:
			conditionsOK = conditionsOK && strings.Compare(mapParams[cond.Variable], cond.Value) <= 0

		case WorkflowConditionsOperatorGreaterThan:
			conditionsOK = conditionsOK && strings.Compare(mapParams[cond.Variable], cond.Value) > 0

		case WorkflowConditionsOperatorGreaterOrEqualThan:
			conditionsOK = conditionsOK && strings.Compare(mapParams[cond.Variable], cond.Value) >= 0

		case WorkflowConditionsOperatorRegex:
			match, err := regexp.MatchString(cond.Value, mapParams[cond.Variable])
			if err != nil {
				return false, err
			}
			conditionsOK = conditionsOK && match
		}
	}

	return conditionsOK, nil
}
