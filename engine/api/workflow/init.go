package workflow

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk/log"
)

var baseUIURL, defaultOS, defaultArch string

//Initialize starts goroutines for workflows
func Initialize(c context.Context, DBFunc func() *gorp.DbMap, uiURL, confDefaultOS, confDefaultArch string) {
	baseUIURL = uiURL
	defaultOS = confDefaultOS
	defaultArch = confDefaultArch
	tickStop := time.NewTicker(30 * time.Minute)
	defer tickStop.Stop()

	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting workflow ticker: %v", c.Err())
				return
			}
		case <-tickStop.C:
			if err := stopRunsBlocked(DBFunc()); err != nil {
				log.Warning("workflow.stopRunsBlocked> Error on stopRunsBlocked : %v", err)
			}
		}
	}
}
