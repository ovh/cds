package sdk

import (
	"fmt"
)

var (
	//VERSION is set with -ldflags "-X github.com/ovh/cds/sdk.VERSION=$(VERSION)"
	VERSION = "snapshot"

	//GOOS is set with -ldflags "-X github.com/ovh/cds/sdk.GOOS=$(GOOS)"
	GOOS = ""

	//GOARCH is set with -ldflags "-X github.com/ovh/cds/sdk.GOARCH=$(GOARCH)"
	GOARCH = ""

	//GITHASH is set with -ldflags "-X github.com/ovh/cds/sdk.GITHASH=$(GITHASH)"
	GITHASH = ""

	//BUILDTIME is set with -ldflags "-X github.com/ovh/cds/sdk.BUILDTIME=$(BUILDTIME)"
	BUILDTIME = ""

	//BINARY is set with -ldflags "-X github.com/ovh/cds/sdk.BINARY=$(BINARY)"
	BINARY = ""
)

// Version is used by /mon/version
type Version struct {
	Version      string `json:"version"`
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
	GitHash      string `json:"git_hash"`
	BuildTime    string `json:"build_time"`
}

// VersionCurrent returns the current version
func VersionCurrent() Version {
	return Version{
		Version:      VERSION,
		Architecture: GOARCH,
		OS:           GOOS,
		GitHash:      GITHASH,
		BuildTime:    BUILDTIME,
	}
}

// VersionString returns a string contains all about current version
func VersionString() string {
	return fmt.Sprintf("CDS %s version:%s os:%s architecture:%s git.hash:%s build.time:%s", BINARY, VERSION, GOOS, GOARCH, GITHASH, BUILDTIME)
}
