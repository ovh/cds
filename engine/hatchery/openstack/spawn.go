package openstack

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"math"
	"strings"
	"text/template"
	"time"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
)

// SpawnWorker creates a new cloud instances
// requirements are not supported
func (h *HatcheryOpenstack) SpawnWorker(ctx context.Context, spawnArgs hatchery.SpawnArguments) error {
	if spawnArgs.JobID != "0" {
		log.Debug(ctx, "spawnWorker> spawning worker %s model:%s for job %d", spawnArgs.WorkerName, spawnArgs.Model.GetName(), spawnArgs.JobID)
	} else {
		log.Debug(ctx, "spawnWorker> spawning worker %s model:%s", spawnArgs.WorkerName, spawnArgs.Model.GetName())
	}

	if spawnArgs.JobID == "0" && !spawnArgs.RegisterOnly {
		return sdk.WithStack(fmt.Errorf("no job ID and no register"))
	}

	if err := h.checkSpawnLimits(ctx, spawnArgs.Model); err != nil {
		ctx = sdk.ContextWithStacktrace(ctx, err)
		log.Error(ctx, err.Error())
		return nil
	}

	// Get flavor for target model
	flavor, err := h.flavor(spawnArgs.Model.GetFlavor())
	if err != nil {
		return err
	}

	// Get image ID
	imageID, err := h.imageID(ctx, spawnArgs.Model.GetOpenstackImage())
	if err != nil {
		return err
	}

	var withExistingImage bool
	if spawnArgs.Model.ModelV1 == nil || (!spawnArgs.Model.ModelV1.NeedRegistration && !spawnArgs.RegisterOnly) {
		start := time.Now()
		imgs := h.getImages(ctx)
		log.Debug(ctx, "spawnWorker> call images.List on openstack took %fs, nbImages:%d", time.Since(start).Seconds(), len(imgs))
		for _, img := range imgs {
			if !strings.HasPrefix(img.Name, "cds_image") {
				continue
			}
			workerModelName := img.Metadata["worker_model_name"] // Temporary check on name for old registred model but new snapshot will only have path
			workerModelPath := img.Metadata["worker_model_path"]
			workerModelLastModified := img.Metadata["worker_model_last_modified"]
			nameOrPathMatch := (workerModelName != "" && workerModelName == spawnArgs.Model.GetName()) || workerModelPath == spawnArgs.Model.GetFullPath()
			if nameOrPathMatch && fmt.Sprintf("%s", workerModelLastModified) == spawnArgs.Model.GetLastModified() {
				withExistingImage = true
				imageID = img.ID
				break
			}
		}
	}
	workerConfig := h.GenerateWorkerConfig(ctx, h, spawnArgs)

	var cmdSuffix string
	if spawnArgs.RegisterOnly {
		cmdSuffix = fmt.Sprintf(" --config %s register", workerConfig.EncodeBase64())
	} else {
		cmdSuffix += fmt.Sprintf(" --config %s", workerConfig.EncodeBase64())
	}

	udata := spawnArgs.Model.GetPreCmd() + "\n" + spawnArgs.Model.GetCmd() + cmdSuffix + "\n" + spawnArgs.Model.GetPostCmd()
	tmpl, err := template.New("udata").Parse(udata)
	if err != nil {
		return err
	}

	//workerConfig.Basedir =
	udataParam := struct {
		// All fields below are deprecated
		API               string `json:"api"`
		Token             string `json:"token"`
		Name              string `json:"name"`
		BaseDir           string `json:"base_dir"`
		HTTPInsecure      bool   `json:"http_insecure"`
		Model             string `json:"model"`
		HatcheryName      string `json:"hatchery_name"`
		WorkflowJobID     int64  `json:"workflow_job_id"`
		TTL               int    `json:"ttl"`
		FromWorkerImage   bool   `json:"from_worker_image"`
		GraylogHost       string `json:"graylog_host"`
		GraylogPort       int    `json:"graylog_port"`
		GraylogExtraKey   string `json:"graylog_extra_key"`
		GraylogExtraValue string `json:"graylog_extra_value"`
		WorkerBinary      string
		InjectEnvVars     map[string]string `json:"inject_env_vars"`
		// All fields above are deprecated
		Config string
	}{
		API:             workerConfig.APIEndpoint,
		FromWorkerImage: withExistingImage,
		Config:          workerConfig.EncodeBase64(),
	}

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
		"flavor":                     spawnArgs.Model.GetFlavor(),
		"model":                      spawnArgs.Model.GetOpenstackImage(),
		"worker_model_path":          spawnArgs.Model.GetFullPath(),
		"worker_model_last_modified": spawnArgs.Model.GetLastModified(),
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
			log.Debug(ctx, "Found %s as available IP", ip)
		}

		networks := []servers.Network{{UUID: h.networkID, FixedIP: ip}}
		opts := servers.CreateOpts{
			Name:      spawnArgs.WorkerName,
			FlavorRef: flavor.ID,
			ImageRef:  imageID,
			Metadata:  meta,
			UserData:  []byte(udata64),
			Networks:  networks,
		}
		if len(h.Config.DefaultSecurityGroups) > 0 {
			opts.SecurityGroups = h.Config.DefaultSecurityGroups
		}
		r := servers.Create(h.openstackClient, opts)

		server, err := r.Extract()
		if err != nil {
			if strings.Contains(err.Error(), "is already in use on instance") && try < maxTries { // Fixed IP address X.X.X.X is already in use on instance
				log.Warn(ctx, "SpawnWorker> Unable to create server: name:%s flavor:%s image:%s metadata:%v networks:%s err:%v body:%s - Try %d/%d", spawnArgs.WorkerName, flavor.ID, imageID, meta, networks, err, r.Body, try, maxTries)
				continue
			}
			return fmt.Errorf("SpawnWorker> Unable to create server: name:%s flavor:%s image:%s metadata:%v networks:%s err:%v body:%s", spawnArgs.WorkerName, flavor.ID, imageID, meta, networks, err, r.Body)
		}
		log.Debug(ctx, "SpawnWorker> Created Server ID: %s", server.ID)
		break
	}
	return nil
}

