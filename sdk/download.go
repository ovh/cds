package sdk

// Download contains a association name of binary / arch-os available
type Download struct {
	Name    string   `json:"name"`
	OSArchs []OSArch `json:"osArchs"`
}

// OSArch contains a association OS / Arch
type OSArch struct {
	OS    string   `json:"os"`
	Archs []string `json:"archs"`
}

// IsBinaryOSArchValid returns err if name (worker, engine, cdsctl..) is not
// valid with os and arch. Returns "fixed Arch" 386 / amd64 or arm
// example: if arch == i386 or i686, return 386
func IsBinaryOSArchValid(name, os, arch string) (string, error) {
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
		return arch, ErrDownloadInvalidName
	}

	var osFound bool
	for _, o := range download.OSArchs {
		if os == o.OS {
			osFound = true
			for _, a := range o.Archs {
				if a == fixedArch {
					// name, os, arch found, it's valid
					return fixedArch, nil
				}
			}
		}
	}
	if !osFound {
		return fixedArch, ErrDownloadInvalidOS
	}

	return fixedArch, ErrDownloadInvalidArch
}

func getArchName(a string) string {
	switch a {
	case "386", "i386", "i686":
		return "386"
	case "amd64", "x86_64":
		return "amd64"
	}
	return a
}

// GetStaticDownloads returns default builded CDS Binaries
func GetStaticDownloads() []Download {
	defaultArch := []OSArch{
		{OS: "window", Archs: []string{"386", "amd64"}},
		{OS: "linux", Archs: []string{"386", "amd64", "arm"}},
		{OS: "darwin", Archs: []string{"amd64"}},
		{OS: "freebsd", Archs: []string{"386", "amd64"}},
	}

	downloads := []Download{
		{
			Name:    "worker",
			OSArchs: defaultArch,
		},
		{
			Name:    "engine",
			OSArchs: defaultArch,
		},
		{
			Name:    "cdsctl",
			OSArchs: []OSArch{{OS: "linux", Archs: []string{"amd64"}}},
		},
		{
			Name:    "cds",
			OSArchs: defaultArch,
		},
	}

	return downloads
}
