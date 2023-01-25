package sdk

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
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
	Disabled  bool    `json:"disabled" db:"disabled" cli:"disabled"`
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

func ImportGPGKey(dir string, keyName string, publicKey string) (string, []byte, error) {
	gpg2Found := false

	if _, err := exec.LookPath("gpg2"); err == nil {
		gpg2Found = true
	}

	if !gpg2Found {
		if _, err := exec.LookPath("gpg"); err != nil {
			return "", nil, NewErrorFrom(ErrNotFound, "command gpg/gpg2 not found")
		}
	}
	content := []byte(publicKey)
	tmpfile, errTmpFile := os.CreateTemp(dir, keyName)
	if errTmpFile != nil {
		return "", content, NewError(ErrUnknownError, fmt.Errorf("cannot setup pgp key %s : %v", keyName, errTmpFile))
	}
	defer func() {
		_ = os.Remove(tmpfile.Name())
	}()

	if _, err := tmpfile.Write(content); err != nil {
		return tmpfile.Name(), content, NewError(ErrUnknownError, fmt.Errorf("cannot setup pgp key file %s : %v", keyName, err))
	}

	if err := tmpfile.Close(); err != nil {
		return tmpfile.Name(), content, NewError(ErrUnknownError, fmt.Errorf("cannot setup pgp key file %s (close) : %v", keyName, err))
	}

	gpgBin := "gpg"
	if gpg2Found {
		gpgBin = "gpg2"
	}
	cmd := exec.Command(gpgBin, "--import", tmpfile.Name())
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return tmpfile.Name(), content, NewError(ErrUnknownError, fmt.Errorf("cannot import pgp key %s : %v", keyName, err))
	}
	return tmpfile.Name(), content, nil
}
