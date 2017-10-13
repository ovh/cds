package bitbucket

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
	Hash      string  `json:"Id"`
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
}

type Lines struct {
	Text string `"json:text"`
}

type Content struct {
	Lines []Lines `"json:lines"`
}

type HookDetail struct {
	Key           string `json:"key"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Description   string `json:"description"`
	Version       string `json:"version"`
	ConfigFormKey string `json:"configFormKey"`
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
