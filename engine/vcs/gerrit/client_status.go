package gerrit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/andygrunwald/go-gerrit"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// SetStatus set build status on Gerrit
func (c *gerritClient) SetStatus(ctx context.Context, event sdk.Event) error {
	var eventNR sdk.EventRunWorkflowNode
	if err := json.Unmarshal(event.Payload, &eventNR); err != nil {
		return sdk.WrapError(err, "cannot unmarshal payload")
	}

	if eventNR.GerritChange == nil {
		log.Debug("gerrit.setStatus> no gerrit change provided: %s/%s", eventNR.Status, eventNR.NodeName)
		return nil
	}

	// Use reviewer account to post the review
	c.client.Authentication.SetBasicAuth(c.reviewerName, c.reviewerToken)

	// https://gerrit-review.googlesource.com/Documentation/rest-api-changes.html#review-input
	ri := gerrit.ReviewInput{
		Message: c.buildMessage(eventNR),
		Tag:     "CDS",
		Labels:  c.buildLabel(eventNR),
		Notify:  "OWNER", // Send notification to the owner
	}

	// Check if we already send the message
	changeDetail, _, err := c.client.Changes.GetChangeDetail(eventNR.GerritChange.ID, nil)
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
		if _, _, err := c.client.Changes.SetReview(eventNR.GerritChange.ID, eventNR.GerritChange.Revision, &ri); err != nil {
			return sdk.WrapError(err, "unable to set gerrit review")
		}
	}

	return nil
}

func (c *gerritClient) ListStatuses(ctx context.Context, repo string, ref string) ([]sdk.VCSCommitStatus, error) {
	return nil, nil
}

func (c *gerritClient) buildMessage(eventNR sdk.EventRunWorkflowNode) string {
	var message string
	switch eventNR.Status {
	case sdk.StatusSuccess:
		message += fmt.Sprintf("Build Success on %s\n%s", eventNR.NodeName, eventNR.GerritChange.URL)
	case sdk.StatusSkipped:
		message += fmt.Sprintf("Build Skipped on %s\n%s", eventNR.NodeName, eventNR.GerritChange.URL)
	case sdk.StatusFail, sdk.StatusStopped:
		message += fmt.Sprintf("Build Failed on %s\n%s \n%s", eventNR.NodeName, eventNR.GerritChange.URL, eventNR.GerritChange.Report)
	case sdk.StatusBuilding:
		message += fmt.Sprintf("CDS starts working on %s\n%s", eventNR.NodeName, eventNR.GerritChange.URL)
	}
	return message
}

func (c *gerritClient) buildLabel(eventNR sdk.EventRunWorkflowNode) map[string]string {
	labels := make(map[string]string)
	switch eventNR.Status {
	case sdk.StatusSuccess:
		labels["Verified"] = "1"
	case sdk.StatusFail, sdk.StatusStopped:
		labels["Verified"] = "-1"
	default:
		return nil
	}
	return labels
}
