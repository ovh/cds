package sdk

var (
	//VERSION is set with -ldflags "-X github.com/ovh/cds/sdk.VERSION=$(VERSION)"
	VERSION = "snapshot"
)

// Version is used by /mon/version
type Version struct {
	Version      string `json:"version"`
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
}
