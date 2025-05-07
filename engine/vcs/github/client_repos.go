package github

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

// Repos list repositories that are accessible to the authenticated user
// https://developer.github.com/v3/repos/#list-your-repositories
func (g *githubClient) Repos(ctx context.Context) ([]sdk.VCSRepo, error) {
	var repos = []Repository{}
	var noEtag bool
	var attempt int

	var nextPage = "/user/repos"
	for nextPage != "" {
		if ctx.Err() != nil {
			break
		}

		var opt getArgFunc
		if noEtag {
			opt = withoutETag
		} else {
			opt = withETag
		}

		attempt++
		status, body, headers, err := g.get(ctx, nextPage, opt)
		if err != nil {
			log.Warn(ctx, "githubClient.Repos> Error %s", err)
			return nil, err
		}
		if status >= 400 {
			return nil, sdk.NewError(sdk.ErrUnknownError, errorAPI(body))
		}
		nextRepos := []Repository{}

		//Github may return 304 status because we are using conditional request with ETag based headers
		if status == http.StatusNotModified {
			//If repos aren't updated, lets get them from cache
			k := cache.Key("vcs", "github", "repos", sdk.Hash512(g.OAuthToken+g.username), "/user/repos")
			if _, err := g.Cache.Get(k, &repos); err != nil {
				log.Error(ctx, "cannot get from cache %s: %v", k, err)
			}
			if len(repos) != 0 || attempt > 5 {
				//We found repos, let's exit the loop
				break
			}
			//If we did not found any repos in cache, let's retry (same nextPage) without etag
			noEtag = true
			continue
		} else {
			if err := sdk.JSONUnmarshal(body, &nextRepos); err != nil {
				log.Warn(ctx, "githubClient.Repos> Unable to parse github repositories: %s", err)
				return nil, err
			}
		}

		repos = append(repos, nextRepos...)
		nextPage = getNextPage(headers)
	}

	//Put the body on cache for one hour and one minute
	k := cache.Key("vcs", "github", "repos", sdk.Hash512(g.OAuthToken+g.username), "/user/repos")
	if err := g.Cache.SetWithTTL(k, repos, 61*60); err != nil {
		log.Error(ctx, "cannot SetWithTTL: %s: %v", k, err)
	}

	responseRepos := []sdk.VCSRepo{}
	for _, repo := range repos {
		responseRepos = append(responseRepos, g.ToVCSRepo(repo))
	}

	return responseRepos, nil
}

// RepoByFullname Get only one repo
// https://developer.github.com/v3/repos/#list-your-repositories
func (g *githubClient) RepoByFullname(ctx context.Context, fullname string) (sdk.VCSRepo, error) {
	repo, err := g.repoByFullname(ctx, fullname)
	if err != nil {
		return sdk.VCSRepo{}, err
	}

	if repo.ID == 0 {
		return sdk.VCSRepo{}, err
	}

	return g.ToVCSRepo(repo), nil
}

func (g *githubClient) repoByFullname(ctx context.Context, fullname string) (Repository, error) {
	url := "/repos/" + fullname
	status, body, _, err := g.get(ctx, url)
	if err != nil {
		log.Warn(ctx, "githubClient.Repos> Error %s", err)
		return Repository{}, err
	}
	if status >= 400 {
		return Repository{}, sdk.NewError(sdk.ErrRepoNotFound, errorAPI(body))
	}
	repo := Repository{}

	//Github may return 304 status because we are using conditional request with ETag based headers
	if status == http.StatusNotModified {
		//If repo isn't updated, lets get them from cache
		k := cache.Key("vcs", "github", "repo", sdk.Hash512(g.OAuthToken+g.username), url)
		if _, err := g.Cache.Get(k, &repo); err != nil {
			log.Error(ctx, "cannot get from cache %s: %v", k, err)
		}
	} else {
		if err := sdk.JSONUnmarshal(body, &repo); err != nil {
			log.Warn(ctx, "githubClient.Repos> Unable to parse github repository: %s", err)
			return Repository{}, err
		}
		//Put the body on cache for one hour and one minute
		k := cache.Key("vcs", "github", "repo", sdk.Hash512(g.OAuthToken+g.username), url)
		if err := g.Cache.SetWithTTL(k, repo, 61*60); err != nil {
			log.Error(ctx, "cannot SetWithTTL: %s: %v", k, err)
		}
	}

	return repo, nil
}

func (g *githubClient) UserHasWritePermission(ctx context.Context, fullname string) (bool, error) {
	owner := strings.SplitN(fullname, "/", 2)[0]
	if g.token != "" {
		if g.username == "" {
			return false, sdk.WrapError(sdk.ErrUserNotFound, "No user found in configuration")
		}
		if g.username == owner {
			log.Debug(ctx, "githubClient.UserHasWritePermission> nothing to do ¯\\_(ツ)_/¯")
			return true, nil
		}
	}

	url := "/repos/" + fullname + "/collaborators/" + g.username + "/permission"
	k := cache.Key("vcs", "github", "user-write", sdk.Hash512(g.OAuthToken+g.username), url)

	status, resp, _, err := g.get(ctx, url)
	if err != nil {
		return false, err
	}
	if status >= 400 {
		return false, sdk.NewError(sdk.ErrUnknownError, errorAPI(resp))
	}
	var permResp UserPermissionResponse
	if status == http.StatusNotModified {
		if _, err := g.Cache.Get(k, &permResp); err != nil {
			log.Error(ctx, "cannot get from cache %s: %v", k, err)
		}
	} else {
		if err := sdk.JSONUnmarshal(resp, &permResp); err != nil {
			return false, sdk.WrapError(err, "unable to unmarshal: %s", string(resp))
		}
		if err := g.Cache.SetWithTTL(k, permResp, 61*60); err != nil {
			log.Error(ctx, "cannot SetWithTTL: %s: %v", k, err)
		}
	}
	return permResp.Permission == "write" || permResp.Permission == "admin", nil
}

func (g *githubClient) ToVCSRepo(repo Repository) sdk.VCSRepo {
	return sdk.VCSRepo{
		ID:              strconv.Itoa(repo.ID),
		Name:            repo.Name,
		Slug:            strings.Split(repo.FullName, "/")[0],
		Fullname:        repo.FullName,
		URL:             repo.HTMLURL,
		URLCommitFormat: repo.HTMLURL + "/commit/%s",
		URLTagFormat:    repo.HTMLURL + "/commits/%s",
		URLBranchFormat: repo.HTMLURL + "/commits/%s",
		HTTPCloneURL:    repo.CloneURL,
		SSHCloneURL:     repo.SSHURL,
	}
}
