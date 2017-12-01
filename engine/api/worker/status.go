package worker

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-gorp/gorp"
)

var (
	lastRequest time.Time
	status      string
	mux         sync.Mutex
)

//Status returns info about worker Model Status
func Status(db *gorp.DbMap) string {
	if time.Now().Sub(lastRequest) > 2*time.Second {
		queryCount := `select count(worker_model.id)
    from worker_model
    where nb_spawn_err > 0`

		count, errc := db.SelectInt(queryCount)
		mux.Lock()
		if errc != nil {
			status = fmt.Sprintf("Status> unable to load worker_model in error:%s", errc)
		} else {
			status = fmt.Sprintf("%d", count)
		}
		mux.Unlock()

		lastRequest = time.Now()
	}
	return status
}
