package sdk

import (
	"context"
	"strings"

	"github.com/rockbears/log"
)

var (
	DefaultFuncs = map[string]ActionFunc{
		"contains": contains,
	}
)

type ActionFunc func(ctx context.Context, a *ActionParser, inputs ...interface{}) (interface{}, error)

// contains(search, item)
func contains(ctx context.Context, _ *ActionParser, inputs ...interface{}) (interface{}, error) {
	// TODO manage that arg 0 can be an array of string
	log.Debug(ctx, "function: contains with args: %v", inputs)
	if len(inputs) != 2 {
		return nil, NewErrorFrom(ErrInvalidData, "wrong number of arguments to call contains(search, item)")
	}
	inputSearch, ok := inputs[0].(string)
	if !ok {
		return nil, NewErrorFrom(ErrInvalidData, "search argument must be a string")
	}
	inputItem, ok := inputs[1].(string)
	if !ok {
		return nil, NewErrorFrom(ErrInvalidData, "item argument must be a string")
	}
	return strings.Contains(inputSearch, inputItem), nil
}
