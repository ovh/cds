package api

import (
	"context"
	"strconv"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/sdk"
)

func (a *API) listenMaintenance(c context.Context) error {
	pubSub, err := a.Cache.Subscribe(c, sdk.MaintenanceQueueName)
	if err != nil {
		return sdk.WrapError(err, "listenMaintenance> unable to subscribe to %s", sdk.MaintenanceQueueName)
	}
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				return sdk.WrapError(c.Err(), "listenMaintenance> Exiting")
			}
		case <-tick.C:
			msg, err := pubSub.GetMessage(c)
			if err != nil {
				log.Warn(c, "listenMaintenance> Cannot get message %s: %s", msg, err)
				continue
			}
			b, err := strconv.ParseBool(msg)
			if err != nil {
				log.Warn(c, "listenMaintenance> Cannot parse value %s: %s", msg, err)
			}
			a.Maintenance = b
			event.PublishMaintenanceEvent(c, sdk.EventMaintenance{Enable: b})
		}
	}
}
