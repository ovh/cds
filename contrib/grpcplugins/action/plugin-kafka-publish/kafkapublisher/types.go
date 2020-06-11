package kafkapublisher

//Context represent a CDS stream context
type Context struct {
	Version       string          `json:"version"`
	ActionID      int64           `json:"action_id"`
	Directory     string          `json:"directory"`
	Files         []string        `json:"files"`
	ReceivedFiles map[string]bool `json:"-"`
	Closed        bool            `json:"-"`
}

//Ack is an acknoledgemnt for a context
type Ack struct {
	Context Context `json:"context"`
	Result  string  `json:"result"`
	Log     []byte  `json:"log,omitempty"`
}

//Artifact is an artifact send from the plugin receiver to the plugin
type Artifact struct {
	Context Context `json:"context"`
	Name    string  `json:"name"`
	Tag     string  `json:"tag"`
	Content []byte  `json:"content"`
}
