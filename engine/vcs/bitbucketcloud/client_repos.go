package bitbucketcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

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
	path := fmt.Sprintf("/repositories/%s", user.Username)
	params := url.Values{}
	params.Set("pagelen", "100")
	params.Set("role", "contributor")
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
	owner := strings.SplitN(fullname, "/", 2)[0]
	if client.username == "" || owner == client.username {
		log.Debug("bitbucketcloudClient.GrantWritePermission> nothing to do")
		return nil
	}

	// url := "/repos/" + fullname + "/collaborators/" + client.username + "?permission=push"
	// resp, err := client.put(url, "application/json", nil, nil)
	// if err != nil {
	// 	log.Warning("bitbucketcloudClient.GrantWritePermission> Error (%s) %s", url, err)
	// 	return err
	// }

	// // Response when person is already a collaborator
	// if resp.StatusCode == 204 {
	// 	log.Info("bitbucketcloudClient.GrantWritePermission> %s is already a collaborator", client.username)
	// 	return nil
	// }

	// body, err := ioutil.ReadAll(resp.Body)
	// if err != nil {
	// 	return err
	// }
	// defer resp.Body.Close() // nolint

	// log.Debug("bitbucketcloudClient.GrantWritePermission> invitation response: %v", string(body))

	// // Response when a new invitation is created
	// if resp.StatusCode == 201 {
	// 	invit := RepositoryInvitation{}
	// 	if err := json.Unmarshal(body, &invit); err != nil {
	// 		log.Warning("bitbucketcloudClient.GrantWritePermission> unable to unmarshal invitation %s", err)
	// 		return err
	// 	}

	// 	// Accept the invitation
	// 	url := fmt.Sprintf("/user/repository_invitations/%d", invit.ID)
	// 	resp, err := client.patch(url, &postOptions{asUser: true})
	// 	if err != nil {
	// 		log.Warning("bitbucketcloudClient.GrantWritePermission> Error (%s) %s", url, err)
	// 		return err
	// 	}
	// 	body, err := ioutil.ReadAll(resp.Body)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	_ = resp.Body.Close()
	// 	log.Debug("bitbucketcloudClient.GrantWritePermission> accept invitation response: %v", string(body))

	// 	// All is fine
	// 	if resp.StatusCode == 204 {
	// 		return nil
	// 	}

	// 	return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	// }

	return sdk.ErrNotImplemented
}
