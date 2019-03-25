package repo

import (
	"regexp"
	"time"
)

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
	Files    map[string]File
}

type File struct {
	Filename   string
	Status     string
	Diff       string
	DiffDetail FileDiffDetail
}

type FileDiffDetail struct {
	Hunks []Hunk
}

func (d FileDiffDetail) Matches(regexp *regexp.Regexp) (hunks []Hunk, addedLinewMatch bool, removedLinewMatch bool) {
	for _, h := range d.Hunks {
		var hunkMatches bool
		for _, l := range h.RemovedLines {
			if regexp.MatchString(l) {
				removedLinewMatch = true
				break
			}
		}
		for _, l := range h.AddedLines {
			if regexp.MatchString(l) {
				addedLinewMatch = true
				break
			}
		}
		if hunkMatches {
			hunks = append(hunks, h)
		}
	}
	return hunks, addedLinewMatch, removedLinewMatch
}

type Hunk struct {
	Header       string
	Content      string
	RemovedLines []string
	AddedLines   []string
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
