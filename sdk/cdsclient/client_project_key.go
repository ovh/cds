package cdsclient

import (
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectListKeys(key string) ([]sdk.ProjectKey, error) {
	k := []sdk.ProjectKey{}
	code, err := c.GetJSON("/project/"+key+"/keys", &k)
	if code != 200 {
		if err == nil {
			return nil, fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return nil, err
	}
	return k, nil
}
