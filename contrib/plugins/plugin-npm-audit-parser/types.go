package main

type NpmAudit struct {
	Advisories map[int64]Advisory `json:"advisories"`
}

type Advisory struct {
	Findings        []Finding `json:"findings"`
	Title           string    `json:"title"`
	Overview        string    `json:"overview"`
	CVES            []string  `json:"cves"`
	PatchedVersions string    `json:"patched_versions"`
	ModuleName      string    `json:"module_name"`
	Severity        string    `json:"severity"`
	URL             string    `json:"url"`
}

type Finding struct {
	Version string   `json:"version"`
	Paths   []string `json:"paths"`
}
