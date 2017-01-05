package scheduler

import "time"

//Initialize starts the 3 goroutines for pipeline schedulers
func Initialize(cleanerDelay time.Duration) {
	Cleaner(cleanerDelay)
	Scheduler()
	Executer()
}
