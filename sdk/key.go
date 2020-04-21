package sdk

import (
	"fmt"
	"strings"
)

type KeyType string

func (k KeyType) String() string {
	return string(k)
}

// Those are types if key managed in CDS
const (
	KeyTypeSSH KeyType = "ssh"
	KeyTypePGP KeyType = "pgp"
)

func GenerateProjectDefaultKeyName(projectKey string, t KeyType) string {
	return fmt.Sprintf("proj-%s-%s", t, strings.ToLower(projectKey))
}

// Key represent a key of type SSH or GPG.
type Key struct {
	ID      int64   `json:"id" db:"id" cli:"-"`
	Name    string  `json:"name" db:"name" cli:"name"`
	Public  string  `json:"public" db:"public" cli:"publickey"`
	Private string  `json:"private" db:"private" cli:"-"`
	KeyID   string  `json:"keyID" db:"key_id" cli:"-"`
	Type    KeyType `json:"type" db:"type" cli:"type"`
}

// ProjectKey represent a key attach to a project
type ProjectKey struct {
	ID        int64   `json:"id" db:"id" cli:"-"`
	Name      string  `json:"name" db:"name" cli:"name"`
	Public    string  `json:"public" db:"public" cli:"publickey"`
	Private   string  `json:"private" db:"private" cli:"-" gorpmapping:"encrypted,ID,Name"`
	KeyID     string  `json:"key_id" db:"key_id" cli:"-"`
	Type      KeyType `json:"type" db:"type" cli:"type"`
	ProjectID int64   `json:"project_id" db:"project_id" cli:"-"`
	Builtin   bool    `json:"-" db:"builtin" cli:"-"`
}

// ApplicationKey represent a key attach to an application
type ApplicationKey struct {
	ID            int64   `json:"id" db:"id" cli:"-"`
	Name          string  `json:"name" db:"name" cli:"name"`
	Public        string  `json:"public" db:"public" cli:"publickey"`
	Private       string  `json:"private" db:"private" cli:"-" gorpmapping:"encrypted,ID,Name"`
	KeyID         string  `json:"key_id" db:"key_id" cli:"-"`
	Type          KeyType `json:"type" db:"type" cli:"type"`
	ApplicationID int64   `json:"application_id" db:"application_id"`
}

// EnvironmentKey represent a key attach to an environment
type EnvironmentKey struct {
	ID            int64   `json:"id" db:"id" cli:"-"`
	Name          string  `json:"name" db:"name" cli:"name"`
	Public        string  `json:"public" db:"public" cli:"publickey"`
	Private       string  `json:"private" db:"private" cli:"-" gorpmapping:"encrypted,ID,Name"`
	KeyID         string  `json:"key_id" db:"key_id" cli:"-"`
	Type          KeyType `json:"type" db:"type" cli:"type"`
	EnvironmentID int64   `json:"environment_id" db:"environment_id"`
}
