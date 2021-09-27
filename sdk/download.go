package sdk

import (
	"context"
	json "encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/rockbears/log"
)

// URLGithubIssues contains a link to CDS Issues
const URLGithubIssues = "https://github.com/ovh/cds/issues"

// URLGithubReleases contains a link to CDS Official Releases
const URLGithubReleases = "https://github.com/ovh/cds/releases"

type DownloadableResource struct {
	Name      string `json:"name"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	Variant   string `json:"variant,omitempty"`
	Filename  string `json:"filename,omitempty"`
	Available *bool  `json:"available,omitempty"`
}

var (
	binaries        = []string{"engine", "worker", "cdsctl"}
	supportedOS     = []string{"windows", "darwin", "linux", "freebsd", "openbsd"}
	supportedARCH   = []string{"amd64", "arm", "386", "arm64", "ppc64le"}
	supportedOSArch = []string{}
	cdsctlVariant   = []string{"nokeychain"}
)

func InitSupportedOSArch(supportedOSArchConf []string) error {
	if len(supportedOSArchConf) == 0 {
		// if the supportedOSArchConf is empty, we init it with the full list
		for _, os := range supportedOS {
			for _, arch := range supportedARCH {
				supportedOSArch = append(supportedOSArch, os+"/"+arch)
			}
		}
		return nil
	}

	// example of supportedOSArchConf: darwin/amd64,darwin/arm64,linux/amd64,windows/amd64
	for _, v := range supportedOSArchConf {
		t := strings.Split(v, "/")
		if len(t) != 2 {
			return fmt.Errorf("invalid value %q in supportedOSArch configuration", v)
		}
		if !IsInArray(t[0], supportedOS) {
			return fmt.Errorf("invalid value for os %q in supportedOSArch configuration", t[0])
		}
		if !IsInArray(t[1], supportedARCH) {
			return fmt.Errorf("invalid value for arch %q in supportedOSArch configuration", t[1])
		}
		supportedOSArch = append(supportedOSArch, v)
	}
	return nil
}

func AllDownloadableResources() []DownloadableResource {
	var all []DownloadableResource
	for _, b := range binaries {

		for _, osArch := range supportedOSArch {
			t := strings.Split(osArch, "/")
			os := t[0]
			arch := t[1]

			all = append(all, DownloadableResource{
				Filename: BinaryFilename(b, os, arch, ""),
				Name:     b,
				OS:       os,
				Arch:     arch,
			})
			if b == "cdsctl" && (os == "linux" || os == "darwin" || os == "windows") && (arch == "amd64" || arch == "arm64") {
				for _, v := range cdsctlVariant {
					all = append(all, DownloadableResource{
						Filename: BinaryFilename(b, os, arch, v),
						Name:     b,
						OS:       os,
						Arch:     arch,
						Variant:  v,
					})
				}
			}
		}
	}
	return all
}

// getArchName returns 386 for "386", "i386", "i686"
// amd64 for "amd64", "x86_64" (uname -m)
func getArchName(a string) string {
	switch a {
	case "386", "i386", "i686":
		return "386"
	case "amd64", "x86_64":
		return "amd64"
	}
	return a
}

// BinaryFilename returns artifact name cds-name-os-arch-variant
// this name is used on Github Releases
func BinaryFilename(name, os, arch, variant string) string {
	if variant != "" {
		variant = "-" + variant
	}
	var prefix = "cds-"
	if name == "cdsctl" {
		prefix = ""
	}
	var suffix string
	if os == "windows" {
		suffix = ".exe"
	}
	return fmt.Sprintf("%s%s-%s-%s%s%s", prefix, name, os, getArchName(arch), variant, suffix)
}

// IsDownloadedBinary returns true if the binary is already downloaded, false otherwise
func IsDownloadedBinary(directoriesDownload, filename string) bool {
	if _, err := os.Stat(path.Join(directoriesDownload, filename)); err == nil {
		return true
	}
	return false
}

// AllDownloadableResourcesWithAvailability set flag Available on downloads list
func AllDownloadableResourcesWithAvailability(directoriesDownload string) []DownloadableResource {
	resources := AllDownloadableResources()
	// for each download, check if binary exists in directoriesDownload
	for i := range resources {
		filename := BinaryFilename(resources[i].Name, resources[i].OS, resources[i].Arch, resources[i].Variant)
		if IsDownloadedBinary(directoriesDownload, filename) {
			resources[i].Available = &True
		} else {
			resources[i].Available = &False
		}
	}
	return resources
}

// CheckContentTypeBinary returns an error if Content-Type is not application/octet-stream
func CheckContentTypeBinary(resp *http.Response) error {
	var contentType string
	for k, v := range resp.Header {
		if k == "Content-Type" && len(v) >= 1 {
			contentType = v[0]
			break
		}
	}
	if contentType != "application/octet-stream" {
		return fmt.Errorf("invalid Binary (Content-Type: %s). Please try again or download it manually from %s", contentType, URLGithubReleases)
	}
	return nil
}

// GetContentType returns the content-type header from a http.Response
func GetContentType(resp *http.Response) string {
	for k, v := range resp.Header {
		if k == "Content-Type" && len(v) >= 1 {
			return v[0]
		}
	}
	return ""
}

func DownloadFromGitHub(ctx context.Context, directory, filename string, version string) error {
	urlBinary, err := DownloadURLFromGithub(filename, version)
	if err != nil {
		return WrapError(err, "error while getting %s from %s", filename, urlBinary)
	}

	resp, err := http.Get(urlBinary)
	if err != nil {
		return WrapError(err, "error while getting binary %s", urlBinary)
	}
	defer resp.Body.Close()

	if err := CheckContentTypeBinary(resp); err != nil {
		return WrapError(err, "error while checking content-type of binary %s", urlBinary)
	}

	if resp.StatusCode != 200 {
		return WrapError(err, "error http code: %d, url called: %s", resp.StatusCode, urlBinary)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return WrapError(err, "error while reading file content for %s", filename)
	}

	fullpath := path.Join(directory, filename)
	log.Debug(ctx, "downloading %v into  %v", urlBinary, fullpath)
	if err := ioutil.WriteFile(fullpath, body, 0755); err != nil {
		return WrapError(err, "error while write file content for %s in %s", filename, directory)
	}

	return nil
}

func DownloadURLFromGithub(filename string, version string) (string, error) {
	if version == "snapshot" {
		version = "latest"
	}
	urlFile := fmt.Sprintf("https://github.com/ovh/cds/releases/download/%s/%s", version, filename)
	if version != "latest" {
		return urlFile, nil
	}

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
			if *asset.Name == filename {
				return *asset.BrowserDownloadURL, nil
			}
		}
	}

	text := fmt.Sprintf("Invalid Artifacts on latest release (%s). Please try again in few minutes.\n", filename)
	text += fmt.Sprintf("If the problem persists, please open an issue on %s\n", URLGithubIssues)
	text += fmt.Sprintf("You can manually download binary from latest release: %s\n", URLGithubReleases)
	return "", errors.New(text)
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
