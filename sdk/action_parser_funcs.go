package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/ovh/cds/sdk/glob"
	"github.com/pkg/errors"
	"github.com/rockbears/log"
	"github.com/spf13/cast"
)

var (
	DefaultFuncs = map[string]ActionFunc{
		"contains":   contains,
		"startsWith": startsWith,
		"endsWith":   endsWith,
		"format":     format,
		"join":       join,
		"toJSON":     toJSON,
		"fromJSON":   fromJSON,
		"hashFiles":  hashFiles,
		"success":    success,
		"always":     always,
		"cancelled":  cancelled,
		"failure":    failure,
		"result":     result,
	}
)

type ActionFunc func(ctx context.Context, a *ActionParser, inputs ...interface{}) (interface{}, error)

func result(ctx context.Context, a *ActionParser, inputs ...interface{}) (interface{}, error) {
	log.Debug(ctx, "function: contains with args: %v", inputs)
	if len(inputs) != 2 {
		return nil, NewErrorFrom(ErrInvalidData, "contains: wrong number of arguments to call contains(type, name)")
	}

	typ, ok := inputs[0].(string)
	if !ok {
		return nil, NewErrorFrom(ErrInvalidData, "contains: item argument must be a string")
	}

	name, ok := inputs[1].(string)
	if !ok {
		return nil, NewErrorFrom(ErrInvalidData, "contains: item argument must be a string")
	}

	glob := glob.New(name)

	btes, _ := json.Marshal(a.contexts)
	log.Error(context.Background(), "result::context= %s", string(btes))

	jobsMap := findMapStringFromMapAsInterface(a.contexts, "jobs")
	if jobsMap == nil {
		log.Error(ctx, "map jobs not found in context")
	}
	for jobID, jobContext := range jobsMap { // Iterate over all the jobs

		log.Debug(ctx, "result::trying to find result in job %q", jobID)

		resultsMap := findMapStringFromMapAsInterface(jobContext, "results")
		for k, v := range resultsMap {
			if strings.HasPrefix(k, typ+":") {
				g, err := glob.MatchString(k)
				if err != nil {
					return nil, err
				}
				if g != nil {
					return v, nil
				}
			}
		}
	}

	return nil, errors.Errorf("unable to find result %q", typ+":"+name)
}

func findMapStringFromMapAsInterface(target interface{}, key string) map[string]interface{} {
	jobContextValue := reflect.ValueOf(target)
	if jobContextValue.Kind() == reflect.Map {
		mapIter := jobContextValue.MapRange()
		for mapIter.Next() {
			kValue := mapIter.Key()
			vValue := mapIter.Value()
			log.Error(context.Background(), "checking %v", kValue.Interface())
			if kValue.CanInterface() && kValue.Interface() == key {
				if vValue.Kind() == reflect.Map && vValue.CanInterface() {
					vMapKeys := vValue.MapKeys()
					if len(vMapKeys) > 0 {
						if vMapKeys[0].CanInterface() && vMapKeys[0].Kind() == reflect.Slice {
							x, err := cast.ToStringMapE(vValue.Interface())
							if err != nil {
								log.ErrorWithStackTrace(context.Background(), err)
								continue
							}
							return x
						}
					}
				}
			}
		}
	} else {
		log.Error(context.Background(), "is not a Mas. its a %v", jobContextValue.Kind())
	}
	return nil
}

// contains(search, item)
func contains(ctx context.Context, _ *ActionParser, inputs ...interface{}) (interface{}, error) {
	log.Debug(ctx, "function: contains with args: %v", inputs)
	if len(inputs) != 2 {
		return nil, NewErrorFrom(ErrInvalidData, "contains: wrong number of arguments to call contains(search, item)")
	}

	inputToSearch, ok := inputs[1].(string)
	if !ok {
		return nil, NewErrorFrom(ErrInvalidData, "contains: item argument must be a string")
	}

	inputSearch, ok := inputs[0].(string)
	if ok {
		return strings.Contains(strings.ToLower(inputSearch), strings.ToLower(inputToSearch)), nil
	}

	inputSearchArray, ok := inputs[0].([]interface{})
	if !ok {
		return nil, NewErrorFrom(ErrInvalidData, "contains: search argument must be a string or an array")
	}

	// If search is an array, inputToSearch must be an item of the array
	for _, i := range inputSearchArray {
		if fmt.Sprintf("%v", i) == inputToSearch {
			return true, nil
		}
	}
	return false, nil
}

