package sdk

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/ovh/cds/sdk/glob"
	"github.com/pkg/errors"
	"github.com/rockbears/log"
	"github.com/spf13/cast"
)

var (
	DefaultFuncs = map[string]ActionFunc{
		"contains":     contains,
		"startsWith":   startsWith,
		"endsWith":     endsWith,
		"format":       format,
		"join":         join,
		"toJSON":       toJSON,
		"fromJSON":     fromJSON,
		"hashFiles":    hashFiles,
		"success":      success,
		"always":       always,
		"cancelled":    cancelled,
		"failure":      failure,
		"result":       result,
		"toLower":      newStringActionFunc("toLower", nilerr(strings.ToLower)),
		"toUpper":      newStringActionFunc("toUpper", nilerr(strings.ToUpper)),
		"toTitle":      newStringActionFunc("toTitle", nilerr(strings.ToTitle)),
		"title":        newStringActionFunc("title", nilerr(strings.Title)),
		"b64enc":       newStringActionFunc("b64enc", nilerr(base64encode)),
		"b64dec":       newStringActionFunc("b64dec", base64decode),
		"b32enc":       newStringActionFunc("b32enc", nilerr(base32encode)),
		"b32dec":       newStringActionFunc("b32dec", base32decode),
		"trimAll":      newStringStringActionFunc("trimAll", strings.Trim),
		"trimPrefix":   newStringStringActionFunc("trimPrefix", strings.TrimPrefix),
		"trimSuffix":   newStringStringActionFunc("trimSuffix", strings.TrimSuffix),
		"toArray":      toArray,
		"match":        match,
		"replace":      replace,
		"contextValue": contextValue,
		"default":      dfault,
		"coalesce":     coalesce,
	}
)

type ActionFunc func(ctx context.Context, a *ActionParser, inputs ...interface{}) (interface{}, error)

func dfault(_ context.Context, a *ActionParser, inputs ...interface{}) (interface{}, error) {
	switch len(inputs) {
	case 0:
		return nil, nil // default
	case 1:
		return inputs[0], nil // default "var" OR default ""
	case 2:
		if !empty(inputs[0]) {
			return inputs[0], nil // default "var" "" OR default "var" "another"
		}
		return inputs[1], nil // default "" "var" OR default "" ""
	default:
		return nil, NewErrorFrom(ErrInvalidData, "default: wrong number of arguments")
	}
}

// coalesce returns the first non-empty value.
func coalesce(_ context.Context, _ *ActionParser, inputs ...interface{}) (interface{}, error) {
	for _, val := range inputs {
		if !empty(val) {
			return val, nil
		}
	}
	return nil, nil
}

func replace(_ context.Context, _ *ActionParser, inputs ...interface{}) (interface{}, error) {
	if len(inputs) != 3 && len(inputs) != 4 {
		return nil, NewErrorFrom(ErrInvalidData, "replace: wrong number of arguments")
	}
	input, ok := inputs[0].(string)
	if !ok {
		return nil, NewErrorFrom(ErrInvalidData, "replace: input must be a string")
	}
	old, ok := inputs[1].(string)
	if !ok {
		return nil, NewErrorFrom(ErrInvalidData, "replace: old must be a string")
	}
	new, ok := inputs[2].(string)
	if !ok {
		return nil, NewErrorFrom(ErrInvalidData, "replace: new must be a string")
	}

	nbOfReplacements := -1
	if len(inputs) == 4 {
		value, err := strconv.Atoi(fmt.Sprintf("%v", inputs[3]))
		if err != nil {
			return nil, NewErrorFrom(ErrInvalidData, "replace: nbOfReplacements must be an int")
		}
		nbOfReplacements = value
	}

	return strings.Replace(input, old, new, nbOfReplacements), nil
}

func contextValue(_ context.Context, a *ActionParser, inputs ...interface{}) (interface{}, error) {
	if len(inputs) == 0 {
		return nil, NewErrorFrom(ErrInvalidData, "contextValue: wrong number of arguments")
	}

	var currentObject any
	for i, index := range inputs {
		if i > 0 && currentObject == nil {
			return nil, NewErrorFrom(ErrInvalidData, "contextValue: object %v doesn't exist", inputs[0:i])
		}
		switch t := index.(type) {
		case string:
			if currentObject == nil {
				obj, has := a.contexts[t]
				if !has {
					return nil, NewErrorFrom(ErrInvalidData, "contextValue: unable to find context %s", t)
				}
				currentObject = obj
				continue
			}
			switch m := currentObject.(type) {
			case map[string]interface{}:
				currentObject = m[t]
			default:
				return nil, NewErrorFrom(ErrInvalidData, "contextValue: cannot get value at index %s in object of type %T", t, currentObject)
			}
		case int:
			if currentObject == nil {
				return nil, NewErrorFrom(ErrInvalidData, "contextValue: first argument must be a context name")
			}
			switch m := currentObject.(type) {
			case map[int]interface{}:
				currentObject = m[t]
			case map[int64]interface{}:
				currentObject = m[int64(t)]
			case []map[int64]interface{}:
				currentObject = m[t]
			case []map[int]interface{}:
				currentObject = m[t]
			case []map[string]interface{}:
				currentObject = m[t]
			case []interface{}:
				currentObject = m[t]
			default:
				return nil, NewErrorFrom(ErrInvalidData, "contextValue: cannot get value at int index %d in object of type %T", t, currentObject)
			}
		default:
			return nil, NewErrorFrom(ErrInvalidData, "contextValue: wrong type of argument. Got %T, Need string or integrer ", index)
		}
	}
	return currentObject, nil
}

