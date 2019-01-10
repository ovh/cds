package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (g *githubClient) PullRequest(ctx context.Context, fullname string, id int) (sdk.VCSPullRequest, error) {
	cachePullRequestKey := cache.Key("vcs", "github", "pullrequests", g.OAuthToken, fmt.Sprintf("/repos/%s/pulls/%d"+fullname, id))

	url := fmt.Sprintf("/repos/%s/pulls/%d"+fullname, id)
	status, body, _, err := g.get(url)
	if err != nil {
		g.Cache.Delete(cachePullRequestKey)
		return sdk.VCSPullRequest{}, err
	}
	if status >= 400 {
		g.Cache.Delete(cachePullRequestKey)
		return sdk.VCSPullRequest{}, sdk.NewError(sdk.ErrUnknownError, errorAPI(body))
	}

	//Github may return 304 status because we are using conditional request with ETag based headers
	var pr PullRequest
	if status == http.StatusNotModified {
		//If repos aren't updated, lets get them from cache
		if !g.Cache.Get(cachePullRequestKey, &pr) {
			log.Error("Unable to get pullrequest (%s) from the cache", cachePullRequestKey)
		}

	} else {
		if err := json.Unmarshal(body, &pr); err != nil {
			log.Warning("githubClient.PullRequest> Unable to parse github pullrequest: %s", err)
			return sdk.VCSPullRequest{}, err
		}
	}

	if pr.ID != id {
		log.Warning("githubClient.PullRequest> Cannot find pullrequest %d", id)
		g.Cache.Delete(cachePullRequestKey)
		return sdk.VCSPullRequest{}, fmt.Errorf("githubClient.PullRequest > Cannot find pullrequest %d", id)
	}

	//Put the body on cache for one hour and one minute
	g.Cache.SetWithTTL(cachePullRequestKey, pr, 61*60)

	return pr.ToVCSPullRequest(), nil
}

// PullRequests fetch all the pull request for a repository
func (g *githubClient) PullRequests(ctx context.Context, fullname string) ([]sdk.VCSPullRequest, error) {
	var pullRequests = []PullRequest{}
	var nextPage = "/repos/" + fullname + "/pulls"

	for {
		if nextPage != "" {
			status, body, headers, err := g.get(nextPage)
			if err != nil {
				log.Warning("githubClient.PullRequests> Error %s", err)
				return nil, err
			}
			if status >= 400 {
				return nil, sdk.NewError(sdk.ErrUnknownError, errorAPI(body))
			}
			nextPullRequests := []PullRequest{}

			//Github may return 304 status because we are using conditional request with ETag based headers
			if status == http.StatusNotModified {
				//If repos aren't updated, lets get them from cache
				g.Cache.Get(cache.Key("vcs", "github", "pullrequests", g.OAuthToken, "/repos/"+fullname+"/pulls"), &pullRequests)
				break
			} else {
				if err := json.Unmarshal(body, &nextPullRequests); err != nil {
					log.Warning("githubClient.Branches> Unable to parse github branches: %s", err)
					return nil, err
				}
			}

			pullRequests = append(pullRequests, nextPullRequests...)

			nextPage = getNextPage(headers)
		} else {
			break
		}
	}

	//Put the body on cache for one hour and one minute
	g.Cache.SetWithTTL(cache.Key("vcs", "github", "pullrequests", g.OAuthToken, "/repos/"+fullname+"/pulls"), pullRequests, 61*60)

	prResults := []sdk.VCSPullRequest{}
	for _, pullr := range pullRequests {
		pr := pullr.ToVCSPullRequest()
		prResults = append(prResults, pr)
	}

	return prResults, nil
}

// PullRequestComment push a new comment on a pull request
func (g *githubClient) PullRequestComment(ctx context.Context, repo string, id int, text string) error {
	if g.DisableStatus {
		log.Warning("github.PullRequestComment>  ⚠ Github statuses are disabled")
		return nil
	}

	path := fmt.Sprintf("/repos/%s/issues/%d/comments", repo, id)
	payload := map[string]string{
		"body": text,
	}
	values, _ := json.Marshal(payload)
	res, err := g.post(path, "application/json", bytes.NewReader(values), &postOptions{skipDefaultBaseURL: false, asUser: true})
	if err != nil {
		return sdk.WrapError(err, "Unable to post status")
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return sdk.WrapError(err, "Unable to read body")
	}

	log.Debug("%v", string(body))

	if res.StatusCode != 201 {
		return sdk.WrapError(err, "Unable to create status on github. Status code : %d - Body: %s", res.StatusCode, body)
	}

	return nil
}

func (g *githubClient) PullRequestCreate(ctx context.Context, repo string, pr sdk.VCSPullRequest) (sdk.VCSPullRequest, error) {
	path := fmt.Sprintf("/repos/%s/pulls", repo)
	payload := map[string]string{
		"title": pr.Title,
		"head":  pr.Head.Branch.DisplayID,
		"base":  pr.Base.Branch.DisplayID,
	}
	values, _ := json.Marshal(payload)
	res, err := g.post(path, "application/json", bytes.NewReader(values), &postOptions{skipDefaultBaseURL: false, asUser: true})
	if err != nil {
		return sdk.VCSPullRequest{}, sdk.WrapError(err, "Unable to post status")
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return sdk.VCSPullRequest{}, sdk.WrapError(err, "Unable to read body")
	}

	var prResponse PullRequest
	if err := json.Unmarshal(body, &prResponse); err != nil {
		return sdk.VCSPullRequest{}, sdk.WrapError(err, "Unable to unmarshal pullrequest %s", string(body))
	}

	return prResponse.ToVCSPullRequest(), nil
}

func (pullr PullRequest) ToVCSPullRequest() sdk.VCSPullRequest {
	return sdk.VCSPullRequest{
		ID: pullr.Number,
		Base: sdk.VCSPushEvent{
			Repo: pullr.Base.Repo.FullName,
			Branch: sdk.VCSBranch{
				ID:           pullr.Base.Ref,
				DisplayID:    pullr.Base.Ref,
				LatestCommit: pullr.Base.Sha,
			},
			CloneURL: pullr.Base.Repo.CloneURL,
			Commit: sdk.VCSCommit{
				Author: sdk.VCSAuthor{
					Avatar:      pullr.Base.User.AvatarURL,
					DisplayName: pullr.Base.User.Login,
					Name:        pullr.Base.User.Name,
				},
				Hash:      pullr.Base.Sha,
				Message:   pullr.Base.Label,
				Timestamp: pullr.UpdatedAt.Unix(),
			},
		},
		Head: sdk.VCSPushEvent{
			Repo: pullr.Head.Repo.FullName,
			Branch: sdk.VCSBranch{
				ID:           pullr.Head.Ref,
				DisplayID:    pullr.Head.Ref,
				LatestCommit: pullr.Head.Sha,
			},
			CloneURL: pullr.Head.Repo.CloneURL,
			Commit: sdk.VCSCommit{
				Author: sdk.VCSAuthor{
					Avatar:      pullr.Head.User.AvatarURL,
					DisplayName: pullr.Head.User.Login,
					Name:        pullr.Head.User.Name,
				},
				Hash:      pullr.Head.Sha,
				Message:   pullr.Head.Label,
				Timestamp: pullr.UpdatedAt.Unix(),
			},
		},
		URL: pullr.URL,
		User: sdk.VCSAuthor{
			Avatar:      pullr.User.AvatarURL,
			DisplayName: pullr.User.Login,
			Name:        pullr.User.Name,
		},
		Closed: pullr.State == "closed",
		Merged: pullr.Merged,
	}
}
