package microvm

import (
	"bytes"
	"context"
	"fmt"
	"text/template"
	"time"

	"github.com/rockbears/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/slug"
	"github.com/ovh/cds/sdk/telemetry"
)

// Keys of the worker secret and the guest files written from them
const (
	secretKeyConfig = "worker-config"
	secretKeyScript = "boot.sh"
	secretKeyUnit   = "cds-worker.service"

	guestPathConfig = "/etc/cds/worker-config"
	guestPathScript = "/usr/local/bin/cds-worker-boot.sh"
	guestPathUnit   = "/etc/systemd/system/cds-worker.service"
	guestUnit       = "cds-worker.service"
	guestPathWorker = "/usr/local/bin/worker"
	// guestWorkerBasedir is the worker basedir unless the hatchery configures
	// one. Not /tmp: systemd-tmpfiles empties it at boot, preloaded plugins
	// included.
	guestWorkerBasedir = "/var/lib/cds-worker"
)

// The worker binary comes from the platform cache; it is downloaded from
// the API only when the platform could not provide it. The worker reads its
// base64 configuration from CDS_CONFIG. The guest reboots at the end:
// Firecracker exits on a guest reboot, which ends the MicroVM (a poweroff
// would leave the VMM running).
// Its output goes to the serial console through the unit.
var bootScriptTemplate = template.Must(template.New("boot").Parse(`#!/bin/bash
if [ ! -x ` + guestPathWorker + ` ]; then
  if ! curl -fsSL{{if .Insecure}} -k{{end}} --retry 60 --retry-delay 1 --retry-all-errors --retry-max-time 60 "{{.API}}/download/worker/{{.OSArch}}" -o ` + guestPathWorker + `; then
    echo "unable to download the CDS worker from {{.API}}"
    systemctl reboot
    exit 1
  fi
  chmod +x ` + guestPathWorker + `
fi
export CDS_CONFIG="$(cat ` + guestPathConfig + `)"
` + guestPathWorker + `
systemctl reboot
`))

// The worker starts as soon as the platform unit configured the NICs, the
// routes and resolv.conf, instead of waiting for network-online.target and
// basic.target. Without the default dependencies, the unit keeps those it
// needs: local filesystems (and /tmp), and being stopped at shutdown so that
// the reboot of the boot script goes through. microvm-net.service keeps its
// own default dependencies, so it starts after sysinit.target (journald,
// tmpfiles). Wants rather than Requires: when the network unit fails, the
// worker still starts, fails to reach the API and ends the VM instead of
// leaving it idle until the registration timeout.
// The platform links the enabled units into multi-user.target.wants
// whatever their install section: that only queues the unit at boot, its
// ordering comes from After.
const workerUnit = `[Unit]
Description=CDS worker
DefaultDependencies=no
Wants=microvm-net.service
After=microvm-net.service local-fs.target
Conflicts=shutdown.target
Before=shutdown.target

[Service]
Type=simple
ExecStart=` + guestPathScript + `
StandardOutput=journal+console
StandardError=journal+console

[Install]
WantedBy=multi-user.target
`

