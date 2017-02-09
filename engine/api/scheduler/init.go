package scheduler

import "github.com/go-gorp/gorp"

//Initialize starts the 3 goroutines for pipeline schedulers
func Initialize(DBFunc func() *gorp.DbMap, nbExecToKeep int) {
	go Cleaner(DBFunc, nbExecToKeep)
	go Executer(DBFunc)
	go Scheduler(DBFunc)
}
