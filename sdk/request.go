package sdk

import (
	"net/http"
)

var (

	// AuthHeader is used as HTTP header
	AuthHeader = "X_AUTH_HEADER"
	// RequestedWithHeader is used as HTTP header
	RequestedWithHeader = "X-Requested-With"
	// RequestedWithValue is used as HTTP header
	RequestedWithValue = "X-CDS-SDK"
	//SessionTokenHeader is user as HTTP header
	SessionTokenHeader = "Session-Token"

	// ResponseWorkflowNameHeader is used as HTTP header
	ResponseWorkflowNameHeader = "X-Api-Workflow-Name"
	// ResponseWorkflowIDHeader is used as HTTP header
	ResponseWorkflowIDHeader = "X-Api-Workflow-Id"
	// WorkflowAsCodeHeader is used as HTTP header
	WorkflowAsCodeHeader = "X-Api-Workflow-As-Code"

	// ResponseTemplateGroupNameHeader is used as HTTP header
	ResponseTemplateGroupNameHeader = "X-Api-Template-Group-Name"
	// ResponseTemplateSlugHeader is used as HTTP header
	ResponseTemplateSlugHeader = "X-Api-Template-Slug"
)

// Different values of agent
const (
	SDKAgent     = "CDS/sdk"
	WorkerAgent  = "CDS/worker"
	ServiceAgent = "CDS/service"
)

// RequestModifier is used to modify behavior of Request and Steam functions
type RequestModifier func(req *http.Request)

// HTTPClient is a interface for HTTPClient mock
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// SetHeader modify headers of http.Request
func SetHeader(key, value string) RequestModifier {
	return func(req *http.Request) {
		req.Header.Set(key, value)
	}
}
