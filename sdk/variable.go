package sdk

import (
	"fmt"
	"strings"
	"time"
)

type Secret struct {
	ProjectID int64  `json:"project_id" db:"project_id" cli:"-"`
	Name      string `json:"name" db:"content_name" cli:"name,key"`
	Token     string `json:"token" db:"token" cli:"token"`
	Status    string `json:"status" db:"-" cli:"status"`
}

// Variable represent a variable for a project or pipeline
type Variable struct {
	ID    int64  `json:"id,omitempty" cli:"-"`
	Name  string `json:"name" cli:"name,key"`
	Value string `json:"value" cli:"value"`
	Type  string `json:"type" cli:"type"`
}

func FromProjectVariables(appVars []ProjectVariable) []Variable {
	vars := make([]Variable, len(appVars))
	for i, a := range appVars {
		vars[i] = Variable{
			Value: a.Value,
			Name:  a.Name,
			Type:  a.Type,
			ID:    a.ID,
		}
	}
	return vars
}

func FromAplicationVariables(appVars []ApplicationVariable) []Variable {
	vars := make([]Variable, len(appVars))
	for i, a := range appVars {
		vars[i] = Variable{
			Value: a.Value,
			Name:  a.Name,
			Type:  a.Type,
			ID:    a.ID,
		}
	}
	return vars
}

func FromEnvironmentVariables(envVars []EnvironmentVariable) []Variable {
	vars := make([]Variable, len(envVars))
	for i, a := range envVars {
		vars[i] = Variable{
			Value: a.Value,
			Name:  a.Name,
			Type:  a.Type,
			ID:    a.ID,
		}
	}
	return vars
}

func (v *Variable) ToApplicationVariable(appID int64) *ApplicationVariable {
	return &ApplicationVariable{
		ID:            v.ID,
		ApplicationID: appID,
		Type:          v.Type,
		Name:          v.Name,
		Value:         v.Value,
	}
}

type ProjectVariable struct {
	ID        int64  `json:"id,omitempty" cli:"-"`
	Name      string `json:"name" cli:"name,key"`
	Value     string `json:"value" cli:"value"`
	Type      string `json:"type" cli:"type"`
	ProjectID int64  `json:"project_id" cli:"-"`
}

type ApplicationVariable struct {
	ID            int64  `json:"id,omitempty" cli:"-"`
	Name          string `json:"name" cli:"name,key"`
	Value         string `json:"value" cli:"value"`
	Type          string `json:"type" cli:"type"`
	ApplicationID int64  `json:"application_id" cli:"-"`
}

type EnvironmentVariable struct {
	ID            int64  `json:"id,omitempty" cli:"-"`
	Name          string `json:"name" cli:"name,key"`
	Value         string `json:"value" cli:"value"`
	Type          string `json:"type" cli:"type"`
	EnvironmentID int64  `json:"environment_id" cli:"-"`
}

func (v Variable) ToParameter(prefix string) Parameter {
	name := v.Name
	if prefix != "" {
		name = fmt.Sprintf("%s.%s", prefix, v.Name)
	}
	return Parameter{
		Name:  name,
		Value: v.Value,
		Type:  v.Type,
	}
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
	SSHKeyVariable     = "ssh"
	PGPKeyVariable     = "pgp"
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

	BasicVariableNames = []string{
		"cds.version",
		"cds.application",
		"cds.environment",
		"cds.job",
		"cds.manual",
		"cds.pipeline",
		"cds.project",
		"cds.run",
		"cds.run.number",
		"cds.run.subnumber",
		"cds.stage",
		"cds.triggered_by.email",
		"cds.triggered_by.fullname",
		"cds.triggered_by.username",
		"cds.ui.pipeline.run",
		"cds.worker",
		"cds.workflow",
		"cds.workspace",
		"payload",
	}

	BasicGitVariableNames = []string{
		"git.repository",
		"git.branch",
		"git.message",
		"git.author",
		"git.hash",
		"git.hash.short",
		"git.url",
		"git.http_url",
		"git.server",
	}
)

// NeedPlaceholder returns true if variable type is either secret or key
func NeedPlaceholder(t string) bool {
	switch t {
	case SecretVariable, KeyVariable, SSHKeyVariable, PGPKeyVariable:
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

func EnvVartoENV(p Parameter) []string {
	var env []string
	if !strings.HasPrefix(p.Name, "cds.env.") {
		return nil
	}

	pName := strings.TrimPrefix(p.Name, "cds.env.")

	oneLineValue := OneLineValue(p.Value)

	envName := strings.Replace(pName, ".", "_", -1)
	envName = strings.Replace(envName, "-", "_", -1)
	env = append(env, fmt.Sprintf("CDS_ENV_%s=%s", strings.ToUpper(envName), oneLineValue)) // CDS_ENV_MYSTRINGVARIABLE
	env = append(env, fmt.Sprintf("CDS_ENV_%s=%s", pName, oneLineValue))                    //CDS_ENV_MyStringVariable
	env = append(env, fmt.Sprintf("%s=%s", pName, oneLineValue))                            // MyStringVariable
	env = append(env, fmt.Sprintf("%s=%s", strings.ToUpper(envName), oneLineValue))         // MYSTRINGVARIABLE
	return env
}

func OneLineValue(v string) string {
	if strings.Contains(v, "\n") {
		return strings.Replace(v, "\n", "\\n", -1)
	}
	return v
}
