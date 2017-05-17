package hatchery

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/facebookgo/httpcontrol"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Create creates hatchery
func Create(h Interface, api, token string, maxWorkers, provision int, requestSecondsTimeout int, maxFailures int, insecureSkipVerifyTLS bool, provisionSeconds, registerSeconds, warningSeconds, criticalSeconds, graceSeconds int) {
	Client = &http.Client{
		Transport: &httpcontrol.Transport{
			RequestTimeout:  time.Duration(requestSecondsTimeout) * time.Second,
			MaxTries:        5,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerifyTLS},
		},
	}

	sdk.SetHTTPClient(Client)
	// No user / password, only token used for auth hatchery
	sdk.Options(api, "", "", token)

	if err := h.Init(); err != nil {
		log.Error("Create> Init error: %s", err)
		os.Exit(10)
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Error("Create> Cannot retrieve hostname: %s", err)
		os.Exit(10)
	}

	go hearbeat(h, token, maxFailures)

	var spawnIds []int64
	var errR error

	tickerRoutine := time.NewTicker(2 * time.Second).C
	tickerProvision := time.NewTicker(time.Duration(provisionSeconds) * time.Second).C
	tickerRegister := time.NewTicker(time.Duration(registerSeconds) * time.Second).C
	for {
		select {
		case <-tickerRoutine:
			spawnIds, errR = routine(h, maxWorkers, provision, hostname, time.Now().Unix(), spawnIds, warningSeconds, criticalSeconds, graceSeconds)
			if errR != nil {
				log.Warning("Error on routine: %s", errR)
			}
		case <-tickerProvision:
			provisioning(h, provision)
		case <-tickerRegister:
			if err := workerRegister(h); err != nil {
				log.Warning("Error on workerRegister: %s", err)
			}
		}
	}
}

// Register calls CDS API to register current hatchery
func Register(h *sdk.Hatchery, token string) error {
	log.Info("Register> Hatchery %s", h.Name)

	h.UID = token
	data, errm := json.Marshal(h)
	if errm != nil {
		return errm
	}

	data, code, errr := sdk.Request("POST", "/hatchery", data)
	if errr != nil {
		return errr
	}

	if code >= 300 {
		return fmt.Errorf("Register> HTTP %d", code)
	}

	if err := json.Unmarshal(data, h); err != nil {
		return err
	}

	// Here, h.UID contains token generated by API
	sdk.Authorization(h.UID)

	log.Info("Register> Hatchery %s registered with id:%d", h.Name, h.ID)

	return nil
}

// GenerateName generate a hatchery's name
func GenerateName(add, name string) string {
	if name == "" {
		var errHostname error
		name, errHostname = os.Hostname()
		if errHostname != nil {
			log.Warning("Cannot retrieve hostname: %s", errHostname)
			name = "cds-hatchery"
		}
		name += "-" + namesgenerator.GetRandomName(0)
	}

	if add != "" {
		name += "-" + add
	}

	return name
}

func hearbeat(m Interface, token string, maxFailures int) {
	var failures int
	for {
		time.Sleep(5 * time.Second)
		if m.Hatchery().ID == 0 {
			log.Info("hearbeat> %s Disconnected from CDS engine, trying to register...", m.Hatchery().Name)
			if err := Register(m.Hatchery(), token); err != nil {
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

		if _, _, err := sdk.Request("PUT", fmt.Sprintf("/hatchery/%d", m.Hatchery().ID), nil); err != nil {
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

func workerRegister(h Interface) error {
	models, errwm := sdk.GetWorkerModelsEnabled()
	if errwm != nil {
		return fmt.Errorf("workerRegister> error on GetWorkerModels: %e", errwm)
	}

	if len(models) == 0 {
		return fmt.Errorf("workerRegister> No model returned by GetWorkerModels")
	}
	log.Debug("workerRegister> models received: %d", len(models))

	var nRegistered int
	for _, m := range models {
		if m.Type != h.ModelType() {
			continue
		}
		// limit to 5 registration per ticker
		if nRegistered > 5 {
			continue
		}
		if !m.NeedRegistration {
			log.Debug("workerRegister> no need to register worker model %s (%d)", m.Name, m.ID)
			continue
		}

		log.Debug("workerRegister> spawn a worker for register worker model %s (%d)", m.Name, m.ID)
		if err := h.SpawnWorker(&m, nil, true); err != nil {
			log.Warning("workerRegister> cannot spawn worker for register: %s", m.Name, err)
		}
		nRegistered++
	}
	return nil
}
