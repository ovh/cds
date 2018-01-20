package tonic

import (
	"encoding"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/satori/go.uuid"
)

const (
	queryTag        = "query"
	pathTag         = "path"
	enumTag         = "enum"
	DefaultMaxBytes = 256 * 1024 // default to max 256ko body
)

// An ErrorHook lets you interpret errors returned by your handlers.
// After analysis, the hook should return a suitable http status code and
// error payload.
// This lets you deeply inspect custom error types.
//
// See sub-package 'jujuerrhook' for a ready-to-use implementation
// that relies on juju/errors (https://github.com/juju/errors).
type ErrorHook func(*gin.Context, error) (int, interface{})

// An ExecHook is the func called to handle a request.
//
// It is given as input:
//	- the gin context
//	- the wrapping gin-handler
//	- the function name of the tonic-handler
//
// The default ExecHook simply calle the wrapping gin-handler
// with the gin context.
type ExecHook func(*gin.Context, gin.HandlerFunc, string)

// A BindHook is the func called by the wrapping gin-handler when binding
// an incoming request to the tonic-handler's input object.
type BindHook func(*gin.Context, interface{}) error

// A RenderHook is the last func called by the wrapping gin-handler before returning.
//
// It is given as input:
//	- the gin context
//	- the HTTP status code
//	- the response payload
//
// Its role is to render the payload to the client.
type RenderHook func(*gin.Context, int, interface{})

var (
	errorHook  ErrorHook  = DefaultErrorHook
	execHook   ExecHook   = DefaultExecHook
	bindHook   BindHook   = DefaultBindingHook
	renderHook RenderHook = DefaultRenderHook

	// routes is a global map of routes handled with a tonic-enabled handler.
	// The map is made available through the GetRoutes helper.
	routes = make(map[string]*Route)
)

// DefaultExecHook is the default exec hook.
//
// It simply executes the wrapping gin-handler.
func DefaultExecHook(c *gin.Context, h gin.HandlerFunc, fname string) { h(c) }

// DefaultErrorHook is the default error hook.
//
// It returns a 400 HTTP status with a payload containing
// the error message.
func DefaultErrorHook(c *gin.Context, e error) (int, interface{}) {
	return 400, gin.H{`error`: e.Error()}
}

// DefaultBindingHook is the default binding hook.
//
// It uses gin to bind body parameters to input object.
// Returns an error if gin binding fails.
func DefaultBindingHook(c *gin.Context, i interface{}) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, DefaultMaxBytes)
	if c.Request.ContentLength == 0 || c.Request.Method == http.MethodGet {
		return nil
	}
	if err := c.ShouldBindWith(i, binding.JSON); err != nil && err != io.EOF {
		return fmt.Errorf("error parsing request body: %s", err.Error())
	}
	return nil
}

// DefaultRenderHook is the default render hook.
//
// It serializes the payload to JSON, or returns an empty body is payload is nil.
// If gin is running in debug mode, the serialized JSON is indented.
func DefaultRenderHook(c *gin.Context, status int, payload interface{}) {
	// Either serialize custom output object or send empty body
	if payload != nil {
		if gin.IsDebugging() {
			c.IndentedJSON(status, payload)
		} else {
			c.JSON(status, payload)
		}
	} else {
		c.String(status, "")
	}
}

// GetRoutes returns the routes handled by a tonic-enabled handler.
// TODO: maybe remove this func and export tonic.Routes var ?
func GetRoutes() map[string]*Route {
	return routes
}

// GetErrorHook returns the current error hook.
func GetErrorHook() ErrorHook {
	return errorHook
}

// SetErrorHook lets you set your own error hook.
func SetErrorHook(eh ErrorHook) {
	if eh != nil {
		errorHook = eh
	}
}

// GetExecHook returns the current exec hook.
func GetExecHook() ExecHook {
	return execHook
}

// SetExecHook lets you set your own exec hook.
func SetExecHook(eh ExecHook) {
	if eh != nil {
		execHook = eh
	}
}

// GetBindHook returns the current bind hook.
func GetBindHook() BindHook {
	return bindHook
}

// SetBindHook lets you set your own bind hook.
func SetBindHook(bh BindHook) {
	if bh != nil {
		bindHook = bh
	}
}

// GetRenderHook returns the current render hook.
func GetRenderHook() RenderHook {
	return renderHook
}

// SetRenderHook lets you set your own render hook.
func SetRenderHook(rh RenderHook) {
	if rh != nil {
		renderHook = rh
	}
}

// InputError is an error type returned when tonic fails to bind parameters.
type InputError string

// Error makes InputError implement the error interface.
func (ie InputError) Error() string {
	return string(ie)
}

