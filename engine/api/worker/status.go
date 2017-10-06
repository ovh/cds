package worker

import (
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
)

var (
	lastRequest time.Time

	status      string
	healthy     bool
	statusError error
)

//Status returns info about worker Model Status
func Status(db *gorp.DbMap) (string, bool, error) {
	if time.Now().Sub(lastRequest) > 2*time.Second {
		queryCount := `select count(worker_model.id)
    from worker_model
    where nb_spawn_err > 0`

		count, errc := db.SelectInt(queryCount)
		if errc != nil {
			status = "Unable to load worker_model in error"
			statusError = errc
			healthy = false
		} else {
			healthy = true
			status = fmt.Sprintf("%d Errors", count)
			if count > 0 {
				statusError = fmt.Errorf("%d Errors", count)
			}
		}

		lastRequest = time.Now()
	}
	return status, healthy, statusError
}
