package sdk

var (
	//OS could be windows darwin linux freebsd, setted by Makefile
	OS = ""

	//ARCH could be amd64 arm 386, setted by Makefile
	ARCH = ""

	//VERSION is set with -ldflags "-X github.com/ovh/cds/sdk.VERSION=$(VERSION)"
	VERSION = "snapshot"
)

// Version is used by /mon/version
type Version struct {
	Version      string `json:"version"`
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
}
