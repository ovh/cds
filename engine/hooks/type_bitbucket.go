package hooks

import (
	"github.com/ovh/cds/sdk"
	"time"
)

// BitbucketPushEvent represents payload send by github on a push event
type BitbucketPushEvent struct {
	Actor      User       `json:"actor"`
	Repository Repository `json:"repository"`
	Push       struct {
		Changes []struct {
			New struct {
				Type   string `json:"type"`
				Name   string `json:"name"`
				Target struct {
					Type    string    `json:"type"`
					Hash    string    `json:"hash"`
					Author  User      `json:"author"`
					Message string    `json:"message"`
					Date    time.Time `json:"date"`
					Parents []struct {
						Type  string `json:"type"`
						Hash  string `json:"hash"`
						Links struct {
							Self struct {
								Href string `json:"href"`
							} `json:"self"`
							HTML struct {
								Href string `json:"href"`
							} `json:"html"`
						} `json:"links"`
					} `json:"parents"`
					Links struct {
						Self struct {
							Href string `json:"href"`
						} `json:"self"`
						HTML struct {
							Href string `json:"href"`
						} `json:"html"`
					} `json:"links"`
				} `json:"target"`
				Links struct {
					Self struct {
						Href string `json:"href"`
					} `json:"self"`
					Commits struct {
						Href string `json:"href"`
					} `json:"commits"`
					HTML struct {
						Href string `json:"href"`
					} `json:"html"`
				} `json:"links"`
			} `json:"new"`
			Old struct {
				Type   string `json:"type"`
				Name   string `json:"name"`
				Target struct {
					Type    string    `json:"type"`
					Hash    string    `json:"hash"`
					Author  User      `json:"author"`
					Message string    `json:"message"`
					Date    time.Time `json:"date"`
					Parents []struct {
						Type  string `json:"type"`
						Hash  string `json:"hash"`
						Links struct {
							Self struct {
								Href string `json:"href"`
							} `json:"self"`
							HTML struct {
								Href string `json:"href"`
							} `json:"html"`
						} `json:"links"`
					} `json:"parents"`
					Links struct {
						Self struct {
							Href string `json:"href"`
						} `json:"self"`
						HTML struct {
							Href string `json:"href"`
						} `json:"html"`
					} `json:"links"`
				} `json:"target"`
				Links struct {
					Self struct {
						Href string `json:"href"`
					} `json:"self"`
					Commits struct {
						Href string `json:"href"`
					} `json:"commits"`
					HTML struct {
						Href string `json:"href"`
					} `json:"html"`
				} `json:"links"`
			} `json:"old"`
			Links struct {
				HTML struct {
					Href string `json:"href"`
				} `json:"html"`
				Diff struct {
					Href string `json:"href"`
				} `json:"diff"`
				Commits struct {
					Href string `json:"href"`
				} `json:"commits"`
			} `json:"links"`
			Created bool `json:"created"`
			Forced  bool `json:"forced"`
			Closed  bool `json:"closed"`
			Commits []struct {
				Hash    string `json:"hash"`
				Type    string `json:"type"`
				Message string `json:"message"`
				Author  User   `json:"author"`
				Links   struct {
					Self struct {
						Href string `json:"href"`
					} `json:"self"`
					HTML struct {
						Href string `json:"href"`
					} `json:"html"`
				} `json:"links"`
			} `json:"commits"`
			Truncated bool `json:"truncated"`
		} `json:"changes"`
	} `json:"push"`
}

type User struct {
	Type        string `json:"type"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	UUID        string `json:"uuid"`
	Links       struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
		Avatar struct {
			Href string `json:"href"`
		} `json:"avatar"`
	} `json:"links"`
}

type Project struct {
	Type    string `json:"type"`
	Project string `json:"project"`
	UUID    string `json:"uuid"`
	Links   struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
		Avatar struct {
			Href string `json:"href"`
		} `json:"avatar"`
	} `json:"links"`
	Key string `json:"key"`
}

type Repository struct {
	Type  string `json:"type"`
	Links struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
		Avatar struct {
			Href string `json:"href"`
		} `json:"avatar"`
	} `json:"links"`
	UUID      string  `json:"uuid"`
	Project   Project `json:"project"`
	FullName  string  `json:"full_name"`
	Name      string  `json:"name"`
	Website   string  `json:"website"`
	Owner     User    `json:"owner"`
	Scm       string  `json:"scm"`
	IsPrivate bool    `json:"is_private"`
}

func (b *BitbucketPushEvent) GetCommits() []sdk.VCSCommit {
	commits := []sdk.VCSCommit{}
	for _, c := range b.Push.Changes[0].Commits {
		commit := sdk.VCSCommit{
			Hash: c.Hash,
			Author: sdk.VCSAuthor{
				Name:        c.Author.Username,
				DisplayName: c.Author.DisplayName,
				Avatar:      c.Author.Links.Avatar.Href,
			},
			Message: c.Message,
			URL:     c.Links.HTML.Href,
		}
		commits = append(commits, commit)
	}
	return commits
}
