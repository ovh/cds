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

type SemverNpmYarnPackage struct {
	Version string `json:"version"`
}

type SemverPoetry struct {
	Project SemverPoetryProject `toml:"project"`
	Tool    SemverPoetryTool    `toml:"tool"`
}
type SemverPoetryProject struct {
	Version string `toml:"version"`
}
type SemverPoetryTool struct {
	Poetry SemverPoetryToolPoetry `toml:"poetry"`
}
type SemverPoetryToolPoetry struct {
	Version string `toml:"version"`
}
