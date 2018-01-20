package tonic

import (
	"reflect"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
)

// A Route contains information about a tonic-enabled route.
type Route struct {
	gin.RouteInfo

	// defaultStatusCode is the HTTP status code returned when everything goes well.
	defaultStatusCode int

	description string
	summary     string

	// handler is the tonic handler.
	handler reflect.Value
	// handlerType is the type of the tonic handler.
	handlerType reflect.Type
	// inputType is the type of the input object, if any.
	// Can be nil.
	inputType reflect.Type
	// outputType is the type of the output object, if any.
	// Can be nil.
	outputType reflect.Type

	deprecated bool
}

// GetVerb returns the HTTP verb of the route.
func (r *Route) GetVerb() string {
	return r.Method
}

// GetPath returns the path of the route.
func (r *Route) GetPath() string {
	return r.Path
}

// GetDescription returns the description of the route.
func (r *Route) GetDescription() string {
	return r.description
}

// GetSummary returns the summary of the route.
func (r *Route) GetSummary() string {
	return r.summary
}

// GetDefaultStatusCode returns the default status code of the route.
func (r *Route) GetDefaultStatusCode() int {
	return r.defaultStatusCode
}

// GetHandler returns the handler of the route.
func (r *Route) GetHandler() reflect.Value {
	return r.handler
}

// GetDeprecated returns the deprecated flag of the route.
func (r *Route) GetDeprecated() bool {
	return r.deprecated
}

// GetInType returns the input type of the route.
// reflect.Ptr types are dereferenced.
func (r *Route) GetInType() reflect.Type {
	if in := r.inputType; in != nil && in.Kind() == reflect.Ptr {
		return in.Elem()
	}
	return r.inputType
}

// GetOutType returns the output type of the route.
// reflect.Ptr types are dereferenced.
func (r *Route) GetOutType() reflect.Type {
	if out := r.outputType; out != nil && out.Kind() == reflect.Ptr { // should always be true
		return out.Elem()
	}
	return r.outputType
}

// GetHandlerName returns the name of the handler func.
func (r *Route) GetHandlerName() string {
	p := strings.Split(r.GetHandlerNameWithPackage(), ".")
	return p[len(p)-1]
}

// GetHandlerNameWithPackage returns the name of the handler func,
// with its package path.
func (r *Route) GetHandlerNameWithPackage() string {
	f := runtime.FuncForPC(r.handler.Pointer()).Name()
	p := strings.Split(f, "/")
	return p[len(p)-1]
}

// GetTags generates a list of tags for the swagger spec
// from one route definition.
// Currently it only takes the first path of the route as the tag.
func (r *Route) GetTags() []string {
	tags := make([]string, 0, 1)
	paths := strings.SplitN(r.GetPath(), "/", 3)
	if len(paths) > 1 {
		tags = append(tags, paths[1])
	}
	return tags
}
