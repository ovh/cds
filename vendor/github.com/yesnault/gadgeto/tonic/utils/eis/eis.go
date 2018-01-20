package eis

import "github.com/gin-gonic/gin"

var (
	enabler = func() {}
)

// BEWARE: THIS IS _NOT_ CONCURRENT SAFE
// Make sure your calls to Freeze() and Melt() happen in the same goroutine

// eis lets you setup gin middlewares
// and activate them later
// this is useful for weird init routines (e.g. swagger generation in tonic)

func Freeze(h gin.HandlerFunc) gin.HandlerFunc {
	f := noopMiddleware
	currentEnabler := enabler
	enabler = func() { currentEnabler(); f = h }
	return func(c *gin.Context) { f(c) }
}

func Melt() {
	enabler()
}

func noopMiddleware(c *gin.Context) {
	c.Next()
}
