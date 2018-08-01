package github

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (g *githubClient) ListForks(ctx context.Context, repo string) ([]sdk.VCSRepo, error) {
	var repos = []Repository{}
	var nextPage = "/repos/" + repo + "/forks"

	var noEtag bool
	var attempt int
	for {
		if nextPage != "" {
			var opt getArgFunc
			if noEtag {
				opt = withoutETag
			} else {
				opt = withETag
			}

			attempt++
			status, body, headers, err := g.get(nextPage, opt)
			if err != nil {
				log.Warning("githubClient.ListForks> Error %s", err)
				return nil, err
			}
			if status >= 400 {
				return nil, sdk.NewError(sdk.ErrUnknownError, errorAPI(body))
			}
			nextRepos := []Repository{}

			//Github may return 304 status because we are using conditional request with ETag based headers
			if status == http.StatusNotModified {
				//If repos aren't updated, lets get them from cache
				g.Cache.Get(cache.Key("vcs", "github", "forks", g.OAuthToken, "/user/", repo, "/forks"), &repos)
				if len(repos) != 0 || attempt > 5 {
					//We found repos, let's exit the loop
					break
				}
				//If we did not found any repos in cache, let's retry (same nextPage) without etag
				noEtag = true
				continue
			} else {
				if err := json.Unmarshal(body, &nextRepos); err != nil {
					log.Warning("githubClient.ListForks> Unable to parse github repositories: %s", err)
					return nil, err
				}
			}

			repos = append(repos, nextRepos...)
			nextPage = getNextPage(headers)
		} else {
			break
		}
	}

	//Put the body on cache for one hour and one minute
	g.Cache.SetWithTTL(cache.Key("vcs", "github", "forks", g.OAuthToken, "/user/", repo, "/forks"), repos, 61*60)

	responseRepos := []sdk.VCSRepo{}
	for _, repo := range repos {
		r := sdk.VCSRepo{
			ID:           strconv.Itoa(repo.ID),
			Name:         repo.Name,
			Slug:         strings.Split(repo.FullName, "/")[0],
			Fullname:     repo.FullName,
			URL:          repo.HTMLURL,
			HTTPCloneURL: repo.CloneURL,
			SSHCloneURL:  repo.SSHURL,
		}
		responseRepos = append(responseRepos, r)
	}

	return responseRepos, nil
}
