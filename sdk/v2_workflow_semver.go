package sdk

type SemverHelmChart struct {
	Version string `yaml:"version"`
}

type SemverCargoFile struct {
	Package SemverCargoFilePackage `toml:"package"`
}

type SemverCargoFilePackage struct {
	Version string `toml:"version"`
}
