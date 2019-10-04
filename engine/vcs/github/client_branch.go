package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Branches returns list of branches for a repo
// https://developer.github.com/v3/repos/branches/#list-branches
func (g *githubClient) Branches(ctx context.Context, fullname string) ([]sdk.VCSBranch, error) {
	var branches = []Branch{}
	var nextPage = "/repos/" + fullname + "/branches"

	repo, err := g.repoByFullname(fullname)
	if err != nil {
		return nil, err
	}

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
				log.Warning("githubClient.Branches> Error %s", err)
				return nil, err
			}
			if status >= 400 {
				return nil, sdk.NewError(sdk.ErrUnknownError, errorAPI(body))
			}
			nextBranches := []Branch{}

			//Github may return 304 status because we are using conditional request with ETag based headers
			if status == http.StatusNotModified {
				//If repos aren't updated, lets get them from cache
				k := cache.Key("vcs", "github", "branches", g.OAuthToken, "/repos/"+fullname+"/branches")
				_, err := g.Cache.Get(k, &branches)
				if err != nil {
					log.Error("cannot get from cache %s: %v", k, err)
				}
				if len(branches) != 0 || attempt > 5 {
					//We found branches, let's exit the loop
					break
				}
				//If we did not found any branch in cache, let's retry (same nextPage) without etag
				noEtag = true
				continue
			} else {
				if err := json.Unmarshal(body, &nextBranches); err != nil {
					log.Warning("githubClient.Branches> Unable to parse github branches: %s", err)
					return nil, err
				}
			}

			branches = append(branches, nextBranches...)
			nextPage = getNextPage(headers)
		} else {
			break
		}
	}

	//Put the body on cache for one hour and one minute
	k := cache.Key("vcs", "github", "branches", g.OAuthToken, "/repos/"+fullname+"/branches")
	if err := g.Cache.SetWithTTL(k, branches, 61*60); err != nil {
		log.Error("cannot SetWithTTL: %s: %v", k, err)
	}

	branchesResult := []sdk.VCSBranch{}
	for _, b := range branches {
		branch := sdk.VCSBranch{
			DisplayID:    b.Name,
			ID:           b.Name,
			LatestCommit: b.Commit.Sha,
			Default:      b.Name == repo.DefaultBranch,
		}
		for _, p := range b.Commit.Parents {
			branch.Parents = append(branch.Parents, p.Sha)
		}
		branchesResult = append(branchesResult, branch)
	}

	return branchesResult, nil
}

// Branch returns only detail of a branch
func (g *githubClient) Branch(ctx context.Context, fullname, theBranch string) (*sdk.VCSBranch, error) {
	cacheBranchKey := cache.Key("vcs", "github", "branches", g.OAuthToken, "/repos/"+fullname+"/branch/"+theBranch)
	repo, err := g.repoByFullname(fullname)
	if err != nil {
		return nil, err
	}

	url := "/repos/" + fullname + "/branches/" + theBranch
	status, body, _, err := g.get(url)
	if err != nil {
		if err := g.Cache.Delete(cacheBranchKey); err != nil {
			log.Error("githubClient.Branch> unable to delete cache key %v: %v", cacheBranchKey, err)
		}
		return nil, err
	}
	if status >= 400 {
		if err := g.Cache.Delete(cacheBranchKey); err != nil {
			log.Error("githubClient.Branch> unable to delete cache key %v: %v", cacheBranchKey, err)
		}
		return nil, sdk.NewError(sdk.ErrUnknownError, errorAPI(body))
	}

	//Github may return 304 status because we are using conditional request with ETag based headers
	var branch Branch
	if status == http.StatusNotModified {
		//If repos aren't updated, lets get them from cache
		find, err := g.Cache.Get(cacheBranchKey, &branch)
		if err != nil {
			log.Error("cannot get from cache %s: %v", cacheBranchKey, err)
		}
		if !find {
			log.Error("Unable to get branch (%s) from the cache", cacheBranchKey)
		}
	} else {
		if err := json.Unmarshal(body, &branch); err != nil {
			log.Warning("githubClient.Branch> Unable to parse github branch: %s", err)
			return nil, err
		}
	}

	if branch.Name == "" {
		log.Warning("githubClient.Branch> Cannot find branch %v: %s", branch, theBranch)
		if err := g.Cache.Delete(cacheBranchKey); err != nil {
			log.Error("githubClient.Branch> unable to delete cache key %v: %v", cacheBranchKey, err)
		}
		return nil, fmt.Errorf("githubClient.Branch > Cannot find branch %s", theBranch)
	}

	//Put the body on cache for one hour and one minute
	k := cache.Key("vcs", "github", "branches", g.OAuthToken, "/repos/"+fullname+"/branch/"+theBranch)
	if err := g.Cache.SetWithTTL(k, branch, 61*60); err != nil {
		log.Error("cannot SetWithTTL: %s: %v", k, err)
	}

	branchResult := &sdk.VCSBranch{
		DisplayID:    branch.Name,
		ID:           branch.Name,
		LatestCommit: branch.Commit.Sha,
		Default:      branch.Name == repo.DefaultBranch,
	}

	if branch.Commit.Sha != "" {
		for _, p := range branch.Commit.Parents {
			branchResult.Parents = append(branchResult.Parents, p.Sha)
		}
	}

	return branchResult, nil
}
