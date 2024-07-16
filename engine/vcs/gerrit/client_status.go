package gerrit

import (
	"context"
	"fmt"
	"strings"

	"github.com/andygrunwald/go-gerrit"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

func (client *gerritClient) CreateInsightReport(ctx context.Context, repo string, sha string, insightKey string, vcsReport sdk.VCSInsight) error {
	// not implemented
	return nil
}

// SetStatus set build status on Gerrit
func (client *gerritClient) SetStatus(ctx context.Context, buildStatus sdk.VCSBuildStatus) error {
	if buildStatus.GerritChange == nil {
		log.Debug(ctx, "gerrit.setStatus> no gerrit change provided - context %s", buildStatus.Context)
		return nil
	}

	// Use reviewer account to post the review
	client.client.Authentication.SetBasicAuth(client.reviewerName, client.reviewerToken)

	// https://gerrit-review.googlesource.com/Documentation/rest-api-changes.html#review-input
	ri := gerrit.ReviewInput{
		Message: client.buildMessage(buildStatus),
		Tag:     "CDS",
		Labels:  client.buildLabel(buildStatus),
		Notify:  "OWNER", // Send notification to the owner
	}

	// Check if we already send the message
	changeDetail, _, err := client.client.Changes.GetChangeDetail(buildStatus.GerritChange.ID, nil)
	if err != nil {
		return sdk.WrapError(err, "error while getting change detail")
	}
	found := false
	for _, m := range changeDetail.Messages {
		if m.Tag != "CDS" {
			continue
		}
		if strings.Contains(m.Message, ri.Message) {
			found = true
			break
		}
	}

	if !found {
		if _, _, err := client.client.Changes.SetReview(buildStatus.GerritChange.ID, buildStatus.GerritChange.Revision, &ri); err != nil {
			return sdk.WrapError(err, "unable to set gerrit review")
		}
	}

	return nil
}

func (client *gerritClient) ListStatuses(ctx context.Context, repo string, ref string) ([]sdk.VCSCommitStatus, error) {
	return nil, nil
}

func (client *gerritClient) buildMessage(buildStatus sdk.VCSBuildStatus) string {
	var message string
	switch buildStatus.Status {
	case sdk.StatusSuccess:
		message += fmt.Sprintf("Build Success on %s\n%s", buildStatus.Context, buildStatus.GerritChange.URL)
	case sdk.StatusSkipped:
		message += fmt.Sprintf("Build Skipped on %s\n%s", buildStatus.Context, buildStatus.GerritChange.URL)
	case sdk.StatusFail, sdk.StatusStopped:
		message += fmt.Sprintf("Build Failed on %s\n%s \n%s", buildStatus.Context, buildStatus.GerritChange.URL, buildStatus.GerritChange.Report)
	case sdk.StatusBuilding:
		message += fmt.Sprintf("CDS starts working on %s\n%s", buildStatus.Context, buildStatus.GerritChange.URL)
	}
	return message
}

func (client *gerritClient) buildLabel(buildStatus sdk.VCSBuildStatus) map[string]string {
	labels := make(map[string]string)
	switch buildStatus.Status {
	case sdk.StatusSuccess:
		labels["Verified"] = "1"
	case sdk.StatusFail, sdk.StatusStopped:
		labels["Verified"] = "-1"
	default:
		return nil
	}
	return labels
}
