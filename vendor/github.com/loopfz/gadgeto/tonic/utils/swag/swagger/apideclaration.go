package swagger

import (
	"encoding/json"
)

type SwaggerApiInfo struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

// ApiDeclaration is https://github.com/swagger-api/swagger-spec/blob/master/versions/1.2.md#52-api-declaration
type ApiDeclaration struct {
	SwaggerApiInfo SwaggerApiInfo                  `json:"info"`
	Swagger        string                          `json:"swagger"`
	BasePath       string                          `json:"basePath,omitempty"`
	Paths          map[string]map[string]Operation `json:"paths"`
	Definitions    map[string]Model                `json:"definitions"`
}

// NewApiDeclaration returns a bootstrapedApiDeclaration
func NewApiDeclaration(version string, basePath string) *ApiDeclaration {

	decl := &ApiDeclaration{}

	apiInfo := SwaggerApiInfo{}
	apiInfo.Version = version
	apiInfo.Title = "Swagger API"
	decl.SwaggerApiInfo = apiInfo

	decl.Swagger = swaggerVersion
	decl.BasePath = basePath
	decl.Definitions = map[string]Model{}
	decl.Paths = map[string]map[string]Operation{}

	return decl
}

// AddModel does just what it says
func (decl *ApiDeclaration) AddModel(m Model) {
	decl.Definitions[m.Id] = m
}

//ToJSON is a shortcut to json.MarshalIndent
func (decl *ApiDeclaration) ToJSON() string {
	b, _ := json.MarshalIndent(decl, "", "    ")
	return string(b)
}

//GetSDKPaths filters out non relevant paths/operations for the SDKS
// at the moment just skips Operation marked as IsMonitoring
func (decl *ApiDeclaration) GetSDKPaths() map[string]map[string]Operation {
	sdksPaths := make(map[string]map[string]Operation)

	for route, ops := range decl.Paths {
		sdkOps := make(map[string]Operation)
		for method, op := range ops {
			if op.IsMonitoring {
				continue
			}
			sdkOps[method] = op
		}
		sdksPaths[route] = sdkOps
	}
	return sdksPaths
}
