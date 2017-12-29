package cdsclient

import (
	"context"
	"fmt"

	"github.com/google/go-github/github"

	"github.com/ovh/cds/sdk"
)

func (c *client) Download() ([]sdk.Download, error) {
	var res []sdk.Download
	if _, err := c.GetJSON("/download", &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *client) DownloadURLFromAPI(name, os, arch string) string {
	return fmt.Sprintf("%s/download/%s/%s/%s", c.APIURL(), name, os, arch)
}

func (c *client) DownloadURLFromGithub(ctx context.Context, name, os, arch string) (string, error) {
	client := github.NewClient(nil)
	release, resp, err := client.Repositories.GetLatestRelease(ctx, "ovh", "cds")
	if err != nil {
		return "", fmt.Errorf("Repositories.GetLatestRelease returned error:%v response:%v", err, resp.Body)
	}

	if len(release.Assets) > 0 {
		for _, asset := range release.Assets {
			if *asset.Name == sdk.GetArtifactFilename(name, os, arch) {
				return *asset.BrowserDownloadURL, nil
			}
		}
	}

	text := "Invalid Artifacts on latest release. Please try again in few minutes.\n"
	text += "If the problem persists, please open an issue on https://github.com/ovh/cds/issues\n"
	return "", fmt.Errorf(text)
}
