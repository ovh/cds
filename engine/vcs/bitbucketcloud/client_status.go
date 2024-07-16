package bitbucketcloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

func (b *bitbucketcloudClient) CreateInsightReport(ctx context.Context, repo string, sha string, insightKey string, vcsReport sdk.VCSInsight) error {
	// not implemented
	return nil
}

// SetStatus Users with push access can create commit statuses for a given ref:
func (client *bitbucketcloudClient) SetStatus(ctx context.Context, buildStatus sdk.VCSBuildStatus) error {
	if buildStatus.Status == "" {
		log.Debug(ctx, "bitbucketcloud.SetStatus> Do not process event for empty status")
		return nil
	}

	if buildStatus.Status == sdk.StatusChecking ||
		buildStatus.Status == sdk.StatusDisabled ||
		buildStatus.Status == sdk.StatusNeverBuilt ||
		buildStatus.Status == sdk.StatusSkipped ||
		buildStatus.Status == sdk.StatusUnknown ||
		buildStatus.Status == sdk.StatusWaiting {
		return nil
	}

	var state string
	switch buildStatus.Status {
	case sdk.StatusFail:
		state = "FAILED"
	case sdk.StatusSuccess, sdk.StatusSkipped:
		state = "SUCCESSFUL"
	case sdk.StatusStopped:
		state = "STOPPED"
	default:
		state = "INPROGRESS"
	}

	bbStatus := Status{
		Description: buildStatus.Description,
		URL:         buildStatus.URLCDS,
		State:       state,
		Name:        buildStatus.Title,
		Key:         buildStatus.Context,
	}

	if len(buildStatus.Context) > 36 { // 40 maxlength on bitbucket cloud
		buildStatus.Context = buildStatus.Context[:36]
	}

	path := fmt.Sprintf("/repositories/%s/commit/%s/statuses/build", buildStatus.RepositoryFullname, buildStatus.GitHash)
	b, err := json.Marshal(bbStatus)
	if err != nil {
		return sdk.WrapError(err, "Unable to marshal bitbucketcloud status")
	}
	buf := bytes.NewBuffer(b)

	res, err := client.post(ctx, path, "application/json", buf, nil)
	if err != nil {
		return sdk.WrapError(err, "Unable to post status")
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return sdk.WrapError(err, "Unable to read body")
	}
	if res.StatusCode != 201 && res.StatusCode != 200 {
		return fmt.Errorf("unable to create status on bitbucket cloud. Status code : %d - Body: %s - context:%s", res.StatusCode, body, buildStatus.Context)
	}

	var resp Status
	if err := sdk.JSONUnmarshal(body, &resp); err != nil {
		return sdk.WrapError(err, "Unable to unmarshal body")
	}

	log.Debug(ctx, "bitbucketcloud.SetStatus> Status %s %s created at %v", resp.UUID, resp.Links.Self.Href, resp.CreatedOn)

	return nil
}

func (client *bitbucketcloudClient) ListStatuses(ctx context.Context, repo string, ref string) ([]sdk.VCSCommitStatus, error) {
	url := fmt.Sprintf("/repositories/%s/commit/%s/statuses", repo, ref)
	status, body, _, err := client.get(ctx, url)
	if err != nil {
		return []sdk.VCSCommitStatus{}, sdk.WrapError(err, "bitbucketcloudClient.ListStatuses")
	}
	if status >= 400 {
		return []sdk.VCSCommitStatus{}, sdk.NewError(sdk.ErrRepoNotFound, errorAPI(body))
	}
	var ss Statuses
	if err := sdk.JSONUnmarshal(body, &ss); err != nil {
		return []sdk.VCSCommitStatus{}, sdk.WrapError(err, "Unable to parse bitbucket cloud commit: %s", ref)
	}

	vcsStatuses := make([]sdk.VCSCommitStatus, 0, ss.Size)
	for _, s := range ss.Values {
		if !strings.HasPrefix(s.Name, "CDS/") {
			continue
		}
		vcsStatuses = append(vcsStatuses, sdk.VCSCommitStatus{
			CreatedAt:  s.CreatedOn,
			Decription: s.Description,
			Ref:        ref,
			State:      processBbitbucketState(s),
		})
	}

	return vcsStatuses, nil
}

func processBbitbucketState(s Status) string {
	switch s.State {
	case "SUCCESSFUL":
		return sdk.StatusSuccess
	case "FAILED":
		return sdk.StatusFail
	case "STOPPED":
		return sdk.StatusStopped
	default:
		return sdk.StatusBuilding
	}
}
