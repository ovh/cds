package sdk

import "time"

// Variable represent a variable for a project or pipeline
type Variable struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

// VariableAudit represent audit for a variable
type VariableAudit struct {
	ID         int        `json:"id"`
	Variables  []Variable `json:"variables"`
	Versionned time.Time  `json:"versionned"`
	Author     string     `json:"author"`
}

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
	// AvailableVariableType list all exising variable type in CDS
	AvailableVariableType = []string{
		SecretVariable,
		TextVariable,
		StringVariable,
		KeyVariable,
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

// VariablerFind return a variable given its name if it exists in array
func VariablerFind(vars []Variable, s string) *Variable {
	for _, v := range vars {
		if v.Name == s {
			return &v
		}
	}
	return nil
}
