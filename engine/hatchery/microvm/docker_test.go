package microvm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
)

const testRegistryPassword = "s3cr3t-r3gistry-p4ss"

func testDockerBase() DockerBaseConfig {
	return DockerBaseConfig{
		KernelURL:     "https://example.org/docker-vmlinux",
		KernelSHA256:  testSHA,
		RootfsURL:     "https://example.org/docker-rootfs.squashfs",
		RootfsSHA256:  testSHA,
		RootfsFormat:  "squashfs",
		RootfsSizeMiB: 8192,
	}
}

func newTestDockerHatchery(t *testing.T) *HatcheryMicroVM {
	h, _, _ := newTestHatchery(t)
	h.Config.ModelType = sdk.WorkerModelTypeDocker
	h.Config.DockerBase = testDockerBase()
	return h
}

func testDockerModel(spec sdk.V2WorkerModelDockerSpec) sdk.WorkerStarterWorkerModel {
	raw, _ := json.Marshal(spec)
	return sdk.WorkerStarterWorkerModel{
		ModelV2:    &sdk.V2WorkerModel{Name: "golang", Type: sdk.WorkerModelTypeDocker, OSArch: "linux/amd64", Spec: raw},
		DockerSpec: spec,
	}
}

// spawnDocker spawns a docker worker and returns its MicroVM, its Secret and
// the job infos pushed
func spawnDocker(t *testing.T, h *HatcheryMicroVM, model sdk.WorkerStarterWorkerModel) (*unstructured.Unstructured, map[string]string, []sdk.V2SendJobRunInfo) {
	api := newTestAPI(t)
	h.Config.API.HTTP.URL = api.URL
	jobInfos := mockPluginInfos(t, h, api.plugin, nil)
	require.NoError(t, h.SpawnWorker(context.TODO(), hatchery.SpawnArguments{WorkerName: "w1", WorkerToken: "worker-token", Model: model, JobID: sdk.UUID()}))
	mvm, err := h.dynClient.Resource(microVMResource).Namespace("cds-workers").Get(context.TODO(), "w1", metav1.GetOptions{})
	require.NoError(t, err)
	secret, err := h.kubeClient.CoreV1().Secrets("cds-workers").Get(context.TODO(), "cds-worker-w1", metav1.GetOptions{})
	require.NoError(t, err)
	return mvm, secret.StringData, *jobInfos
}

func guestFiles(mvm *unstructured.Unstructured) map[string]map[string]interface{} {
	files, _, _ := unstructured.NestedSlice(mvm.Object, "spec", "guest", "files")
	byPath := map[string]map[string]interface{}{}
	for _, f := range files {
		file := f.(map[string]interface{})
		byPath[file["path"].(string)] = file
	}
	return byPath
}

