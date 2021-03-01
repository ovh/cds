package luascript

import (
	"context"
	"strings"

	"github.com/yuin/gluare"
	lua "github.com/yuin/gopher-lua"
)

// Check is a type which helps to call a lua script with variables to check something.
// The lua script must return true/false
// re.find , re.gsub, re.match, re.gmatch are available. These functions have the same API as Lua pattern match. gluare uses the Go regexp package, so you can use regular expressions that are supported in the Go regexp package.

type Check struct {
	state                    *lua.LState
	exceptionHandlerFunction *lua.LFunction
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

	// Opening a subset of builtin modules
	for _, pair := range []struct {
		n string
		f lua.LGFunction
	}{
		{lua.LoadLibName, lua.OpenPackage}, // Must be first
		{lua.BaseLibName, lua.OpenBase},
		{lua.TabLibName, lua.OpenTable},
		{lua.StringLibName, lua.OpenString},
		{lua.OsLibName, lua.OpenOs},
	} {
		if err := state.CallByParam(lua.P{
			Fn:      state.NewFunction(pair.f),
			NRet:    0,
			Protect: true,
		}, lua.LString(pair.n)); err != nil {
			return nil, err
		}
	}

	//Open gluare module
	state.PreloadModule("re", gluare.Loader)

	// Sandboxing lua engine
	if err := state.DoString("coroutine=nil;debug=nil;io=nil;open=nil;os.rename=nil;os.remove=nil;os.exit=nil;os.clock=nil;os.execute=nil;os.getenv=nil;os.setlocale=nil;os.tmpname=nil"); err != nil {
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
	for k, v := range vars {
		k = strings.Replace(k, ".", "_", -1)
		k = strings.Replace(k, "-", "_", -1)
		c.state.SetGlobal(k, lua.LString(v))
	}
}

func (c *Check) SetFloatVariables(vars map[string]float64) {
	for k, v := range vars {
		k = strings.Replace(k, ".", "_", -1)
		k = strings.Replace(k, "-", "_", -1)
		c.state.SetGlobal(k, lua.LNumber(v))
	}
}

func (c *Check) EnableStrict() error {
	// This code will override __index lua func that is called when reading a variable.
	// If the variable is not define we want to return an error.
	code := `
    local mt = getmetatable(_G)
    if mt == nil then
      mt = {}
      setmetatable(_G, mt)
    end
    mt.__index = function (t, n)
      if n ~= "C" then
        error("variable '"..n.."' is not declared", 2)
      end
      return rawget(t, n)
    end
  `
	if err := c.state.DoString(code); err != nil {
		return err
	}
	return nil
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
