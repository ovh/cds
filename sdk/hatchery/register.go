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

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// Create creates hatchery
func Create(h Interface, api, token string, provision int, requestSecondsTimeout int, insecureSkipVerifyTLS bool) {
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
		log.Critical("Create> Init error: %s\n", err)
		os.Exit(10)
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Critical("Create> Cannot retrieve hostname: %s\n", err)
		os.Exit(10)
	}

	go hearbeat(h, token)

	for {
		time.Sleep(2 * time.Second)
		if h.Hatchery() == nil || h.Hatchery().ID == 0 {
			log.Debug("Create> continue")
			continue
		}

		if err := routine(h, provision, hostname); err != nil {
			log.Warning("Create> Error: %s\n", err)
		}
	}
}

// Register calls CDS API to register current hatchery
func Register(h *sdk.Hatchery, token string) error {
	log.Notice("Register> Hatchery %s\n", h.Name)

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

	sdk.Authorization(h.UID)

	log.Notice("Register> Hatchery registered with uid:%s\n", h.UID)

	return nil
}

// GenerateName generate a hatchery's name
func GenerateName(add string, withRandom bool) string {
	// Register without declaring model
	name, err := os.Hostname()
	if err != nil {
		log.Warning("Cannot retrieve hostname: %s\n", err)
		name = "cds-hatchery"
	}

	if add != "" {
		name += "-" + add
	}

	if withRandom {
		name += "-" + namesgenerator.GetRandomName(0)
	}

	return name
}

func hearbeat(m Interface, token string) {
	for {
		time.Sleep(5 * time.Second)
		if m.Hatchery().ID == 0 {
			log.Notice("hearbeat> Disconnected from CDS engine, trying to register...\n")
			if err := Register(m.Hatchery(), token); err != nil {
				log.Notice("hearbeat> Cannot register: %s\n", err)
				continue
			}
			log.Notice("hearbeat> Registered back: ID %d", m.Hatchery().Model.ID)
		}

		if _, _, err := sdk.Request("PUT", fmt.Sprintf("/hatchery/%d", m.Hatchery().ID), nil); err != nil {
			log.Notice("heartbeat> cannot refresh beat: %s\n", err)
			m.Hatchery().ID = 0
			continue
		}
	}
}
