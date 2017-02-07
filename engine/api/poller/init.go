package poller

//Initialize starts the 3 goroutines for pipeline schedulers
func Initialize(nbExecToKeep int) {
	go Cleaner(nbExecToKeep)
	go Executer()
	go Scheduler()
}
