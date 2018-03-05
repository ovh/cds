package repo

import "time"

// Repo is the main type of this lib
type Repo struct {
	path    string
	url     string
	sshKey  *sshKey
	pgpKey  *pgpKey
	verbose bool
	logger  func(format string, i ...interface{})
}

// Commit represent a git commit
type Commit struct {
	LongHash string
	Hash     string
	Author   string
	Subject  string
	Body     string
	Date     time.Time
}

// CloneOpts is a optional structs for git clone command
type CloneOpts struct {
	Recursive               *bool
	NoStrictHostKeyChecking *bool
	Auth                    *AuthOpts
}

// AuthOpts is a optional structs for git command
type AuthOpts struct {
	Username   string
	Password   string
	PrivateKey *SSHKey
	SignKey    *PGPKey
}

// SSHKey is a type for a ssh key
type SSHKey struct {
	Filename string
	Content  []byte
}

// PGPKey is a type for a pgp key
type PGPKey struct {
	Name    string
	Public  string
	Private string
	ID      string
}
