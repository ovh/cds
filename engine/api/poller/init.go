package poller

import (
	"context"

	"github.com/go-gorp/gorp"
)

//Initialize starts the 3 goroutines for pipeline schedulers
func Initialize(c context.Context, nbExecToKeep int, DBFunc func() *gorp.DbMap) {
	go Cleaner(c, DBFunc, nbExecToKeep)
	go Executer(c, DBFunc)
	go Scheduler(c, DBFunc)
}
