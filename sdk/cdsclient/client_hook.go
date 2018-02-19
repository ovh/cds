package cdsclient

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ovh/cds/sdk"
)

func (c *client) PollVCSEvents(uuid string, workflowID int64, vcsServer string, timestamp int64) (events sdk.RepositoryEvents, interval time.Duration, err error) {
	url := fmt.Sprintf("/hook/%s/workflow/%d/vcsevent/%s", uuid, workflowID, vcsServer)
	header, _, errGet := c.GetJSONWithHeaders(url, &events, SetHeader("X-CDS-Last-Execution", fmt.Sprint(timestamp)))
	if errGet != nil {
		return events, interval, errGet
	}

	//Check poll interval
	if header.Get("X-CDS-Poll-Interval") != "" {
		f, errParse := strconv.ParseFloat(header.Get("X-CDS-Poll-Interval"), 64)
		if errParse == nil {
			interval = time.Duration(f) * time.Second
		}
	}

	return events, interval, nil
}
