package kafkapublisher

import "bytes"

//File represents a file
type File struct {
	ContextID    *int64        `json:"context_id,omitempty"`
	Name         string        `json:"filename,omitempty"`
	ID           string        `json:"file_id,omitempty"`
	Content      *bytes.Buffer `json:"-"`
	ChunksNumber int           `json:"chunks_number,omitempty"`
}

//Chunk represents a piece of file
type Chunk struct {
	ContextID *int64 `json:"context_id,omitempty"`
	Filename  string `json:"filename,omitempty"`
	FileID    string `json:"file_id,omitempty"`
	Content   []byte `json:"content,omitempty"`
	Offset    int    `json:"offset,omitempty"`
}

//Chunks is a list of chunks
type Chunks []Chunk

//Context represent a CDS stream context
type Context struct {
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
