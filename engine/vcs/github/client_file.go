package github

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func (g *githubClient) ListContent(ctx context.Context, repo string, commit, dir string) ([]sdk.VCSContent, error) {
	url := fmt.Sprintf("/repos/%s/contents/%s?ref=%s", repo, dir, commit)
	status, body, _, err := g.get(ctx, url)
	if err != nil {
		log.Warn(ctx, "githubClient.ListContent> Error %s", err)
		return nil, err
	}
	if status >= 400 {
		return nil, sdk.NewError(sdk.ErrRepoNotFound, errorAPI(body))
	}
	var c []Content

	//Github may return 304 status because we are using conditional request with ETag based headers
	if status == http.StatusNotModified {
		//If repo isn't updated, lets get them from cache
		k := cache.Key("vcs", "github", "content", sdk.Hash512(g.OAuthToken+g.username), url)
		if _, err := g.Cache.Get(ctx, k, &c); err != nil {
			log.Error(ctx, "cannot get from cache %s: %v", k, err)
		}
	} else {
		if err := sdk.JSONUnmarshal(body, &c); err != nil {
			log.Warn(ctx, "githubClient.ListContent> Unable to parse github content: %s", err)
			return nil, err
		}
		//Put the body on cache for one hour and one minute
		k := cache.Key("vcs", "github", "content", sdk.Hash512(g.OAuthToken+g.username), url)
		if err := g.Cache.SetWithTTL(ctx, k, c, 61*60); err != nil {
			log.Error(ctx, "cannot SetWithTTL: %s: %v", k, err)
		}
	}
	contents := make([]sdk.VCSContent, 0, len(c))
	for _, co := range c {
		contents = append(contents, g.ToVCSContent(co))
	}
	return contents, nil

}

func (g *githubClient) GetContent(ctx context.Context, repo string, commit, filePath string) (sdk.VCSContent, error) {
	url := fmt.Sprintf("/repos/%s/contents/%s?ref=%s", repo, filePath, commit)
	status, body, _, err := g.get(ctx, url)
	if err != nil {
		log.Warn(ctx, "githubClient.ListContent> Error %s", err)
		return sdk.VCSContent{}, err
	}
	if status >= 400 {
		return sdk.VCSContent{}, sdk.NewError(sdk.ErrRepoNotFound, errorAPI(body))
	}
	var c Content

	//Github may return 304 status because we are using conditional request with ETag based headers
	if status == http.StatusNotModified {
		//If repo isn't updated, lets get them from cache
		k := cache.Key("vcs", "github", "content", sdk.Hash512(g.OAuthToken+g.username), url)
		if _, err := g.Cache.Get(ctx, k, &c); err != nil {
			log.Error(ctx, "cannot get from cache %s: %v", k, err)
		}
	} else {
		if err := sdk.JSONUnmarshal(body, &c); err != nil {
			log.Warn(ctx, "githubClient.ListContent> Unable to parse github content: %s", err)
			return sdk.VCSContent{}, err
		}
		//Put the body on cache for one hour and one minute
		k := cache.Key("vcs", "github", "content", sdk.Hash512(g.OAuthToken+g.username), url)
		if err := g.Cache.SetWithTTL(ctx, k, c, 61*60); err != nil {
			log.Error(ctx, "cannot SetWithTTL: %s: %v", k, err)
		}
	}
	return g.ToVCSContent(c), nil
}

func (g *githubClient) GetArchive(ctx context.Context, repo, dir, format, commit string) (io.Reader, http.Header, error) {
	return nil, nil, sdk.WithStack(sdk.ErrNotImplemented)
}

func (g *githubClient) ToVCSContent(c Content) sdk.VCSContent {
	return sdk.VCSContent{
		Content:     c.Content,
		Name:        c.Name,
		IsFile:      c.Type == "file",
		IsDirectory: c.Type == "dir",
	}
}
