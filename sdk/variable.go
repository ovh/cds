package sdk

import "time"

// Variable represent a variable for a project or pipeline
type Variable struct {
	ID    int64        `json:"id"`
	Name  string       `json:"name"`
	Value string       `json:"value"`
	Type  VariableType `json:"type"`
}

// VariableAudit represent audit for a variable
type VariableAudit struct {
	ID         int        `json:"id"`
	Variables  []Variable `json:"variables"`
	Versionned time.Time  `json:"versionned"`
	Author     string     `json:"author"`
}

// VariableType defines the types of project, application and environment variable
type VariableType string

// Different type of Variable
const (
	SecretVariable     VariableType = "password"
	TextVariable       VariableType = "text"
	StringVariable     VariableType = "string"
	KeyVariable        VariableType = "key"
	BooleanVariable    VariableType = "boolean"
	NumberVariable     VariableType = "number"
	RepositoryVariable VariableType = "repository"
)

var (
	// AvailableVariableType list all exising variable type in CDS
	AvailableVariableType = []VariableType{
		SecretVariable,
		TextVariable,
		StringVariable,
		KeyVariable,
		BooleanVariable,
		NumberVariable,
	}
)

// NeedPlaceholder returns true if variable type is either secret or key
func NeedPlaceholder(t VariableType) bool {
	switch t {
	case SecretVariable, KeyVariable:
		return true
	default:
		return false
	}
}

// VariableTypeFromString return a valid VariableType from a string
// Defaults to String
func VariableTypeFromString(in string) VariableType {
	switch in {
	case string(SecretVariable):
		return SecretVariable
	case string(TextVariable):
		return TextVariable
	case string(StringVariable):
		return StringVariable
	case string(KeyVariable):
		return KeyVariable
	case string(BooleanVariable):
		return BooleanVariable
	default:
		return StringVariable
	}
}
