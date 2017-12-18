package worker

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

var (
	lastRequest    time.Time
	statusNbErrors string
	mux            sync.Mutex
)

//Status returns info about worker Model Status
func Status(db *gorp.DbMap) sdk.MonitoringStatusLine {
	status := sdk.MonitoringStatusOK
	if time.Now().Sub(lastRequest) > 2*time.Second {
		queryCount := `select count(worker_model.id) from worker_model where nb_spawn_err > 0`

		count, errc := db.SelectInt(queryCount)
		mux.Lock()
		if errc != nil {
			statusNbErrors = fmt.Sprintf("Status> unable to load worker_model in error:%s", errc)
			status = sdk.MonitoringStatusAlert
		} else {
			statusNbErrors = fmt.Sprintf("%d", count)
			if count > 0 {
				status = sdk.MonitoringStatusWarn
			} else {
				status = sdk.MonitoringStatusOK
			}
		}
		mux.Unlock()

		lastRequest = time.Now()
	}

	return sdk.MonitoringStatusLine{Component: "Worker Model Errors", Value: statusNbErrors, Status: status}
}
