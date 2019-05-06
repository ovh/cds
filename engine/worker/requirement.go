package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/mem"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var requirementCheckFuncs = map[string]func(w *currentWorker, r sdk.Requirement) (bool, error){
	sdk.BinaryRequirement:        checkBinaryRequirement,
	sdk.HostnameRequirement:      checkHostnameRequirement,
	sdk.ModelRequirement:         checkModelRequirement,
	sdk.NetworkAccessRequirement: checkNetworkAccessRequirement,
	sdk.PluginRequirement:        checkPluginRequirement,
	sdk.ServiceRequirement:       checkServiceRequirement,
	sdk.MemoryRequirement:        checkMemoryRequirement,
	sdk.VolumeRequirement:        checkVolumeRequirement,
	sdk.OSArchRequirement:        checkOSArchRequirement,
}

func checkRequirements(w *currentWorker, a *sdk.Action, bookedJobID int64) (bool, []sdk.Requirement) {
	requirementsOK := true
	errRequirements := []sdk.Requirement{}

	log.Debug("checkRequirements> for JobID:%d model of worker: %s", bookedJobID, w.model.Name)

	log.Debug("requirements for %s >>> %+v\n", a.Name, a.Requirements)
	for _, r := range a.Requirements {
		ok, err := checkRequirement(w, r)
		if err != nil {
			log.Warning("checkQueue> error on checkRequirement %s", err)
		}
		if !ok {
			requirementsOK = false
			errRequirements = append(errRequirements, r)
			continue
		}
	}

	log.Debug("checkRequirements> checkRequirements:%t errRequirements:%v", requirementsOK, errRequirements)
	return requirementsOK, errRequirements
}

func checkRequirement(w *currentWorker, r sdk.Requirement) (bool, error) {
	check := requirementCheckFuncs[r.Type]
	if check == nil {
		return false, fmt.Errorf("checkRequirement> Unknown type of requirement: %s", r.Type)
	}
	return check(w, r)
}

func checkPluginRequirement(w *currentWorker, r sdk.Requirement) (bool, error) {
	var currentOS = strings.ToLower(sdk.GOOS)
	var currentARCH = strings.ToLower(sdk.GOARCH)

	binary, err := w.client.PluginGetBinaryInfos(r.Name, currentOS, currentARCH)
	if err != nil {
		return false, err
	}

	// then try to download the plugin
	pluginBinary := path.Join(w.basedir, binary.Name)
	if _, err := os.Stat(pluginBinary); os.IsNotExist(err) {
		log.Debug("Downloading the plugin %s", binary.Name)
		//If the file doesn't exist. Download it.
		fi, err := os.OpenFile(pluginBinary, os.O_CREATE|os.O_RDWR, os.FileMode(binary.Perm))
		if err != nil {
			return false, err
		}

		log.Debug("Get the binary plugin %s", r.Name)
		if err := w.client.PluginGetBinary(r.Name, currentOS, currentARCH, fi); err != nil {
			_ = fi.Close()
			return false, err
		}
		//It's downloaded. Close the file
		_ = fi.Close()
	} else {
		log.Debug("plugin binary is in cache %s", pluginBinary)
	}

	return true, nil
}

// checkHostnameRequirement returns true if current hostname is a requirement
func checkHostnameRequirement(w *currentWorker, r sdk.Requirement) (bool, error) {
	h, err := os.Hostname()
	if err != nil {
		return false, err
	}
	return h == r.Value, nil
}

// checkBinaryRequirement returns true is binary requirement is in worker's PATH
func checkBinaryRequirement(w *currentWorker, r sdk.Requirement) (bool, error) {
	if _, err := exec.LookPath(r.Value); err != nil {
		// Return nil because the error contains 'Executable file not found', that's what we wanted
		return false, nil
	}
	return true, nil
}

func checkModelRequirement(w *currentWorker, r sdk.Requirement) (bool, error) {
	// if there is a model req and no model on worker -> return false
	if w.model.ID == 0 {
		return false, nil
	}
	modelName := strings.Split(r.Value, " ")[0]
	modelPath := strings.SplitN(modelName, "/", 2)
	if len(modelPath) == 2 {
		modelName = modelPath[1]
	}
	return modelName == w.model.Name, nil
}

func checkNetworkAccessRequirement(w *currentWorker, r sdk.Requirement) (bool, error) {
	conn, err := net.DialTimeout("tcp", r.Value, 10*time.Second)
	if err != nil {
		return false, nil
	}
	conn.Close()

	return true, nil
}

