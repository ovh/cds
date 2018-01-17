package sdk

import (
	"fmt"
	"net/http"
	"os"
	"path"
)

// URLGithubIssues contains a link to CDS Issues
const URLGithubIssues = "https://github.com/ovh/cds/issues"

// URLGithubReleases contains a link to CDS Official Releases
const URLGithubReleases = "https://github.com/ovh/cds/releases"

// Download contains a association name of binary / arch-os available
type Download struct {
	Name    string   `json:"name"`
	OSArchs []OSArch `json:"osArchs"`
}

// OSArch contains a association OS / Arch
type OSArch struct {
	OS        string `json:"os"`
	Archs     []Arch `json:"archs"`
	Extension string `json:"extension"`
}

// Arch contains a association Arch / available
type Arch struct {
	Arch      string `json:"arch"`
	Available bool   `json:"available"`
}

// IsBinaryOSArchValid returns err if name (worker, engine, cdsctl..) is not
// valid with os and arch. Returns "fixed Arch" 386 / amd64 or arm
// example: if arch == i386 or i686, return 386
func IsBinaryOSArchValid(directoriesDownload, name, osBinary, arch string) (string, string, error) {
	v := GetStaticDownloads()
	var fixedArch = getArchName(arch)

	var nameFound bool
	var download Download
	for _, n := range v {
		if n.Name == name {
			nameFound = true
			download = n
		}
	}

	if !nameFound {
		return arch, "", ErrDownloadInvalidName
	}

	var osFound bool
	for _, o := range download.OSArchs {
		if osBinary == o.OS {
			osFound = true
			for _, a := range o.Archs {
				if a.Arch == fixedArch {
					// name, os, arch found, it's valid
					if _, err := os.Stat(path.Join(directoriesDownload, fmt.Sprintf("cds-%s-%s-%s%s", name, osBinary, fixedArch, o.Extension))); err == nil {
						return fixedArch, o.Extension, nil
					}
					return fixedArch, "", ErrDownloadDoesNotExist
				}
			}
		}
	}
	if !osFound {
		return fixedArch, "", ErrDownloadInvalidOS
	}

	return fixedArch, "", ErrDownloadInvalidArch
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

// GetArtifactFilename returns artifact name cds-name-os-arch
// this name is used on Github Releases
func GetArtifactFilename(name, os, arch string) string {
	return fmt.Sprintf("cds-%s-%s-%s", name, os, getArchName(arch))
}

func getDefaultArch() []OSArch {
	return []OSArch{
		{OS: "windows", Archs: []Arch{{Arch: "amd64"}}, Extension: ".exe"},
		{OS: "linux", Archs: []Arch{{Arch: "386"}, {Arch: "amd64"}, {Arch: "arm"}}},
		{OS: "darwin", Archs: []Arch{{Arch: "amd64"}}},
		{OS: "freebsd", Archs: []Arch{{Arch: "amd64"}}},
	}
}

// GetStaticDownloads returns default builded CDS Binaries
func GetStaticDownloads() []Download {
	downloads := []Download{
		{
			Name:    "worker",
			OSArchs: getDefaultArch(),
		},
		{
			Name:    "engine",
			OSArchs: getDefaultArch(),
		},
		{
			Name:    "cdsctl",
			OSArchs: getDefaultArch(),
		},
	}

	return downloads
}

// GetStaticDownloadsWithAvailability set flag Available on downloads list
func GetStaticDownloadsWithAvailability(directoriesDownload string) []Download {
	downloads := GetStaticDownloads()
	// for each download, check if binary exists in directoriesDownload
	for k, d := range downloads {
		for ks, o := range downloads[k].OSArchs {
			for ka, a := range downloads[k].OSArchs[ks].Archs {
				if _, err := os.Stat(path.Join(directoriesDownload, fmt.Sprintf("cds-%s-%s-%s%s", d.Name, o.OS, a.Arch, o.Extension))); err == nil {
					downloads[k].OSArchs[ks].Archs[ka].Available = true
				}
			}
		}
	}
	return downloads
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
