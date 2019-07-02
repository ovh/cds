package sdk

import "time"

// Variable represent a variable for a project or pipeline
type Variable struct {
	ID    int64  `json:"id,omitempty" cli:"-"`
	Name  string `json:"name" cli:"name,key"`
	Value string `json:"value" cli:"value"`
	Type  string `json:"type" cli:"type"`
}

// VariableAudit represent audit for a variable
type VariableAudit struct {
	ID         int        `json:"id"`
	Variables  []Variable `json:"variables"`
	Versionned time.Time  `json:"versionned"`
	Author     string     `json:"author"`
}

const (
	// SecretMinLength is the minimal size of a secret
	// variable to be considered as a secret
	// a secret variable displayed, with less then 6, will
	// displayed, instead of appears as **cds.app.my-password**
	SecretMinLength = 6
)

// Different type of Variable
const (
	SecretVariable     = "password"
	TextVariable       = "text"
	StringVariable     = "string"
	KeyVariable        = "key"
	BooleanVariable    = "boolean"
	NumberVariable     = "number"
	RepositoryVariable = "repository"
)

var (
	// AvailableVariableType list all existing variable type in CDS
	AvailableVariableType = []string{
		SecretVariable,
		TextVariable,
		StringVariable,
		BooleanVariable,
		NumberVariable,
	}
)

// NeedPlaceholder returns true if variable type is either secret or key
func NeedPlaceholder(t string) bool {
	switch t {
	case SecretVariable, KeyVariable:
		return true
	default:
		return false
	}
}

// VariableFind return a variable given its name if it exists in array
func VariableFind(vars []Variable, s string) *Variable {
	for _, v := range vars {
		if v.Name == s {
			return &v
		}
	}
	return nil
}

// VariablesFilter return a slice of variables filtered by type
func VariablesFilter(vars []Variable, types ...string) []Variable {
	res := []Variable{}
	for _, v := range vars {
		for _, s := range types {
			if v.Type == s {
				res = append(res, v)
			}
		}
	}
	return res
}

// VariablesPrefix add a prefix on all the variable in the slice
func VariablesPrefix(vars []Variable, prefix string) []Variable {
	res := make([]Variable, len(vars))
	for i := range vars {
		s := vars[i]
		s.Name = prefix + s.Name
		res[i] = s
	}
	return res
}
