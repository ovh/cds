package bitbucketserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func (client *bitbucketClient) SetStatus(ctx context.Context, buildStatus sdk.VCSBuildStatus) error {
	ctx, end := telemetry.Span(ctx, "bitbucketserver.SetStatus")
	defer end()

	state := getBitbucketStateFromStatus(buildStatus.Status)
	status := Status{
		Key:         buildStatus.Context,
		Name:        buildStatus.Title,
		State:       state,
		URL:         buildStatus.URLCDS,
		Description: buildStatus.Description,
	}

	values, err := json.Marshal(status)
	if err != nil {
		return sdk.WrapError(err, "Unable to marshall status")
	}

	log.Info(ctx, "sending build status for %s : %s %s - %s", buildStatus.GitHash, status.Key, status.Name, state)

	if err := client.do(ctx, "POST", "build-status", fmt.Sprintf("/commits/%s", buildStatus.GitHash), nil, values, nil, Options{}); err != nil {
		return sdk.WrapError(err, "Unable to post build-status name:%s status:%s", status.Name, state)
	}
	return nil
}

func (client *bitbucketClient) ListStatuses(ctx context.Context, repo string, ref string) ([]sdk.VCSCommitStatus, error) {
	ss := []Status{}

	path := fmt.Sprintf("/commits/%s", ref)
	params := url.Values{}
	nextPage := 0
	for {
		if ctx.Err() != nil {
			break
		}

		if nextPage != 0 {
			params.Set("start", fmt.Sprintf("%d", nextPage))
		}

		var response ResponseStatus
		if err := client.do(ctx, "GET", "build-status", path, params, nil, &response, Options{}); err != nil {
			return nil, sdk.WrapError(err, "Unable to get statuses")
		}

		ss = append(ss, response.Values...)

		if response.IsLastPage {
			break
		} else {
			nextPage = response.NextPageStart
		}
	}

	vcsStatuses := []sdk.VCSCommitStatus{}
	for _, s := range ss {
		if !strings.HasPrefix(s.Description, "CDS/") {
			continue
		}
		vcsStatuses = append(vcsStatuses, sdk.VCSCommitStatus{
			CreatedAt:  time.Unix(s.Timestamp/1000, 0),
			Decription: s.Description,
			Ref:        ref,
			State:      processBitbucketState(s),
		})
	}

	return vcsStatuses, nil
}

func processBitbucketState(s Status) string {
	switch s.State {
	case successful:
		return sdk.StatusSuccess
	case failed:
		return sdk.StatusFail
	default:
		return sdk.StatusDisabled
	}
}

const (
	// "state": "<INPROGRESS|SUCCESSFUL|FAILED>"
	// doc from https://developer.atlassian.com/server/bitbucket/how-tos/updating-build-status-for-commits/
	inProgress = "INPROGRESS"
	successful = "SUCCESSFUL"
	failed     = "FAILED"
)

func getBitbucketStateFromStatus(status string) string {
	switch status {
	case sdk.StatusSuccess, sdk.StatusSkipped, sdk.StatusDisabled:
		return successful
	case sdk.StatusWaiting, sdk.StatusBuilding:
		return inProgress
	case sdk.StatusFail:
		return failed
	default:
		return failed
	}
}