func startsWith(ctx context.Context, _ *ActionParser, inputs ...interface{}) (interface{}, error) {
	log.Debug(ctx, "function: startsWith with args: %v", inputs)
	searchString, ok := inputs[0].(string)
	if !ok {
		return nil, NewErrorFrom(ErrInvalidData, "startsWith: searchString argument must be a string")
	}

	searchValue, ok := inputs[1].(string)
	if !ok {
		return nil, NewErrorFrom(ErrInvalidData, "startsWith: searchValue argument must be a string")
	}
	return strings.HasPrefix(strings.ToLower(searchString), strings.ToLower(searchValue)), nil
}

func endsWith(ctx context.Context, _ *ActionParser, inputs ...interface{}) (interface{}, error) {
	log.Debug(ctx, "function: endsWith with args: %v", inputs)
	searchString, ok := inputs[0].(string)
	if !ok {
		return nil, NewErrorFrom(ErrInvalidData, "endsWith: searchString argument must be a string")
	}

	searchValue, ok := inputs[1].(string)
	if !ok {
		return nil, NewErrorFrom(ErrInvalidData, "endsWith: searchValue argument must be a string")
	}
	return strings.HasSuffix(strings.ToLower(searchString), strings.ToLower(searchValue)), nil
}

func format(ctx context.Context, _ *ActionParser, inputs ...interface{}) (interface{}, error) {
	log.Debug(ctx, "function: format with args: %v", inputs)
	if len(inputs) < 2 {
		return nil, NewErrorFrom(ErrInvalidData, "format: you must specify at least one replace value")
	}
	inputString, ok := inputs[0].(string)
	if !ok {
		return nil, NewErrorFrom(ErrInvalidData, "format: first argument must be a string")
	}

	for i := 1; i < len(inputs); i++ {
		inputString = strings.Replace(inputString, fmt.Sprintf("{%d}", i-1), fmt.Sprintf("%v", inputs[i]), -1)
	}
	return inputString, nil
}

func join(ctx context.Context, _ *ActionParser, inputs ...interface{}) (interface{}, error) {
	log.Debug(ctx, "function: join with args: %v", inputs)
	if len(inputs) < 1 || len(inputs) > 2 {
		return nil, NewErrorFrom(ErrInvalidData, "join: you must specify an array as first argument, and an optional separator")
	}
	separator := ","
	if len(inputs) == 2 {
		separator = fmt.Sprintf("%v", inputs[1])
	}

	var arrayString []string

	arrayInput, ok := inputs[0].([]interface{})
	if ok {
		for _, item := range arrayInput {
			arrayString = append(arrayString, fmt.Sprintf("%v", item))
		}
		return strings.Join(arrayString, separator), nil
	}
	stringInput, ok := inputs[0].(string)
	if !ok {
		return nil, NewErrorFrom(ErrInvalidData, "join: first argument must be an array or a string")
	}
	return stringInput, nil
}

func toJSON(ctx context.Context, _ *ActionParser, inputs ...interface{}) (interface{}, error) {
	log.Debug(ctx, "function: toJSON with args: %v", inputs)
	if len(inputs) != 1 {
		return nil, NewErrorFrom(ErrInvalidData, "toJSON: you must have one argument")
	}
	bts, err := json.MarshalIndent(inputs[0], "", "  ")
	if err != nil {
		return nil, NewErrorFrom(ErrInvalidData, "toJSON: given input cannot be convert to JSON")
	}
	return string(bts), nil
}

