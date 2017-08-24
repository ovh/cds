package sdk

const (
	KeyTypeSsh = "ssh"
	KeyTypeGpg = "gpg"
)

// Key represent a key of type SSH or GPG.
type Key struct {
	Name    string `json:"name" db:"name"`
	Public  string `json:"public" db:"public"`
	Private string `json:"private" db:"private"`
	Type    string `json:"type" db:"type"`
}

// Key represent a key attach to a project
type ProjectKey struct {
	Key
	ProjectID int64 `json:"project_id" db:"project_id"`
}

// Key represent a key attach to an application
type ApplicationKey struct {
	Key
	ApplicationID int64 `json:"application_id" db:"application_id"`
}

// Key represent a key attach to an environment
type EnvironmentKey struct {
	Key
	EnvironmentID int64 `json:"environment_id" db:"environment_id"`
}
