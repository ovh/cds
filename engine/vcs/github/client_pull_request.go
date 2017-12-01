package github

import (
	"encoding/json"
	"net/http"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// PullRequests fetch all the pull request for a repository
func (g *githubClient) PullRequests(fullname string) ([]sdk.VCSPullRequest, error) {
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
		pr := sdk.VCSPullRequest{
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
		}
		prResults = append(prResults, pr)
	}

	return prResults, nil
}
