package sdk

//VERSION is set with -ldflags "-X main.VERSION={{.cds.proj.version}}+{{.cds.version}}"
var VERSION = "snapshot"

type Version struct {
	Version string `json:"version"`
}
