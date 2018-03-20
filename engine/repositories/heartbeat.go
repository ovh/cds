package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) heartbeat(ctx context.Context) error {
	ticker := time.NewTicker(30 * time.Second)
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	var heartbeatFailures int
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := s.doHeartbeat(); err != nil {
				log.Error("Repositories> heartbeat> Heartbeat failed")
				heartbeatFailures++
				if heartbeatFailures > s.Cfg.API.MaxHeartbeatFailures {
					return fmt.Errorf("Heartbeat failed excedeed")
				}
			}
			heartbeatFailures = 0
		}
	}
}

func (s *Service) doHeartbeat() error {
	srv := sdk.Service{
		Name:          s.Cfg.Name,
		HTTPURL:       s.Cfg.URL,
		LastHeartbeat: time.Time{},
		Token:         s.Cfg.API.Token,
		Type:          services.TypeRepositories,
	}
	log.Debug("Repositories> doHeartbeat: %+v", srv)
	hash, err := s.cds.ServiceRegister(srv)
	if err != nil {
		return sdk.WrapError(err, "doHeartbeat")
	}
	s.hash = hash
	return nil
}
