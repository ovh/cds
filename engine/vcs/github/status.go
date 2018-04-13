package github

import (
	"fmt"
	"time"

	"github.com/ovh/cds/sdk"
)

// GetStatus returns github status
func GetStatus() []sdk.MonitoringStatusLine {
	var statusRemaining string
	switch {
	case RateLimitRemaining < 100:
		statusRemaining = sdk.MonitoringStatusAlert
	case RateLimitRemaining < 1000:
		statusRemaining = sdk.MonitoringStatusWarn
	default:
		statusRemaining = sdk.MonitoringStatusOK
	}
	a := sdk.MonitoringStatusLine{Component: "Github-RateLimitRemaining", Value: fmt.Sprintf("%d", RateLimitRemaining), Status: statusRemaining}

	var statusReset, resetTime string
	if RateLimitReset <= 0 {
		statusReset = sdk.MonitoringStatusAlert
	} else {
		tm := time.Unix(int64(RateLimitReset), 0)
		resetTime = fmt.Sprintf("%dh%dm%ds", tm.Hour(), tm.Minute(), tm.Second())
	}
	b := sdk.MonitoringStatusLine{Component: "Github-RateLimitReset", Value: resetTime, Status: statusReset}

	var statusRateLimit string
	if RateLimitLimit < 5000 {
		statusRateLimit = sdk.MonitoringStatusAlert
	}
	c := sdk.MonitoringStatusLine{Component: "Github-RateLimit", Value: fmt.Sprintf("%d", RateLimitLimit), Status: statusRateLimit}

	return []sdk.MonitoringStatusLine{a, b, c}
}
