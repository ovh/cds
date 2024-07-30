package bitbucketserver

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func (b *bitbucketClient) ListContent(ctx context.Context, repo string, commit, dir string, offset, limit string) ([]sdk.VCSContent, error) {
	_, end := telemetry.Span(ctx, "bitbucketserver.GetArchive", telemetry.Tag(telemetry.TagRepository, repo))
	defer end()

	t := strings.Split(repo, "/")
	if len(t) != 2 {
		return nil, sdk.WithStack(sdk.ErrRepoNotFound)
	}

	path := fmt.Sprintf("/projects/%s/repos/%s/files/%s", t[0], t[1], dir)
	params := url.Values{}
	params.Set("at", commit)
	params.Set("start", offset)
	params.Set("limit", limit)

	var contents ListContentResponse
	if err := b.do(ctx, http.MethodGet, "core", path, params, nil, &contents, Options{DisableCache: true}); err != nil {
		return nil, err
	}

	vcsContents := make([]sdk.VCSContent, 0)
	for _, c := range contents.Values {
		parts := strings.Split(c, "/")
		for i, p := range parts {
			var isFile, isDirectory bool

			if i == len(parts)-1 {
				isFile = true
			} else {
				isDirectory = true
			}
			vcsContents = append(vcsContents, sdk.VCSContent{
				Name:        p,
				IsFile:      isFile,
				IsDirectory: isDirectory,
			})
		}
	}

	return vcsContents, nil
}

func (b *bitbucketClient) GetContent(ctx context.Context, repo string, commit, file string) (sdk.VCSContent, error) {
	return sdk.VCSContent{}, sdk.WithStack(sdk.ErrNotImplemented)
}

func (b *bitbucketClient) GetArchive(ctx context.Context, repo string, dir string, format string, commit string) (io.Reader, http.Header, error) {
	_, end := telemetry.Span(ctx, "bitbucketserver.GetArchive", telemetry.Tag(telemetry.TagRepository, repo))
	defer end()

	t := strings.Split(repo, "/")
	if len(t) != 2 {
		return nil, nil, sdk.WithStack(sdk.ErrRepoNotFound)
	}

	path := fmt.Sprintf("/projects/%s/repos/%s/archive", t[0], t[1])
	params := url.Values{}
	params.Set("path", dir)
	params.Set("format", format)

	if commit != "" {
		params.Set("at", commit)
	}
	return b.stream(ctx, "GET", "core", path, params, nil)
}
