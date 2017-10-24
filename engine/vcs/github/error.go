package github

import (
	"encoding/json"
	"fmt"
)

//Error wraps github error format
type ghError struct {
	ID   string `json:"error"`
	Desc string `json:"error_description"`
	URI  string `json:"error_uri"`
}

func (e ghError) Error() string {
	return fmt.Sprintf("(gh_%s) %s", e.ID, e.Desc)
}

func (e ghError) String() string {
	return e.Error()
}

//Github errors
var (
	ErrorRateLimit = &ghError{
		ID:   "rate_limit",
		Desc: "Rate Limit reached",
		URI:  "https://developer.github.com/v3/#rate-limiting",
	}

	ErrorUnauthorized = &ghError{
		ID:   "bad_credentials",
		Desc: "Bad credentials",
		URI:  "https://developer.github.com/v3",
	}
)

//errorAPI creates a new error
func errorAPI(body []byte) error {
	res := map[string]string{}
	json.Unmarshal(body, &res)
	return ghError{
		ID:   "api_error",
		Desc: res["message"],
		URI:  res["documentation_url"],
	}
}
