package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

type statusData struct {
	desc         string
	status       string
	repoFullName string
	hash         string
	urlPipeline  string
	context      string
}

// SetStatus Users with push access can create commit statuses for a given ref:
// https://developer.github.com/v3/repos/statuses/#create-a-status
func (g *githubClient) SetStatus(ctx context.Context, buildStatus sdk.VCSBuildStatus) error {
	if buildStatus.Status == "" {
		log.Debug(ctx, "github.SetStatus> Do not process event for empty status")
		return nil
	}

	ghStatus := CreateStatus{
		Description: buildStatus.Description,
		State:       buildStatus.Status,
		Context:     buildStatus.Context,
		TargetURL:   buildStatus.URLCDS,
	}

	switch buildStatus.Status {
	case sdk.StatusSuccess:
		ghStatus.State = "success"
	case sdk.StatusFail:
		ghStatus.State = "failure"
	case sdk.StatusBuilding:
		ghStatus.State = "pending"
	default:
		log.Debug(ctx, "SetStatus> github setStatus not managed for %s", buildStatus.Status)
		return nil
	}

	path := fmt.Sprintf("/repos/%s/statuses/%s", buildStatus.RepositoryFullname, buildStatus.GitHash)

	b, err := json.Marshal(ghStatus)
	if err != nil {
		return sdk.WrapError(err, "Unable to marshal github status")
	}
	buf := bytes.NewBuffer(b)

	log.Debug(ctx, "SetStatus> github post on %v body:%v", path, string(b))

	res, err := g.post(ctx, path, "application/json", buf, nil, nil)
	if err != nil {
		return sdk.WrapError(err, "Unable to post status")
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return sdk.WrapError(err, "Unable to read body")
	}

	log.Debug(ctx, "SetStatus> github response for %v body:%v", path, string(body))

	if res.StatusCode != 201 {
		return sdk.WrapError(err, "Unable to create status on github. Status code : %d - Body: %s - context:%s", res.StatusCode, body, buildStatus.Context)
	}

	s := &Status{}
	if err := sdk.JSONUnmarshal(body, s); err != nil {
		return sdk.WrapError(err, "Unable to unmarshal body")
	}

	log.Debug(ctx, "SetStatus> Status %d %s created at %v", s.ID, s.URL, s.CreatedAt)

	return nil
}

func (g *githubClient) ListStatuses(ctx context.Context, repo string, ref string) ([]sdk.VCSCommitStatus, error) {
	url := "/repos/" + repo + "/statuses/" + ref
	status, body, _, err := g.get(ctx, url)
	if err != nil {
		return []sdk.VCSCommitStatus{}, sdk.WrapError(err, "githubClient.ListStatuses")
	}
	if status >= 400 {
		return []sdk.VCSCommitStatus{}, sdk.NewError(sdk.ErrRepoNotFound, errorAPI(body))
	}
	ss := []Status{}

	//Github may return 304 status because we are using conditional request with ETag based headers
	if status == http.StatusNotModified {
		//If repo isn't updated, lets get them from cache
		k := cache.Key("vcs", "github", "statuses", sdk.Hash512(g.OAuthToken+g.username), url)
		if _, err := g.Cache.Get(ctx, k, &ss); err != nil {
			log.Error(ctx, "cannot get from cache %s: %v", k, err)
		}
	} else {
		if err := sdk.JSONUnmarshal(body, &ss); err != nil {
			return []sdk.VCSCommitStatus{}, sdk.WrapError(err, "Unable to parse github commit: %s", ref)
		}
		//Put the body on cache for one hour and one minute
		k := cache.Key("vcs", "github", "statuses", sdk.Hash512(g.OAuthToken+g.username), url)
		if err := g.Cache.SetWithTTL(ctx, k, ss, 61*60); err != nil {
			log.Error(ctx, "cannot SetWithTTL: %s: %v", k, err)
		}
	}

	vcsStatuses := []sdk.VCSCommitStatus{}
	for _, s := range ss {
		if !strings.HasPrefix(s.Context, "CDS/") {
			continue
		}
		vcsStatuses = append(vcsStatuses, sdk.VCSCommitStatus{
			CreatedAt:  s.CreatedAt,
			Decription: s.Context,
			Ref:        ref,
			State:      processGithubState(s),
		})
	}

	return vcsStatuses, nil
}

func processGithubState(s Status) string {
	switch s.State {
	case "success":
		return sdk.StatusSuccess
	case "error", "failure":
		return sdk.StatusFail
	default:
		return sdk.StatusDisabled
	}
}
