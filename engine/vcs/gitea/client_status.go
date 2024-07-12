package gitea

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
)

func (client *giteaClient) CreateInsightReport(ctx context.Context, repo string, sha string, insightKey string, vcsReport sdk.VCSInsight) error {
	// not implemented
	return nil
}

func (client *giteaClient) SetStatus(ctx context.Context, buildStatus sdk.VCSBuildStatus) error {

	// POST /repos/{owner}/{repo}/statuses/{sha}
	// {
	// 	"context": "string",
	// 	"description": "string",
	// 	"state": "string",
	// 	"target_url": "string"
	//   }

	if buildStatus.Status == "" {
		log.Debug(ctx, "gitea.SetStatus> Do not process event for empty status")
		return nil
	}

	giteaStatus := gitea.CreateStatusOption{
		Context:   buildStatus.Context,
		TargetURL: buildStatus.URLCDS,
	}

	// gitea display on the UI, the context contact with the description.
	// so that, we remove the context from the description:
	// description":"WorkflowNotificationsLog:Success","context":"ITV2WFNOTIF-WorkflowNotificationsLog
	// we want only ITV2WFNOTIF-WorkflowNotificationsLog:Success display
	td := strings.Split(buildStatus.Description, ":")
	if len(td) == 2 {
		giteaStatus.Description = td[1]
	} else {
		giteaStatus.Description = buildStatus.Description
	}

	switch buildStatus.Status {
	case sdk.StatusChecking, sdk.StatusPending, sdk.StatusBuilding:
		giteaStatus.State = gitea.StatusPending
	case sdk.StatusSuccess:
		giteaStatus.State = gitea.StatusSuccess
	case sdk.StatusFail:
		giteaStatus.State = gitea.StatusFailure
	}

	path := fmt.Sprintf("/repos/%s/statuses/%s", buildStatus.RepositoryFullname, buildStatus.GitHash)

	b, err := json.Marshal(giteaStatus)
	if err != nil {
		return sdk.WrapError(err, "unable to marshal gitea status")
	}

	log.Debug(ctx, "SetStatus> gitea post on %v body:%v", path, string(b))

	t := strings.Split(buildStatus.RepositoryFullname, "/")
	if len(t) != 2 {
		return sdk.WrapError(err, "invalid gitRepositoryFullname gitea: %s", buildStatus.RepositoryFullname)
	}
	s, resp, err := client.client.CreateStatus(t[0], t[1], buildStatus.GitHash, giteaStatus)
	if err != nil {
		return sdk.WrapError(err, "unable to post gitea status")
	}

	log.Debug(ctx, "SetStatus> gitea response for %v status: %d:", path, resp.StatusCode)

	if resp.StatusCode != 201 {
		return sdk.WrapError(err, "unable to create status on gitea. Status code : %d - Body: %s - context:%s", resp.StatusCode, resp.Body, buildStatus.Context)
	}

	log.Debug(ctx, "SetStatus> Status %d %s created at %v", s.ID, s.URL, s.Created)
	return nil
}

func (client *giteaClient) ListStatuses(ctx context.Context, fullname string, ref string) ([]sdk.VCSCommitStatus, error) {

	t := strings.Split(fullname, "/")
	if len(t) != 2 {
		return nil, fmt.Errorf("giteaCliet.Tags> invalid fullname %s", fullname)
	}
	statuses, _, err := client.client.ListStatuses(t[0], t[1], ref, gitea.ListStatusesOption{})
	if err != nil {
		return nil, err
	}

	var vcsStatuses []sdk.VCSCommitStatus
	for _, s := range statuses {
		vcsStatuses = append(vcsStatuses, sdk.VCSCommitStatus{
			CreatedAt:  s.Created,
			Decription: s.Context,
			Ref:        ref,
			State:      processGiteaState(s.State),
		})
	}

	return vcsStatuses, nil
}

func processGiteaState(s gitea.StatusState) string {
	switch s {
	case "success":
		return sdk.StatusSuccess
	case "error", "failure":
		return sdk.StatusFail
	default:
		return sdk.StatusDisabled
	}
}