func (h *HatcheryOpenstack) checkSpawnLimits(ctx context.Context, model sdk.WorkerStarterWorkerModel) error {
	existingServers := h.getServers(ctx)
	if len(existingServers) >= h.Configuration().Provision.MaxWorker {
		return sdk.WithStack(fmt.Errorf("MaxWorker limit (%d) reached", h.Configuration().Provision.MaxWorker))
	}

	// Get flavor for target model
	flavor, err := h.flavor(model.GetFlavor())
	if err != nil {
		return err
	}

	// If a max CPUs count is set in configuration we will check that there are enough CPUs available to spawn the model
	var totalCPUsUsed int
	if h.Config.MaxCPUs > 0 {
		for i := range existingServers {
			flavorName, _ := existingServers[i].Metadata["flavor"]
			flavor, err := h.flavor(flavorName)
			if err == nil {
				totalCPUsUsed += flavor.VCPUs
			}
		}
		if totalCPUsUsed+flavor.VCPUs > h.Config.MaxCPUs {
			return sdk.WithStack(fmt.Errorf("MaxCPUs limit (%d) reached", h.Config.MaxCPUs))
		}
	}

	// If the CountSmallerFlavorToKeep is set in config, we should check that there will be enough CPUs to spawn a smaller flavor after this one
	if h.Config.MaxCPUs > 0 && h.Config.CountSmallerFlavorToKeep > 0 {
		smallerFlavor := h.getSmallerFlavorThan(flavor)
		// If same id, means that the requested flavor is the smallest one so we want to start it.
		log.Debug(ctx, "checkSpawnLimits> smaller flavor found for %s is %s", flavor.Name, smallerFlavor.Name)
		if smallerFlavor.ID != flavor.ID {
			minCPUsNeededToStart := flavor.VCPUs + h.Config.CountSmallerFlavorToKeep*smallerFlavor.VCPUs
			countCPUsLeft := int(math.Max(.0, float64(h.Config.MaxCPUs-totalCPUsUsed))) // Set zero as min value in case that the limit changed and count of used greater than max count
			if minCPUsNeededToStart > countCPUsLeft {
				return sdk.WithStack(fmt.Errorf("CountSmallerFlavorToKeep limit reached, can't start model %s with flavor %s that requires %d CPUs. Smaller flavor is %s and need %d CPUs. There are currently %d/%d left CPUs",
					model.GetFullPath(), flavor.Name, flavor.VCPUs, smallerFlavor.Name, smallerFlavor.VCPUs, countCPUsLeft, h.Config.MaxCPUs))
			}
			log.Debug(ctx, "checkSpawnLimits> %d/%d CPUs left is enough to start model %s with flavor %s that require %d CPUs",
				countCPUsLeft, h.Config.MaxCPUs, model.GetFullPath(), flavor.Name, flavor.VCPUs)
		}
	}

	return nil
}
