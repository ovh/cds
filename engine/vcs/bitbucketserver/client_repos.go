package bitbucketserver

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func (b *bitbucketClient) Repos(ctx context.Context) ([]sdk.VCSRepo, error) {
	ctx, end := telemetry.Span(ctx, "bitbucketserver.Repos")
	defer end()

	bbRepos := []Repo{}

	path := "/repos"
	params := url.Values{}
	params.Set("limit", "200")
	nextPage := 0
	for {
		if ctx.Err() != nil {
			break
		}

		if nextPage != 0 {
			params.Set("start", fmt.Sprintf("%d", nextPage))
		}

		var response Response
		if err := b.do(ctx, "GET", "core", path, params, nil, &response); err != nil {
			return nil, sdk.WrapError(err, "Unable to get repos")
		}

		bbRepos = append(bbRepos, response.Values...)

		if response.IsLastPage {
			break
		} else {
			nextPage = response.NextPageStart
		}
	}

	repos := []sdk.VCSRepo{}
	for _, r := range bbRepos {
		repos = append(repos, b.ToVCSRepo(r))
	}
	return repos, nil
}

func (b *bitbucketClient) RepoByFullname(ctx context.Context, fullname string) (sdk.VCSRepo, error) {
	t := strings.SplitN(fullname, "/", 2)
	r := Repo{}
	path := fmt.Sprintf("/projects/%s/repos/%s", t[0], t[1])

	if err := b.do(ctx, "GET", "core", path, nil, nil, &r); err != nil {
		return sdk.VCSRepo{}, sdk.WrapError(err, "Unable to get repo")
	}

	return b.ToVCSRepo(r), nil
}

func (b *bitbucketClient) UserHasWritePermission(ctx context.Context, repo string) (bool, error) {
	if b.username == "" {
		return false, sdk.WrapError(sdk.ErrUserNotFound, "No user found in configuration")
	}

	project, slug, err := getRepo(repo)
	if err != nil {
		return false, sdk.WithStack(err)
	}
	path := fmt.Sprintf("/projects/%s/repos/%s/permissions/users", project, slug)
	params := url.Values{}
	params.Add("filter", b.username)

	var response UsersPermissionResponse
	if err := b.do(ctx, "GET", "core", path, params, nil, &response); err != nil {
		return false, sdk.WithStack(err)
	}

	for _, v := range response.Values {
		if v.User.Slug == b.username {
			return v.Permission == "REPO_WRITE" || v.Permission == "REPO_ADMIN", nil
		}
	}
	return false, nil
}

func (b *bitbucketClient) ToVCSRepo(repo Repo) sdk.VCSRepo {
	var webURL, sshURL, httpURL string
	if repo.Links != nil {
		for _, c := range repo.Links.Clone {
			if c.Name == "http" {
				httpURL = c.URL
			}
			if c.Name == "ssh" {
				sshURL = c.URL
			}
		}
		for _, s := range repo.Links.Self {
			webURL = s.URL
			break
		}
	}

	return sdk.VCSRepo{
		Name:            repo.Name,
		Slug:            repo.Slug,
		Fullname:        fmt.Sprintf("%s/%s", repo.Project.Key, repo.Slug),
		URL:             webURL,
		URLCommitFormat: strings.TrimSuffix(webURL, "/browse") + "/commits/%s",
		URLTagFormat:    webURL + "?at=refs%%2Ftags%%2F%s",
		URLBranchFormat: webURL + "?at=refs%%2Fheads%%2F%s",
		HTTPCloneURL:    httpURL,
		SSHCloneURL:     sshURL,
	}
}
