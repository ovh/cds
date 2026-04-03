package forgejo

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/ovh/cds/sdk"
)

func (f *forgejoClient) Commits(ctx context.Context, fullname, branch, since, until string) ([]sdk.VCSCommit, error) {
	owner, repo, err := getRepo(fullname)
	if err != nil {
		return nil, err
	}

	const maxResults = 50
	const pageSize = 50
	basePath := fmt.Sprintf("/repos/%s/%s/commits", owner, repo)
	if branch != "" {
		basePath += "?sha=" + url.QueryEscape(branch)
	}

	var allCommits []*Commit
	for page := 1; ; page++ {
		var pageCommits []*Commit
		apiPath := buildPaginatedPath(basePath, ListOptions{Page: page, PageSize: pageSize})
		if _, err := f.client.get(ctx, apiPath, &pageCommits); err != nil {
			return nil, sdk.WrapError(err, "cannot load commits on branch %s", branch)
		}
		allCommits = append(allCommits, pageCommits...)
		if len(pageCommits) < pageSize || len(allCommits) >= maxResults {
			break
		}
	}
	if len(allCommits) > maxResults {
		allCommits = allCommits[:maxResults]
	}

	commitsResult := make([]sdk.VCSCommit, 0, len(allCommits))
	for _, c := range allCommits {
		commitsResult = append(commitsResult, f.toVCSCommit(c))
	}

	return commitsResult, nil
}

func (f *forgejoClient) Commit(ctx context.Context, fullname, hash string) (sdk.VCSCommit, error) {
	owner, repo, err := getRepo(fullname)
	if err != nil {
		return sdk.VCSCommit{}, err
	}

	var commit Commit
	path := fmt.Sprintf("/repos/%s/%s/git/commits/%s", owner, repo, url.PathEscape(hash))
	if _, err = f.client.get(ctx, path, &commit); err != nil {
		return sdk.VCSCommit{}, err
	}
	if commit.RepoCommit == nil {
		return sdk.VCSCommit{}, sdk.NewErrorFrom(sdk.ErrNotFound, "commit %s data not found", hash)
	}
	return f.toVCSCommit(&commit), nil
}

func (f *forgejoClient) CommitsBetweenRefs(ctx context.Context, repo, base, head string) ([]sdk.VCSCommit, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}

func (f *forgejoClient) toVCSCommit(commit *Commit) sdk.VCSCommit {
	vcsCommit := sdk.VCSCommit{
		Signature: "",
		Verified:  false,
		Hash:      commit.SHA,
		URL:       commit.HTMLURL,
		Timestamp: commit.Created.Unix() * 1000,
	}
	if commit.RepoCommit != nil {
		vcsCommit.Message = commit.RepoCommit.Message
		if commit.RepoCommit.Verification != nil && commit.RepoCommit.Verification.Signature != "" {
			vcsCommit.Signature = commit.RepoCommit.Verification.Signature
			vcsCommit.Verified = commit.RepoCommit.Verification.Verified
		}
	}
	if commit.Author != nil {
		vcsCommit.Author = sdk.VCSAuthor{
			Name:        commit.Author.UserName,
			Avatar:      commit.Author.AvatarURL,
			Email:       commit.Author.Email,
			DisplayName: commit.Author.FullName,
			Slug:        commit.Author.UserName,
			ID:          strconv.FormatInt(commit.Author.ID, 10),
		}
	}
	if commit.RepoCommit != nil && commit.RepoCommit.Committer != nil {
		vcsCommit.Committer = sdk.VCSAuthor{
			Name:  commit.RepoCommit.Committer.Name,
			Email: commit.RepoCommit.Committer.Email,
			Slug:  commit.RepoCommit.Committer.Name,
		}
	}

	return vcsCommit
}
