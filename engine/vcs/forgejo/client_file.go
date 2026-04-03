package forgejo

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (f *forgejoClient) ListContent(ctx context.Context, repo string, commit, dir string, _, _ string) ([]sdk.VCSContent, error) {
	owner, repoName, err := getRepo(repo)
	if err != nil {
		return nil, err
	}

	var contents []*ContentsResponse
	apiPath := fmt.Sprintf("/repos/%s/%s/contents/%s?ref=%s", owner, repoName, url.PathEscape(dir), url.QueryEscape(commit))
	httpResp, err := f.client.get(ctx, apiPath, &contents)
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			return []sdk.VCSContent{}, nil
		}
		return nil, err
	}
	resp := make([]sdk.VCSContent, 0, len(contents))
	for _, c := range contents {
		resp = append(resp, f.ToVCSContent(c))
	}
	return resp, nil
}

func (f *forgejoClient) GetContent(ctx context.Context, repo string, commit, filePath string) (sdk.VCSContent, error) {
	owner, repoName, err := getRepo(repo)
	if err != nil {
		return sdk.VCSContent{}, err
	}

	var content ContentsResponse
	apiPath := fmt.Sprintf("/repos/%s/%s/contents/%s?ref=%s", owner, repoName, url.PathEscape(filePath), url.QueryEscape(commit))
	if _, err = f.client.get(ctx, apiPath, &content); err != nil {
		return sdk.VCSContent{}, err
	}
	return f.ToVCSContent(&content), nil
}

func (f *forgejoClient) GetArchive(ctx context.Context, repo string, dir string, format string, commit string) (io.Reader, http.Header, error) {
	return nil, nil, sdk.WithStack(sdk.ErrNotImplemented)
}

func (f *forgejoClient) ToVCSContent(content *ContentsResponse) sdk.VCSContent {
	var fileContent string
	if content.Content != nil {
		fileContent = *content.Content
	}
	return sdk.VCSContent{
		Name:        content.Name,
		Content:     fileContent,
		IsDirectory: content.Type == "dir",
		IsFile:      content.Type == "file",
	}
}
