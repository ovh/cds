package swagger

const swaggerVersion = "2.0"

// ResourceListing is https://github.com/swagger-api/swagger-spec/blob/master/versions/1.2.md#51-resource-listing
type ResourceListing struct {
	ApiVersion     string `json:"apiVersion"`
	SwaggerVersion string `json:"swaggerVersion"` // e.g 1.2
}

//Swagger Path https://github.com/swagger-api/swagger-spec/blob/master/versions/2.0.md#pathItemObject
type Path struct {
	Path       string `json:"-"`
	Operations map[string]Operation
}

//Api is https://github.com/swagger-api/swagger-spec/blob/master/versions/1.2.md#522-api-object
type Api struct {
	Operations []Operation      `json:"operations,omitempty"`
	Models     map[string]Model `json:"models,omitempty"`
}

//ResponseMessage is https://github.com/swagger-api/swagger-spec/blob/master/versions/1.2.md#525-response-message-object
type ResponseMessage struct {
	Code          int    `json:"code"`
	Message       string `json:"message"`
	ResponseModel string `json:"responseModel"`
}

// Authorization is https://github.com/wordnik/swagger-core/wiki/authorizations
type Authorization struct {
	LocalOAuth OAuth  `json:"local-oauth"`
	ApiKey     ApiKey `json:"apiKey"`
}

//OAuth is  https://github.com/wordnik/swagger-core/wiki/authorizations
type OAuth struct {
	Type       string               `json:"type"`   // e.g. oauth2
	Scopes     []string             `json:"scopes"` // e.g. PUBLIC
	GrantTypes map[string]GrantType `json:"grantTypes"`
}

//GrantType is https://github.com/wordnik/swagger-core/wiki/authorizations
type GrantType struct {
	LoginEndpoint        Endpoint `json:"loginEndpoint"`
	TokenName            string   `json:"tokenName"` // e.g. access_code
	TokenRequestEndpoint Endpoint `json:"tokenRequestEndpoint"`
	TokenEndpoint        Endpoint `json:"tokenEndpoint"`
}

//Endpoint is  https://github.com/wordnik/swagger-core/wiki/authorizations
type Endpoint struct {
	Url              string `json:"url"`
	ClientIdName     string `json:"clientIdName"`
	ClientSecretName string `json:"clientSecretName"`
	TokenName        string `json:"tokenName"`
}

//ApiKey is https://github.com/wordnik/swagger-core/wiki/authorizations
type ApiKey struct {
	Type   string `json:"type"`   // e.g. apiKey
	PassAs string `json:"passAs"` // e.g. header
}
