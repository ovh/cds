package scheduler

import "time"

//Initialize starts the 3 goroutines for pipeline schedulers
func Initialize(cleanerDelay time.Duration) {
	go Cleaner(cleanerDelay)
	go Executer()
	go Scheduler()
}
