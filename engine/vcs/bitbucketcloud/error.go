package bitbucketcloud

import (
	"fmt"

	"github.com/ovh/cds/sdk"
)

//Error wraps bitbucketcloud error format
type Error struct {
	Type        string       `json:"type"`
	ErrorStruct ErrorDetails `json:"error"`
}

type ErrorDetails struct {
	Details string `json:"details"`
	Message string `json:"message"`
}

func (e Error) Error() string {
	return fmt.Sprintf("(bitbucketcloud_%s) %s", e.Type, e.ErrorStruct.Message)
}

func (e Error) String() string {
	return e.Error()
}

var (
	ErrorUnauthorized = &Error{
		Type: "bad_credentials",
		ErrorStruct: ErrorDetails{
			Message: "Bad credentials",
		},
	}
)

//errorAPI creates a new error
func errorAPI(body []byte) error {
	var res Error
	_ = sdk.JSONUnmarshal(body, &res)
	return res
}
