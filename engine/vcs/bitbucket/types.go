package bitbucket

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// WebHook Represent a webhook in bitbucket model
type WebHook struct {
	ID            int               `json:"id,omitempty"`
	Active        bool              `json:"active"`
	Configuration map[string]string `json:"configuration"`
	Events        []string          `json:"events"`
	Name          string            `json:"name"`
	URL           string            `json:"url"`
}

type Branch struct {
	ID         string `json:"id"`
	DisplayID  string `json:"displayId"`
	LatestHash string `json:"latestChangeset"`
	IsDefault  bool   `json:"isDefault"`
}

type BranchResponse struct {
	Values     []Branch `json:"values"`
	Size       int      `json:"size"`
	IsLastPage bool     `json:"isLastPage"`
}

type Author struct {
	Name  string `json:"name"`
	Email string `json:"emailAddress"`
}

type CommitsResponse struct {
	Values        []Commit `json:"values"`
	Size          int      `json:"size"`
	NextPageStart int      `json:"nextPageStart"`
	IsLastPage    bool     `json:"isLastPage"`
}

type Commit struct {
	Hash      string  `json:"id"`
	Author    *Author `json:"author"`
	Timestamp int64   `json:"authorTimestamp"`
	Message   string  `json:"message"`
}

type Status struct {
	Description string `json:"description"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	State       string `json:"state"`
	URL         string `json:"url"`
	Timestamp   int64  `json:"dateAdded"`
}

type Lines struct {
	Text string `json:"text"`
}

type Content struct {
	Lines []Lines `json:"lines"`
}

type HookDetail struct {
	Key           string `json:"key"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Description   string `json:"description"`
	Version       string `json:"version"`
	ConfigFormKey string `json:"configFormKey"`
}

type HooksConfig struct {
	Version       string
	LocationCount string
	Details       []HookConfigDetail
}

func (h *HooksConfig) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	m["version"] = h.Version
	m["locationCount"] = fmt.Sprintf("%d", len(h.Details))
	for i, d := range h.Details {
		var keySuffix string
		if i > 0 {
			keySuffix = fmt.Sprintf("%d", i+1)
		}

		m["httpMethod"+keySuffix] = d.Method
		m["url"+keySuffix] = d.URL
		m["postContentType"+keySuffix] = d.PostContentType
		m["postData"+keySuffix] = d.PostData
		m["branchFilter"+keySuffix] = d.BranchFilter
		m["tagFilter"+keySuffix] = d.TagFilter
		m["userFilter"+keySuffix] = d.UserFilter
		m["skipSsl"+keySuffix] = d.SkipSsl
		m["useAuth"+keySuffix] = d.UseAuth
	}
	return json.Marshal(m)
}

func (h *HooksConfig) UnmarshalJSON(b []byte) error {
	m := make(map[string]interface{})
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	var nbLocation int
	for k := range m {
		if strings.HasPrefix(k, "url") {
			nbLocation++
		}
	}
	h.LocationCount = fmt.Sprintf("%d", nbLocation)
	h.Version = "3"
	h.Details = make([]HookConfigDetail, nbLocation)
	for i := 0; i < nbLocation; i++ {
		var keySuffix string
		if i > 0 {
			keySuffix = fmt.Sprintf("%d", i+1)
		}
		skipSsl, _ := strconv.ParseBool(fmt.Sprintf("%s", m["skipSsl"+keySuffix]))
		useAuth, _ := strconv.ParseBool(fmt.Sprintf("%s", m["useAuth"+keySuffix]))
		h.Details[i] = HookConfigDetail{
			BranchFilter:    fmt.Sprintf("%s", m["branchFilter"+keySuffix]),
			Method:          fmt.Sprintf("%s", m["httpMethod"+keySuffix]),
			PostContentType: fmt.Sprintf("%s", m["postContentType"+keySuffix]),
			PostData:        fmt.Sprintf("%s", m["postData"+keySuffix]),
			TagFilter:       fmt.Sprintf("%s", m["tagFilter"+keySuffix]),
			URL:             fmt.Sprintf("%s", m["url"+keySuffix]),
			UserFilter:      fmt.Sprintf("%s", m["userFilter"+keySuffix]),
			SkipSsl:         skipSsl,
			UseAuth:         useAuth,
		}
	}
	return nil
}

type HookConfigDetail struct {
	Method          string `json:"httpMethod"`
	URL             string `json:"url"`
	PostContentType string `json:"postContentType"`
	PostData        string `json:"postData"`
	BranchFilter    string `json:"branchFilter"`
	TagFilter       string `json:"tagFilter"`
	UserFilter      string `json:"userFilter"`
	SkipSsl         bool   `json:"skipSsl"`
	UseAuth         bool   `json:"useAuth"`
}

