# CORS for Gin [![GoDoc](https://godoc.org/github.com/itsjamie/gin-cors?status.svg)](https://godoc.org/github.com/itsjamie/gin-cors) [![Build Status](https://travis-ci.org/itsjamie/gin-cors.svg?branch=master)](https://travis-ci.org/itsjamie/gin-cors) [![Coverage Status](https://coveralls.io/repos/itsjamie/gin-cors/badge.svg?branch=master)](https://coveralls.io/r/itsjamie/gin-cors?branch=master)

gin-cors is a middleware written in [Go (Golang)](http://golang.org) specifically for the [Gin Framework](https://gin-gonic.github.io/gin/) that implements the [Cross Origin Resource Sharing specification](http://www.w3.org/TR/cors/) from the W3C.  Implementing CORS headers enable pages within a modern web browser to consume resources (such as REST APIs) from servers that are on a different domain.

## Getting Started
To use this library, add the following code into your Gin router setup:

```go
import "github.com/itsjamie/gin-cors"

// Initialize a new Gin router
router := gin.New()

// Apply the middleware to the router (works with groups too)
router.Use(cors.Middleware(cors.Config{
	Origins:        "*",
	Methods:        "GET, PUT, POST, DELETE",
	RequestHeaders: "Origin, Authorization, Content-Type",
	ExposedHeaders: "",
	MaxAge: 50 * time.Second,
	Credentials: true,
	ValidateHeaders: false,
}))
```

## Setup Options
The middleware can be configured with four options, which match the HTTP headers that it generates:

Parameter          | Type            | Details
-------------------|-----------------|----------------------------------
Origins            | *string*        | A comma delimited list of origins which have access. For example: ```"http://localhost, http://api.server.com, http://files.server.com"```
RequestHeaders     | *string*        | A comma delimited list of allowed HTTP  that is passed to the browser in the **Access-Control-Allow-Headers** header.
ExposeHeaders      | *string*        | A comma delimited list of HTTP headers that should be exposed to the CORS client via the **Access-Control-Expose-Headers** header.
Methods            | *string*        | A comma delimited list of allowed HTTP methods that is passed to the browser in the **Access-Control-Allow-Methods**.
MaxAge             | *time.Duration* | The amount of time a preflight request should be cached, if not specified, the header **Access-Control-Max-Age** will not be set.
Credentials        | *bool*          | This is passed in the **Access-Control-Allow-Credentials** header. If ```true``` Cookies, HTTP authentication and the client-side SSL certificates will be sent on previous interactions with the origin.
ValidateHeaders    | *bool*          | If ```false``` we skip validating the requested headers/methods with the list of allowed ones, and instead just respond with all what we support, it is up to the client implementating CORS to deny the request. This is an optimization allowed by the specification. 


## CORS Resources

* [HTML Rocks Tutorial: Using CORS](http://www.html5rocks.com/en/tutorials/cors/)
* [Mozilla Developer Network: CORS Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Access_control_CORS)
* [CORS Specification from W3C](http://www.w3.org/TR/cors/)

## Special Thanks
Special thanks to [benpate](https://github.com/benpate) for providing a foundation to work from.

## License
The code is licensed under the MIT License. See LICENSE file for more details.
