package defaultctx

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ovh/venom"
)

// Name is Context Type name.
const Name = "default"

// New returns a new TestCaseContext.
func New() venom.TestCaseContext {
	ctx := &DefaultTestCaseContext{}
	ctx.Name = Name
	return ctx
}

// DefaultTestCaseContext represents the context of a testcase.
type DefaultTestCaseContext struct {
	venom.CommonTestCaseContext
	datas map[string]interface{}
}

// Init Initialize the context.
func (tcc *DefaultTestCaseContext) Init() error {
	tcc.datas = tcc.TestCase.Context
	return nil
}

// Close the context.
func (tcc *DefaultTestCaseContext) Close() error {
	return nil
}

// GetString returns string from default context.
func (tcc *DefaultTestCaseContext) GetString(key string) (string, error) {
	if tcc.datas[key] == nil {
		return "", NotFound(key)
	}
	result, ok := tcc.datas[key].(string)
	if !ok {
		return "", InvalidArgument(key)
	}
	return result, nil
}

// GetFloat returns float64 from default context.
func (tcc *DefaultTestCaseContext) GetFloat(key string) (float64, error) {
	if tcc.datas[key] == nil {
		return 0, NotFound(key)
	}
	result, ok := tcc.datas[key].(float64)
	if !ok {
		return 0, InvalidArgument(key)
	}
	return result, nil
}

// GetInt returns int from default context.
func (tcc *DefaultTestCaseContext) GetInt(key string) (int, error) {
	res, err := tcc.GetFloat(key)
	if err != nil {
		return 0, err
	}

	return int(res), nil
}

// GetBool returns bool from default context.
func (tcc *DefaultTestCaseContext) GetBool(key string) (bool, error) {
	if tcc.datas[key] == nil {
		return false, NotFound(key)
	}
	result, ok := tcc.datas[key].(bool)
	if !ok {
		return false, InvalidArgument(key)
	}
	return result, nil
}

// GetStringSlice returns string slice from default context.
func (tcc *DefaultTestCaseContext) GetStringSlice(key string) ([]string, error) {
	if tcc.datas[key] == nil {
		return nil, NotFound(key)
	}

	stringSlice, ok := tcc.datas[key].([]string)
	if ok {
		return stringSlice, nil
	}

	slice, ok := tcc.datas[key].([]interface{})
	if !ok {
		return nil, InvalidArgument(key)
	}

	res := make([]string, len(slice))

	for k, v := range slice {
		s, ok := v.(string)
		if !ok {
			return nil, errors.New("cannot cast to string")
		}

		res[k] = s
	}

	return res, nil
}

// GetComplex unmarshal argument in struct from default context.
func (tcc *DefaultTestCaseContext) GetComplex(key string, arg interface{}) error {
	if tcc.datas[key] == nil {
		return NotFound(key)
	}

	val, err := json.Marshal(tcc.datas[key])
	if err != nil {
		return err
	}

	err = json.Unmarshal(val, arg)
	if err != nil {
		return err
	}
	return nil
}

// NotFound is error returned when trying to get missing argument
func NotFound(key string) error { return fmt.Errorf("missing context argument '%s'", key) }

// InvalidArgument is error returned when trying to cast argument with wrong type
func InvalidArgument(key string) error { return fmt.Errorf("invalid context argument type '%s'", key) }
