package xray

type ComponentLicences struct {
	ComponentID   string `json:"component_id"`
	ComponentName string `json:"component_name"`
	Version       string `json:"version"`
	PkgType       string `json:"pkg_type"`
	PackageID     string `json:"package_id"`
	Licenses      []struct {
		Key     string `json:"key"`
		Link    string `json:"link"`
		Sources []struct {
			Source      string `json:"source"`
			Occurrences int    `json:"occurrences"`
		} `json:"sources"`
	} `json:"licenses"`
}

type ComponentDetails struct {
	ComponentLicences []ComponentLicences `json:"licenses"`
}

type ComponentDetailsRequest struct {
	ComponentName            string `json:"component_name,omitempty"`             // "image:tag",
	PackageType              string `json:"package_type,omitempty"`               // "build | releaseBundle | docker | debian | npm | rpm | go | pypi | conan | terraform | alpine | nuget | cran | conan | maven",
	Sha256                   string `json:"sha_256,omitempty"`                    //: "1d36301476dc57eb479e03d9e37a885dd751a6e6979f6f916a92c10cb7520e4e",
	Violations               bool   `json:"violations,omitempty"`                 // true | false,
	IncludeIgnoredViolations bool   `json:"include_ignored_violations,omitempty"` // true | false
	License                  bool   `json:"license,omitempty"`                    // true | false,
	ExcludeUnknown           bool   `json:"exclude_unknown,omitempty"`            // true | false,
	Security                 bool   `json:"security,omitempty"`                   // true | false,
	MaliciousCode            bool   `json:"malicious_code,omitempty"`             // true | false,
	Iac                      bool   `json:"iac,omitempty"`                        // true | false,
	Services                 bool   `json:"services,omitempty"`                   // true | false,
	Applications             bool   `json:"applications,omitempty"`               // true | false,
	OutputFormat             string `json:"output_format,omitempty"`              // "pdf | csv | json | json_full",
	Spdx                     bool   `json:"spdx,omitempty"`                       // true | false,
	SpdxFormat               string `json:"spdx_format,omitempty"`                // "json | tag:value | xlsx",
	Cyclonedx                bool   `json:"cyclonedx,omitempty"`                  // true | false,
	CyclonedxFormat          string `json:"cyclonedx_format,omitempty"`           // "json | xml"
}
