package sdk

const (
	//BinaryRequirement refers to the need to a specific binary on host running the action
	BinaryRequirement = "binary"
	// NetworkAccessRequirement refers to the need of an opened network acces to given endpoint.
	NetworkAccessRequirement = "network"
	// ModelRequirement refers to the need fo a specific model
	ModelRequirement = "model"
	// HostnameRequirement checks the hostname of the worker
	HostnameRequirement = "hostname"
	//PluginRequirement installs & checks plugins of the worker
	PluginRequirement = "plugin"
	//ServiceRequirement links a service to a worker
	ServiceRequirement = "service"
	//MemoryRequirement set memory limit on a container
	MemoryRequirement = "memory"
	// VolumeRequirement set Volume limit on a container
	VolumeRequirement = "volume"
	// OSArchRequirement checks the 'dist' of a worker eg {GOOS}/{GOARCH}
	OSArchRequirement = "os-architecture"
)

// RequirementsList is a list of requirement
type RequirementList []Requirement

// Values returns all Requirement.Value
func (l RequirementList) Values() []string {
	values := make([]string, len(l))
	for i := range l {
		values[i] = l[i].Value
	}
	return values
}

var (
	// AvailableRequirementsType List of all requirements
	AvailableRequirementsType = []string{
		BinaryRequirement,
		NetworkAccessRequirement,
		ModelRequirement,
		HostnameRequirement,
		PluginRequirement,
		ServiceRequirement,
		MemoryRequirement,
		VolumeRequirement,
		OSArchRequirement,
	}

	// OSArchRequirementValues comes from go tool dist list
	OSArchRequirementValues = RequirementList{
		{"linux/amd64", OSArchRequirement, "linux/amd64"},
		{"linux/386", OSArchRequirement, "linux/386"},
		//{"android/386", OSArchRequirement, "android/386"},
		//{"android/amd64", OSArchRequirement, "android/amd64"},
		//{"android/arm", OSArchRequirement, "android/arm"},
		//{"android/arm64", OSArchRequirement, "android/arm64"},
		//{"darwin/386", OSArchRequirement, "darwin/386"},
		{"darwin/amd64", OSArchRequirement, "darwin/amd64"},
		//{"darwin/arm", OSArchRequirement, "darwin/arm"},
		//{"darwin/arm64", OSArchRequirement, "darwin/arm64"},
		//{"dragonfly/amd64", OSArchRequirement, "dragonfly/amd64"},
		{"freebsd/386", OSArchRequirement, "freebsd/386"},
		{"freebsd/amd64", OSArchRequirement, "freebsd/amd64"},
		//{"freebsd/arm", OSArchRequirement, "freebsd/arm"},
		//{"linux/arm", OSArchRequirement, "linux/arm"},
		{"linux/arm64", OSArchRequirement, "linux/arm64"},
		//{"linux/mips", OSArchRequirement, "linux/mips"},
		//{"linux/mips64", OSArchRequirement, "linux/mips64"},
		//{"linux/mips64le", OSArchRequirement, "linux/mips64le"},
		//{"linux/mipsle", OSArchRequirement, "linux/mipsle"},
		//{"linux/ppc64", OSArchRequirement, "linux/ppc64"},
		//{"linux/ppc64le", OSArchRequirement, "linux/ppc64le"},
		//{"linux/s390x", OSArchRequirement, "linux/s390x"},
		//{"nacl/386", OSArchRequirement, "nacl/386"},
		//{"nacl/amd64p32", OSArchRequirement, "nacl/amd64p32"},
		//{"nacl/arm", OSArchRequirement, "nacl/arm"},
		{"netbsd/386", OSArchRequirement, "netbsd/386"},
		{"netbsd/amd64", OSArchRequirement, "netbsd/amd64"},
		//{"netbsd/arm", OSArchRequirement, "netbsd/arm"},
		{"openbsd/386", OSArchRequirement, "openbsd/386"},
		{"openbsd/amd64", OSArchRequirement, "openbsd/amd64"},
		//{"openbsd/arm", OSArchRequirement, "openbsd/arm"},
		//{"plan9/386", OSArchRequirement, "plan9/386"},
		//{"plan9/amd64", OSArchRequirement, "plan9/amd64"},
		//{"plan9/arm", OSArchRequirement, "plan9/arm"},
		//{"solaris/amd64", OSArchRequirement, "solaris/amd64"},
		{"windows/386", OSArchRequirement, "windows/386"},
		{"windows/amd64", OSArchRequirement, "windows/amd64"},
	}
)

// Requirement can be :
// - a binary "which /usr/bin/docker"
// - a network access "telnet google.com 443"
//easyjson:json
type Requirement struct {
	Name  string `json:"name"`
	Type  string `json:"type" yaml:"-"`
	Value string `json:"value" yaml:"-"`
}

// AddRequirement append a requirement in a requirement array
func AddRequirement(array *RequirementList, name string, requirementType string, value string) {
	requirements := append(*array, Requirement{
		Name:  name,
		Value: value,
		Type:  requirementType,
	})
	*array = requirements
}

// Requirement add given requirement to Action
func (a *Action) Requirement(name string, t string, value string) *Action {
	r := Requirement{
		Name:  name,
		Type:  t,
		Value: value,
	}

	a.Requirements = append(a.Requirements, r)
	return a
}
