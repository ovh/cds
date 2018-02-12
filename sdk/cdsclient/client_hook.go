package cdsclient

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ovh/cds/sdk"
)

func (c *client) PollVCSEvents(uuid string) (events sdk.RepositoryEvents, interval time.Duration, err error) {
	url := fmt.Sprintf("/hook/%s/vcsevent", uuid)
	header, _, errGet := c.GetJSONWithHeaders(url, &events)
	if errGet != nil {
		return events, interval, errGet
	}

	//Check poll interval
	if header.Get("X-Poll-Interval") != "" {
		f, errParse := strconv.ParseFloat(header.Get("X-Poll-Interval"), 64)
		if errParse == nil {
			interval = time.Duration(f) * time.Second
		}
	}

	return events, interval, nil
}
