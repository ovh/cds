package hatchery

import (
	"os"
	"time"

	"github.com/ovh/cds/sdk/log"
)

func hearbeat(m Interface, token string, maxFailures int) {
	var failures int
	for {
		time.Sleep(10 * time.Second)
		if m.Hatchery().ID == 0 {
			log.Info("hearbeat> %s Disconnected from CDS engine, trying to register...", m.Hatchery().Name)
			if err := Register(m); err != nil {
				log.Info("hearbeat> %s Cannot register: %s", m.Hatchery().Name, err)
				checkFailures(maxFailures, failures)
				continue
			}
			if m.Hatchery().ID == 0 {
				log.Error("hearbeat> Cannot register hatchery. ID %d", m.Hatchery().ID)
				checkFailures(maxFailures, failures)
				continue
			}
			log.Info("hearbeat> %s Registered back: ID %d with model ID %d", m.Hatchery().Name, m.Hatchery().ID, m.Hatchery().Model.ID)
		}

		if err := m.CDSClient().HatcheryRefresh(m.Hatchery().ID); err != nil {
			log.Info("heartbeat> %s cannot refresh beat: %s", m.Hatchery().Name, err)
			m.Hatchery().ID = 0
			checkFailures(maxFailures, failures)
			continue
		}
		failures = 0
	}
}

func checkFailures(maxFailures, nb int) {
	if nb > maxFailures {
		log.Error("Too many failures on try register. This hatchery is killed")
		os.Exit(10)
	}
}
