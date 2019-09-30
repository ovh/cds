package bitbucketcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Repos list repositories that are accessible to the authenticated user
func (client *bitbucketcloudClient) Repos(ctx context.Context) ([]sdk.VCSRepo, error) {
	var repos []Repository

	user, err := client.CurrentUser(ctx)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot load user info")
	}
	reposForUser, err := client.reposForUser(ctx, user.Username)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot load repositories for user %s", user.Username)
	}
	repos = append(repos, reposForUser...)

	teams, err := client.Teams(ctx)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot teams info")
	}

	for _, team := range teams.Values {
		reposForTeam, err := client.reposForUser(ctx, team.Username)
		if err != nil {
			return nil, sdk.WrapError(err, "cannot load repositories for team %s", team.Username)
		}
		repos = append(repos, reposForTeam...)
	}

	responseRepos := make([]sdk.VCSRepo, 0, len(repos))
	for _, repo := range repos {
		r := sdk.VCSRepo{
			ID:           repo.UUID,
			Name:         repo.Name,
			Slug:         repo.Slug,
			Fullname:     repo.FullName,
			URL:          repo.Links.HTML.Href,
			HTTPCloneURL: repo.Links.Clone[0].Href,
			SSHCloneURL:  repo.Links.Clone[1].Href,
		}
		responseRepos = append(responseRepos, r)
	}

	return responseRepos, nil
}

// reposForUser list repositories that are accessible for an user
func (client *bitbucketcloudClient) reposForUser(ctx context.Context, username string) ([]Repository, error) {
	var repos []Repository
	path := fmt.Sprintf("/repositories/%s", username)
	params := url.Values{}
	params.Set("pagelen", "100")
	params.Set("role", "member")
	nextPage := 1
	for {
		if nextPage != 1 {
			params.Set("page", fmt.Sprintf("%d", nextPage))
		}

		var response Repositories
		if err := client.do(ctx, "GET", "core", path, params, nil, &response); err != nil {
			return nil, sdk.WrapError(err, "Unable to get repos")
		}
		if cap(repos) == 0 {
			repos = make([]Repository, 0, response.Size)
		}

		repos = append(repos, response.Values...)

		if response.Next == "" {
			break
		} else {
			nextPage++
		}
	}

	return repos, nil
}

// RepoByFullname Get only one repo
func (client *bitbucketcloudClient) RepoByFullname(ctx context.Context, fullname string) (sdk.VCSRepo, error) {
	repo, err := client.repoByFullname(fullname)
	if err != nil {
		return sdk.VCSRepo{}, err
	}

	if repo.UUID == "" {
		return sdk.VCSRepo{}, err
	}

	r := sdk.VCSRepo{
		ID:           repo.UUID,
		Name:         repo.Name,
		Slug:         repo.Slug,
		Fullname:     repo.FullName,
		URL:          repo.Links.HTML.Href,
		HTTPCloneURL: repo.Links.Clone[0].Href,
		SSHCloneURL:  repo.Links.Clone[1].Href,
	}
	return r, nil
}

func (client *bitbucketcloudClient) repoByFullname(fullname string) (Repository, error) {
	var repo Repository
	url := fmt.Sprintf("/repositories/%s", fullname)
	status, body, _, err := client.get(url)
	if err != nil {
		log.Warning("bitbucketcloudClient.Repos> Error %s", err)
		return repo, err
	}
	if status >= 400 {
		return repo, sdk.NewError(sdk.ErrRepoNotFound, errorAPI(body))
	}

	if err := json.Unmarshal(body, &repo); err != nil {
		return repo, sdk.WrapError(err, "Unable to parse github repository")
	}

	return repo, nil
}

func (client *bitbucketcloudClient) GrantWritePermission(ctx context.Context, fullname string) error {
	return sdk.ErrNotImplemented
}
