package sdk

const (
	KeyTypeSsh = "ssh"
	KeyTypePgp = "pgp"
)

// Key represent a key of type SSH or GPG.
type Key struct {
	Name    string `json:"name" db:"name" cli:"name"`
	Public  string `json:"public" db:"public" cli:"publickey"`
	Private string `json:"private" db:"private" cli:"-"`
	KeyID   string `json:"keyID" db:"key_id" cli:"-"`
	Type    string `json:"type" db:"type" cli:"type"`
}

// ProjectKey represent a key attach to a project
type ProjectKey struct {
	Key
	ProjectID int64 `json:"project_id" db:"project_id" cli:"-"`
	Builtin   bool  `json:"-" db:"builtin" cli:"-"`
}

// ApplicationKey represent a key attach to an application
type ApplicationKey struct {
	Key
	ApplicationID int64 `json:"application_id" db:"application_id"`
}

// EnvironmentKey represent a key attach to an environment
type EnvironmentKey struct {
	Key
	EnvironmentID int64 `json:"environment_id" db:"environment_id"`
}
