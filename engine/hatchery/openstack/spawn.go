package openstack

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/moby/moby/pkg/namesgenerator"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

// SpawnWorker creates a new cloud instances
// requirements are not supported
func (h *HatcheryOpenstack) SpawnWorker(spawnArgs hatchery.SpawnArguments) (string, error) {
	//generate a pretty cool name
	name := spawnArgs.Model.Name + "-" + strings.Replace(namesgenerator.GetRandomName(0), "_", "-", -1)
	if spawnArgs.RegisterOnly {
		name = "register-" + name
	}

	if spawnArgs.JobID > 0 {
		log.Info("spawnWorker> spawning worker %s model:%s for job %d - %s", name, spawnArgs.Model.Name, spawnArgs.JobID, spawnArgs.LogInfo)
	} else {
		log.Info("spawnWorker> spawning worker %s model:%s - %s", name, spawnArgs.Model.Name, spawnArgs.LogInfo)
	}

	var omd sdk.OpenstackModelData

	if h.hatch == nil {
		return "", fmt.Errorf("hatchery disconnected from engine")
	}

	if len(h.getServers()) == h.Configuration().Provision.MaxWorker {
		log.Debug("MaxWorker limit (%d) reached", h.Configuration().Provision.MaxWorker)
		return "", nil
	}

	if err := json.Unmarshal([]byte(spawnArgs.Model.Image), &omd); err != nil {
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
	if h.Configuration().Provision.WorkerLogsOptions.Graylog.Host != "" {
		graylog += fmt.Sprintf("export CDS_GRAYLOG_HOST=%s ", h.Configuration().Provision.WorkerLogsOptions.Graylog.Host)
	}
	if h.Configuration().Provision.WorkerLogsOptions.Graylog.Port > 0 {
		graylog += fmt.Sprintf("export CDS_GRAYLOG_PORT=%d ", h.Configuration().Provision.WorkerLogsOptions.Graylog.Port)
	}
	if h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraKey != "" {
		graylog += fmt.Sprintf("export CDS_GRAYLOG_EXTRA_KEY=%s ", h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraKey)
	}
	if h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraValue != "" {
		graylog += fmt.Sprintf("export CDS_GRAYLOG_EXTRA_VALUE=%s ", h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraValue)
	}

	grpc := ""
	if h.Configuration().API.GRPC.URL != "" && spawnArgs.Model.Communication == sdk.GRPC {
		grpc += fmt.Sprintf("export CDS_GRPC_API=%s ", h.Configuration().API.GRPC.URL)
		grpc += fmt.Sprintf("export CDS_GRPC_INSECURE=%t ", h.Configuration().API.GRPC.Insecure)
	}

	udataEnd := `
cd $HOME
# Download and start worker with curl
rm -f worker
curl  "{{.API}}/download/worker/$(uname -m)" -o worker --retry 10 --retry-max-time 120 -C - >> /tmp/user_data 2>&1
chmod +x worker
export CDS_SINGLE_USE=1
export CDS_FORCE_EXIT=1
export CDS_API={{.API}}
export CDS_TOKEN={{.Key}}
export CDS_NAME={{.Name}}
export CDS_MODEL={{.Model}}
export CDS_HATCHERY={{.Hatchery}}
export CDS_HATCHERY_NAME={{.HatcheryName}}
export CDS_BOOKED_PB_JOB_ID={{.PipelineBuildJobID}}
export CDS_BOOKED_WORKFLOW_JOB_ID={{.WorkflowJobID}}
export CDS_TTL={{.TTL}}
{{.Graylog}}
{{.Grpc}}
./worker`

	if spawnArgs.RegisterOnly {
		udataEnd += " register"
	}
	udataEnd += " ; sudo shutdown -h now;"

	var withExistingImage bool
	if !spawnArgs.Model.NeedRegistration && !spawnArgs.RegisterOnly {
		start := time.Now()
		imgs := h.getImages()
		log.Debug("spawnWorker> call images.List on openstack took %fs, nbImages:%d", time.Since(start).Seconds(), len(imgs))
		for _, img := range imgs {
			workerModelName, _ := img.Metadata["worker_model_name"]
			if workerModelName == spawnArgs.Model.Name {
				withExistingImage = true
				var jobInfo string
				if spawnArgs.JobID != 0 {
					jobInfo = fmt.Sprintf(" job:%d", spawnArgs.JobID)
				}
				log.Info("spawnWorker> existing image found for worker:%s model:%s img:%s %s %s", name, spawnArgs.Model.Name, img.ID, jobInfo, spawnArgs.LogInfo)
				imageID = img.ID
				break
			}
		}
	}

	tmpl, errt := template.New("udata").Parse(string(udataEnd))
	if errt != nil {
		return "", errt
	}
	udataParam := struct {
		API                string
		Name               string
		Key                string
		Model              int64
		Hatchery           int64
		HatcheryName       string
		PipelineBuildJobID int64
		WorkflowJobID      int64
		TTL                int
		Graylog            string
		Grpc               string
	}{
		API:          h.Configuration().API.HTTP.URL,
		Name:         name,
		Key:          h.Configuration().API.Token,
		Model:        spawnArgs.Model.ID,
		Hatchery:     h.hatch.ID,
		HatcheryName: h.hatch.Name,
		TTL:          h.Config.WorkerTTL,
		Graylog:      graylog,
		Grpc:         grpc,
	}

	if spawnArgs.IsWorkflowJob {
		udataParam.WorkflowJobID = spawnArgs.JobID
	} else {
		udataParam.PipelineBuildJobID = spawnArgs.JobID
	}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, udataParam); err != nil {
		return "", err
	}

	var udataBegin, udata string

	if withExistingImage {
		log.Debug("spawnWorker> using userdata from existing image")
		udataBegin = `#!/bin/bash
set +e
export CDS_FROM_WORKER_IMAGE="true";
`
	} else {
		log.Debug("spawnWorker> using userdata from worker model")
		udataBegin = `#!/bin/bash
set +e
export CDS_FROM_WORKER_IMAGE="false";
`
	}
	udata = udataBegin + string(udataModel) + buffer.String()

	// Encode again
	udata64 := base64.StdEncoding.EncodeToString([]byte(udata))

	// Create openstack vm
	meta := map[string]string{
		"worker":                     name,
		"hatchery_name":              h.Hatchery().Name,
		"register_only":              fmt.Sprintf("%t", spawnArgs.RegisterOnly),
		"flavor":                     omd.Flavor,
		"model":                      omd.Image,
		"worker_model_name":          spawnArgs.Model.Name,
		"worker_model_last_modified": fmt.Sprintf("%d", spawnArgs.Model.UserLastModified.Unix()),
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
	r := servers.Create(h.openstackClient, servers.CreateOpts{
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