type Hook struct {
	Enabled bool        `json:"enabled"`
	Details *HookDetail `json:"details"`
}

type Key struct {
	ID    int64  `json:"id"`
	Text  string `json:"text"`
	Label string `json:"label"`
}

type Keys struct {
	Values []Key `json:"values"`
}

type Response struct {
	Values        []Repo `json:"values"`
	Size          int    `json:"size"`
	NextPageStart int    `json:"nextPageStart"`
	IsLastPage    bool   `json:"isLastPage"`
}

type ResponseStatus struct {
	Values        []Status `json:"values"`
	Size          int      `json:"size"`
	NextPageStart int      `json:"nextPageStart"`
	IsLastPage    bool     `json:"isLastPage"`
}

type Project struct {
	Key string `json:"key"`
}

type Repo struct {
	Name    string   `json:"name"`
	Slug    string   `json:"slug"`
	Public  bool     `json:"public"`
	ScmId   string   `json:"scmId"`
	Project *Project `json:"project"`
	Link    *Link    `json:"link"`
	Links   *Links   `json:"links"`
}

type Links struct {
	Clone []Clone `json:"clone"`
	Self  []Clone `json:"self"`
}

type Clone struct {
	URL  string `json:"href"`
	Name string `json:"name"`
}

type Link struct {
	URL string `json:"url"`
	Rel string `json:"rel"`
}

type UsersResponse struct {
	Values        []User `json:"values"`
	Size          int    `json:"size"`
	NextPageStart int    `json:"nextPageStart"`
	IsLastPage    bool   `json:"isLastPage"`
}

type User struct {
	Username     string `json:"name"`
	EmailAddress string `json:"emailAddress"`
	DisplayName  string `json:"displayName"`
	Slug         string `json:"slug"`
}

type PullRequest struct {
	ID          int    `json:"id"`
	Version     int    `json:"version"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	Open        bool   `json:"open"`
	Closed      bool   `json:"closed"`
	CreatedDate int    `json:"createdDate"`
	UpdatedDate int    `json:"updatedDate"`
	FromRef     struct {
		ID         string `json:"id"`
		Repository struct {
			Slug    string      `json:"slug"`
			Name    interface{} `json:"name"`
			Project struct {
				Key string `json:"key"`
			} `json:"project"`
		} `json:"repository"`
	} `json:"fromRef"`
	ToRef struct {
		ID         string `json:"id"`
		Repository struct {
			Slug    string      `json:"slug"`
			Name    interface{} `json:"name"`
			Project struct {
				Key string `json:"key"`
			} `json:"project"`
		} `json:"repository"`
	} `json:"toRef"`
	Locked bool `json:"locked"`
	Author struct {
		User struct {
			Name         string `json:"name"`
			EmailAddress string `json:"emailAddress"`
			ID           int    `json:"id"`
			DisplayName  string `json:"displayName"`
			Active       bool   `json:"active"`
			Slug         string `json:"slug"`
			Type         string `json:"type"`
		} `json:"user"`
		Role     string `json:"role"`
		Approved bool   `json:"approved"`
		Status   string `json:"status"`
	} `json:"author"`
	Reviewers []struct {
		User struct {
			Name         string `json:"name"`
			EmailAddress string `json:"emailAddress"`
			ID           int    `json:"id"`
			DisplayName  string `json:"displayName"`
			Active       bool   `json:"active"`
			Slug         string `json:"slug"`
			Type         string `json:"type"`
		} `json:"user"`
		LastReviewedCommit string `json:"lastReviewedCommit"`
		Role               string `json:"role"`
		Approved           bool   `json:"approved"`
		Status             string `json:"status"`
	} `json:"reviewers"`
	Participants []struct {
		User struct {
			Name         string `json:"name"`
			EmailAddress string `json:"emailAddress"`
			ID           int    `json:"id"`
			DisplayName  string `json:"displayName"`
			Active       bool   `json:"active"`
			Slug         string `json:"slug"`
			Type         string `json:"type"`
		} `json:"user"`
		Role     string `json:"role"`
		Approved bool   `json:"approved"`
		Status   string `json:"status"`
	} `json:"participants"`
	Links struct {
		Self []struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"links"`
}

type PullRequestResponse struct {
	Values        []PullRequest `json:"values"`
	Size          int           `json:"size"`
	NextPageStart int           `json:"nextPageStart"`
	IsLastPage    bool          `json:"isLastPage"`
}