// A docker job boots the base image of the hatchery, not an image of the
// model, and runs its image in a container with the VM worker binary as
// entrypoint: the job image needs no curl, and the job gets no Docker access
func TestSpawnDockerWorker(t *testing.T) {
	h := newTestDockerHatchery(t)
	h.Config.Provision.InjectEnvVars = []string{"INJECTED=from-hatchery", "OVERRIDDEN=by-hatchery"}
	model := testDockerModel(sdk.V2WorkerModelDockerSpec{
		Image:    "registry.example.org/team/golang:1.26",
		Username: "robot",
		Password: testRegistryPassword,
		Envs:     map[string]string{"OVERRIDDEN": "by-model", "GOFLAGS": "-mod=mod"},
	})
	model.Flavor = "S"
	mvm, secret, jobInfos := spawnDocker(t, h, model)

	spec := mvm.Object["spec"].(map[string]interface{})
	require.Equal(t, map[string]interface{}{"url": "https://example.org/docker-vmlinux", "sha256": testSHA}, spec["kernel"])
	require.Equal(t, map[string]interface{}{"url": "https://example.org/docker-rootfs.squashfs", "sha256": testSHA, "format": "squashfs", "sizeMiB": int64(8192)}, spec["rootfs"])
	require.Equal(t, int64(2), spec["vcpus"])
	require.Equal(t, int64(8192), spec["memoryMiB"])
	require.Equal(t, int64(20*1024), spec["diskMiB"])
	policy, _, _ := unstructured.NestedString(mvm.Object, "spec", "networking", "egress", "policyRef", "name")
	require.Equal(t, "cds-workers", policy)
	require.Equal(t, h.bootArgs(), spec["bootArgs"])

	// The worker binary and the preloaded plugins come from the platform cache
	// as for a microvm worker; the environment and the registry password are
	// root-only files
	files := guestFiles(mvm)
	require.Len(t, files, 6)
	require.Equal(t, map[string]interface{}{"name": "cds-worker-w1", "key": secretKeyEnv}, files[guestPathEnv]["secretKeyRef"])
	require.Equal(t, "0600", files[guestPathEnv]["mode"], "the environment holds the worker token")
	require.Equal(t, map[string]interface{}{"name": "cds-worker-w1", "key": secretKeyRegistryPassword}, files[guestPathRegistryPassword]["secretKeyRef"])
	require.Equal(t, "0600", files[guestPathRegistryPassword]["mode"])
	require.Contains(t, files, guestPathWorker)
	require.Contains(t, files, "/var/lib/cds-worker/script-linux-amd64")
	require.Contains(t, files, guestPathScript)
	require.Contains(t, files, guestPathUnit)
	require.Equal(t, workerUnit, secret[secretKeyUnit], "same unit ordering as a microvm worker")

	script := secret[secretKeyScript]
	// Login, then pull, then run: a private image cannot be pulled before the login
	login := strings.Index(script, `docker login --username 'robot' --password-stdin "$registry" < /etc/cds/registry-password`)
	pull := strings.Index(script, `docker pull --quiet "$image"`)
	run := strings.Index(script, "docker run ")
	require.True(t, login > 0 && pull > login && run > pull, script)
	require.Contains(t, script, "registry='registry.example.org'\n")
	require.Contains(t, script, "image='registry.example.org/team/golang:1.26'\n")

	runLine := script[run : run+strings.Index(script[run:], "\n")]
	require.Equal(t, `docker run --rm --network host --env-file /etc/cds/worker-env --volume /usr/local/bin/worker:/usr/local/bin/worker:ro --volume "$basedir:$basedir" --volume /etc/ssl/certs/ca-certificates.crt:/etc/cds/certs/ca-certificates.crt:ro --entrypoint /usr/local/bin/worker "$image"`, runLine)
	require.NotContains(t, script, "docker.sock", "the job has no Docker access")
	require.Contains(t, script, "basedir='/var/lib/cds-worker'\n")
	// An image running as non-root can write the basedir and run the plugins
	require.Contains(t, script, `chmod 1777 "$basedir"`)
	require.Contains(t, script, `chmod -R a+rX "$basedir"`)
	require.True(t, strings.HasSuffix(script, "\"$image\"\nsystemctl reboot\n"), "the end of the container ends the VM")

	// The registry password is only in the Secret key of its file
	require.Equal(t, testRegistryPassword, secret[secretKeyRegistryPassword])
	require.NotContains(t, script, testRegistryPassword)
	rawMVM, err := json.Marshal(mvm.Object)
	require.NoError(t, err)
	require.NotContains(t, string(rawMVM), testRegistryPassword)
	require.NotContains(t, secret[secretKeyEnv], testRegistryPassword)

	// The environment of a swarm v2 docker worker: injected variables, the
	// flavor memory, the worker config, then the model variables win
	env := map[string]string{}
	for _, line := range strings.Split(strings.TrimSuffix(secret[secretKeyEnv], "\n"), "\n") {
		name, value, _ := strings.Cut(line, "=")
		env[name] = value
	}
	require.Equal(t, "from-hatchery", env["INJECTED"])
	require.Equal(t, "by-model", env["OVERRIDDEN"])
	require.Equal(t, "-mod=mod", env["GOFLAGS"])
	require.Equal(t, "8192", env["CDS_MODEL_MEMORY"])
	raw, err := base64.StdEncoding.DecodeString(env["CDS_CONFIG"])
	require.NoError(t, err)
	var workerConfig workerruntime.WorkerConfig
	require.NoError(t, json.Unmarshal(raw, &workerConfig))
	require.Equal(t, "worker-token", workerConfig.APIToken)
	require.Equal(t, "/var/lib/cds-worker", workerConfig.Basedir, "mounted at the same path, where the plugins are preloaded")

	require.Len(t, jobInfos, 1)
	require.Equal(t, `Worker "w1" will run image "registry.example.org/team/golang:1.26" with flavor "S" (2 vCPUs, 8192 MiB RAM, 20480 MiB disk)`, jobInfos[0].Message)
}

