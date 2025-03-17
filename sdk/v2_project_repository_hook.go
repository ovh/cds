package sdk

import "time"

type ProjectRepositoryHook struct {
	ID         string    `json:"id" db:"id" cli:"id"`
	ProjectKey string    `json:"project_key" db:"project_key" cli:"project_key"`
	VCSServer  string    `json:"vcs_server" db:"vcs_server" cli:"vcs_server"`
	Repository string    `json:"repository" db:"repository" cli:"repository"`
	Created    time.Time `json:"created" db:"created" cli:"created"`
	Username   string    `json:"username" db:"username" cli:"username"`
}

type PostProjectRepositoryHook struct {
	VCSServer  string `json:"vcs_server"`
	Repository string `json:"repository"`
}
