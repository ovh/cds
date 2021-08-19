package kafkapublisher

import (
	"fmt"
	"path"

	"github.com/ovh/cds/sdk"
)

//GetContext returns a context
func GetContext(data []byte) (*Context, bool) {
	c := &Context{}
	if err := sdk.JSONUnmarshal(data, c); err != nil {
		return nil, false
	}
	c.ReceivedFiles = map[string]bool{}
	for _, f := range c.Files {
		c.ReceivedFiles[f] = false
	}
	return c, true
}

//NewContext returns a context
func NewContext(actionID int64, files []string) *Context {
	fileNames := []string{}
	for _, f := range files {
		fileNames = append(fileNames, path.Base(f))
	}

	return &Context{
		ActionID:  actionID,
		Directory: path.Join(".", fmt.Sprintf("%d", actionID)),
		Files:     fileNames,
	}
}

//IsComplete return true if all files for this context have been received
func (c Context) IsComplete() bool {
	for _, v := range c.ReceivedFiles {
		if !v {
			return false
		}
	}
	return true
}
