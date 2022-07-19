package hooks

import "time"

type GiteaEventPayload struct {
	Secret     string `json:"secret"`
	Ref        string `json:"ref"`    // refs/heads/branch_name
	Before     string `json:"before"` // commit before
	After      string `json:"after"`  // commit aster
	CompareUrl string `json:"compare_url"`
	Commits    []struct {
		Id      string `json:"id"`
		Message string `json:"message"`
		Url     string `json:"url"`
		Author  struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Username string `json:"username"`
		} `json:"author"`
		Committer struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Username string `json:"username"`
		} `json:"committer"`
		Timestamp time.Time `json:"timestamp"`
	} `json:"commits"`
	Repository struct {
		Id    int `json:"id"`
		Owner struct {
			Id        int    `json:"id"`
			Login     string `json:"login"`
			FullName  string `json:"full_name"`
			Email     string `json:"email"`
			AvatarUrl string `json:"avatar_url"`
			Username  string `json:"username"`
		} `json:"owner"`
		Name            string    `json:"name"`
		FullName        string    `json:"full_name"`
		Description     string    `json:"description"`
		Private         bool      `json:"private"`
		Fork            bool      `json:"fork"`
		HtmlUrl         string    `json:"html_url"`
		SshUrl          string    `json:"ssh_url"`
		CloneUrl        string    `json:"clone_url"`
		Website         string    `json:"website"`
		StarsCount      int       `json:"stars_count"`
		ForksCount      int       `json:"forks_count"`
		WatchersCount   int       `json:"watchers_count"`
		OpenIssuesCount int       `json:"open_issues_count"`
		DefaultBranch   string    `json:"default_branch"`
		CreatedAt       time.Time `json:"created_at"`
		UpdatedAt       time.Time `json:"updated_at"`
	} `json:"repository"`
	Pusher struct {
		Id        int    `json:"id"`
		Login     string `json:"login"`
		FullName  string `json:"full_name"`
		Email     string `json:"email"`
		AvatarUrl string `json:"avatar_url"`
		Username  string `json:"username"`
	} `json:"pusher"`
	Sender struct {
		Id        int    `json:"id"`
		Login     string `json:"login"`
		FullName  string `json:"full_name"`
		Email     string `json:"email"`
		AvatarUrl string `json:"avatar_url"`
		Username  string `json:"username"`
	} `json:"sender"`
}