// SpawnWorker creates the worker secret then its MicroVM
func (h *HatcheryMicroVM) SpawnWorker(ctx context.Context, spawnArgs hatchery.SpawnArguments) error {
	ctx, end := telemetry.Span(ctx, "HatcheryMicroVM.SpawnWorker",
		telemetry.Tag(telemetry.TagWorkflowNodeJobRun, spawnArgs.JobID),
		telemetry.Tag(telemetry.TagWorker, spawnArgs.WorkerName))
	defer end()

	if spawnArgs.Model.ModelV2 == nil || spawnArgs.RegisterOnly {
		return sdk.WithStack(fmt.Errorf("hatchery microvm only spawns workers for v2 jobs"))
	}

	flavorName := spawnArgs.Model.GetFlavor(spawnArgs.Requirements, h.Config.DefaultFlavor)
	flavor := h.Config.flavor(flavorName)
	if flavor == nil {
		return sdk.WithStack(fmt.Errorf("flavor %q not found in hatchery configuration", flavorName))
	}

	workerConfig := h.GenerateWorkerConfig(ctx, h, spawnArgs)
	if workerConfig.Basedir == "" {
		workerConfig.Basedir = guestWorkerBasedir
	}
	secret, err := h.buildSecret(spawnArgs, workerConfig)
	if err != nil {
		return err
	}
	plugins := h.pluginFiles(ctx, spawnArgs.Model.ModelV2.OSArch, workerConfig.Basedir, workerConfig.APIEndpoint, workerConfig.APIEndpointInsecure)
	mvm := h.buildMicroVM(spawnArgs, *flavor, secret.Name, h.workerBinary(ctx, spawnArgs, workerConfig), plugins)

	if sdk.IsValidUUID(spawnArgs.JobID) {
		flavorInfo := sdk.V2SendJobRunInfo{
			Level:   sdk.WorkflowRunInfoLevelInfo,
			Time:    time.Now(),
			Message: fmt.Sprintf("Worker %q will use flavor %q (%d vCPUs, %d MiB RAM, %d MiB disk)", spawnArgs.WorkerName, flavor.Name, flavor.VCPUs, flavor.MemoryMiB, diskMiB(*flavor, spawnArgs.Model.MicroVMSpec)),
		}
		if err := h.CDSClientV2().V2QueuePushJobInfo(ctx, spawnArgs.Region, spawnArgs.JobID, flavorInfo); err != nil {
			log.ErrorWithStackTrace(ctx, err)
		}
	}

	// The secret goes first: the platform reads it before the boot
	if _, err := h.kubeClient.CoreV1().Secrets(h.Config.Namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
		return sdk.WrapError(err, "cannot create secret %s", secret.Name)
	}

	created, err := h.dynClient.Resource(microVMResource).Namespace(h.Config.Namespace).Create(ctx, mvm, metav1.CreateOptions{})
	if err != nil {
		if errDel := h.kubeClient.CoreV1().Secrets(h.Config.Namespace).Delete(ctx, secret.Name, metav1.DeleteOptions{}); errDel != nil {
			log.Error(ctx, "cannot delete secret %s: %v", secret.Name, errDel)
		}
		return sdk.WrapError(err, "cannot create MicroVM %s", spawnArgs.WorkerName)
	}

	// The secret is garbage-collected with its MicroVM. When this fails, the
	// orphan secrets cleanup deletes it.
	patch := fmt.Sprintf(`{"metadata":{"ownerReferences":[{"apiVersion":%q,"kind":%q,"name":%q,"uid":%q}]}}`,
		microVMAPIVersion, microVMKind, created.GetName(), created.GetUID())
	if _, err := h.kubeClient.CoreV1().Secrets(h.Config.Namespace).Patch(ctx, secret.Name, types.MergePatchType, []byte(patch), metav1.PatchOptions{}); err != nil {
		log.Warn(ctx, "cannot set the owner of secret %s: %v", secret.Name, err)
	}

	log.Debug(ctx, "hatchery> microvm> SpawnWorker> %s > MicroVM created", spawnArgs.WorkerName)
	return nil
}

func (h *HatcheryMicroVM) labels(workerName string) map[string]string {
	return map[string]string{
		LABEL_HATCHERY_NAME: h.Config.Name,
		LABEL_WORKER_NAME:   workerName,
	}
}

func (h *HatcheryMicroVM) buildSecret(spawnArgs hatchery.SpawnArguments, workerConfig workerruntime.WorkerConfig) (*corev1.Secret, error) {
	// Fall back to the guest arch; the API normalizes uname's spelling (x86_64 -> amd64)
	osArch := spawnArgs.Model.ModelV2.OSArch
	if osArch == "" {
		osArch = "linux/$(uname -m)"
	}
	var script bytes.Buffer
	if err := bootScriptTemplate.Execute(&script, struct {
		API      string
		Insecure bool
		OSArch   string
	}{
		API:      workerConfig.APIEndpoint,
		Insecure: workerConfig.APIEndpointInsecure,
		OSArch:   osArch,
	}); err != nil {
		return nil, sdk.WithStack(err)
	}

	labels := h.labels(spawnArgs.WorkerName)
	// The platform only writes guest files from Secrets carrying this label
	labels[labelGuestFiles] = "true"

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      slug.Convert("cds-worker-" + spawnArgs.WorkerName),
			Namespace: h.Config.Namespace,
			Labels:    labels,
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			secretKeyConfig: workerConfig.EncodeBase64(),
			secretKeyScript: script.String(),
			secretKeyUnit:   workerUnit,
		},
	}, nil
}

// workerBinary is the worker binary the platform downloads and caches
type workerBinary struct {
	URL    string
	SHA256 string
}

