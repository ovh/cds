package luascript

import (
	"context"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

// Check is a type which helps to call a lua script with variables to check something.
// The lua script must return true/false
type Check struct {
	state                    *lua.LState
	exceptionHandlerFunction *lua.LFunction
	variables                map[string]string
	IsError                  bool
	Result                   bool
	ctx                      context.Context
}

// NewCheck instanciates a check
func NewCheck() (*Check, error) {
	state := lua.NewState(
		lua.Options{
			SkipOpenLibs:  true,
			CallStackSize: 120,
			RegistrySize:  120 * 20,
		})

	// Sandboxing lua engine
	if err := state.DoString("coroutine=nil;debug=nil;io=nil;open=nil;os=nil"); err != nil {
		return nil, err
	}

	c := &Check{
		state: state,
	}
	c.exceptionHandlerFunction = state.NewFunction(c.exceptionHandler)
	return c, nil
}

func (c *Check) exceptionHandler(L *lua.LState) int {
	c.IsError = true
	return 0
}

func (c *Check) SetVariables(vars map[string]string) {
	c.variables = vars
	for k, v := range vars {
		k = strings.Replace(k, ".", "_", -1)
		c.state.SetGlobal(k, lua.LString(v))
	}
}

//Perform the lua script
func (c *Check) Perform(script string) error {
	var ok bool

	if err := c.state.DoString(script); err != nil {
		c.IsError = true
		return err
	}

	lv := c.state.Get(-1) // get the value at the top of the stack
	if lua.LVAsBool(lv) { // lv is neither nil nor false
		ok = true
	}

	c.IsError = false
	c.Result = ok
	return nil
}