func toArray(_ context.Context, _ *ActionParser, inputs ...interface{}) (interface{}, error) {
	if len(inputs) == 0 {
		return []any{}, nil
	}

	if len(inputs) == 1 {
		val := reflect.ValueOf(inputs[0])
		switch val.Kind() {
		case reflect.Array, reflect.Slice:
			return inputs[0], nil
		default:
			if inputs[0] == nil {
				return []any{}, nil
			}
			return []any{inputs[0]}, nil
		}
	}

	return inputs, nil
}

func result(ctx context.Context, a *ActionParser, inputs ...interface{}) (interface{}, error) {
	if len(inputs) != 2 {
		return nil, NewErrorFrom(ErrInvalidData, "result: wrong number of arguments to call result(type, name)")
	}

	typ, ok := inputs[0].(string)
	if !ok {
		return nil, NewErrorFrom(ErrInvalidData, "result: item argument must be a string")
	}

	name, ok := inputs[1].(string)
	if !ok {
		return nil, NewErrorFrom(ErrInvalidData, "result: item argument must be a string")
	}
	glob := glob.New(name)

	jobsMap := cast.ToStringMap(a.contexts["jobs"])
	if jobsMap == nil {
		return nil, errors.New("result: map jobs not found in context")
	}

	var results []any

	for _, jobContextI := range jobsMap { // Iterate over all the jobs
		var jobRunResultsAsMap map[string]interface{}

		jobContext := cast.ToStringMap(jobContextI)
		if jobContext == nil {
			return nil, errors.New("result: unable to cast job context to map")
		}

		resultsMapI := jobContext["results"]
		if resultsMapI != nil {
			resultsMap := cast.ToStringMap(resultsMapI)
			jobRunResultsI := resultsMap["JobRunResults"]
			if jobRunResultsI != nil {
				jobRunResults := cast.ToStringMap(jobRunResultsI)
				if jobRunResults == nil {
					return nil, errors.New("result: unable to cast jobRunResults context to map")
				}
				var err error
				jobRunResultsAsMap, err = cast.ToStringMapE(jobRunResults)
				if err != nil {
					continue
				}
			}
		} else {
			jobRunResultsAsMapI := jobContext["JobRunResults"]
			if jobRunResultsAsMapI != nil {
				jobRunResultsAsMap = cast.ToStringMap(jobRunResultsAsMapI)
				if jobRunResultsAsMap == nil {
					return nil, errors.New("result: unable to cast jobRunResultsAsMap context to map")
				}
			}
		}

		for k, v := range jobRunResultsAsMap {
			if strings.HasPrefix(k, typ+":") {
				g, err := glob.MatchString(strings.TrimPrefix(k, typ+":"))
				if err != nil {
					return nil, err
				}
				if g != nil {
					results = append(results, v)
				}
			}
		}
	}

	if len(results) == 0 {
		return nil, nil
	}

	if len(results) == 1 {
		return results[0], nil
	}

	return results, nil
}

