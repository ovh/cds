package github

import (
	"encoding/json"
	"fmt"
)

//Error wraps github error format
type Error struct {
	ID   string `json:"error"`
	Desc string `json:"error_description"`
	URI  string `json:"error_uri"`
}

func (e Error) Error() string {
	return fmt.Sprintf("(gh_%s) %s", e.ID, e.Desc)
}

func (e Error) String() string {
	return e.Error()
}

//Github errors
var (
	ErrorRateLimit = &Error{
		ID:   "rate_limit",
		Desc: "Rate Limit reached",
		URI:  "https://developer.github.com/v3/#rate-limiting",
	}

	ErrorUnauthorized = &Error{
		ID:   "bad_credentials",
		Desc: "Bad credentials",
		URI:  "https://developer.github.com/v3",
	}
)

//ErrorAPI creates a new error
func ErrorAPI(body []byte) Error {
	res := map[string]string{}
	json.Unmarshal(body, &res)
	return Error{
		ID:   "api_error",
		Desc: res["message"],
		URI:  res["documentation_url"],
	}
}
