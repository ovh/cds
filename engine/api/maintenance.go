package api

import (
	"context"
	"strconv"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (a *API) listenMaintenance(c context.Context) {
	pubSub := a.Cache.Subscribe(sdk.MaintenanceQueueName)
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("listenMaintenance> Exiting: %v", c.Err())
				return
			}
		case <-tick.C:
			msg, err := a.Cache.GetMessageFromSubscription(c, pubSub)
			if err != nil {
				log.Warning("listenMaintenance> Cannot get message %s: %s", msg, err)
				continue
			}
			b, err := strconv.ParseBool(msg)
			if err != nil {
				log.Warning("listenMaintenance> Cannot parse value %s: %s", msg, err)
			}
			a.Maintenance = b
		}
	}
}
