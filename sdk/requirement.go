package sdk

import (
	"context"
	"net"
	"time"
)

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

// RequirementList is a list of requirement
type RequirementList []Requirement

// Values returns all Requirement.Value
func (l RequirementList) Values() []string {
	values := make([]string, len(l))
	for i := range l {
		values[i] = l[i].Value
	}
	return values
}

// RequirementListDeduplicate returns requirements list without duplicate values.
func RequirementListDeduplicate(l RequirementList) RequirementList {
	m := map[string]Requirement{}

	for i := range l {
		m[l[i].Name+l[i].Type+l[i].Value] = l[i]
	}

	newList := make([]Requirement, 0, len(m))
	for i := range m {
		newList = append(newList, m[i])
	}
	return newList
}

// IsValid returns requirement list validity.
func (l RequirementList) IsValid() error {
	// check requirement unicity
	for i := range l {
		for j := range l {
			if l[i].Name == l[j].Name && l[i].Type == l[j].Type && i != j {
				return WrapError(ErrInvalidJobRequirement, "duplicate requirement name %s and type %s", l[i].Name, l[i].Type)
			}
		}
	}

	// check that only one model requirement and hostname exists
	nbModel, nbHostname := 0, 0
	for i := range l {
		switch l[i].Type {
		case ModelRequirement:
			nbModel++
		case HostnameRequirement:
			nbHostname++
		}
	}
	if nbModel > 1 {
		return WithStack(ErrInvalidJobRequirementDuplicateModel)
	}
	if nbHostname > 1 {
		return WithStack(ErrInvalidJobRequirementDuplicateHostname)
	}

	return nil
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
		{Name: "linux/amd64", Type: OSArchRequirement, Value: "linux/amd64"},
		{Name: "linux/386", Type: OSArchRequirement, Value: "linux/386"},
		//{"android/386", OSArchRequirement, "android/386"},
		//{"android/amd64", OSArchRequirement, "android/amd64"},
		//{"android/arm", OSArchRequirement, "android/arm"},
		//{"android/arm64", OSArchRequirement, "android/arm64"},
		//{"darwin/386", OSArchRequirement, "darwin/386"},
		{Name: "darwin/amd64", Type: OSArchRequirement, Value: "darwin/amd64"},
		//{"darwin/arm", OSArchRequirement, "darwin/arm"},
		//{"darwin/arm64", OSArchRequirement, "darwin/arm64"},
		//{"dragonfly/amd64", OSArchRequirement, "dragonfly/amd64"},
		{Name: "freebsd/386", Type: OSArchRequirement, Value: "freebsd/386"},
		{Name: "freebsd/amd64", Type: OSArchRequirement, Value: "freebsd/amd64"},
		//{"freebsd/arm", OSArchRequirement, "freebsd/arm"},
		//{"linux/arm", OSArchRequirement, "linux/arm"},
		{Name: "linux/arm64", Type: OSArchRequirement, Value: "linux/arm64"},
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
		{Name: "netbsd/386", Type: OSArchRequirement, Value: "netbsd/386"},
		{Name: "netbsd/amd64", Type: OSArchRequirement, Value: "netbsd/amd64"},
		//{"netbsd/arm", OSArchRequirement, "netbsd/arm"},
		{Name: "openbsd/386", Type: OSArchRequirement, Value: "openbsd/386"},
		{Name: "openbsd/amd64", Type: OSArchRequirement, Value: "openbsd/amd64"},
		//{"openbsd/arm", OSArchRequirement, "openbsd/arm"},
		//{"plan9/386", OSArchRequirement, "plan9/386"},
		//{"plan9/amd64", OSArchRequirement, "plan9/amd64"},
		//{"plan9/arm", OSArchRequirement, "plan9/arm"},
		//{"solaris/amd64", OSArchRequirement, "solaris/amd64"},
		{Name: "windows/386", Type: OSArchRequirement, Value: "windows/386"},
		{Name: "windows/amd64", Type: OSArchRequirement, Value: "windows/amd64"},
	}
)

// Requirement can be :
// - a binary "which /usr/bin/docker"
// - a network access "telnet google.com 443"
type Requirement struct {
	ID       int64  `json:"id" db:"id"`
	ActionID int64  `json:"action_id" db:"action_id"`
	Name     string `json:"name" yaml:"name" db:"name"`
	Type     string `json:"type" yaml:"type" db:"type"`
	Value    string `json:"value" yaml:"value" db:"value"`
}

// AddRequirement append a requirement in a requirement array
func AddRequirement(array *RequirementList, id int64, name string, requirementType string, value string) {
	requirements := append(*array, Requirement{
		ID:    id,
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

// CheckNetworkAccessRequirement returns true if req.Value can Dial
func CheckNetworkAccessRequirement(req Requirement) bool {
	d := net.Dialer{Timeout: 10 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout)
	defer cancel()

	conn, err := d.DialContext(ctx, "tcp", req.Value)
	if err != nil {
		return false
	}
	conn.Close()

	return true
}
