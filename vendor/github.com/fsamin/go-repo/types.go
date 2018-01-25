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
