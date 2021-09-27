package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/ovh/cds/sdk"
)

// Release Create a release Github
func (g *githubClient) Release(ctx context.Context, fullname string, tagName string, title string, releaseNote string) (*sdk.VCSRelease, error) {
	var url = "/repos/" + fullname + "/releases"

	req := ReleaseRequest{
		TagName: tagName,
		Name:    title,
		Body:    releaseNote,
	}
	b, err := json.Marshal(req)
	if err != nil {
		return nil, sdk.WrapError(err, "Cannot marshal body %+v", req)
	}

	res, err := g.post(ctx, url, "application/json", bytes.NewBuffer(b), nil, nil)
	if err != nil {
		return nil, sdk.WrapError(err, "Cannot create release on github")
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, sdk.WrapError(err, "Cannot read release response")
	}

	if res.StatusCode != 201 {
		return nil, sdk.WrapError(fmt.Errorf("github.Release >Unable to create release on github. Url : %s Status code : %d - Body: %s", url, res.StatusCode, body), "")
	}

	var response ReleaseResponse
	if err := sdk.JSONUnmarshal(body, &response); err != nil {
		return nil, sdk.WrapError(err, "Cannot unmarshal response: %s", string(body))
	}

	release := &sdk.VCSRelease{
		ID:        response.ID,
		UploadURL: response.UploadURL,
	}

	return release, nil
}

// UploadReleaseFile Attach a file into the release
func (g *githubClient) UploadReleaseFile(ctx context.Context, _ string, _ string, uploadURL string, artifactName string, r io.Reader, fileLength int) error {
	var url = strings.Split(uploadURL, "{")[0] + "?name=" + artifactName
	headers := map[string]string{
		"Content-Length": strconv.Itoa(fileLength),
	}

	res, err := g.post(ctx, url, "application/octet-stream", r, headers, &postOptions{skipDefaultBaseURL: true})
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 201 {
		errorMsg, _ := ioutil.ReadAll(res.Body)
		return sdk.WrapError(fmt.Errorf("github.Release >Unable to upload file on release. Url : %s - Status code : %d: %s", url, res.StatusCode, string(errorMsg)), "")
	}

	return nil
}