// workerBinary returns the worker binary of the model os/arch, nil when its
// hash is unavailable: the boot script then downloads it.
func (h *HatcheryMicroVM) workerBinary(ctx context.Context, spawnArgs hatchery.SpawnArguments, workerConfig workerruntime.WorkerConfig) *workerBinary {
	osArch := spawnArgs.Model.ModelV2.OSArch
	if osArch == "" {
		return nil
	}
	sum, err := h.workerSHA256(ctx, workerConfig.APIEndpoint, workerConfig.APIEndpointInsecure, osArch)
	if err != nil {
		log.Warn(ctx, "hatchery> microvm> cannot compute the sha256 of the worker %s, the worker downloads it at boot: %v", osArch, err)
		return nil
	}
	return &workerBinary{URL: workerDownloadURL(workerConfig.APIEndpoint, osArch), SHA256: sum}
}

// bootArgs is the platform default command line with the configured extra
// arguments, and the SSH server masked unless debugging is configured
func (h *HatcheryMicroVM) bootArgs() string {
	args := platformBootArgs
	if h.Config.ExtraBootArgs != "" {
		args += " " + h.Config.ExtraBootArgs
	}
	if len(h.Config.DebugSSHAuthorizedKeys) == 0 {
		args += " " + sshBootArgs
	}
	return args
}

// diskMiB is the disk of the flavor, raised to the rootfs size of the model:
// the platform refuses a disk smaller than the image it converts
func diskMiB(flavor FlavorConfig, spec sdk.V2WorkerModelMicroVMSpec) int {
	return max(flavor.DiskGiB*1024, spec.Rootfs.Size)
}

func (h *HatcheryMicroVM) buildMicroVM(spawnArgs hatchery.SpawnArguments, flavor FlavorConfig, secretName string, worker *workerBinary, plugins []interface{}) *unstructured.Unstructured {
	spec := spawnArgs.Model.MicroVMSpec

	rootfs := map[string]interface{}{
		"url":    spec.Rootfs.URL,
		"sha256": spec.Rootfs.SHA256,
	}
	if spec.Rootfs.Format != "" {
		rootfs["format"] = spec.Rootfs.Format
	}
	if spec.Rootfs.Size > 0 {
		rootfs["sizeMiB"] = int64(spec.Rootfs.Size)
	}

	egress := map[string]interface{}{"enabled": true}
	if h.Config.EgressPolicy != "" {
		egress["policyRef"] = map[string]interface{}{"name": h.Config.EgressPolicy}
	}

	// SSH from the bastion, for debugging only
	networking := map[string]interface{}{
		"egress": egress,
		"admin":  map[string]interface{}{"enabled": len(h.Config.DebugSSHAuthorizedKeys) > 0},
	}
	if len(h.Config.DebugSSHAuthorizedKeys) > 0 {
		keys := make([]interface{}, 0, len(h.Config.DebugSSHAuthorizedKeys))
		for _, k := range h.Config.DebugSSHAuthorizedKeys {
			keys = append(keys, k)
		}
		networking["sshAuthorizedKeys"] = keys
	}

	guestFile := func(path string, mode string, key string) interface{} {
		return map[string]interface{}{
			"path":         path,
			"mode":         mode,
			"secretKeyRef": map[string]interface{}{"name": secretName, "key": key},
		}
	}

	files := []interface{}{
		guestFile(guestPathConfig, "0600", secretKeyConfig),
		guestFile(guestPathScript, "0755", secretKeyScript),
		guestFile(guestPathUnit, "0644", secretKeyUnit),
	}
	if worker != nil {
		files = append(files, map[string]interface{}{
			"path":   guestPathWorker,
			"mode":   "0755",
			"source": map[string]interface{}{"url": worker.URL, "sha256": worker.SHA256},
		})
	}
	files = append(files, plugins...)

	labels := map[string]interface{}{}
	for k, v := range h.labels(spawnArgs.WorkerName) {
		labels[k] = v
	}

	vmSpec := map[string]interface{}{
		"vcpus":     int64(flavor.VCPUs),
		"memoryMiB": int64(flavor.MemoryMiB),
		"kernel": map[string]interface{}{
			"url":    spec.Kernel.URL,
			"sha256": spec.Kernel.SHA256,
		},
		"rootfs":     rootfs,
		"bootArgs":   h.bootArgs(),
		"networking": networking,
		"guest": map[string]interface{}{
			"files":        files,
			"systemdUnits": []interface{}{guestUnit},
		},
	}
	// Unset, the platform keeps the rootfs size
	if disk := diskMiB(flavor, spec); disk > 0 {
		vmSpec["diskMiB"] = int64(disk)
	}

	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": microVMAPIVersion,
		"kind":       microVMKind,
		"metadata": map[string]interface{}{
			"name":      spawnArgs.WorkerName,
			"namespace": h.Config.Namespace,
			"labels":    labels,
		},
		"spec": vmSpec,
	}}
}
