package internal

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/rockbears/log"
	"github.com/shirou/gopsutil/mem"

	"github.com/ovh/cds/sdk"
)

var requirementCheckFuncs = map[string]func(w *CurrentWorker, r sdk.Requirement) (bool, error){
	sdk.BinaryRequirement:   checkBinaryRequirement,
	sdk.HostnameRequirement: checkHostnameRequirement,
	sdk.ModelRequirement:    checkModelRequirement,
	sdk.PluginRequirement:   checkPluginRequirement,
	sdk.ServiceRequirement:  checkServiceRequirement,
	sdk.MemoryRequirement:   checkMemoryRequirement,
	sdk.OSArchRequirement:   checkOSArchRequirement,
	sdk.RegionRequirement:   checkRegionRequirement,
	sdk.SecretRequirement:   checkSecretRequirement,
}

func checkRequirements(ctx context.Context, w *CurrentWorker, a *sdk.Action) (bool, []sdk.Requirement) {
	requirementsOK := true
	errRequirements := make([]sdk.Requirement, 0)

	log.Debug(ctx, "requirements for %s >>> %+v\n", a.Name, a.Requirements)
	for _, r := range a.Requirements {
		ok, err := checkRequirement(w, r)
		if err != nil {
			log.Warn(ctx, "checkQueue> error on checkRequirement %s", err)
		}
		if !ok {
			requirementsOK = false
			errRequirements = append(errRequirements, r)
			continue
		}
	}

	log.Debug(ctx, "checkRequirements> checkRequirements:%t errRequirements:%v", requirementsOK, errRequirements)
	return requirementsOK, errRequirements
}

func checkRequirement(w *CurrentWorker, r sdk.Requirement) (bool, error) {
	check := requirementCheckFuncs[r.Type]
	if check == nil {
		return false, fmt.Errorf("checkRequirement> Unknown type of requirement: %s", r.Type)
	}
	return check(w, r)
}

func checkPluginRequirement(w *CurrentWorker, r sdk.Requirement) (bool, error) {
	var ctx = context.TODO()
	var currentOS = strings.ToLower(sdk.GOOS)
	var currentARCH = strings.ToLower(sdk.GOARCH)

	binary, err := w.client.PluginGetBinaryInfos(r.Name, currentOS, currentARCH)
	if err != nil {
		return false, err
	}

	// then try to download the plugin
	if _, err := w.BaseDir().Stat(binary.Name); os.IsNotExist(err) {
		log.Debug(ctx, "Downloading the plugin %s", binary.Name)
		//If the file doesn't exist. Download it.
		fi, err := w.BaseDir().OpenFile(binary.Name, os.O_CREATE|os.O_RDWR, os.FileMode(binary.Perm))
		if err != nil {
			return false, err
		}
		log.Debug(ctx, "Get the binary plugin %s", r.Name)
		if err := w.client.PluginGetBinary(r.Name, currentOS, currentARCH, fi); err != nil {
			_ = fi.Close()
			return false, err
		}
		//It's downloaded. Close the file
		_ = fi.Close()
	} else {
		log.Debug(ctx, "plugin binary is in cache %s", binary.Name)
	}

	return true, nil
}

// checkHostnameRequirement returns true if current hostname is a requirement
func checkHostnameRequirement(_ *CurrentWorker, r sdk.Requirement) (bool, error) {
	h, err := os.Hostname()
	if err != nil {
		return false, err
	}
	return h == r.Value, nil
}

// checkBinaryRequirement returns true is binary requirement is in worker's PATH
func checkBinaryRequirement(_ *CurrentWorker, r sdk.Requirement) (bool, error) {
	if _, err := exec.LookPath(r.Value); err != nil {
		// Return nil because the error contains 'Executable file not found', that's what we wanted
		return false, nil
	}
	return true, nil
}

func checkModelRequirement(w *CurrentWorker, r sdk.Requirement) (bool, error) {
	// if there is a model req and no model on worker -> return false
	if w.model.ID == 0 {
		return false, nil
	}

	modelName := strings.Split(r.Value, " ")[0]
	modelPath := strings.SplitN(modelName, "/", 2)
	if len(modelPath) == 2 {
		// if the requirement contains group info (myGroup/myModel) check that it match current worker model
		return modelName == fmt.Sprintf("%s/%s", w.model.Group.Name, w.model.Name), nil
	}

	isSharedInfra := w.model.Group.Name == sdk.SharedInfraGroupName && modelName == w.model.Name

	return isSharedInfra, nil
}