func checkServiceRequirement(w *currentWorker, r sdk.Requirement) (bool, error) {
	// service are supported only for Model Docker
	if w.model.Type != sdk.Docker {
		return false, nil
	}

	retry := 3
	for attempt := 0; attempt < retry; attempt++ {
		ips, err := net.LookupIP(r.Name)
		if err != nil {
			log.Debug("Error checking requirement : %s", err)
			time.Sleep(2 * time.Second)
			continue
		}
		var s string
		for _, ip := range ips {
			s += s + ip.String() + " "
		}
		log.Info("Service requirement %s is ready %s", r.Name, s)
		return true, nil
	}

	return false, nil
}

func checkMemoryRequirement(w *currentWorker, r sdk.Requirement) (bool, error) {
	var totalMemory int64
	neededMemory, err := strconv.ParseInt(r.Value, 10, 64)
	if err != nil {
		return false, err
	}

	switch w.model.Type {
	// Check env variables in a docker is safer than mem.VirtualMemory
	case sdk.Docker:
		var err error
		// Useful for provisioned worker
		memoryEnv := os.Getenv("CDS_MODEL_MEMORY")
		totalMemory, err = strconv.ParseInt(memoryEnv, 10, 64)
		if err != nil {
			return false, err
		}
		totalMemory = totalMemory * 1024 * 1024
	default:
		v, err := mem.VirtualMemory()
		if err != nil {
			return false, err
		}
		totalMemory = int64(v.Total)
	}
	//Assuming memory is in megabytes
	//If we have more than 90% of neededMemory, lets do it
	return totalMemory >= (neededMemory*1024*1024)*90/100, nil
}

func checkVolumeRequirement(w *currentWorker, r sdk.Requirement) (bool, error) {
	// volume are supported only for Model Docker
	if w.model.Type != sdk.Docker {
		return false, nil
	}

	for _, v := range strings.Split(r.Value, ",") {
		if strings.HasPrefix(v, "destination=") {
			theMountedDir := strings.Split(v, "=")[1]
			if stat, err := os.Stat(theMountedDir); err != nil || !stat.IsDir() {
				return true, nil
			}
		}
	}
	return false, nil
}

func checkOSArchRequirement(w *currentWorker, r sdk.Requirement) (bool, error) {
	osarch := strings.Split(r.Value, "/")
	if len(osarch) != 2 {
		return false, fmt.Errorf("invalid requirement %s", r.Value)
	}

	return osarch[0] == strings.ToLower(sdk.GOOS) && osarch[1] == strings.ToLower(sdk.GOARCH), nil
}

// checkPluginDeployment returns true if current job:
//  - is not linked to a deployment integration
//  - is linked to a deployement integration, plugin well downloaded (in this func) and
//    requirements on the plugins are OK too
func checkPluginDeployment(w *currentWorker, job sdk.WorkflowNodeJobRun) (bool, error) {
	var currentOS = strings.ToLower(sdk.GOOS)
	var currentARCH = strings.ToLower(sdk.GOARCH)
	var binary *sdk.GRPCPluginBinary

	if len(job.IntegrationPluginBinaries) == 0 {
		// current job is not linked to a deployment integration (in pipeline context)
		return true, nil
	}

	log.Debug("Checking plugins...(%#v)", job.IntegrationPluginBinaries)

	// first check OS and Architecture
	for _, b := range job.IntegrationPluginBinaries {
		if b.OS == currentOS && b.Arch == currentARCH {
			binary = &b
			break
		}
	}
	if binary == nil {
		return false, fmt.Errorf("%s %s not supported by this plugin", currentOS, currentARCH)
	}

	// then check plugin requirements
	for _, r := range binary.Requirements {
		ok, err := checkRequirement(w, r)
		if err != nil {
			log.Warning("checkQueue> error on checkRequirement %s", err)
		}
		if !ok {
			return false, fmt.Errorf("plugin requirement %s does not match", r.Name)
		}
	}

	// then try to download the plugin
	integrationPluginBinary := path.Join(w.basedir, binary.Name)
	if _, err := os.Stat(integrationPluginBinary); os.IsNotExist(err) {
		log.Debug("Downloading the plugin %s", binary.PluginName)
		//If the file doesn't exist. Download it.
		fi, err := os.OpenFile(integrationPluginBinary, os.O_CREATE|os.O_RDWR, os.FileMode(binary.Perm))
		if err != nil {
			return false, err
		}

		if err := w.client.PluginGetBinary(binary.PluginName, currentOS, currentARCH, fi); err != nil {
			_ = fi.Close()
			return false, err
		}
		//It's downloaded. Close the file
		_ = fi.Close()
	} else {
		log.Debug("plugin binary is in cache")
	}

	log.Info("plugin successfully downloaded: %#v", binary.Name)

	return true, nil
}
