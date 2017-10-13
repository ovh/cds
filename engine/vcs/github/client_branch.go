package github

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Branches returns list of branches for a repo
// https://developer.github.com/v3/repos/branches/#list-branches
func (g *githubClient) Branches(fullname string) ([]sdk.VCSBranch, error) {
	var branches = []Branch{}
	var nextPage = "/repos/" + fullname + "/branches"

	repo, err := g.repoByFullname(fullname)
	if err != nil {
		return nil, err
	}

	for {
		if nextPage != "" {
			status, body, headers, err := g.get(nextPage)
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
				g.Cache.Get(cache.Key("vcs", "github", "branches", g.OAuthToken, "/repos/"+fullname+"/branches"), &branches)
				break
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
	g.Cache.SetWithTTL(cache.Key("vcs", "github", "branches", g.OAuthToken, "/repos/"+fullname+"/branches"), branches, 61*60)

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
func (g *githubClient) Branch(fullname, theBranch string) (*sdk.VCSBranch, error) {
	cacheBranchKey := cache.Key("vcs", "github", "branches", g.OAuthToken, "/repos/"+fullname+"/branch/"+theBranch)
	repo, err := g.repoByFullname(fullname)
	if err != nil {
		return nil, err
	}

	url := "/repos/" + fullname + "/branches/" + theBranch
	status, body, _, err := g.get(url)
	if err != nil {
		g.Cache.Delete(cacheBranchKey)
		return nil, err
	}
	if status >= 400 {
		g.Cache.Delete(cacheBranchKey)
		return nil, sdk.NewError(sdk.ErrUnknownError, errorAPI(body))
	}

	//Github may return 304 status because we are using conditional request with ETag based headers
	var branch Branch
	if status == http.StatusNotModified {
		//If repos aren't updated, lets get them from cache
		if !g.Cache.Get(cacheBranchKey, &branch) {
			log.Error("Unable to get branch (%s) from the cache", cacheBranchKey)
		}

	} else {
		if err := json.Unmarshal(body, &branch); err != nil {
			log.Warning("githubClient.Branch> Unable to parse github branch: %s", err)
			return nil, err
		}
	}

	if branch.Name == "" {
		log.Warning("githubClient.Branch> Cannot find branch %s: %v", branch, theBranch)
		g.Cache.Delete(cacheBranchKey)
		return nil, fmt.Errorf("githubClient.Branch > Cannot find branch %s", theBranch)
	}

	//Put the body on cache for one hour and one minute
	g.Cache.SetWithTTL(cache.Key("vcs", "github", "branches", g.OAuthToken, "/repos/"+fullname+"/branch/"+theBranch), branch, 61*60)

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
