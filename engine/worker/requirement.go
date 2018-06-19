package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/shirou/gopsutil/mem"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/platformplugin"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/plugin"
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

func checkRequirements(w *currentWorker, a *sdk.Action, execGroups []sdk.Group, bookedJobID int64) (bool, []sdk.Requirement) {
	requirementsOK := true
	errRequirements := []sdk.Requirement{}
	if err := w.client.WorkerSetStatus(sdk.StatusChecking); err != nil {
		log.Error("WorkerSetStatus> error on WorkerSetStatus(sdk.StatusChecking): %s", err)
	}

	log.Debug("checkRequirements> for JobID:%d model of worker: %s", bookedJobID, w.model.Name)

	// DEPRECATED
	// this code is useful for pipelineBuildJob
	// with CDS Workflows, the queue contains only jobs executable by worker
	// after removing pbBuildJob, check execGroups here can be removed
	if execGroups != nil && len(execGroups) > 0 && w.model.GroupID > 0 {
		checkGroup := false
		for _, g := range execGroups {
			if g.ID == w.model.GroupID {
				checkGroup = true
				break
			}
		}
		if !checkGroup {
			requirementsOK = false
			log.Debug("checkRequirements> model %s attached to group %d can't run this job", w.model.Name, w.model.GroupID)
			return requirementsOK, nil
		}
	}

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

	log.Debug("checkRequirements> checkRequirements:%t errRequirements:%s", requirementsOK, errRequirements)
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
	pluginBinary := path.Join(w.basedir, r.Name)

	if _, err := os.Stat(pluginBinary); os.IsNotExist(err) {
		//If the file doesn't exist. Download it.
		if err := sdk.DownloadPlugin(r.Name, w.basedir); err != nil {
			return false, err
		}
		if err := os.Chmod(pluginBinary, 0700); err != nil {
			return false, err
		}
	}

	pluginClient := plugin.NewClient(context.Background(), r.Name, pluginBinary, "", "", false)
	defer pluginClient.Kill()

	_plugin, err := pluginClient.Instance()
	if err != nil {
		log.Warning("checkPluginRequirement> Error Checking %s requirement : %s", r.Name, err)
		return false, err
	}
	log.Debug("checkPluginRequirement> Plugin %s successfully started", _plugin.Name())

	return true, nil
}

func checkHostnameRequirement(w *currentWorker, r sdk.Requirement) (bool, error) {
	h, err := os.Hostname()
	if err != nil {
		return false, err
	}

	if h == r.Value {
		return true, nil
	}

	return false, nil
}

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
	t := strings.Split(r.Value, " ")
	wm, err := w.client.WorkerModel(t[0])
	return wm.ID == w.model.ID, err
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
		if _, err := net.LookupIP(r.Name); err != nil {
			log.Debug("Error checking requirement : %s", err)
			time.Sleep(2 * time.Second)
			continue
		}
		return true, nil
	}

	return false, nil
}

func checkMemoryRequirement(w *currentWorker, r sdk.Requirement) (bool, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return false, err
	}
	totalMemory := v.Total

	neededMemory, err := strconv.ParseInt(r.Value, 10, 64)
	if err != nil {
		return false, err
	}
	//Assuming memory is in megabytes
	//If we have more than 90% of neededMemory, lets do it
	return int64(totalMemory) >= (neededMemory*1024*1024)*90/100, nil
}

func checkVolumeRequirement(w *currentWorker, r sdk.Requirement) (bool, error) {
	// available only on worker booked
	if w.bookedPBJobID == 0 && w.bookedWJobID == 0 {
		return false, nil
	}

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

	return osarch[0] == strings.ToLower(runtime.GOOS) && osarch[1] == strings.ToLower(runtime.GOARCH), nil
}

func checkPlugins(w *currentWorker, j sdk.WorkflowNodeJobRun) (bool, error) {
	var currentOS = strings.ToLower(runtime.GOOS)
	var currentARCH = strings.ToLower(runtime.GOARCH)
	var binary *sdk.GRPCPluginBinary

	if len(j.PlatformPluginBinaries) == 0 {
		return true, nil
	}

	log.Debug("Checking plugins...(%#v)", j.PlatformPluginBinaries)

	//First check OS and Architecture
	for _, b := range j.PlatformPluginBinaries {
		if b.OS == currentOS && b.Arch == currentARCH {
			binary = &b
			break
		}
	}
	if binary == nil {
		return false, fmt.Errorf("%s %s not supported by this plugin", currentOS, currentARCH)
	}

	//Then check plugin requirements
	for _, r := range binary.Requirements {
		ok, err := checkRequirement(w, r)
		if err != nil {
			log.Warning("checkQueue> error on checkRequirement %s", err)
		}
		if !ok {
			return false, fmt.Errorf("plugin requirement %s does not match", r.Name)
		}
	}

	//Then try to download the plugin
	pluginBinary := path.Join(w.basedir, binary.Name)
	if _, err := os.Stat(pluginBinary); os.IsNotExist(err) {
		log.Debug("Downloading the plugin %s", binary.PluginName)
		//If the file doesn't exist. Download it.
		fi, err := os.OpenFile(pluginBinary, os.O_CREATE|os.O_RDWR, os.FileMode(binary.Perm))
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

	//Last but not least: start the plugin
	socket, err := startGRPCPlugin(context.Background(), w, *binary, startGRPCPluginOptions{
		out: os.Stdout,
		err: os.Stderr,
	})
	if err != nil {
		return false, err
	}

	c, err := platformplugin.Client(context.Background(), socket.Socket)
	if err != nil {
		return false, fmt.Errorf("unable to call grpc plugin: %v", err)
	}

	socket.Client = c

	m, err := c.Manifest(context.Background(), new(empty.Empty))
	if err != nil {
		return false, fmt.Errorf("unable to call grpc plugin manifest: %v", err)
	}

	log.Info("plugin successfully initialized: %#v", m)

	registerPluginClient(w, binary.PluginName, socket)

	return true, nil
}
