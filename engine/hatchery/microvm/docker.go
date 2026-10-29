package microvm

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/distribution/reference"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
)

// Keys of the worker secret and the guest files of a docker worker. Both are
// read by root only: the environment holds the worker token, the password
// file the registry password.
const (
	secretKeyEnv              = "worker-env"
	secretKeyRegistryPassword = "registry-password"

	guestPathEnv              = "/etc/cds/worker-env"
	guestPathRegistryPassword = "/etc/cds/registry-password"

	// The CA bundle of the VM, mounted for a worker whose image has none
	guestPathCABundle     = "/etc/ssl/certs/ca-certificates.crt"
	containerPathCAs      = "/etc/cds/certs"
	containerCertDirs     = "/etc/ssl/certs:/etc/pki/tls/certs:" + containerPathCAs
	containerCABundleFile = containerPathCAs + "/ca-certificates.crt"
)

// The docker worker runs the job image with the worker binary of the VM
// bind-mounted as its entrypoint: a static binary, so the image needs
// neither curl nor a shell. Docker is socket-activated in the base image:
// the first docker command starts it. The container shares the VM network
// (its egress policy and MTU), not its docker socket, and has no resource
// limit: the VM is the limit. The worker basedir is mounted at the same path
// for the worker to find the preloaded plugins; it is writable by any user
// and the plugins executable by any user, for images running as non-root.
// A login or pull failure ends the VM with a line on the serial console.
var dockerBootScriptTemplate = template.Must(template.New("docker").Funcs(template.FuncMap{"quote": shellQuote}).Parse(`#!/bin/bash
` + workerDownloadScript + `image={{quote .Image}}
basedir={{quote .Basedir}}
{{- if .Registry}}
registry={{quote .Registry}}
if ! out=$(docker login --username {{quote .Username}} --password-stdin "$registry" < ` + guestPathRegistryPassword + ` 2>&1); then
  echo "unable to log in to $registry to pull $image: $out"
  systemctl reboot
  exit 1
fi
{{- end}}
for i in 1 2 3 4 5; do
  if out=$(docker pull --quiet "$image" 2>&1); then
    break
  fi
  if [ "$i" -eq 5 ]; then
    echo "unable to pull $image: $out"
    systemctl reboot
    exit 1
  fi
  sleep 5
done
mkdir -p "$basedir"
chmod 1777 "$basedir"
chmod -R a+rX "$basedir"
docker run --rm --network host --env-file ` + guestPathEnv + ` --volume ` + guestPathWorker + `:` + guestPathWorker + `:ro --volume "$basedir:$basedir" --volume ` + guestPathCABundle + `:` + containerCABundleFile + `:ro --entrypoint ` + guestPathWorker + ` "$image"
systemctl reboot
`))

// shellQuote quotes a string for bash
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// registryLogin tells whether the worker logs in to the registry of its image
func registryLogin(model sdk.WorkerStarterWorkerModel) bool {
	return model.DockerSpec.Username != ""
}

// vmSpec is the image the worker boots: the docker base image for a docker
// model, the model itself for a microvm model
func (h *HatcheryMicroVM) vmSpec(model sdk.WorkerStarterWorkerModel) sdk.V2WorkerModelMicroVMSpec {
	if h.ModelType() == sdk.WorkerModelTypeDocker {
		return h.Config.DockerBase.spec()
	}
	return model.MicroVMSpec
}

// dockerBootScript runs the image of the model with the worker
func dockerBootScript(spawnArgs hatchery.SpawnArguments, workerConfig workerruntime.WorkerConfig, data bootScriptData) (string, error) {
	image := spawnArgs.Model.DockerSpec.Image
	ref, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return "", sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid image %q: %v", image, err)
	}
	data.Image = image
	data.Basedir = workerConfig.Basedir
	if registryLogin(spawnArgs.Model) {
		data.Registry = reference.Domain(ref)
		data.Username = spawnArgs.Model.DockerSpec.Username
	}
	var script strings.Builder
	if err := dockerBootScriptTemplate.Execute(&script, data); err != nil {
		return "", sdk.WithStack(err)
	}
	return script.String(), nil
}

// dockerEnvFile is the environment the swarm hatchery gives a v2 docker
// worker, as a docker env file
func dockerEnvFile(spawnArgs hatchery.SpawnArguments, workerConfig workerruntime.WorkerConfig, flavor FlavorConfig) (string, error) {
	envs := maps.Clone(workerConfig.InjectEnvVars)
	if envs == nil {
		envs = map[string]string{}
	}
	envs["CDS_MODEL_MEMORY"] = strconv.Itoa(flavor.MemoryMiB)
	envs["CDS_CONFIG"] = workerConfig.EncodeBase64()
	// An image without CA certificates (ubuntu, debian without the
	// ca-certificates package) gets those of the VM to reach the CDS API.
	// The worker is a Go binary: it reads the certificates of every
	// directory listed, the default ones first, so the image's own CAs
	// still count. A model env can set another list.
	envs["SSL_CERT_DIR"] = containerCertDirs
	maps.Copy(envs, spawnArgs.Model.DockerSpec.Envs)

	var file strings.Builder
	for _, name := range slices.Sorted(maps.Keys(envs)) {
		// An env file has one variable per line
		if name == "" || strings.ContainsAny(name, "=\n\r") || strings.ContainsAny(envs[name], "\n\r") {
			return "", sdk.NewErrorFrom(sdk.ErrInvalidData, "environment variable %q: multi-line values and names with '=' are not supported", name)
		}
		fmt.Fprintf(&file, "%s=%s\n", name, envs[name])
	}
	return file.String(), nil
}
