package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/mem"

	"github.com/ovh/cds/sdk"
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
}

func checkRequirements(w *currentWorker, a *sdk.Action) (bool, []sdk.Requirement) {
	requirementsOK := true
	errRequirements := []sdk.Requirement{}
	w.client.WorkerSetStatus(sdk.StatusChecking)
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

	return requirementsOK, errRequirements
}

func checkRequirement(w *currentWorker, r sdk.Requirement) (bool, error) {
	check := requirementCheckFuncs[r.Type]
	if check == nil {
		return false, fmt.Errorf("checkRequirement> Unknown type of requirement: %s supported requirements are : %v", r.Type, requirementCheckFuncs)
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
		log.Warning("[WARNING] Error Checkin %s requirement : %s", r.Name, err)
		return false, err
	}
	log.Warning("[NOTICE] Plugin %s successfully started", _plugin.Name())

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
	wm, err := sdk.GetWorkerModel(r.Value)
	if err != nil {
		return false, nil
	}

	if wm.ID == w.modelID {
		return true, nil
	}

	return false, nil
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
	if _, err := net.LookupIP(r.Name); err != nil {
		log.Warning("Error checking requirement : %s\n", err)
		return false, nil
	}

	return true, nil
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
