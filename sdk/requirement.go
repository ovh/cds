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
)

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
	}
)

// Requirement can be :
// - a binary "which /usr/bin/docker"
// - a network access "telnet google.com 443"
type Requirement struct {
	Name  string `json:"name"`
	Type  string `json:"type" yaml:"-"`
	Value string `json:"value" yaml:"-"`
}

// AddRequirement append a requirement in a requirement array
func AddRequirement(array *[]Requirement, name string, requirementType string, value string) {
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
