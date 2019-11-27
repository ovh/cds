package github

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func isRateLimitReached() bool {
	if RateLimitReset > 0 && RateLimitReset < int(time.Now().Unix()) {
		log.Debug("RateLimitReset reached, it's ok to call github")
		return false
	}
	return RateLimitRemaining < 100
}

// RateLimit Get your current rate limit status
// https://developer.github.com/v3/rate_limit/#get-your-current-rate-limit-status
func (g *githubClient) RateLimit(ctx context.Context) error {
	url := "/rate_limit"
	status, body, _, err := g.get(ctx, url)
	if err != nil {
		log.Warning(ctx, "githubClient.RateLimit> Error %s", err)
		return err
	}
	// If the GitHub instance does not have Rate Limitting enabled you will see a 404.
	if status == http.StatusNotFound && strings.Contains(string(body), "Rate limiting is not enabled.") {
		return nil
	}
	if status >= 400 {
		return sdk.NewError(sdk.ErrUnknownError, errorAPI(body))
	}
	rateLimit := &RateLimit{}
	if err := json.Unmarshal(body, rateLimit); err != nil {
		log.Warning(ctx, "githubClient.RateLimit> Error %s", err)
		return err
	}
	if rateLimit.Rate.Remaining < 100 {
		log.Error(ctx, "Github Rate Limit nearly exceeded %v", rateLimit)
		return ErrorRateLimit
	}
	return nil
}
