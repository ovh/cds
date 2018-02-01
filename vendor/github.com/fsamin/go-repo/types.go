package repo

import "time"

type Repo struct {
	path string
}

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
