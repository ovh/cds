package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (g *githubClient) PullRequest(ctx context.Context, fullname string, id int) (sdk.VCSPullRequest, error) {
	var pr PullRequest
	cachePullRequestKey := cache.Key("vcs", "github", "pullrequests", g.OAuthToken, fmt.Sprintf("/repos/%s/pulls/%d", fullname, id))
	opts := []getArgFunc{withETag}

	for {
		url := fmt.Sprintf("/repos/%s/pulls/%d", fullname, id)

		status, body, _, err := g.get(ctx, url, opts...)
		if err != nil {
			if err := g.Cache.Delete(cachePullRequestKey); err != nil {
				log.Error(ctx, "githubclient.PullRequest > unable to delete cache key %v: %v", cachePullRequestKey, err)
			}
			return sdk.VCSPullRequest{}, err
		}
		if status >= 400 {
			if err := g.Cache.Delete(cachePullRequestKey); err != nil {
				log.Error(ctx, "githubclient.PullRequest > unable to delete cache key %v: %v", cachePullRequestKey, err)
			}
			return sdk.VCSPullRequest{}, sdk.NewError(sdk.ErrUnknownError, errorAPI(body))
		}

		//Github may return 304 status because we are using conditional request with ETag based headers
		if status == http.StatusNotModified {
			//If repos aren't updated, lets get them from cache
			find, err := g.Cache.Get(cachePullRequestKey, &pr)
			if err != nil {
				log.Error(ctx, "cannot get from cache %s: %v", cachePullRequestKey, err)
			}
			if !find {
				opts = []getArgFunc{withoutETag}
				log.Error(ctx, "Unable to get pullrequest (%s) from the cache", strings.ReplaceAll(cachePullRequestKey, g.OAuthToken, ""))
				continue
			}

		} else {
			if err := json.Unmarshal(body, &pr); err != nil {
				log.Warning(ctx, "githubClient.PullRequest> Unable to parse github pullrequest: %s", err)
				return sdk.VCSPullRequest{}, sdk.WithStack(err)
			}
		}

		if pr.Number != id {
			log.Warning(ctx, "githubClient.PullRequest> Cannot find pullrequest %d", id)
			if err := g.Cache.Delete(cachePullRequestKey); err != nil {
				log.Error(ctx, "githubclient.PullRequest > unable to delete cache key %v: %v", cachePullRequestKey, err)
			}
			return sdk.VCSPullRequest{}, sdk.WithStack(fmt.Errorf("cannot find pullrequest %d", id))
		}

		//Put the body on cache for one hour and one minute
		if err := g.Cache.SetWithTTL(cachePullRequestKey, pr, 61*60); err != nil {
			log.Error(ctx, "cannot SetWithTTL: %s: %v", cachePullRequestKey, err)
		}
		break
	}

	return pr.ToVCSPullRequest(), nil
}

// PullRequests fetch all the pull request for a repository
func (g *githubClient) PullRequests(ctx context.Context, fullname string, opts sdk.VCSPullRequestOptions) ([]sdk.VCSPullRequest, error) {
	var pullRequests = []PullRequest{}
	cacheKey := cache.Key("vcs", "github", "pullrequests", g.OAuthToken, "/repos/"+fullname+"/pulls")
	githubOpts := []getArgFunc{withETag}

	var nextPage = "/repos/" + fullname + "/pulls"
	for nextPage != "" {
		if ctx.Err() != nil {
			break
		}

		status, body, headers, err := g.get(ctx, nextPage, githubOpts...)
		if err != nil {
			log.Warning(ctx, "githubClient.PullRequests> Error %s", err)
			return nil, err
		}
		if status >= 400 {
			return nil, sdk.NewError(sdk.ErrUnknownError, errorAPI(body))
		}
		githubOpts[0] = withETag
		nextPullRequests := []PullRequest{}

		//Github may return 304 status because we are using conditional request with ETag based headers
		if status == http.StatusNotModified {
			//If repos aren't updated, lets get them from cache
			find, err := g.Cache.Get(cacheKey, &pullRequests)
			if err != nil {
				log.Error(ctx, "cannot get from cache %s: %v", cacheKey, err)
			}
			if !find {
				githubOpts[0] = withoutETag
				log.Error(ctx, "Unable to get pullrequest (%s) from the cache", strings.ReplaceAll(cacheKey, g.OAuthToken, ""))
				continue
			}
			break
		} else {
			if err := json.Unmarshal(body, &nextPullRequests); err != nil {
				log.Warning(ctx, "githubClient.Branches> Unable to parse github branches: %s", err)
				return nil, err
			}
		}

		pullRequests = append(pullRequests, nextPullRequests...)

		nextPage = getNextPage(headers)
	}

	//Put the body on cache for one hour and one minute
	k := cache.Key("vcs", "github", "pullrequests", g.OAuthToken, "/repos/"+fullname+"/pulls")
	if err := g.Cache.SetWithTTL(k, pullRequests, 61*60); err != nil {
		log.Error(ctx, "cannot SetWithTTL: %s: %v", k, err)
	}

	prResults := []sdk.VCSPullRequest{}
	for _, pullr := range pullRequests {
		// If a state is given we want to filter PRs
		switch opts.State {
		case sdk.VCSPullRequestStateOpen:
			if pullr.State == "closed" || pullr.Merged {
				continue
			}
		case sdk.VCSPullRequestStateMerged:
			if !pullr.Merged {
				continue
			}
		case sdk.VCSPullRequestStateClosed:
			if pullr.State != "closed" || pullr.Merged {
				continue
			}
		}
		pr := pullr.ToVCSPullRequest()
		prResults = append(prResults, pr)
	}

	return prResults, nil
}

// PullRequestComment push a new comment on a pull request
func (g *githubClient) PullRequestComment(ctx context.Context, repo string, prReq sdk.VCSPullRequestCommentRequest) error {
	if g.DisableStatus {
		log.Warning(ctx, "github.PullRequestComment>  ⚠ Github statuses are disabled")
		return nil
	}

	canWrite, err := g.UserHasWritePermission(ctx, repo)
	if err != nil {
		return err
	}
	if !canWrite {
		if err := g.GrantWritePermission(ctx, repo); err != nil {
			return err
		}
	}

	path := fmt.Sprintf("/repos/%s/issues/%d/comments", repo, prReq.ID)
	payload := map[string]string{
		"body": prReq.Message,
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
	canWrite, err := g.UserHasWritePermission(ctx, repo)
	if err != nil {
		return sdk.VCSPullRequest{}, nil
	}
	if !canWrite {
		if err := g.GrantWritePermission(ctx, repo); err != nil {
			return sdk.VCSPullRequest{}, nil
		}
	}

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
		URL: pullr.HTMLURL,
		User: sdk.VCSAuthor{
			Avatar:      pullr.User.AvatarURL,
			DisplayName: pullr.User.Login,
			Name:        pullr.User.Name,
		},
		Closed: pullr.State == "closed",
		Merged: pullr.Merged,
	}
}
