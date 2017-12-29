package cdsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

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

func (c *client) DownloadURLFromGithub(name, os, arch string) (string, error) {
	var httpClient = &http.Client{Timeout: 10 * time.Second}

	r, err := httpClient.Get("https://api.github.com/repos/ovh/cds/releases/latest")
	if err != nil {
		return "", err
	}
	defer r.Body.Close()

	release := RepositoryRelease{}
	errDecode := json.NewDecoder(r.Body).Decode(&release)
	if errDecode != nil {
		return "", errDecode
	}

	if len(release.Assets) > 0 {
		for _, asset := range release.Assets {
			if *asset.Name == sdk.GetArtifactFilename(name, os, arch) {
				return *asset.BrowserDownloadURL, nil
			}
		}
	}

	text := "Invalid Artifacts on latest release. Please try again in few minutes.\n"
	text += fmt.Sprintf("If the problem persists, please open an issue on %s\n", sdk.URLGithubIssues)
	text += fmt.Sprintf("You can manually download binary from latest release: %s\n", sdk.URLGithubReleases)
	return "", fmt.Errorf(text)
}

// code below is from https://github.com/google/go-github/tree/master/github
// This library is distributed under the BSD-style license found in the LICENSE file.

// RepositoryRelease represents a GitHub release in a repository.
type RepositoryRelease struct {
	ID              *int           `json:"id,omitempty"`
	TagName         *string        `json:"tag_name,omitempty"`
	TargetCommitish *string        `json:"target_commitish,omitempty"`
	Name            *string        `json:"name,omitempty"`
	Body            *string        `json:"body,omitempty"`
	Draft           *bool          `json:"draft,omitempty"`
	Prerelease      *bool          `json:"prerelease,omitempty"`
	CreatedAt       *Timestamp     `json:"created_at,omitempty"`
	PublishedAt     *Timestamp     `json:"published_at,omitempty"`
	URL             *string        `json:"url,omitempty"`
	HTMLURL         *string        `json:"html_url,omitempty"`
	AssetsURL       *string        `json:"assets_url,omitempty"`
	Assets          []ReleaseAsset `json:"assets,omitempty"`
	UploadURL       *string        `json:"upload_url,omitempty"`
	ZipballURL      *string        `json:"zipball_url,omitempty"`
	TarballURL      *string        `json:"tarball_url,omitempty"`
}

// ReleaseAsset represents a GitHub release asset in a repository.
type ReleaseAsset struct {
	ID                 *int       `json:"id,omitempty"`
	URL                *string    `json:"url,omitempty"`
	Name               *string    `json:"name,omitempty"`
	Label              *string    `json:"label,omitempty"`
	State              *string    `json:"state,omitempty"`
	ContentType        *string    `json:"content_type,omitempty"`
	Size               *int       `json:"size,omitempty"`
	DownloadCount      *int       `json:"download_count,omitempty"`
	CreatedAt          *Timestamp `json:"created_at,omitempty"`
	UpdatedAt          *Timestamp `json:"updated_at,omitempty"`
	BrowserDownloadURL *string    `json:"browser_download_url,omitempty"`
}

// Timestamp represents a time that can be unmarshalled from a JSON string
// formatted as either an RFC3339 or Unix timestamp. This is necessary for some
// fields since the GitHub API is inconsistent in how it represents times. All
// exported methods of time.Time can be called on Timestamp.
type Timestamp struct {
	time.Time
}

func (t Timestamp) String() string {
	return t.Time.String()
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// Time is expected in RFC3339 or Unix format.
func (t *Timestamp) UnmarshalJSON(data []byte) (err error) {
	str := string(data)
	i, err := strconv.ParseInt(str, 10, 64)
	if err == nil {
		(*t).Time = time.Unix(i, 0)
	} else {
		(*t).Time, err = time.Parse(`"`+time.RFC3339+`"`, str)
	}
	return
}

// Equal reports whether t and u are equal based on time.Equal
func (t Timestamp) Equal(u Timestamp) bool {
	return t.Time.Equal(u.Time)
}
