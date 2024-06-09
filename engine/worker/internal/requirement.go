package internal

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/rockbears/log"

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
	sdk.FlavorRequirement:   checkFlavorRequirement,
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
	if len(strings.Split(r.Value, "/")) == 5 {
		return true, nil
	}

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
	isSameName := modelName == w.model.Name // for backward compatibility with runs, if only the name match we considered that the model can be used, keep this condition until the workflow runs were not migrated.
	return isSharedInfra || isSameName, nil
}

func checkServiceRequirement(w *CurrentWorker, r sdk.Requirement) (bool, error) {
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
	return true, nil
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

func checkFlavorRequirement(w *CurrentWorker, r sdk.Requirement) (bool, error) {
	return true, nil
}

// checkPlugins returns true if current job:
//   - is not linked to a deployment integration
//   - is linked to a deployment integration, plugin well downloaded (in this func) and
//     requirements on the plugins are OK too
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
			return sdk.NewErrorFrom(sdk.ErrPluginInvalid, "unable to download plugin %q %q %q: %v", binary.PluginName, currentOS, currentARCH, err)
		}
		//It's downloaded. Close the file
		_ = fi.Close()
	} else {
		log.Debug(ctx, "plugin binary is in cache")
	}

	log.Info(ctx, "plugin successfully downloaded: %#v", binary.Name)
	return nil
}
