package sdk

import (
	"fmt"
	"runtime"
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

	//DBMIGRATE is set with -ldflags "-X github.com/ovh/cds/sdk.DBMIGRATE=$(DBMIGRATE)"
	// this flag contains the number of sql files to migrate for this version
	DBMIGRATE = ""
)

func init() {
	if GOOS == "" {
		GOOS = runtime.GOOS
	}
	if GOARCH == "" {
		GOARCH = runtime.GOARCH
	}
}

// Version is used by /mon/version
type Version struct {
	Version      string `json:"version" yaml:"version"`
	Architecture string `json:"architecture" yaml:"architecture"`
	OS           string `json:"os" yaml:"os"`
	GitHash      string `json:"git_hash" yaml:"git_hash"`
	BuildTime    string `json:"build_time" yaml:"build_time"`
	DBMigrate    string `json:"db_migrate,omitempty" yaml:"db_migrate,omitempty"`
}

// VersionCurrent returns the current version
func VersionCurrent() Version {
	return Version{
		Version:      VERSION,
		Architecture: GOARCH,
		OS:           GOOS,
		GitHash:      GITHASH,
		BuildTime:    BUILDTIME,
		DBMigrate:    DBMIGRATE,
	}
}

// VersionString returns a string contains all about current version
func VersionString() string {
	return fmt.Sprintf("CDS %s version:%s os:%s architecture:%s git.hash:%s build.time:%s db.migrate:%s", BINARY, VERSION, GOOS, GOARCH, GITHASH, BUILDTIME, DBMIGRATE)
}
