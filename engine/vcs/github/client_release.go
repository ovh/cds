package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/ovh/cds/sdk"
)

// Release Create a release Github
func (g *githubClient) Release(fullname string, tagName string, title string, releaseNote string) (*sdk.VCSRelease, error) {
	var url = "/repos/" + fullname + "/releases"

	req := ReleaseRequest{
		TagName: tagName,
		Name:    title,
		Body:    releaseNote,
	}
	b, err := json.Marshal(req)
	if err != nil {
		return nil, sdk.WrapError(err, "github.Release > Cannot marshal body %+v", req)
	}

	res, err := g.post(url, "application/json", bytes.NewBuffer(b), false)
	if err != nil {
		return nil, sdk.WrapError(err, "github.Release > Cannot create release on github")
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, sdk.WrapError(err, "github.Release > Cannot read release response")
	}

	if res.StatusCode != 201 {
		return nil, sdk.WrapError(fmt.Errorf("github.Release >Unable to create release on github. Url : %s Status code : %d - Body: %s", url, res.StatusCode, body), "")
	}

	var response ReleaseResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, sdk.WrapError(err, "github.Release>  Cannot unmarshal response: %s", string(body))
	}

	release := &sdk.VCSRelease{
		ID:        response.ID,
		UploadURL: response.UploadURL,
	}

	return release, nil
}

// UploadReleaseFile Attach a file into the release
func (g *githubClient) UploadReleaseFile(repo string, release *sdk.VCSRelease, runArtifact sdk.WorkflowNodeRunArtifact, buf *bytes.Buffer) error {
	var url = strings.Split(release.UploadURL, "{")[0] + "?name=" + runArtifact.Name
	res, err := g.post(url, "application/octet-stream", buf, true)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 201 {
		return sdk.WrapError(fmt.Errorf("github.Release >Unable to upload file on release. Url : %s - Status code : %d", url, res.StatusCode), "")
	}
	return nil
}
