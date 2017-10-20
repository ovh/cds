package gitlab

import (
	"fmt"
	"time"

	"github.com/ovh/cds/sdk"
)

//GetEvents is not implemented
func (c *gitlabClient) GetEvents(repo string, dateRef time.Time) ([]interface{}, time.Duration, error) {
	return nil, 0.0, fmt.Errorf("Not implemented on Gitlab")
}

//PushEvents is not implemented
func (c *gitlabClient) PushEvents(string, []interface{}) ([]sdk.VCSPushEvent, error) {
	return nil, fmt.Errorf("Not implemented on Gitlab")
}

//CreateEvents is not implemented
func (c *gitlabClient) CreateEvents(string, []interface{}) ([]sdk.VCSCreateEvent, error) {
	return nil, fmt.Errorf("Not implemented on Gitlab")
}

//DeleteEvents is not implemented
func (c *gitlabClient) DeleteEvents(string, []interface{}) ([]sdk.VCSDeleteEvent, error) {
	return nil, fmt.Errorf("Not implemented on Gitlab")
}

//PullRequestEvents is not implemented
func (c *gitlabClient) PullRequestEvents(string, []interface{}) ([]sdk.VCSPullRequestEvent, error) {
	return nil, fmt.Errorf("Not implemented on Gitlab")
}
