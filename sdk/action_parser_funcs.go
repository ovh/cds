package sdk

import (
  "context"
  "fmt"
  "strings"

  "github.com/rockbears/log"
)

var (
  DefaultFuncs = map[string]ActionFunc{
    "contains":   contains,
    "startsWith": startsWith,
    "endsWith":   endsWith,
    "format":     format,
  }
)

type ActionFunc func(ctx context.Context, a *ActionParser, inputs ...interface{}) (interface{}, error)

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

func startsWith(_ context.Context, _ *ActionParser, inputs ...interface{}) (interface{}, error) {
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

func endsWith(_ context.Context, _ *ActionParser, inputs ...interface{}) (interface{}, error) {
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

func format(_ context.Context, _ *ActionParser, inputs ...interface{}) (interface{}, error) {
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

func join(_ context.Context, _ *ActionParser, inputs ...interface{}) (interface{}, error) {
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
