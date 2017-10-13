package github

import (
	"encoding/json"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// RateLimit Get your current rate limit status
// https://developer.github.com/v3/rate_limit/#get-your-current-rate-limit-status
func (g *githubClient) RateLimit() error {
	url := "/rate_limit"
	status, body, _, err := g.get(url)
	if err != nil {
		log.Warning("githubClient.RateLimit> Error %s", err)
		return err
	}
	if status >= 400 {
		return sdk.NewError(sdk.ErrUnknownError, errorAPI(body))
	}
	rateLimit := &RateLimit{}
	if err := json.Unmarshal(body, rateLimit); err != nil {
		log.Warning("githubClient.RateLimit> Error %s", err)
		return err
	}
	if rateLimit.Rate.Remaining < 100 {
		log.Error("Github Rate Limit nearly exceeded %v", rateLimit)
		return ErrorRateLimit
	}
	return nil
}
