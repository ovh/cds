package forgejo

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ovh/cds/sdk"
)

// Tags retrieve tags
func (c *forgejoClient) Tags(ctx context.Context, fullname string) ([]sdk.VCSTag, error) {
	owner, repo, err := getRepo(fullname)
	if err != nil {
		return nil, err
	}

	const maxResults = 200
	const pageSize = 50
	basePath := fmt.Sprintf("/repos/%s/%s/tags", owner, repo)

	var allTags []*Tag
	for page := 1; ; page++ {
		var pageTags []*Tag
		apiPath := buildPaginatedPath(basePath, ListOptions{Page: page, PageSize: pageSize})
		if _, err := c.client.get(ctx, apiPath, &pageTags); err != nil {
			return nil, err
		}
		allTags = append(allTags, pageTags...)
		if len(pageTags) < pageSize || len(allTags) >= maxResults {
			break
		}
	}
	if len(allTags) > maxResults {
		allTags = allTags[:maxResults]
	}

	ret := make([]sdk.VCSTag, 0, len(allTags))
	for _, tag := range allTags {
		vcsTag := sdk.VCSTag{
			Tag: tag.Name,
			Sha: tag.ID,
		}
		if tag.Commit != nil {
			vcsTag.Hash = tag.Commit.SHA
		}
		ret = append(ret, vcsTag)
	}

	return ret, nil
}

func (c *forgejoClient) Tag(ctx context.Context, fullname string, tagName string) (sdk.VCSTag, error) {
	owner, repo, err := getRepo(fullname)
	if err != nil {
		return sdk.VCSTag{}, err
	}

	// Step 1: Get the tag to obtain its sha and commit sha
	var tag Tag
	apiPath := fmt.Sprintf("/repos/%s/%s/tags/%s", owner, repo, url.PathEscape(tagName))
	if _, err := c.client.get(ctx, apiPath, &tag); err != nil {
		return sdk.VCSTag{}, err
	}

	vcsTag := sdk.VCSTag{
		Tag:     tag.Name,
		Sha:     tag.ID,
		Message: tag.Message,
	}
	if tag.Commit != nil {
		vcsTag.Hash = tag.Commit.SHA
	}

	// Step 2: Try to get the annotated tag object via git/tags/{sha} for signature info
	var annotated AnnotatedTag
	gitTagPath := fmt.Sprintf("/repos/%s/%s/git/tags/%s", owner, repo, url.PathEscape(tag.ID))
	if resp, err := c.client.get(ctx, gitTagPath, &annotated); err == nil {
		// Annotated tag found
		if annotated.Tagger != nil {
			vcsTag.Tagger = sdk.VCSAuthor{
				Name:        annotated.Tagger.Name,
				Email:       annotated.Tagger.Email,
				DisplayName: annotated.Tagger.Name,
			}
		}
		if annotated.Message != "" {
			vcsTag.Message = annotated.Message
		}
		if annotated.Object != nil {
			vcsTag.Hash = annotated.Object.SHA
		}
		if annotated.Verification != nil {
			vcsTag.Signature = annotated.Verification.Signature
			vcsTag.Verified = annotated.Verification.Verified
		}
		return vcsTag, nil
	} else if resp != nil && resp.StatusCode == 404 {
		// Lightweight tag: no annotated tag object, get signature from the commit
		commitHash := vcsTag.Hash
		if commitHash == "" {
			commitHash = tag.ID
		}
		commit, err := c.Commit(ctx, fullname, commitHash)
		if err != nil {
			return sdk.VCSTag{}, fmt.Errorf("unable to get commit %s for tag %s: %w", commitHash, tagName, err)
		}
		vcsTag.Signature = commit.Signature
		vcsTag.Verified = commit.Verified
		return vcsTag, nil
	} else {
		// Unexpected error on git/tags, return tag without signature
		return vcsTag, nil
	}
}