func fromJSON(ctx context.Context, _ *ActionParser, inputs ...interface{}) (interface{}, error) {
	log.Debug(ctx, "function: fromJSON with args: %v", inputs)
	if len(inputs) != 1 {
		return nil, NewErrorFrom(ErrInvalidData, "fromJSON: you must have one argument")
	}
	if strings.HasPrefix(inputs[0].(string), "[{") {
		var result []map[string]interface{}
		if err := json.Unmarshal([]byte(inputs[0].(string)), &result); err != nil {
			return nil, NewErrorFrom(ErrInvalidData, "fromJSON: given input is not a valid json")
		}
		return result, nil
	} else if strings.HasPrefix(inputs[0].(string), "[") {
		var result []interface{}
		if err := json.Unmarshal([]byte(inputs[0].(string)), &result); err != nil {
			return nil, NewErrorFrom(ErrInvalidData, "fromJSON: given input is not a valid json")
		}
		return result, nil
	} else {
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(inputs[0].(string)), &result); err != nil {
			return nil, NewErrorFrom(ErrInvalidData, "fromJSON: given input is not a valid json")
		}
		return result, nil
	}
}

func hashFiles(_ context.Context, _ *ActionParser, _ ...interface{}) (interface{}, error) {
	return nil, NewErrorFrom(ErrNotImplemented, "hashFiles is not implemented yet")
}

func success(_ context.Context, a *ActionParser, inputs ...interface{}) (interface{}, error) {
	if len(inputs) > 0 {
		return nil, NewErrorFrom(ErrInvalidData, "success function must not have arguments")
	}
	// Check scope
	if stepContext, has := a.contexts["steps"]; has && stepContext != nil {
		var steps StepsContext
		stepContextBts, _ := json.Marshal(stepContext)
		if err := json.Unmarshal(stepContextBts, &steps); err != nil {
			return nil, NewErrorFrom(ErrInvalidData, "unable to read step context")
		}
		for _, v := range steps {
			if v.Conclusion != V2WorkflowRunJobStatusSuccess {
				return false, nil
			}
		}
		return true, nil
	} else if needsContext, has := a.contexts["needs"]; has && needsContext != nil {
		var needs NeedsContext
		needsCtxBts, _ := json.Marshal(needsContext)
		if err := json.Unmarshal(needsCtxBts, &needs); err != nil {
			return nil, NewErrorFrom(ErrInvalidData, "unable to read step context")
		}
		for _, v := range needs {
			if v.Result != V2WorkflowRunJobStatusSuccess {
				return false, nil
			}
		}
		return true, nil
	}
	return nil, NewErrorFrom(ErrInvalidData, "missing steps and needs context")
}

func always(_ context.Context, _ *ActionParser, inputs ...interface{}) (interface{}, error) {
	if len(inputs) > 0 {
		return nil, NewErrorFrom(ErrInvalidData, "always function must not have arguments")
	}
	return true, nil
}

func cancelled(_ context.Context, _ *ActionParser, inputs ...interface{}) (interface{}, error) {
	if len(inputs) > 0 {
		return nil, NewErrorFrom(ErrInvalidData, "cancelled function must not have arguments")
	}
	return nil, NewErrorFrom(ErrNotImplemented, "cancelled is not implemented yet")
}

func failure(_ context.Context, a *ActionParser, inputs ...interface{}) (interface{}, error) {
	if len(inputs) > 0 {
		return nil, NewErrorFrom(ErrInvalidData, "failure function must not have arguments")
	}
	// Check scope
	if stepContext, has := a.contexts["steps"]; has && stepContext != nil {
		var steps StepsContext
		stepContextBts, _ := json.Marshal(stepContext)
		if err := json.Unmarshal(stepContextBts, &steps); err != nil {
			return nil, NewErrorFrom(ErrInvalidData, "unable to read step context")
		}
		for _, v := range steps {
			if v.Conclusion == V2WorkflowRunJobStatusFail {
				return true, nil
			}
		}
		return false, nil
	} else if jobsContext, has := a.contexts["jobs"]; has && jobsContext != nil {
		var jobs JobsResultContext
		jobsCtxBts, _ := json.Marshal(jobsContext)
		if err := json.Unmarshal(jobsCtxBts, &jobs); err != nil {
			return nil, NewErrorFrom(ErrInvalidData, "unable to read jobs context")
		}
		for _, v := range jobs {
			if v.Result == V2WorkflowRunJobStatusFail {
				return true, nil
			}
		}
		return false, nil
	}
	return nil, NewErrorFrom(ErrInvalidData, "missing step and jobs contexts")
}