// Handler returns a wrapping gin-compatible handler that calls the tonic-handler
// passed in parameter.
//
// The tonic-handler may use the following signature:
//
//  func(*gin.Context, [input object ptr]) ([output object], error)
//
// Input and output objects are both optional (tonic analyzes the handler signature reflexively).
// As such, the minimal accepted signature is:
//
//  func(*gin.Context) error
//
// The wrapping gin-handler will handle the binding code (JSON + path/query)
// and the error handling.
//
// Handler will panic if the tonic-handler is of incompatible type.
func Handler(f interface{}, retcode int, options ...func(*Route)) gin.HandlerFunc {

	fval := reflect.ValueOf(f)
	if fval.Kind() != reflect.Func {
		panic(fmt.Sprintf("Handler parameter must be a function, got %T", f))
	}

	ftype := fval.Type()
	fname := fmt.Sprintf("%s_%s", runtime.FuncForPC(fval.Pointer()).Name(), uuid.Must(uuid.NewV4()).String())

	var typeIn reflect.Type
	var typeOut reflect.Type

	// Check tonic-handler inputs
	numIn := ftype.NumIn()
	if numIn < 1 || numIn > 2 {
		panic(fmt.Sprintf("Incorrect number of handler '%s' input params: expected 1 or 2, got %d", fname, numIn))
	}
	hasIn := (numIn == 2)
	if !ftype.In(0).ConvertibleTo(reflect.TypeOf(&gin.Context{})) {
		panic(fmt.Sprintf("Unsupported type for handler '%s' input parameter: expected *gin.Context, got %v", fname, ftype.In(0)))
	}
	if hasIn {
		if ftype.In(1).Kind() != reflect.Ptr || ftype.In(1).Elem().Kind() != reflect.Struct {
			panic(fmt.Sprintf("Unsupported type for handler '%s' input parameter: expected struct ptr, got %v", fname, ftype.In(1)))
		} else {
			typeIn = ftype.In(1).Elem()
		}
	}

	// Check tonic handler outputs
	numOut := ftype.NumOut()
	if numOut < 1 || numOut > 2 {
		panic(fmt.Sprintf("Incorrect number of handler '%s' output params: expected 1 or 2, got %d", fname, numOut))
	}
	hasOut := (numOut == 2)
	errIdx := 0
	if hasOut {
		errIdx++
		// Output type can be lots of things, we should let it as it is
		typeOut = ftype.Out(0)
		switch ftype.Out(0).Kind() {
		case reflect.Ptr:
			// According to reflect.Type.Elem() doc:
			// It panics if the type's Kind is not Array, Chan, Map, Ptr, or Slice.
			typeOut = ftype.Out(0).Elem()
		default:
			typeOut = ftype.Out(0)
		}
	}
	typeOfError := reflect.TypeOf((*error)(nil)).Elem()
	if !ftype.Out(errIdx).Implements(typeOfError) {
		panic(fmt.Sprintf("Unsupported type for handler '%s' output parameter: expected error implementation, got %v", fname, ftype.Out(errIdx)))
	}

	// Wrapping gin-handler
	retfunc := func(c *gin.Context) {

		// funcIn contains the input parameters of the tonic-handler call
		funcIn := []reflect.Value{reflect.ValueOf(c)}

		if hasIn {
			// tonic-handler has custom input object, handle binding
			input := reflect.New(typeIn)
			err := bindHook(c, input.Interface())
			if err != nil {
				handleError(c, InputError(err.Error()))
				return
			}
			err = bindQueryPath(c, input, "query", extractQuery)
			if err != nil {
				handleError(c, InputError(err.Error()))
				return
			}
			err = bindQueryPath(c, input, "path", extractPath)
			if err != nil {
				handleError(c, InputError(err.Error()))
				return
			}
			funcIn = append(funcIn, input)
		}

		// Call tonic-handler
		ret := fval.Call(funcIn)
		var errOut interface{}
		var outVal interface{}
		if hasOut {
			outVal = ret[0].Interface()
			errOut = ret[1].Interface()
		} else {
			errOut = ret[0].Interface()
		}
		// Raised error, handle it
		if errOut != nil {
			handleError(c, errOut.(error))
			return
		}
		// Normal output
		renderHook(c, retcode, outVal)
	}

	// Register route in tonic-enabled routes map
	route := &Route{
		defaultStatusCode: retcode,
		handler:           fval,
		handlerType:       ftype,
		inputType:         typeIn,
		outputType:        typeOut,
	}
	for _, opt := range options {
		opt(route)
	}
	routes[fname] = route

	return func(c *gin.Context) { execHook(c, retfunc, fname) }
}

// Description set the description of a route.
func Description(s string) func(*Route) {
	return func(r *Route) {
		r.description = s
	}
}

// Description set the summary of a route.
func Summary(s string) func(*Route) {
	return func(r *Route) {
		r.summary = s
	}
}

// Deprecated set the deprecated flag of a route.
func Deprecated(b bool) func(*Route) {
	return func(r *Route) {
		r.deprecated = b
	}
}

// handleError handles any error raised during the execution of the wrapping gin-handler.
func handleError(c *gin.Context, err error) {
	if len(c.Errors) == 0 {
		// Push error into gin context
		c.Error(err)
	}
	errcode, errpl := errorHook(c, err)
	renderHook(c, errcode, errpl)
}