// Without credentials there is nothing to log in with; an image without
// registry logs in to the Docker Hub
func TestSpawnDockerWorkerRegistry(t *testing.T) {
	h := newTestDockerHatchery(t)
	mvm, secret, _ := spawnDocker(t, h, testDockerModel(sdk.V2WorkerModelDockerSpec{Image: "golang:1.26"}))
	require.NotContains(t, secret[secretKeyScript], "docker login")
	require.NotContains(t, secret, secretKeyRegistryPassword)
	require.NotContains(t, guestFiles(mvm), guestPathRegistryPassword)

	h = newTestDockerHatchery(t)
	_, secret, _ = spawnDocker(t, h, testDockerModel(sdk.V2WorkerModelDockerSpec{Image: "team/golang:1.26", Username: "robot", Password: testRegistryPassword}))
	require.Contains(t, secret[secretKeyScript], "registry='docker.io'\n")
}

// The flavor disk is raised to the base image size, never to a model size
func TestSpawnDockerWorkerDiskNeverSmallerThanBase(t *testing.T) {
	h := newTestDockerHatchery(t)
	h.Config.DockerBase.RootfsSizeMiB = 30000
	mvm, _, _ := spawnDocker(t, h, testDockerModel(sdk.V2WorkerModelDockerSpec{Image: "golang:1.26"}))
	disk, _, _ := unstructured.NestedInt64(mvm.Object, "spec", "diskMiB")
	require.Equal(t, int64(30000), disk)
}