func checkServiceRequirement(w *CurrentWorker, r sdk.Requirement) (bool, error) {
	// service are supported only for Model Docker
	if w.model.Type != sdk.Docker {
		return false, nil
	}

	retry := 3
	for attempt := 0; attempt < retry; attempt++ {
		ips, err := net.LookupIP(r.Name)
		if err != nil {
			log.Debug(context.TODO(), "Error checking requirement : %s", err)
			time.Sleep(2 * time.Second)
			continue
		}
		var s string
		for _, ip := range ips {
			s += s + ip.String() + " "
		}
		log.Info(context.TODO(), "Service requirement %s is ready %s", r.Name, s)
		return true, nil
	}

	return false, nil
}

func checkMemoryRequirement(w *CurrentWorker, r sdk.Requirement) (bool, error) {
	var totalMemory int64
	neededMemory, err := strconv.ParseInt(r.Value, 10, 64)
	if err != nil {
		return false, err
	}

	switch w.model.Type {
	// Check env variables in a docker is safer than mem.VirtualMemory
	case sdk.Docker:
		var err error
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

func checkOSArchRequirement(_ *CurrentWorker, r sdk.Requirement) (bool, error) {
	osarch := strings.Split(r.Value, "/")
	if len(osarch) != 2 {
		return false, fmt.Errorf("invalid requirement %s", r.Value)
	}

	return osarch[0] == strings.ToLower(sdk.GOOS) && osarch[1] == strings.ToLower(sdk.GOARCH), nil
}

// region is checked by hatchery only
func checkRegionRequirement(_ *CurrentWorker, _ sdk.Requirement) (bool, error) {
	return true, nil
}

// secret is checked by api only
func checkSecretRequirement(_ *CurrentWorker, _ sdk.Requirement) (bool, error) {
	return true, nil
}

// checkPlugins returns true if current job:
//  - is not linked to a deployment integration
//  - is linked to a deployement integration, plugin well downloaded (in this func) and
//    requirements on the plugins are OK too
func checkPlugins(ctx context.Context, w *CurrentWorker, job sdk.WorkflowNodeJobRun) (bool, error) {

	if len(job.IntegrationPlugins) == 0 {
		// current job is not linked to a deployment integration (in pipeline context)
		return true, nil
	}

	log.Debug(ctx, "Checking plugins...(%#v)", job.IntegrationPlugins)

	for _, p := range job.IntegrationPlugins {
		if err := checkPluginBinary(ctx, w, p); err != nil {
			return false, err
		}
	}
	return true, nil
}

func checkPluginBinary(ctx context.Context, w *CurrentWorker, p sdk.GRPCPlugin) error {
	var binary *sdk.GRPCPluginBinary
	var currentOS = strings.ToLower(sdk.GOOS)
	var currentARCH = strings.ToLower(sdk.GOARCH)

	// first check OS and Architecture
	for _, b := range p.Binaries {
		if b.OS == currentOS && b.Arch == currentARCH {
			binary = &b
			break
		}
	}
	if binary == nil {
		return fmt.Errorf("%s/%s not supported by plugin %s", currentOS, currentARCH, p.Name)
	}

	// then check plugin requirements
	for _, r := range binary.Requirements {
		ok, err := checkRequirement(w, r)
		if err != nil {
			log.Warn(ctx, "checkQueue> error on checkRequirement %s", err)
		}
		if !ok {
			return fmt.Errorf("plugin requirement %s does not match", r.Name)
		}
	}
	// then try to download the plugin
	//integrationPluginBinary := path.Join(w.BaseDir().Name(), binary.Name)
	if _, err := w.BaseDir().Stat(binary.Name); os.IsNotExist(err) {
		log.Info(ctx, "Downloading the plugin %s", binary.PluginName)
		//If the file doesn't exist. Download it.
		fi, err := w.BaseDir().OpenFile(binary.Name, os.O_CREATE|os.O_RDWR, os.FileMode(binary.Perm))
		if err != nil {
			ctx := log.ContextWithStackTrace(ctx, err)
			log.Error(ctx, "unable to openfile %q: %v", binary.Name, err)
			return err
		}

		if err := w.client.PluginGetBinary(binary.PluginName, currentOS, currentARCH, fi); err != nil {
			ctx := log.ContextWithStackTrace(ctx, err)
			log.Error(ctx, "unable to download plugin %q, %q , %q: %v", binary.PluginName, currentOS, currentARCH, err)
			_ = fi.Close()
			return err
		}
		//It's downloaded. Close the file
		_ = fi.Close()
	} else {
		log.Debug(ctx, "plugin binary is in cache")
	}

	log.Info(ctx, "plugin successfully downloaded: %#v", binary.Name)
	return nil
}
