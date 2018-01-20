package swag

import (
	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"
	"github.com/loopfz/gadgeto/tonic/utils/bootstrap"
	"github.com/loopfz/gadgeto/tonic/utils/swag/doc"
	"github.com/loopfz/gadgeto/tonic/utils/swag/swagger"
)

var (
	api *swagger.ApiDeclaration // singleton api declaration, generated once
)

func Swagger(e *gin.Engine, title string, options ...func(*swagger.ApiDeclaration)) gin.HandlerFunc {
	if api == nil {
		bootstrap.Bootstrap(e)

		// generate Api Declaration
		gen := NewSchemaGenerator()
		if err := gen.GenerateSwagDeclaration(tonic.GetRoutes(), "", "", &doc.Infos{}); err != nil {
			panic(err)
		}
		api = gen.apiDeclaration

		api.SwaggerApiInfo.Title = title
		for _, opt := range options {
			opt(api)
		}
	}
	return func(c *gin.Context) {
		c.JSON(200, api)
	}
}

func Version(version string) func(*swagger.ApiDeclaration) {
	return func(a *swagger.ApiDeclaration) {
		a.SwaggerApiInfo.Version = version
	}
}

func BasePath(path string) func(*swagger.ApiDeclaration) {
	return func(a *swagger.ApiDeclaration) {
		a.BasePath = path
	}
}

func Description(desc string) func(*swagger.ApiDeclaration) {
	return func(a *swagger.ApiDeclaration) {
		a.SwaggerApiInfo.Description = desc
	}
}