// Job services need a network of containers the docker mode does not build:
// refused rather than run without them
func TestSpawnDockerWorkerRefusesServices(t *testing.T) {
	h := newTestDockerHatchery(t)
	err := h.SpawnWorker(context.TODO(), hatchery.SpawnArguments{
		WorkerName: "w1",
		Model:      testDockerModel(sdk.V2WorkerModelDockerSpec{Image: "golang:1.26"}),
		JobID:      sdk.UUID(),
		Services:   map[string]sdk.V2JobService{"postgres": {Image: "postgres:16"}},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "services")
}

// A docker env file has one variable per line: a multi-line value would
// silently become other variables
func TestDockerEnvFileRefusesMultilineValues(t *testing.T) {
	model := testDockerModel(sdk.V2WorkerModelDockerSpec{Image: "golang:1.26", Envs: map[string]string{"CERT": "line1\nCDS_CONFIG=forged"}})
	_, err := dockerEnvFile(hatchery.SpawnArguments{Model: model}, workerruntime.WorkerConfig{}, defaultFlavors[0])
	require.Error(t, err)
}

func TestCanSpawnDocker(t *testing.T) {
	h := newTestDockerHatchery(t)
	ctx := context.TODO()
	require.Equal(t, sdk.WorkerModelTypeDocker, h.ModelType(), "the type sent at signin, which the API filters the jobs on")

	require.True(t, h.CanSpawn(ctx, testDockerModel(sdk.V2WorkerModelDockerSpec{Image: "golang:1.26"}), "job", nil))
	require.False(t, h.CanSpawn(ctx, testDockerModel(sdk.V2WorkerModelDockerSpec{}), "job", nil), "no image to run")
	// A microvm model belongs to the other instance
	require.False(t, h.CanSpawn(ctx, testModel(testSpec()), "job", nil))
	otherArch := testDockerModel(sdk.V2WorkerModelDockerSpec{Image: "golang:1.26"})
	otherArch.ModelV2.OSArch = "linux/arm64"
	require.False(t, h.CanSpawn(ctx, otherArch, "job", nil))
	unknownFlavor := testDockerModel(sdk.V2WorkerModelDockerSpec{Image: "golang:1.26"})
	unknownFlavor.Flavor = "XXL"
	require.False(t, h.CanSpawn(ctx, unknownFlavor, "job", nil))
	tooMuchMemory := testDockerModel(sdk.V2WorkerModelDockerSpec{Image: "golang:1.26"})
	tooMuchMemory.Memory = 65536
	require.False(t, h.CanSpawn(ctx, tooMuchMemory, "job", nil))
}

// runs-on.memory picks the smallest flavor holding it, whatever the
// declaration order, unless the job names a flavor
func TestWorkerFlavor(t *testing.T) {
	cfg := HatcheryConfiguration{Flavors: []FlavorConfig{
		{Name: "L", VCPUs: 6, MemoryMiB: 24576},
		{Name: "S", VCPUs: 2, MemoryMiB: 8192},
		{Name: "M", VCPUs: 4, MemoryMiB: 16384},
		{Name: "XS", VCPUs: 1, MemoryMiB: 4096},
	}, DefaultFlavor: "M"}
	withModelFlavor := testSpec()
	withModelFlavor.Flavor = "L"

	for name, tt := range map[string]struct {
		model  sdk.WorkerStarterWorkerModel
		flavor string
		memory int64
		want   string
	}{
		"smallest holding the memory":     {model: testDockerModel(sdk.V2WorkerModelDockerSpec{Image: "i"}), memory: 5000, want: "S"},
		"memory equal to a flavor":        {model: testDockerModel(sdk.V2WorkerModelDockerSpec{Image: "i"}), memory: 4096, want: "XS"},
		"job flavor wins over memory":     {model: testDockerModel(sdk.V2WorkerModelDockerSpec{Image: "i"}), flavor: "l", memory: 1024, want: "L"},
		"default without flavor nor mem":  {model: testDockerModel(sdk.V2WorkerModelDockerSpec{Image: "i"}), want: "M"},
		"microvm job memory":              {model: testModel(testSpec()), memory: 5000, want: "S"},
		"microvm model flavor wins":       {model: testModel(withModelFlavor), memory: 1024, want: "L"},
		"microvm default without any ask": {model: testModel(testSpec()), want: "M"},
	} {
		t.Run(name, func(t *testing.T) {
			tt.model.Flavor = tt.flavor
			tt.model.Memory = tt.memory
			f, err := cfg.workerFlavor(tt.model, nil)
			require.NoError(t, err)
			require.Equal(t, tt.want, f.Name)
		})
	}

	// No smaller flavor than asked
	model := testDockerModel(sdk.V2WorkerModelDockerSpec{Image: "i"})
	model.Memory = 30000
	_, err := cfg.workerFlavor(model, nil)
	require.Error(t, err)
}

func TestCheckConfigurationModelType(t *testing.T) {
	h := New()
	cfg := HatcheryConfiguration{Namespace: "cds-workers"}
	cfg.Name = "my-hatchery"
	cfg.API.HTTP.URL = "https://cds-api.example.org"
	cfg.API.TokenV2 = "v2-token"

	// Left out, the hatchery keeps running microvm models
	require.NoError(t, h.CheckConfiguration(cfg))
	cfg.applyDefaults()
	require.Equal(t, sdk.WorkerModelTypeMicroVM, cfg.ModelType)

	cfg.ModelType = "openstack"
	err := h.CheckConfiguration(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), `invalid modelType "openstack"`)

	// A docker hatchery without a base image could not boot any worker
	cfg.ModelType = sdk.WorkerModelTypeDocker
	err = h.CheckConfiguration(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "dockerBase")

	cfg.DockerBase = testDockerBase()
	require.NoError(t, h.CheckConfiguration(cfg))
	// The format defaults to squashfs
	cfg.DockerBase.RootfsFormat = ""
	require.NoError(t, h.CheckConfiguration(cfg))

	for name, broken := range map[string]func(*DockerBaseConfig){
		"kernel sha256": func(b *DockerBaseConfig) { b.KernelSHA256 = "not-a-sha" },
		"rootfs url":    func(b *DockerBaseConfig) { b.RootfsURL = "file:///rootfs" },
		"rootfs format": func(b *DockerBaseConfig) { b.RootfsFormat = "qcow2" },
	} {
		t.Run(name, func(t *testing.T) {
			c := cfg
			c.DockerBase = testDockerBase()
			broken(&c.DockerBase)
			require.Error(t, h.CheckConfiguration(c))
		})
	}
}

// An image without CA certificates (ubuntu:24.04) cannot verify the CDS API:
// the worker gets the VM bundle after the default directories, and a model
// env still wins
func TestDockerEnvFileCertDirs(t *testing.T) {
	spawnArgs := hatchery.SpawnArguments{Model: sdk.WorkerStarterWorkerModel{}}
	env, err := dockerEnvFile(spawnArgs, workerruntime.WorkerConfig{}, FlavorConfig{MemoryMiB: 4096})
	require.NoError(t, err)
	require.Contains(t, env, "SSL_CERT_DIR=/etc/ssl/certs:/etc/pki/tls/certs:/etc/cds/certs\n")

	spawnArgs.Model.DockerSpec.Envs = map[string]string{"SSL_CERT_DIR": "/opt/certs"}
	env, err = dockerEnvFile(spawnArgs, workerruntime.WorkerConfig{}, FlavorConfig{MemoryMiB: 4096})
	require.NoError(t, err)
	require.Contains(t, env, "SSL_CERT_DIR=/opt/certs\n")
	require.NotContains(t, env, "/etc/cds/certs")
}
