package sdk

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
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
	binaries      = []string{"engine", "worker", "cdsctl"}
	supportedOS   = []string{"windows", "darwin", "linux", "freebsd"}
	supportedARCH = []string{"amd64", "arm", "386", "arm64"}
	cdsctlVariant = []string{"nokeychain"}
)

func AllDownloadableResources() []DownloadableResource {
	var all []DownloadableResource
	for _, b := range binaries {
		for _, os := range supportedOS {
			for _, arch := range supportedARCH {
				all = append(all, DownloadableResource{
					Name: b,
					OS:   os,
					Arch: arch,
				})
				if b == "cdsctl" {
					for _, v := range cdsctlVariant {
						all = append(all, DownloadableResource{
							Name:    b,
							OS:      os,
							Arch:    arch,
							Variant: v,
						})
					}
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

// GetArtifactFilename returns artifact name cds-name-os-arch-variant
// this name is used on Github Releases
func GetArtifactFilename(name, os, arch, variant string) string {
	if variant != "" {
		variant = "-" + variant
	}
	var prefix = "cds-"
	if name == "cdsctl" {
		prefix = ""
	}
	return fmt.Sprintf("%s%s-%s-%s%s", prefix, name, os, getArchName(arch), variant)
}

// AllDownloadableResourcesWithAvailability set flag Available on downloads list
func AllDownloadableResourcesWithAvailability(directoriesDownload string) []DownloadableResource {
	resources := AllDownloadableResources()
	// for each download, check if binary exists in directoriesDownload
	for i := range resources {
		filename := GetArtifactFilename(resources[i].Name, resources[i].OS, resources[i].Arch, resources[i].Variant)
		if _, err := os.Stat(path.Join(directoriesDownload, filename)); err == nil {
			resources[i].Filename = filepath.Base(filename)
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
