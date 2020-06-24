package main

type npmAudit struct {
	Advisories map[int64]advisory `json:"advisories"`
}

type advisory struct {
	Findings        []finding `json:"findings"`
	Title           string    `json:"title"`
	Overview        string    `json:"overview"`
	CVES            []string  `json:"cves"`
	PatchedVersions string    `json:"patched_versions"`
	ModuleName      string    `json:"module_name"`
	Severity        string    `json:"severity"`
	URL             string    `json:"url"`
	CWE             string    `json:"cwe"`
}

type finding struct {
	Version string   `json:"version"`
	Paths   []string `json:"paths"`
}
