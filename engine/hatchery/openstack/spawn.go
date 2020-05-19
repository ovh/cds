package openstack

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

// SpawnWorker creates a new cloud instances
// requirements are not supported
func (h *HatcheryOpenstack) SpawnWorker(ctx context.Context, spawnArgs hatchery.SpawnArguments) error {
	if spawnArgs.JobID > 0 {
		log.Debug("spawnWorker> spawning worker %s model:%s for job %d", spawnArgs.WorkerName, spawnArgs.Model.Name, spawnArgs.JobID)
	} else {
		log.Debug("spawnWorker> spawning worker %s model:%s", spawnArgs.WorkerName, spawnArgs.Model.Name)
	}

	if spawnArgs.JobID == 0 && !spawnArgs.RegisterOnly {
		return sdk.WithStack(fmt.Errorf("no job ID and no register"))
	}

	existingServers := h.getServers(ctx)
	if len(existingServers) >= h.Configuration().Provision.MaxWorker {
		log.Debug("MaxWorker limit (%d) reached", h.Configuration().Provision.MaxWorker)
		return nil
	}

	// Get flavor for target model
	flavor, err := h.flavor(spawnArgs.Model.ModelVirtualMachine.Flavor)
	if err != nil {
		return err
	}

	// If a max CPUs count is set in configuration we will check that there are enough CPUs available to spawn the model
	if h.Config.MaxCPUs > 0 {
		var totalCPUsUsed int
		for i := range existingServers {
			totalCPUsUsed += existingServers[i].Flavor["vcpus"].(int)
		}
		if totalCPUsUsed+flavor.VCPUs >= h.Config.MaxCPUs {
			log.Debug("MaxCPUs limit (%d) reached", h.Config.MaxCPUs)
			return nil
		}
	}

	// TODO If the CountSmallerFlavorToKeep is set in config, we should check that there will be enough CPUs to spawn a smaller flavor after this one

	// Get image ID
	imageID, err := h.imageID(ctx, spawnArgs.Model.ModelVirtualMachine.Image)
	if err != nil {
		return err
	}

	var withExistingImage bool
	if !spawnArgs.Model.NeedRegistration && !spawnArgs.RegisterOnly {
		start := time.Now()
		imgs := h.getImages(ctx)
		log.Debug("spawnWorker> call images.List on openstack took %fs, nbImages:%d", time.Since(start).Seconds(), len(imgs))
		for _, img := range imgs {
			workerModelName := img.Metadata["worker_model_name"]
			workerModelLastModified := img.Metadata["worker_model_last_modified"]
			if workerModelName == spawnArgs.Model.Name && fmt.Sprintf("%s", workerModelLastModified) == fmt.Sprintf("%d", spawnArgs.Model.UserLastModified.Unix()) {
				withExistingImage = true
				imageID = img.ID
				break
			}
		}
	}

	if spawnArgs.RegisterOnly {
		spawnArgs.Model.ModelVirtualMachine.Cmd += " register"
	}

	udata := spawnArgs.Model.ModelVirtualMachine.PreCmd + "\n" + spawnArgs.Model.ModelVirtualMachine.Cmd + "\n" + spawnArgs.Model.ModelVirtualMachine.PostCmd

	tmpl, errt := template.New("udata").Parse(udata)
	if errt != nil {
		return errt
	}
	udataParam := sdk.WorkerArgs{
		API:               h.Configuration().API.HTTP.URL,
		Name:              spawnArgs.WorkerName,
		Token:             spawnArgs.WorkerToken,
		Model:             spawnArgs.Model.Group.Name + "/" + spawnArgs.Model.Name,
		HatcheryName:      h.Name(),
		TTL:               h.Config.WorkerTTL,
		FromWorkerImage:   withExistingImage,
		GraylogHost:       h.Configuration().Provision.WorkerLogsOptions.Graylog.Host,
		GraylogPort:       h.Configuration().Provision.WorkerLogsOptions.Graylog.Port,
		GraylogExtraKey:   h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraKey,
		GraylogExtraValue: h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraValue,
	}

	udataParam.WorkflowJobID = spawnArgs.JobID

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, udataParam); err != nil {
		return err
	}

	// Encode again
	udata64 := base64.StdEncoding.EncodeToString(buffer.Bytes())

	// Create openstack vm
	meta := map[string]string{
		"worker":                     spawnArgs.WorkerName,
		"hatchery_name":              h.Name(),
		"register_only":              fmt.Sprintf("%t", spawnArgs.RegisterOnly),
		"flavor":                     spawnArgs.Model.ModelVirtualMachine.Flavor,
		"model":                      spawnArgs.Model.ModelVirtualMachine.Image,
		"worker_model_path":          spawnArgs.Model.Group.Name + "/" + spawnArgs.Model.Name,
		"worker_model_name":          spawnArgs.Model.Name,
		"worker_model_last_modified": fmt.Sprintf("%d", spawnArgs.Model.UserLastModified.Unix()),
	}

	maxTries := 3
	for try := 1; try <= maxTries; try++ {
		// Ip len(ipsInfos.ips) > 0, specify one of those
		var ip string
		if len(ipsInfos.ips) > 0 {
			var errai error
			ip, errai = h.findAvailableIP(ctx, spawnArgs.WorkerName)
			if errai != nil {
				return errai
			}
			log.Debug("Found %s as available IP", ip)
		}

		networks := []servers.Network{{UUID: h.networkID, FixedIP: ip}}
		r := servers.Create(h.openstackClient, servers.CreateOpts{
			Name:      spawnArgs.WorkerName,
			FlavorRef: flavor.ID,
			ImageRef:  imageID,
			Metadata:  meta,
			UserData:  []byte(udata64),
			Networks:  networks,
		})

		server, err := r.Extract()
		if err != nil {
			if strings.Contains(err.Error(), "is already in use on instance") && try < maxTries { // Fixed IP address X.X.X.X is already in use on instance
				log.Warning(ctx, "SpawnWorker> Unable to create server: name:%s flavor:%s image:%s metadata:%v networks:%s err:%v body:%s - Try %d/%d", spawnArgs.WorkerName, flavor.ID, imageID, meta, networks, err, r.Body, try, maxTries)
				continue
			}
			return fmt.Errorf("SpawnWorker> Unable to create server: name:%s flavor:%s image:%s metadata:%v networks:%s err:%v body:%s", spawnArgs.WorkerName, flavor.ID, imageID, meta, networks, err, r.Body)
		}
		log.Debug("SpawnWorker> Created Server ID: %s", server.ID)
		break
	}
	return nil
}