// An extractorFunc extracts data from a gin context according to
// parameters specified in a field tag.
//
// An extractorFunc takes a gin context and a tag value as input.
//
// It returns:
//	- the parameter name
//	- the parameter values (there may be several)
//	- an error
type extractorFunc func(*gin.Context, string) (string, []string, error)

// bindQueryPath binds fields of an input object to query and path parameters from the gin context.
// It reads targetTag to know, for each field, what to extract using the given extractor func.
func bindQueryPath(c *gin.Context, in reflect.Value, targetTag string, extractor func(*gin.Context, string) (string, []string, error)) error {

	inType := in.Type().Elem()

	for i := 0; i < in.Elem().NumField(); i++ {
		// Extract value from gin context
		fieldType := inType.Field(i)

		if fieldType.Anonymous {
			inField := in.Elem().Field(i)
			if inField.Kind() == reflect.Ptr {
				if inField.IsNil() {
					inField.Set(reflect.New(inField.Type().Elem()))
				}
			} else {
				inField = inField.Addr()
			}
			err := bindQueryPath(c, inField, targetTag, extractor)
			if err != nil {
				return err
			}
			continue
		}

		tag := fieldType.Tag.Get(targetTag)
		if tag == "" {
			continue
		}
		name, values, err := extractor(c, tag)
		if err != nil {
			return err
		}
		if len(values) == 0 {
			continue
		}

		// Fill value into input object
		field := in.Elem().Field(i)
		if field.Kind() == reflect.Ptr {
			f := reflect.New(field.Type().Elem())
			field.Set(f)
			field = field.Elem()
		}
		if field.Kind() == reflect.Slice {
			for _, v := range values {
				newV := reflect.New(field.Type().Elem()).Elem()
				err := bindValue(v, newV)
				if err != nil {
					return err
				}
				field.Set(reflect.Append(field, newV))
			}
			return nil
		} else if len(values) > 1 {
			return fmt.Errorf("parameter '%s' does not support multiple values", name)
		} else {
			enum := fieldType.Tag.Get(enumTag)
			if enum != "" {
				enumValues := strings.Split(strings.TrimSpace(enum), ",")
				if len(enumValues) != 0 {
					if !sliceContains(enumValues, values[0]) {
						return fmt.Errorf("parameter '%s' has not an acceptable value, enum=%v", name, enumValues)
					}
				}
			}
			err = bindValue(values[0], field)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func sliceContains(in []string, s string) bool {
	for _, v := range in {
		if v == s {
			return true
		}
	}
	return false
}

// extractQuery is an extractorFunc that extracts a query parameter.
//
// It reads the parameter name and whether it is required from the tag,
// and returns the name, values and an error.
func extractQuery(c *gin.Context, tag string) (string, []string, error) {

	name, required, defVal, err := ExtractTag(tag, true)
	if err != nil {
		return "", nil, err
	}

	q := c.Request.URL.Query()[name]

	if defVal != "" && len(q) == 0 {
		q = []string{defVal}
	}

	if required && len(q) == 0 {
		return "", nil, fmt.Errorf("missing required field: %s", name)
	}

	return name, q, nil
}

// extractPath is an extractorFunc that extracts a path parameter.
//
// It reads the parameter name and whether it is required from the tag,
// and returns the name, values and an error.
func extractPath(c *gin.Context, tag string) (string, []string, error) {

	name, required, _, err := ExtractTag(tag, false)
	if err != nil {
		return "", nil, err
	}

	out := c.Param(name)
	if required && out == "" {
		return "", nil, fmt.Errorf("field %s is missing: required", name)
	}
	return name, []string{out}, nil
}

// ExtractTag extracts information from the given tag.
//
// Informations returned are:
//	- string: parameter name
//	- bool:   whether the parameter is required or not
//	- string: parameter default value, if any
//	- error:  any error (invalid tag option for example)
func ExtractTag(tag string, defaultValue bool) (string, bool, string, error) {

	parts := strings.Split(tag, ",")
	name, options := parts[0], parts[1:]

	var defVal string
	var required bool
	for _, o := range options {
		o = strings.TrimSpace(o)
		if o == "required" {
			required = true
		} else if defaultValue && strings.HasPrefix(o, "default=") {
			o = strings.TrimPrefix(o, "default=")
			defVal = o
		} else {
			return "", false, "", fmt.Errorf("malformed tag for param '%s': unknown option '%s'", name, o)
		}
	}
	return name, required, defVal, nil
}

// bindValue binds a string value to an actual reflect.Value.
func bindValue(s string, v reflect.Value) error {

	vIntf := reflect.New(v.Type()).Interface()
	unmarshaler, ok := vIntf.(encoding.TextUnmarshaler)
	if ok {
		err := unmarshaler.UnmarshalText([]byte(s))
		if err != nil {
			return err
		}
		v.Set(reflect.Indirect(reflect.ValueOf(unmarshaler)))
		return nil
	}

	switch v.Kind() {
	case reflect.String:
		v.SetString(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(s, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, err := strconv.ParseUint(s, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetUint(i)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		v.SetBool(b)
	default:
		return fmt.Errorf("unsupported type for param bind: %v", v.Kind())
	}

	return nil
}
