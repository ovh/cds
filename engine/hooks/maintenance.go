package hooks

import (
	"context"
	"strconv"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

func (s *Service) listenMaintenance(c context.Context) error {
	pubSub, err := s.Dao.store.Subscribe(c, MaintenanceHookQueue)
	if err != nil {
		return sdk.WrapError(err, "listenMaintenance> unable to subscribe to %s", MaintenanceHookQueue)
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
			s.Maintenance = b
		}
	}
}