func match(ctx context.Context, _ *ActionParser, inputs ...interface{}) (interface{}, error) {
	log.Debug(ctx, "function: match with args: %v", inputs)
	if len(inputs) != 2 {
		return nil, NewErrorFrom(ErrInvalidData, "match: wrong number of arguments to call match(stringToTest, globPattern)")
	}

	globPattern, ok := inputs[1].(string)
	if !ok {
		return nil, NewErrorFrom(ErrInvalidData, "match: globPattern argument must be a string")
	}

	stringToTest, ok := inputs[0].(string)
	if !ok {
		return nil, NewErrorFrom(ErrInvalidData, "match: stringToTest argument must be a string")
	}

	g := glob.New(globPattern)
	result, err := g.MatchString(stringToTest)
	if err != nil {
		return nil, NewErrorFrom(ErrInvalidData, "match: unable to check %s with pattern %s: %v", stringToTest, globPattern, err)
	}
	return result != nil, nil
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

func hashFiles(_ context.Context, a *ActionParser, inputs ...interface{}) (interface{}, error) {
	if len(inputs) == 0 {
		return nil, NewErrorFrom(ErrInvalidData, "hashFiles function must have arguments")
	}
	var cdsContext CDSContext
	ctxInt, has := a.contexts["cds"]
	if has {
		cdsContextBts, _ := json.Marshal(ctxInt)
		if err := json.Unmarshal(cdsContextBts, &cdsContext); err != nil {
			return nil, NewErrorFrom(ErrInvalidData, "unable to read cds context")
		}
	}

	type inputFile struct {
		dirFS fs.FS
		files []string
	}
	files := make([]inputFile, 0)
	for _, i := range inputs {
		input, ok := i.(string)
		if !ok {
			return nil, NewErrorFrom(ErrInvalidData, "%v must be a string", i)
		}
		filesFound, err := glob.Glob(cdsContext.Workspace, input)
		if err != nil {
			return nil, NewErrorFrom(ErrInvalidData, "unable to find files with pattern %s on directory %s: %v", input, cdsContext.Workspace, err)
		}
		ifiles := inputFile{
			dirFS: filesFound.DirFS,
			files: make([]string, 0, len(filesFound.Results)),
		}
		for _, f := range filesFound.Results {
			ifiles.files = append(ifiles.files, f.Path)
		}
		files = append(files, ifiles)
	}
	if len(files) == 0 {
		return nil, NewErrorFrom(ErrInvalidData, "find 0 file with filter %v", inputs)
	}

	allFiles := make([]string, 0)
	for _, inputFile := range files {
		for _, f := range inputFile.files {
			allFiles = append(allFiles, filepath.Join(fmt.Sprintf("%s", inputFile.dirFS), f))
		}
	}
	sort.Strings(allFiles)

	buf := make([]byte, 0)

	for _, file := range allFiles {
		f, err := os.Open(file)
		if err != nil {
			return nil, NewErrorFrom(ErrInvalidData, "unable to read file %s: %v", file, err)
		}
		hasher := sha256.New()
		if _, err := io.Copy(hasher, f); err != nil {
			_ = f.Close()
			return nil, NewErrorFrom(ErrInvalidData, "unable to compute sha256 for file %s: %v", file, err)
		}
		_ = f.Close()
		buf = append(buf, []byte(hex.EncodeToString(hasher.Sum(nil)))...)
	}

	hasher := sha256.New()
	_, err := hasher.Write(buf)
	if err != nil {
		return nil, NewErrorFrom(ErrInvalidData, "unable to compute global sha256: %v", err)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
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
			if v.Conclusion != V2WorkflowRunJobStatusSuccess && v.Conclusion != V2WorkflowRunJobStatusSkipped {
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
	} else if needsContext, has := a.contexts["needs"]; has && needsContext != nil {
		var needs NeedsContext
		jobsCtxBts, _ := json.Marshal(needsContext)
		if err := json.Unmarshal(jobsCtxBts, &needs); err != nil {
			return nil, NewErrorFrom(ErrInvalidData, "unable to read jobs context")
		}
		for _, v := range needs {
			if v.Result == V2WorkflowRunJobStatusFail {
				return true, nil
			}
		}
		return false, nil
	}
	return nil, NewErrorFrom(ErrInvalidData, "missing step and jobs contexts")
}

type (
	stringActionFunc       func(string) (string, error)
	stringStringActionFunc func(string, string) string
)

func newStringActionFunc(name string, fn stringActionFunc) ActionFunc {
	return func(ctx context.Context, _ *ActionParser, inputs ...interface{}) (interface{}, error) {
		log.Debug(ctx, "function: %s with args: %v", name, inputs)

		if len(inputs) != 1 {
			return nil, NewErrorFrom(ErrInvalidData, "%s: requires one argument", name)
		}
		s, ok := inputs[0].(string)
		if !ok {
			return nil, NewErrorFrom(ErrInvalidData, "%s: argument must be a string", name)
		}
		return fn(s)
	}
}

func newStringStringActionFunc(name string, fn stringStringActionFunc) ActionFunc {
	return func(ctx context.Context, _ *ActionParser, inputs ...interface{}) (interface{}, error) {
		log.Debug(ctx, "function: %s with args: %v", name, inputs)

		if len(inputs) != 2 {
			return nil, NewErrorFrom(ErrInvalidData, "%s: requires two argument", name)
		}
		a, ok := inputs[0].(string)
		if !ok {
			return nil, NewErrorFrom(ErrInvalidData, "%s: first argument must be a string", name)
		}
		b, ok := inputs[1].(string)
		if !ok {
			return nil, NewErrorFrom(ErrInvalidData, "%s: second argument must be a string", name)
		}
		return fn(b, a), nil
	}
}

func base64encode(v string) string {
	return base64.StdEncoding.EncodeToString([]byte(v))
}

func base64decode(v string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func base32encode(v string) string {
	return base32.StdEncoding.EncodeToString([]byte(v))
}

func base32decode(v string) (string, error) {
	data, err := base32.StdEncoding.DecodeString(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func nilerr(fn func(string) string) stringActionFunc {
	return func(s string) (string, error) {
		return fn(s), nil
	}
}

// empty returns true if the given value has the zero value for its type.
func empty(given interface{}) bool {
	g := reflect.ValueOf(given)
	if !g.IsValid() {
		return true
	}

	// Basically adapted from text/template.isTrue
	switch g.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return g.Len() == 0
	case reflect.Bool:
		return g.Bool() == false
	case reflect.Complex64, reflect.Complex128:
		return g.Complex() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return g.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return g.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return g.Float() == 0
	case reflect.Struct:
		return false
	default:
		return g.IsNil()
	}
}
