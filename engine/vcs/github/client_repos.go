package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
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
			log.Warning(ctx, "githubClient.Repos> Error %s", err)
			return nil, err
		}
		if status >= 400 {
			return nil, sdk.NewError(sdk.ErrUnknownError, errorAPI(body))
		}
		nextRepos := []Repository{}

		//Github may return 304 status because we are using conditional request with ETag based headers
		if status == http.StatusNotModified {
			//If repos aren't updated, lets get them from cache
			k := cache.Key("vcs", "github", "repos", g.OAuthToken, "/user/repos")
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
			if err := json.Unmarshal(body, &nextRepos); err != nil {
				log.Warning(ctx, "githubClient.Repos> Unable to parse github repositories: %s", err)
				return nil, err
			}
		}

		repos = append(repos, nextRepos...)
		nextPage = getNextPage(headers)
	}

	//Put the body on cache for one hour and one minute
	k := cache.Key("vcs", "github", "repos", g.OAuthToken, "/user/repos")
	if err := g.Cache.SetWithTTL(k, repos, 61*60); err != nil {
		log.Error(ctx, "cannot SetWithTTL: %s: %v", k, err)
	}

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

	r := sdk.VCSRepo{
		ID:           strconv.Itoa(repo.ID),
		Name:         repo.Name,
		Slug:         strings.Split(repo.FullName, "/")[0],
		Fullname:     repo.FullName,
		URL:          repo.HTMLURL,
		HTTPCloneURL: repo.CloneURL,
		SSHCloneURL:  repo.SSHURL,
	}
	return r, nil
}

func (g *githubClient) repoByFullname(ctx context.Context, fullname string) (Repository, error) {
	url := "/repos/" + fullname
	status, body, _, err := g.get(ctx, url)
	if err != nil {
		log.Warning(ctx, "githubClient.Repos> Error %s", err)
		return Repository{}, err
	}
	if status >= 400 {
		return Repository{}, sdk.NewError(sdk.ErrRepoNotFound, errorAPI(body))
	}
	repo := Repository{}

	//Github may return 304 status because we are using conditional request with ETag based headers
	if status == http.StatusNotModified {
		//If repo isn't updated, lets get them from cache
		k := cache.Key("vcs", "github", "repo", g.OAuthToken, url)
		if _, err := g.Cache.Get(k, &repo); err != nil {
			log.Error(ctx, "cannot get from cache %s: %v", k, err)
		}
	} else {
		if err := json.Unmarshal(body, &repo); err != nil {
			log.Warning(ctx, "githubClient.Repos> Unable to parse github repository: %s", err)
			return Repository{}, err
		}
		//Put the body on cache for one hour and one minute
		k := cache.Key("vcs", "github", "repo", g.OAuthToken, url)
		if err := g.Cache.SetWithTTL(k, repo, 61*60); err != nil {
			log.Error(ctx, "cannot SetWithTTL: %s: %v", k, err)
		}
	}

	return repo, nil
}

func (g *githubClient) UserHasWritePermission(ctx context.Context, fullname string) (bool, error) {
	owner := strings.SplitN(fullname, "/", 2)[0]
	if g.username == "" {
		return false, sdk.WrapError(sdk.ErrUserNotFound, "No user found in configuration")
	}
	if g.username == owner {
		log.Debug("githubClient.UserHasWritePermission> nothing to do ¯\\_(ツ)_/¯")
		return true, nil
	}

	url := "/repos/" + fullname + "/collaborators/" + g.username + "/permission"
	k := cache.Key("vcs", "github", "user-write", g.OAuthToken, url)

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
		if err := json.Unmarshal(resp, &permResp); err != nil {
			return false, sdk.WrapError(err, "unable to unmarshal: %s", string(resp))
		}
		if err := g.Cache.SetWithTTL(k, permResp, 61*60); err != nil {
			log.Error(ctx, "cannot SetWithTTL: %s: %v", k, err)
		}
	}
	return permResp.Permission == "write" || permResp.Permission == "admin", nil
}

func (g *githubClient) GrantWritePermission(ctx context.Context, fullname string) error {
	owner := strings.SplitN(fullname, "/", 2)[0]
	if g.username == "" || owner == g.username {
		log.Debug("githubClient.GrantWritePermission> nothing to do ¯\\_(ツ)_/¯")
		return nil
	}
	url := "/repos/" + fullname + "/collaborators/" + g.username + "?permission=push"
	resp, err := g.put(url, "application/json", nil, nil)
	if err != nil {
		log.Warning(ctx, "githubClient.GrantWritePermission> Error (%s) %s", url, err)
		return err
	}

	// Response when person is already a collaborator
	if resp.StatusCode == 204 {
		log.Info(ctx, "githubClient.GrantWritePermission> %s is already a collaborator", g.username)
		return nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close() // nolint

	log.Debug("githubClient.GrantWritePermission> invitation response: %v", string(body))

	// Response when a new invitation is created
	if resp.StatusCode == 201 {
		invit := RepositoryInvitation{}
		if err := json.Unmarshal(body, &invit); err != nil {
			log.Warning(ctx, "githubClient.GrantWritePermission> unable to unmarshal invitation %s", err)
			return err
		}

		// Accept the invitation
		url := fmt.Sprintf("/user/repository_invitations/%d", invit.ID)
		resp, err := g.patch(url, "", nil, &postOptions{asUser: true})
		if err != nil {
			log.Warning(ctx, "githubClient.GrantWritePermission> Error (%s) %s", url, err)
			return err
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		_ = resp.Body.Close()
		log.Debug("githubClient.GrantWritePermission> accept invitation response: %v", string(body))

		// All is fine
		if resp.StatusCode == 204 {
			return nil
		}

		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}
