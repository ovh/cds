package gitea

import (
	"context"
	"io"
	"net/http"

	gg "code.gitea.io/sdk/gitea"

	"github.com/ovh/cds/sdk"
)

func (g *giteaClient) ListContent(_ context.Context, repo string, commit, dir string, _, _ string) ([]sdk.VCSContent, error) {
	owner, repoName, err := getRepo(repo)
	if err != nil {
		return nil, err
	}

	contents, httpResp, err := g.client.ListContents(owner, repoName, commit, dir)
	if err != nil {
		if httpResp.StatusCode == 404 {
			return []sdk.VCSContent{}, nil
		}
		return nil, err
	}
	resp := make([]sdk.VCSContent, 0, len(contents))
	for _, c := range contents {
		resp = append(resp, g.ToVCSContent(c))
	}
	return resp, nil
}

func (g *giteaClient) GetContent(_ context.Context, repo string, commit, filePath string) (sdk.VCSContent, error) {
	owner, repoName, err := getRepo(repo)
	if err != nil {
		return sdk.VCSContent{}, err
	}

	content, _, err := g.client.GetContents(owner, repoName, commit, filePath)
	if err != nil {
		return sdk.VCSContent{}, err
	}
	return g.ToVCSContent(content), nil
}

func (g *giteaClient) GetArchive(ctx context.Context, repo string, dir string, format string, commit string) (io.Reader, http.Header, error) {
	return nil, nil, sdk.WithStack(sdk.ErrNotImplemented)
}

func (g *giteaClient) ToVCSContent(content *gg.ContentsResponse) sdk.VCSContent {
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
