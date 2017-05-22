package openstack

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/spf13/viper"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// SpawnWorker creates a new cloud instances
// requirements are not supported
func (h *HatcheryCloud) SpawnWorker(model *sdk.Model, job *sdk.PipelineBuildJob, registerOnly bool, logInfo string) (string, error) {
	//generate a pretty cool name
	name := model.Name + "-" + strings.Replace(namesgenerator.GetRandomName(0), "_", "-", -1)
	if registerOnly {
		name = "register-" + name
	}

	if job != nil {
		log.Info("spawnWorker> spawning worker %s model:%s for job %d - %s", name, model.Name, job.ID, logInfo)
	} else {
		log.Info("spawnWorker> spawning worker %s model:%s - %s", name, model.Name, logInfo)
	}

	var omd sdk.OpenstackModelData

	if h.hatch == nil {
		return "", fmt.Errorf("hatchery disconnected from engine")
	}

	if len(h.getServers()) == viper.GetInt("max-worker") {
		log.Debug("MaxWorker limit (%d) reached", viper.GetInt("max-worker"))
		return "", nil
	}

	if err := json.Unmarshal([]byte(model.Image), &omd); err != nil {
		return "", err
	}

	// Get image ID
	imageID, erri := h.imageID(omd.Image)
	if erri != nil {
		return "", erri
	}

	// Get flavor ID
	flavorID, errf := h.flavorID(omd.Flavor)
	if errf != nil {
		return "", errf
	}

	// Decode base64 given user data
	udataModel, errd := base64.StdEncoding.DecodeString(omd.UserData)
	if errd != nil {
		return "", errd
	}

	graylog := ""
	if viper.GetString("graylog_host") != "" {
		graylog += fmt.Sprintf("export CDS_GRAYLOG_HOST=%s ", viper.GetString("graylog_host"))
	}
	if viper.GetString("graylog_port") != "" {
		graylog += fmt.Sprintf("export CDS_GRAYLOG_PORT=%s ", viper.GetString("graylog_port"))
	}
	if viper.GetString("graylog_extra_key") != "" {
		graylog += fmt.Sprintf("export CDS_GRAYLOG_EXTRA_KEY=%s ", viper.GetString("graylog_extra_key"))
	}
	if viper.GetString("graylog_extra_value") != "" {
		graylog += fmt.Sprintf("export CDS_GRAYLOG_EXTRA_VALUE=%s ", viper.GetString("graylog_extra_value"))
	}

	udataEnd := `
cd $HOME
# Download and start worker with curl
curl  "{{.API}}/download/worker/$(uname -m)" -o worker --retry 10 --retry-max-time 120 -C - >> /tmp/user_data 2>&1
chmod +x worker
export CDS_SINGLE_USE=1
export CDS_FORCE_EXIT=1
export CDS_API={{.API}}
export CDS_KEY={{.Key}}
export CDS_NAME={{.Name}}
export CDS_MODEL={{.Model}}
export CDS_HATCHERY={{.Hatchery}}
export CDS_HATCHERY_NAME={{.HatcheryName}}
export CDS_BOOKED_JOB_ID={{.JobID}}
export CDS_TTL={{.TTL}}
{{.Graylog}}
./worker`

	if registerOnly {
		udataEnd += " register"
	}
	udataEnd += " ; sudo shutdown -h now;"

	var withExistingImage bool
	if !model.NeedRegistration && !registerOnly {
		start := time.Now()
		imgs := h.getImages()
		log.Debug("spawnWorker> call images.List on openstack took %fs, nbImages:%d", time.Since(start).Seconds(), len(imgs))
		for _, img := range imgs {
			workerModelName, _ := img.Metadata["worker_model_name"]
			if workerModelName == model.Name {
				withExistingImage = true
				log.Info("spawnWorker> existing image found for worker:%s model:%s img:%s", name, model.Name, img.ID)
				imageID = img.ID
				break
			}
		}
	}

	var jobID int64
	if job != nil {
		jobID = job.ID
	}

	tmpl, errt := template.New("udata").Parse(string(udataEnd))
	if errt != nil {
		return "", errt
	}
	udataParam := struct {
		API          string
		Name         string
		Key          string
		Model        int64
		Hatchery     int64
		HatcheryName string
		JobID        int64
		TTL          int
		Graylog      string
	}{
		API:          viper.GetString("api"),
		Name:         name,
		Key:          viper.GetString("token"),
		Model:        model.ID,
		Hatchery:     h.hatch.ID,
		HatcheryName: h.hatch.Name,
		JobID:        jobID,
		TTL:          h.workerTTL,
		Graylog:      graylog,
	}
	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, udataParam); err != nil {
		return "", err
	}

	var udataBegin, udata string

	if withExistingImage {
		log.Debug("spawnWorker> using userdata from existing image")
		udataBegin = `#!/bin/sh
set +e
export CDS_FROM_WORKER_IMAGE=true;
`
	} else {
		log.Debug("spawnWorker> using userdata from worker model")
		udataBegin = `#!/bin/sh
set +e
export CDS_FROM_WORKER_IMAGE=false;
`
	}
	udata = udataBegin + string(udataModel) + buffer.String()

	// Encode again
	udata64 := base64.StdEncoding.EncodeToString([]byte(udata))

	// Create openstack vm
	meta := map[string]string{
		"worker":                     name,
		"hatchery_name":              h.Hatchery().Name,
		"register_only":              fmt.Sprintf("%t", registerOnly),
		"flavor":                     omd.Flavor,
		"model":                      omd.Image,
		"worker_model_name":          model.Name,
		"worker_model_last_modified": fmt.Sprintf("%d", model.UserLastModified.Unix()),
	}

	// Ip len(ipsInfos.ips) > 0, specify one of those
	var ip string
	if len(ipsInfos.ips) > 0 {
		var errai error
		ip, errai = h.findAvailableIP(name)
		if errai != nil {
			return "", errai
		}
		log.Debug("Found %s as available IP", ip)
	}

	networks := []servers.Network{{UUID: h.networkID, FixedIP: ip}}
	r := servers.Create(h.client, servers.CreateOpts{
		Name:      name,
		FlavorRef: flavorID,
		ImageRef:  imageID,
		Metadata:  meta,
		UserData:  []byte(udata64),
		Networks:  networks,
	})

	server, err := r.Extract()
	if err != nil {
		return "", fmt.Errorf("SpawnWorker> Unable to create server: name:%s flavor:%s image:%s metadata:%v networks:%s err:%s body:%s", name, flavorID, imageID, meta, networks, err, r.Body)
	}
	log.Debug("SpawnWorker> Created Server ID: %s", server.ID)
	return name, nil
}
