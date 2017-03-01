/*
This code implements the flow chart that can be found here.
http://www.html5rocks.com/static/images/cors_server_flowchart.png

A Default Config for example is below:

	cors.Config{
		Origins:        "*",
		Methods:        "GET, PUT, POST, DELETE",
		RequestHeaders: "Origin, Authorization, Content-Type",
		ExposedHeaders: "",
		MaxAge: 1 * time.Minute,
		Credentials: true,
		ValidateHeaders: false,
	}
*/
package cors

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	AllowOriginKey      string = "Access-Control-Allow-Origin"
	AllowCredentialsKey        = "Access-Control-Allow-Credentials"
	AllowHeadersKey            = "Access-Control-Allow-Headers"
	AllowMethodsKey            = "Access-Control-Allow-Methods"
	MaxAgeKey                  = "Access-Control-Max-Age"

	OriginKey         = "Origin"
	RequestMethodKey  = "Access-Control-Request-Method"
	RequestHeadersKey = "Access-Control-Request-Headers"
	ExposeHeadersKey  = "Access-Control-Expose-Headers"
)

const (
	optionsMethod = "OPTIONS"
)

/*
Config defines the configuration options available to control how the CORS middleware should function.
*/
type Config struct {
	// Enabling this causes us to compare Request-Method and Request-Headers to confirm they contain a subset of the Allowed Methods and Allowed Headers
	// The spec however allows for the server to always match, and simply return the allowed methods and headers. Either is supported in this middleware.
	ValidateHeaders bool

	// Comma delimited list of origin domains. Wildcard "*" is also allowed, and matches all origins.
	// If the origin does not match an item in the list, then the request is denied.
	Origins string
	origins []string

	// This are the headers that the resource supports, and will accept in the request.
	// Default is "Authorization".
	RequestHeaders string
	requestHeaders []string

	// These are headers that should be accessable by the CORS client, they are in addition to those defined by the spec as "simple response headers"
	//	 Cache-Control
	//	 Content-Language
	//	 Content-Type
	//	 Expires
	//	 Last-Modified
	//	 Pragma
	ExposedHeaders string

	// Comma delimited list of acceptable HTTP methods.
	Methods string
	methods []string

	// The amount of time in seconds that the client should cache the Preflight request
	MaxAge time.Duration
	maxAge string

	// If true, then cookies and Authorization headers are allowed along with the request.  This
	// is passed to the browser, but is not enforced.
	Credentials bool
	credentials string
}

// One time, do the conversion from our the public facing Configuration,
// to all the formats we use internally strings for headers.. slices for looping
func (config *Config) prepare() {
	config.origins = strings.Split(config.Origins, ", ")
	config.methods = strings.Split(config.Methods, ", ")
	config.requestHeaders = strings.Split(config.RequestHeaders, ", ")
	config.maxAge = fmt.Sprintf("%.f", config.MaxAge.Seconds())

	// Generates a boolean of value "true".
	config.credentials = fmt.Sprintf("%t", config.Credentials)

	// Convert to lower-case once as request headers are supposed to be a case-insensitive match
	for idx, header := range config.requestHeaders {
		config.requestHeaders[idx] = strings.ToLower(header)
	}
}

/*
Middleware generates a middleware handler function that works inside of a Gin request
to set the correct CORS headers.  It accepts a cors.Options struct for configuration.
*/
func Middleware(config Config) gin.HandlerFunc {
	forceOriginMatch := false

	if config.Origins == "" {
		panic("You must set at least a single valid origin. If you don't want CORS, to apply, simply remove the middleware.")
	}

	if config.Origins == "*" {
		forceOriginMatch = true
	}

	config.prepare()

	// Create the Middleware function
	return func(context *gin.Context) {
		// Read the Origin header from the HTTP request
		currentOrigin := context.Request.Header.Get(OriginKey)
		context.Writer.Header().Add("Vary", OriginKey)

		// CORS headers are added whenever the browser request includes an "Origin" header
		// However, if no Origin is supplied, they should never be added.
		if currentOrigin == "" {
			return
		}

		originMatch := false
		if !forceOriginMatch {
			originMatch = matchOrigin(currentOrigin, config)
		}

		if forceOriginMatch || originMatch {
			valid := false
			preflight := false

			if context.Request.Method == optionsMethod {
				requestMethod := context.Request.Header.Get(RequestMethodKey)
				if requestMethod != "" {
					preflight = true
					valid = handlePreflight(context, config, requestMethod)
				}
			}

			if !preflight {
				valid = handleRequest(context, config)
			}

			if valid {

				if config.Credentials {
					context.Writer.Header().Set(AllowCredentialsKey, config.credentials)
					// Allowed origins cannot be the string "*" cannot be used for a resource that supports credentials.
					context.Writer.Header().Set(AllowOriginKey, currentOrigin)
				} else if forceOriginMatch {
					context.Writer.Header().Set(AllowOriginKey, "*")
				} else {
					context.Writer.Header().Set(AllowOriginKey, currentOrigin)
				}

				//If this is a preflight request, we are finished, quit.
				//Otherwise this is a normal request and operations should proceed at normal
				if preflight {
					context.AbortWithStatus(200)
				}
				return
			}
		}

		//If it reaches here, it was not a valid request
		context.Abort()
	}
}

func handlePreflight(context *gin.Context, config Config, requestMethod string) bool {
	if ok := validateRequestMethod(requestMethod, config); ok == false {
		return false
	}

	if ok := validateRequestHeaders(context.Request.Header.Get(RequestHeadersKey), config); ok == true {
		context.Writer.Header().Set(AllowMethodsKey, config.Methods)
		context.Writer.Header().Set(AllowHeadersKey, config.RequestHeaders)

		if config.maxAge != "0" {
			context.Writer.Header().Set(MaxAgeKey, config.maxAge)
		}

		return true
	}

	return false
}

func handleRequest(context *gin.Context, config Config) bool {
	if config.ExposedHeaders != "" {
		context.Writer.Header().Set(ExposeHeadersKey, config.ExposedHeaders)
	}

	return true
}

// Case-sensitive match of origin header
func matchOrigin(origin string, config Config) bool {
	for _, value := range config.origins {
		if value == origin {
			return true
		}
	}
	return false
}

// Case-sensitive match of request method
func validateRequestMethod(requestMethod string, config Config) bool {
	if !config.ValidateHeaders {
		return true
	}

	if requestMethod != "" {
		for _, value := range config.methods {
			if value == requestMethod {
				return true
			}
		}
	}

	return false
}

// Case-insensitive match of request headers
func validateRequestHeaders(requestHeaders string, config Config) bool {
	if !config.ValidateHeaders {
		return true
	}

	headers := strings.Split(requestHeaders, ",")

	for _, header := range headers {
		match := false
		header = strings.ToLower(strings.Trim(header, " \t\r\n"))

		for _, value := range config.requestHeaders {
			if value == header {
				match = true
				break
			}
		}

		if !match {
			return false
		}
	}

	return true
}
