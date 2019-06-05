package hooks

import "time"

// BitbucketServerPushEvent represents payload send by bitbucket cloud on a push event
type BitbucketCloudPushEvent struct {
	Push struct {
		Changes []struct {
			Forced bool `json:"forced"`
			Old    struct {
				Name  string `json:"name"`
				Links struct {
					Commits struct {
						Href string `json:"href"`
					} `json:"commits"`
					Self struct {
						Href string `json:"href"`
					} `json:"self"`
					HTML struct {
						Href string `json:"href"`
					} `json:"html"`
				} `json:"links"`
				DefaultMergeStrategy string   `json:"default_merge_strategy"`
				MergeStrategies      []string `json:"merge_strategies"`
				Type                 string   `json:"type"`
				Target               struct {
					Rendered struct {
					} `json:"rendered"`
					Hash  string `json:"hash"`
					Links struct {
						Self struct {
							Href string `json:"href"`
						} `json:"self"`
						HTML struct {
							Href string `json:"href"`
						} `json:"html"`
					} `json:"links"`
					Author struct {
						Raw  string `json:"raw"`
						Type string `json:"type"`
						User struct {
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
							Nickname  string `json:"nickname"`
							Type      string `json:"type"`
							AccountID string `json:"account_id"`
						} `json:"user"`
					} `json:"author"`
					Summary struct {
						Raw    string `json:"raw"`
						Markup string `json:"markup"`
						HTML   string `json:"html"`
						Type   string `json:"type"`
					} `json:"summary"`
					Parents []struct {
						Hash  string `json:"hash"`
						Type  string `json:"type"`
						Links struct {
							Self struct {
								Href string `json:"href"`
							} `json:"self"`
							HTML struct {
								Href string `json:"href"`
							} `json:"html"`
						} `json:"links"`
					} `json:"parents"`
					Date       time.Time `json:"date"`
					Message    string    `json:"message"`
					Type       string    `json:"type"`
					Properties struct {
					} `json:"properties"`
				} `json:"target"`
			} `json:"old"`
			Links struct {
				Commits struct {
					Href string `json:"href"`
				} `json:"commits"`
				HTML struct {
					Href string `json:"href"`
				} `json:"html"`
				Diff struct {
					Href string `json:"href"`
				} `json:"diff"`
			} `json:"links"`
			Created bool `json:"created"`
			Commits []struct {
				Rendered struct {
				} `json:"rendered"`
				Hash  string `json:"hash"`
				Links struct {
					Self struct {
						Href string `json:"href"`
					} `json:"self"`
					Comments struct {
						Href string `json:"href"`
					} `json:"comments"`
					Patch struct {
						Href string `json:"href"`
					} `json:"patch"`
					HTML struct {
						Href string `json:"href"`
					} `json:"html"`
					Diff struct {
						Href string `json:"href"`
					} `json:"diff"`
					Approve struct {
						Href string `json:"href"`
					} `json:"approve"`
					Statuses struct {
						Href string `json:"href"`
					} `json:"statuses"`
				} `json:"links"`
				Author struct {
					Raw  string `json:"raw"`
					Type string `json:"type"`
					User struct {
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
						Nickname  string `json:"nickname"`
						Type      string `json:"type"`
						AccountID string `json:"account_id"`
					} `json:"user"`
				} `json:"author"`
				Summary struct {
					Raw    string `json:"raw"`
					Markup string `json:"markup"`
					HTML   string `json:"html"`
					Type   string `json:"type"`
				} `json:"summary"`
				Parents []struct {
					Hash  string `json:"hash"`
					Type  string `json:"type"`
					Links struct {
						Self struct {
							Href string `json:"href"`
						} `json:"self"`
						HTML struct {
							Href string `json:"href"`
						} `json:"html"`
					} `json:"links"`
				} `json:"parents"`
				Date       time.Time `json:"date"`
				Message    string    `json:"message"`
				Type       string    `json:"type"`
				Properties struct {
				} `json:"properties"`
			} `json:"commits"`
			Truncated bool `json:"truncated"`
			Closed    bool `json:"closed"`
			New       struct {
				Name  string `json:"name"`
				Links struct {
					Commits struct {
						Href string `json:"href"`
					} `json:"commits"`
					Self struct {
						Href string `json:"href"`
					} `json:"self"`
					HTML struct {
						Href string `json:"href"`
					} `json:"html"`
				} `json:"links"`
				DefaultMergeStrategy string   `json:"default_merge_strategy"`
				MergeStrategies      []string `json:"merge_strategies"`
				Type                 string   `json:"type"`
				Target               struct {
					Rendered struct {
					} `json:"rendered"`
					Hash  string `json:"hash"`
					Links struct {
						Self struct {
							Href string `json:"href"`
						} `json:"self"`
						HTML struct {
							Href string `json:"href"`
						} `json:"html"`
					} `json:"links"`
					Author struct {
						Raw  string `json:"raw"`
						Type string `json:"type"`
						User struct {
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
							Nickname  string `json:"nickname"`
							Type      string `json:"type"`
							AccountID string `json:"account_id"`
						} `json:"user"`
					} `json:"author"`
					Summary struct {
						Raw    string `json:"raw"`
						Markup string `json:"markup"`
						HTML   string `json:"html"`
						Type   string `json:"type"`
					} `json:"summary"`
					Parents []struct {
						Hash  string `json:"hash"`
						Type  string `json:"type"`
						Links struct {
							Self struct {
								Href string `json:"href"`
							} `json:"self"`
							HTML struct {
								Href string `json:"href"`
							} `json:"html"`
						} `json:"links"`
					} `json:"parents"`
					Date       time.Time `json:"date"`
					Message    string    `json:"message"`
					Type       string    `json:"type"`
					Properties struct {
					} `json:"properties"`
				} `json:"target"`
			} `json:"new"`
		} `json:"changes"`
	} `json:"push"`
	Actor struct {
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
		Nickname  string `json:"nickname"`
		Type      string `json:"type"`
		AccountID string `json:"account_id"`
	} `json:"actor"`
	Repository struct {
		Scm     string `json:"scm"`
		Website string `json:"website"`
		Name    string `json:"name"`
		Links   struct {
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
		FullName string `json:"full_name"`
		Owner    struct {
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
			Nickname  string `json:"nickname"`
			Type      string `json:"type"`
			AccountID string `json:"account_id"`
		} `json:"owner"`
		Type      string `json:"type"`
		IsPrivate bool   `json:"is_private"`
		UUID      string `json:"uuid"`
	} `json:"repository"`
}
