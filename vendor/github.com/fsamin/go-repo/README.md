# go-repo

Go-Repo is just wrapper arround git commands.

[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](http://godoc.org/github.com/fsamin/go-repo) [![Build Status](https://travis-ci.org/fsamin/go-repo.svg?branch=master)](https://travis-ci.org/fsamin/go-repo) [![Go Report Card](https://goreportcard.com/badge/github.com/fsamin/go-repo)](https://goreportcard.com/report/github.com/fsamin/go-repo)

## Clone a repository

````golang
    r, err := Clone(path, "https://github.com/fsamin/go-repo.git")
    ...
````

## Get current branch
````golang
    r, err := Clone(path, "https://github.com/fsamin/go-repo.git")
    ...
    b, err := r.CurrentBranch()
    ...
    fmt.Println(b)
````

## Fetch & Pull a remote branch
````golang
    r, err := Clone(path, "https://github.com/fsamin/go-repo.git")
    ...
    err = r.FetchRemoteBranch("origin", "tests")
    ...
    err = r.Pull("origin", "tests")
    ...
````

## Git local config
````golang 
    r, err := Clone(path, "https://github.com/fsamin/go-repo.git")
    ...
    r.LocalConfigSet("foo", "bar", "value"))
    val, err := r.LocalConfigGet("foo", "bar")
    ...
````

## Search and open files
````golang 
    r, err := Clone(path, "https://github.com/fsamin/go-repo.git")
    ...
    files, err := r.Glob("**/*.md")
    ...
    f, err := r.Open(files[0])
    ...
````
