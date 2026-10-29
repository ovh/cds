package microvm

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
)

const testSHA = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func newTestHatchery(t *testing.T, objs ...runtime.Object) (*HatcheryMicroVM, *k8sfake.Clientset, *dynamicfake.FakeDynamicClient) {
	log.Factory = log.NewTestingWrapper(t)
	h := New()
	h.Config.Name = "my-hatchery"
	h.Config.Namespace = "cds-workers"
	h.Config.API.HTTP.URL = "https://cds-api.example.org"
	h.Config.EgressPolicy = "cds-workers"
	h.Config.MaxVCPUs = 4
	h.Config.MaxMemory = 4096
	h.Config.applyDefaults()
	h.ServiceInstance = &sdk.Service{CanonicalService: sdk.CanonicalService{ID: 1, Name: "my-hatchery"}}

	kube := k8sfake.NewClientset()
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{microVMResource: "MicroVMList"}, objs...)
	h.kubeClient = kube
	h.dynClient = dyn
	return h, kube, dyn
}

func testModel(spec sdk.V2WorkerModelMicroVMSpec) sdk.WorkerStarterWorkerModel {
	raw, _ := json.Marshal(spec)
	return sdk.WorkerStarterWorkerModel{
		ModelV2:     &sdk.V2WorkerModel{Name: "ubuntu", Type: sdk.WorkerModelTypeMicroVM, OSArch: "linux/amd64", Spec: raw},
		MicroVMSpec: spec,
	}
}

func testSpec() sdk.V2WorkerModelMicroVMSpec {
	return sdk.V2WorkerModelMicroVMSpec{
		Kernel: sdk.V2WorkerModelMicroVMArtifact{URL: "https://example.org/vmlinux", SHA256: testSHA},
		Rootfs: sdk.V2WorkerModelMicroVMRootfs{URL: "https://example.org/rootfs.squashfs", SHA256: testSHA, Format: "squashfs", Size: 4096},
		VCPUs:  2,
		Memory: 1024,
	}
}

// microVM builds a MicroVM as the platform reports it
func microVM(name, hatcheryName string, created time.Time, status map[string]interface{}) *unstructured.Unstructured {
	u := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": microVMAPIVersion,
		"kind":       microVMKind,
		"metadata": map[string]interface{}{
			"name":              name,
			"namespace":         "cds-workers",
			"creationTimestamp": created.UTC().Format(time.RFC3339),
			"labels": map[string]interface{}{
				LABEL_HATCHERY_NAME: hatcheryName,
				LABEL_WORKER_NAME:   name,
			},
		},
		"spec": map[string]interface{}{},
	}}
	if status != nil {
		u.Object["status"] = status
	}
	return u
}

func quotaExceededStatus() map[string]interface{} {
	return map[string]interface{}{
		"phase": "Pending",
		"conditions": []interface{}{
			map[string]interface{}{"type": "QuotaExceeded", "status": "True", "reason": "Forbidden"},
		},
	}
}

func TestCanSpawn(t *testing.T) {
	h, _, _ := newTestHatchery(t)
	ctx := context.TODO()

	require.True(t, h.CanSpawn(ctx, testModel(testSpec()), "job", nil))

	// The model size defaults to the hatchery defaults, which fit the maximum
	require.True(t, h.CanSpawn(ctx, testModel(sdk.V2WorkerModelMicroVMSpec{Kernel: testSpec().Kernel, Rootfs: testSpec().Rootfs}), "job", nil))

	// A model asking for more than the hatchery allows is left to another hatchery
	tooManyCPUs := testSpec()
	tooManyCPUs.VCPUs = 8
	require.False(t, h.CanSpawn(ctx, testModel(tooManyCPUs), "job", nil))
	tooMuchMemory := testSpec()
	tooMuchMemory.Memory = 8192
	require.False(t, h.CanSpawn(ctx, testModel(tooMuchMemory), "job", nil))

	// Only v2 microvm models
	docker := testModel(testSpec())
	docker.ModelV2.Type = sdk.WorkerModelTypeDocker
	require.False(t, h.CanSpawn(ctx, docker, "job", nil))
	require.False(t, h.CanSpawn(ctx, sdk.WorkerStarterWorkerModel{ModelV1: &sdk.Model{Type: sdk.Docker}}, "job", nil))

	otherArch := testModel(testSpec())
	otherArch.ModelV2.OSArch = "linux/arm64"
	require.False(t, h.CanSpawn(ctx, otherArch, "job", nil))

	noKernel := testSpec()
	noKernel.Kernel.URL = ""
	require.False(t, h.CanSpawn(ctx, testModel(noKernel), "job", nil))
}

// A MicroVM refused by the quota means the namespace is full: spawning more
// would only add MicroVMs that cannot be scheduled either.
func TestCanAllocateResourcesWaitsForQuota(t *testing.T) {
	now := time.Now()
	h, _, _ := newTestHatchery(t, microVM("w-running", "my-hatchery", now, map[string]interface{}{"phase": "Running"}))
	can, err := h.CanAllocateResources(context.TODO(), testModel(testSpec()), "job", nil)
	require.NoError(t, err)
	require.True(t, can)

	h, _, _ = newTestHatchery(t, microVM("w-quota", "my-hatchery", now, quotaExceededStatus()))
	can, err = h.CanAllocateResources(context.TODO(), testModel(testSpec()), "job", nil)
	require.NoError(t, err)
	require.False(t, can)

	// The quota of another hatchery's MicroVM is not ours to wait for
	h, _, _ = newTestHatchery(t, microVM("w-other", "other-hatchery", now, quotaExceededStatus()))
	can, err = h.CanAllocateResources(context.TODO(), testModel(testSpec()), "job", nil)
	require.NoError(t, err)
	require.True(t, can)
}

// testAPI serves the API version, the worker binaries and the script
// plugin binary, and counts their downloads
type testAPI struct {
	*httptest.Server
	version         string
	worker          string
	downloads       int
	plugin          string
	pluginDownloads int
}

func newTestAPI(t *testing.T) *testAPI {
	api := &testAPI{version: "1.0.0", worker: "worker binary", plugin: "script plugin"}
	api.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mon/version":
			_ = json.NewEncoder(w).Encode(sdk.Version{Version: api.version})
		case "/download/worker/linux/amd64", "/download/worker/linux/arm64":
			api.downloads++
			_, _ = w.Write([]byte(api.worker + r.URL.Path))
		case "/download/plugin/script/binary/linux/amd64":
			api.pluginDownloads++
			_, _ = w.Write([]byte(api.plugin))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(api.Close)
	return api
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func sha512Hex(s string) string {
	sum := sha512.Sum512([]byte(s))
	return hex.EncodeToString(sum[:])
}

// mockPluginInfos makes the API publish the script plugin binary of
// content under the name script-linux-amd64, or fail with err
func mockPluginInfos(t *testing.T, h *HatcheryMicroVM, content string, err error) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	mockClient := mock_cdsclient.NewMockHatcheryServiceClient(ctrl)
	call := mockClient.EXPECT().PluginsGet("script").AnyTimes()
	if err != nil {
		call.Return(nil, err)
	} else {
		call.Return(&sdk.GRPCPlugin{Name: "script", Binaries: []sdk.GRPCPluginBinary{
			{OS: "linux", Arch: "arm64", Name: "script-linux-arm64", Perm: 0o755, SHA512sum: sha512Hex("arm64 " + content)},
			{OS: "linux", Arch: "amd64", Name: "script-linux-amd64", Perm: 0o755, SHA512sum: sha512Hex(content)},
		}}, nil)
	}
	h.Clientv2 = mockClient
}

func TestSpawnWorker(t *testing.T) {
	h, kube, dyn := newTestHatchery(t)
	api := newTestAPI(t)
	h.Config.API.HTTP.URL = api.URL
	h.CDNConfig.HTTPURL = "https://cds-cdn.example.org"
	h.CDNConfig.TCPURL = "cds-cdn.example.org:8090"
	mockPluginInfos(t, h, api.plugin, nil)

	// The fake dynamic client does not set the UID the API server would
	dyn.PrependReactor("create", "microvms", func(action k8stesting.Action) (bool, runtime.Object, error) {
		obj := action.(k8stesting.CreateAction).GetObject().(*unstructured.Unstructured)
		obj.SetUID(types.UID("mvm-uid"))
		return false, nil, nil
	})

	spawnArgs := hatchery.SpawnArguments{
		WorkerName:  "ubuntu-worker-1",
		WorkerToken: "worker-token",
		Model:       testModel(testSpec()),
		JobID:       sdk.UUID(),
	}
	require.NoError(t, h.SpawnWorker(context.TODO(), spawnArgs))

	mvm, err := dyn.Resource(microVMResource).Namespace("cds-workers").Get(context.TODO(), "ubuntu-worker-1", metav1.GetOptions{})
	require.NoError(t, err)

	// Labels let the hatchery find its workers back
	require.Equal(t, map[string]string{LABEL_HATCHERY_NAME: "my-hatchery", LABEL_WORKER_NAME: "ubuntu-worker-1"}, mvm.GetLabels())

	// Size and boot artifacts come from the worker model
	spec := mvm.Object["spec"].(map[string]interface{})
	require.Equal(t, int64(2), spec["vcpus"])
	require.Equal(t, int64(1024), spec["memoryMiB"])
	require.Equal(t, map[string]interface{}{"url": "https://example.org/vmlinux", "sha256": testSHA}, spec["kernel"])
	require.Equal(t, map[string]interface{}{"url": "https://example.org/rootfs.squashfs", "sha256": testSHA, "format": "squashfs", "sizeMiB": int64(4096)}, spec["rootfs"])

	// The egress policy comes from the hatchery configuration, never from the model
	policy, _, _ := unstructured.NestedString(mvm.Object, "spec", "networking", "egress", "policyRef", "name")
	require.Equal(t, "cds-workers", policy)
	admin, _, _ := unstructured.NestedBool(mvm.Object, "spec", "networking", "admin", "enabled")
	require.False(t, admin, "no SSH access to workers unless debugging is configured")
	_, found, _ := unstructured.NestedSlice(mvm.Object, "spec", "networking", "sshAuthorizedKeys")
	require.False(t, found)

	// The guest gets the worker config, the boot script and an enabled unit
	// from the secret, the worker binary and the script plugin from the
	// platform cache
	secretName := "cds-worker-ubuntu-worker-1"
	files, _, _ := unstructured.NestedSlice(mvm.Object, "spec", "guest", "files")
	require.Len(t, files, 5)
	filesByPath := map[string]map[string]interface{}{}
	for _, f := range files {
		file := f.(map[string]interface{})
		filesByPath[file["path"].(string)] = file
	}
	for _, p := range []string{guestPathConfig, guestPathScript, guestPathUnit} {
		require.Equal(t, secretName, filesByPath[p]["secretKeyRef"].(map[string]interface{})["name"])
	}
	require.Equal(t, "0600", filesByPath[guestPathConfig]["mode"], "the worker config holds the worker token")
	require.Equal(t, "0755", filesByPath[guestPathScript]["mode"])
	require.Equal(t, "0644", filesByPath[guestPathUnit]["mode"])
	// The platform checks the binary it downloads against the hash of the
	// one the API serves for the model os/arch
	require.Equal(t, map[string]interface{}{
		"path": "/usr/local/bin/worker",
		"mode": "0755",
		"source": map[string]interface{}{
			"url":    api.URL + "/download/worker/linux/amd64",
			"sha256": sha256Hex("worker binary/download/worker/linux/amd64"),
		},
	}, filesByPath[guestPathWorker])
	// The worker uses a plugin found in its basedir under the published name
	// instead of downloading it at the first step of the job
	require.Equal(t, map[string]interface{}{
		"path": "/var/lib/cds-worker/script-linux-amd64",
		"mode": "0755",
		"source": map[string]interface{}{
			"url":    api.URL + "/download/plugin/script/binary/linux/amd64",
			"sha256": sha256Hex("script plugin"),
		},
	}, filesByPath["/var/lib/cds-worker/script-linux-amd64"])
	bootArgs, _, _ := unstructured.NestedString(mvm.Object, "spec", "bootArgs")
	require.Equal(t, h.bootArgs(), bootArgs)
	units, _, _ := unstructured.NestedStringSlice(mvm.Object, "spec", "guest", "systemdUnits")
	require.Equal(t, []string{guestUnit}, units)

	secret, err := kube.CoreV1().Secrets("cds-workers").Get(context.TODO(), secretName, metav1.GetOptions{})
	require.NoError(t, err)
	// Same labels as the MicroVM, plus the label without which the platform
	// refuses to write the guest files
	require.Equal(t, map[string]string{
		LABEL_HATCHERY_NAME:       "my-hatchery",
		LABEL_WORKER_NAME:         "ubuntu-worker-1",
		"microvm.poc/guest-files": "true",
	}, secret.Labels)

	// The secret is garbage-collected with its MicroVM
	require.Equal(t, []metav1.OwnerReference{{APIVersion: microVMAPIVersion, Kind: microVMKind, Name: "ubuntu-worker-1", UID: "mvm-uid"}}, secret.OwnerReferences)

	for _, p := range []string{guestPathConfig, guestPathScript, guestPathUnit} {
		key := filesByPath[p]["secretKeyRef"].(map[string]interface{})["key"].(string)
		require.Contains(t, secret.StringData, key)
	}

	// The worker config is what the worker reads from CDS_CONFIG
	raw, err := base64.StdEncoding.DecodeString(secret.StringData[secretKeyConfig])
	require.NoError(t, err)
	var workerConfig workerruntime.WorkerConfig
	require.NoError(t, json.Unmarshal(raw, &workerConfig))
	require.Equal(t, "ubuntu-worker-1", workerConfig.Name)
	require.Equal(t, "worker-token", workerConfig.APIToken)
	require.Equal(t, spawnArgs.JobID, workerConfig.RunJobID)
	// Not /tmp, emptied at boot with the preloaded plugins
	require.Equal(t, "/var/lib/cds-worker", workerConfig.Basedir)

	// The boot script downloads the worker for the model os/arch when the
	// platform did not provide it, runs it and ends the VM
	script := secret.StringData[secretKeyScript]
	require.Contains(t, script, "if [ ! -x /usr/local/bin/worker ]; then\n  if ! curl ")
	require.Contains(t, script, `"`+api.URL+`/download/worker/linux/amd64"`)
	require.Contains(t, script, "--retry-delay 1 ")
	require.Contains(t, script, `export CDS_CONFIG="$(cat /etc/cds/worker-config)"`)
	require.Contains(t, script, "/usr/local/bin/worker\nsystemctl reboot")
	require.NotContains(t, script, " -k ", "TLS is verified unless the API is configured insecure")
	require.Contains(t, secret.StringData[secretKeyUnit], "ExecStart="+guestPathScript)
}

// Without a configured policy the MicroVM gets the platform default policy
func TestBuildMicroVMWithoutPolicy(t *testing.T) {
	h, _, _ := newTestHatchery(t)
	h.Config.EgressPolicy = ""
	mvm := h.buildMicroVM(hatchery.SpawnArguments{WorkerName: "w", Model: testModel(testSpec())}, "s", nil, nil)
	_, found, _ := unstructured.NestedMap(mvm.Object, "spec", "networking", "egress", "policyRef")
	require.False(t, found)
	enabled, _, _ := unstructured.NestedBool(mvm.Object, "spec", "networking", "egress", "enabled")
	require.True(t, enabled)
}

// Debug SSH keys open the admin network and authorize the keys; the egress
// stays as configured
func TestBuildMicroVMWithDebugSSH(t *testing.T) {
	h, _, _ := newTestHatchery(t)
	h.Config.DebugSSHAuthorizedKeys = []string{"ssh-ed25519 AAAA debug@example"}
	mvm := h.buildMicroVM(hatchery.SpawnArguments{WorkerName: "w", Model: testModel(testSpec())}, "s", nil, nil)
	admin, _, _ := unstructured.NestedBool(mvm.Object, "spec", "networking", "admin", "enabled")
	require.True(t, admin)
	keys, _, _ := unstructured.NestedStringSlice(mvm.Object, "spec", "networking", "sshAuthorizedKeys")
	require.Equal(t, []string{"ssh-ed25519 AAAA debug@example"}, keys)
	policy, _, _ := unstructured.NestedString(mvm.Object, "spec", "networking", "egress", "policyRef", "name")
	require.Equal(t, "cds-workers", policy)
	// The SSH server is not masked, or the keys would be useless
	bootArgs, _, _ := unstructured.NestedString(mvm.Object, "spec", "bootArgs")
	require.NotContains(t, bootArgs, "ssh")
}

// The worker unit must not wait for the whole boot: only for the platform
// network unit, and it must still be stopped at shutdown for the reboot
// that ends the VM
func TestWorkerUnit(t *testing.T) {
	require.Contains(t, workerUnit, "\nDefaultDependencies=no\n")
	require.Contains(t, workerUnit, "\nWants=microvm-net.service\n")
	require.Contains(t, workerUnit, "\nAfter=microvm-net.service local-fs.target\n")
	require.Contains(t, workerUnit, "\nConflicts=shutdown.target\n")
	require.Contains(t, workerUnit, "\nBefore=shutdown.target\n")
	require.NotContains(t, workerUnit, "network-online.target")
}

func TestBootArgs(t *testing.T) {
	h, _, _ := newTestHatchery(t)
	// The platform default is kept: the serial console of the worker output,
	// and the reboot that ends the VM
	args := h.bootArgs()
	require.True(t, strings.HasPrefix(args, "console=ttyS0 reboot=k panic=1 pci=off quiet "), args)
	for _, unit := range []string{"serial-getty@ttyS0.service", "apt-daily.timer", "unattended-upgrades.service", "ssh.service", "ssh.socket"} {
		require.Contains(t, strings.Fields(args), "systemd.mask="+unit)
	}
	// Never mask what the platform needs
	for _, unit := range []string{"microvm-net", "journald", "udev"} {
		require.NotContains(t, args, unit)
	}

	h.Config.ExtraBootArgs = "quiet systemd.mask=foo.service"
	require.Equal(t, "console=ttyS0 reboot=k panic=1 pci=off quiet systemd.mask=foo.service systemd.mask=ssh.service systemd.mask=ssh.socket", h.bootArgs())

	// Left out of the configuration file: the default applies
	cfg := HatcheryConfiguration{}
	cfg.applyDefaults()
	require.Equal(t, defaultExtraBootArgs, cfg.ExtraBootArgs)
}

// The worker binary is downloaded once per API build and os/arch, and again
// when the API is upgraded
func TestWorkerSHA256CachedPerBuild(t *testing.T) {
	h, _, _ := newTestHatchery(t)
	api := newTestAPI(t)
	ctx := context.TODO()

	sum, err := h.workerSHA256(ctx, api.URL, false, "linux/amd64")
	require.NoError(t, err)
	require.Equal(t, sha256Hex("worker binary/download/worker/linux/amd64"), sum)
	sum, err = h.workerSHA256(ctx, api.URL, false, "linux/amd64")
	require.NoError(t, err)
	require.Equal(t, sha256Hex("worker binary/download/worker/linux/amd64"), sum)
	require.Equal(t, 1, api.downloads)

	sum, err = h.workerSHA256(ctx, api.URL, false, "linux/arm64")
	require.NoError(t, err)
	require.Equal(t, sha256Hex("worker binary/download/worker/linux/arm64"), sum)
	require.Equal(t, 2, api.downloads)

	api.version = "1.0.1"
	api.worker = "new worker binary"
	sum, err = h.workerSHA256(ctx, api.URL, false, "linux/amd64")
	require.NoError(t, err)
	require.Equal(t, sha256Hex("new worker binary/download/worker/linux/amd64"), sum)
	require.Equal(t, 3, api.downloads)
}

// Without the hashes the MicroVM is spawned as before: the boot script
// downloads the worker, the worker downloads its plugins
func TestSpawnWorkerWithoutWorkerHash(t *testing.T) {
	h, kube, dyn := newTestHatchery(t)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(api.Close)
	h.Config.API.HTTP.URL = api.URL
	mockPluginInfos(t, h, "script plugin", nil)

	require.NoError(t, h.SpawnWorker(context.TODO(), hatchery.SpawnArguments{WorkerName: "w1", Model: testModel(testSpec()), JobID: sdk.UUID()}))

	mvm, err := dyn.Resource(microVMResource).Namespace("cds-workers").Get(context.TODO(), "w1", metav1.GetOptions{})
	require.NoError(t, err)
	files, _, _ := unstructured.NestedSlice(mvm.Object, "spec", "guest", "files")
	require.Len(t, files, 3)
	for _, f := range files {
		require.NotEqual(t, guestPathWorker, f.(map[string]interface{})["path"])
	}
	secret, err := kube.CoreV1().Secrets("cds-workers").Get(context.TODO(), "cds-worker-w1", metav1.GetOptions{})
	require.NoError(t, err)
	require.Contains(t, secret.StringData[secretKeyScript], `"`+api.URL+`/download/worker/linux/amd64"`)
}

// A plugin is downloaded by the hatchery once per published binary, and
// again when the API publishes another one
func TestPluginSHA256(t *testing.T) {
	h, _, _ := newTestHatchery(t)
	api := newTestAPI(t)
	ctx := context.TODO()

	for i := 0; i < 2; i++ {
		sum, err := h.pluginSHA256(ctx, api.URL, false, "script", "linux/amd64", sha512Hex("script plugin"))
		require.NoError(t, err)
		require.Equal(t, sha256Hex("script plugin"), sum)
	}
	require.Equal(t, 1, api.pluginDownloads)

	api.plugin = "new script plugin"
	sum, err := h.pluginSHA256(ctx, api.URL, false, "script", "linux/amd64", sha512Hex("new script plugin"))
	require.NoError(t, err)
	require.Equal(t, sha256Hex("new script plugin"), sum)
	require.Equal(t, 2, api.pluginDownloads)
}

// A plugin that cannot be checked against its published sha512 is left out:
// preloading a binary the worker would reject only delays the job
func TestSpawnWorkerWithoutPreloadedPlugin(t *testing.T) {
	for name, infosErr := range map[string]error{
		"descriptor unavailable": sdk.NewErrorFrom(sdk.ErrNotFound, "no plugin"),
		"binary replaced":        nil,
	} {
		t.Run(name, func(t *testing.T) {
			h, _, dyn := newTestHatchery(t)
			api := newTestAPI(t)
			h.Config.API.HTTP.URL = api.URL
			// The descriptor publishes another content than the one served
			mockPluginInfos(t, h, "older script plugin", infosErr)

			require.NoError(t, h.SpawnWorker(context.TODO(), hatchery.SpawnArguments{WorkerName: "w1", Model: testModel(testSpec()), JobID: sdk.UUID()}))

			mvm, err := dyn.Resource(microVMResource).Namespace("cds-workers").Get(context.TODO(), "w1", metav1.GetOptions{})
			require.NoError(t, err)
			files, _, _ := unstructured.NestedSlice(mvm.Object, "spec", "guest", "files")
			require.Len(t, files, 4)
			for _, f := range files {
				require.NotContains(t, f.(map[string]interface{})["path"], "script")
			}
			require.Empty(t, h.pluginBinaries.sha256s, "a rejected binary is not cached")
		})
	}
}

// A MicroVM that cannot be created must not leave its secret behind
func TestSpawnWorkerDeletesSecretWhenMicroVMCreationFails(t *testing.T) {
	h, kube, dyn := newTestHatchery(t)
	h.Config.API.HTTP.URL = newTestAPI(t).URL
	mockPluginInfos(t, h, "script plugin", nil)
	dyn.PrependReactor("create", "microvms", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, sdk.NewErrorFrom(sdk.ErrForbidden, "denied")
	})
	err := h.SpawnWorker(context.TODO(), hatchery.SpawnArguments{WorkerName: "w1", Model: testModel(testSpec()), JobID: sdk.UUID()})
	require.Error(t, err)
	secrets, err := kube.CoreV1().Secrets("cds-workers").List(context.TODO(), metav1.ListOptions{})
	require.NoError(t, err)
	require.Empty(t, secrets.Items)
}

func TestWorkersStarted(t *testing.T) {
	now := time.Now()
	h, _, _ := newTestHatchery(t,
		microVM("w-running", "my-hatchery", now, map[string]interface{}{"phase": "Running"}),
		microVM("w-new", "my-hatchery", now, nil),
		microVM("w-quota", "my-hatchery", now, quotaExceededStatus()),
		microVM("w-succeeded", "my-hatchery", now, map[string]interface{}{"phase": "Succeeded"}),
		microVM("w-failed", "my-hatchery", now, map[string]interface{}{"phase": "Failed"}),
		microVM("w-other", "other-hatchery", now, map[string]interface{}{"phase": "Running"}),
	)
	names, err := h.WorkersStarted(context.TODO())
	require.NoError(t, err)
	// Waiting for the quota counts in the capacity; ended MicroVMs and other hatcheries' do not
	require.ElementsMatch(t, []string{"w-running", "w-new", "w-quota"}, names)
}

func TestKillAwolWorkers(t *testing.T) {
	now := time.Now()
	old := now.Add(-time.Hour)
	h, _, dyn := newTestHatchery(t,
		// ended: always deleted, the VM is gone
		microVM("w-succeeded", "my-hatchery", now, map[string]interface{}{"phase": "Succeeded", "reason": "GuestExited"}),
		microVM("w-failed", "my-hatchery", now, map[string]interface{}{"phase": "Failed", "reason": "VMFailed"}),
		// running with a registered worker: kept whatever its age
		microVM("w-registered", "my-hatchery", old, map[string]interface{}{"phase": "Running"}),
		// orphan: no worker registered long after the boot
		microVM("w-orphan", "my-hatchery", old, map[string]interface{}{"phase": "Running", "startTime": old.UTC().Format(time.RFC3339)}),
		// created long ago but booted recently (slow scheduling): still has time to register
		microVM("w-booting", "my-hatchery", old, map[string]interface{}{"phase": "Running", "startTime": now.UTC().Format(time.RFC3339)}),
		// recent, not booted yet
		microVM("w-young", "my-hatchery", now, map[string]interface{}{"phase": "Scheduled"}),
		// waiting for the quota: not spawned yet, kept beyond the registration timeout
		microVM("w-quota", "my-hatchery", now.Add(-20*time.Minute), quotaExceededStatus()),
		// waiting for the quota for too long: given up
		microVM("w-quota-old", "my-hatchery", old, quotaExceededStatus()),
		// not ours
		microVM("w-other", "other-hatchery", old, map[string]interface{}{"phase": "Failed"}),
	)

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	mockClient := mock_cdsclient.NewMockHatcheryServiceClient(ctrl)
	mockClient.EXPECT().V2WorkerList(gomock.Any()).Return([]sdk.V2Worker{{Name: "w-registered"}}, nil)
	h.Clientv2 = mockClient

	require.NoError(t, h.killAwolWorkers(context.TODO()))

	list, err := dyn.Resource(microVMResource).Namespace("cds-workers").List(context.TODO(), metav1.ListOptions{})
	require.NoError(t, err)
	var remaining []string
	for _, i := range list.Items {
		remaining = append(remaining, i.GetName())
	}
	require.ElementsMatch(t, []string{"w-registered", "w-booting", "w-young", "w-quota", "w-other"}, remaining)
}

func TestDeleteOrphanSecrets(t *testing.T) {
	now := time.Now()
	h, kube, _ := newTestHatchery(t, microVM("w-alive", "my-hatchery", now, map[string]interface{}{"phase": "Running"}))
	secret := func(name, worker string, created time.Time) *corev1.Secret {
		return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
			Name: name, Namespace: "cds-workers", CreationTimestamp: metav1.NewTime(created),
			Labels: map[string]string{LABEL_HATCHERY_NAME: "my-hatchery", LABEL_WORKER_NAME: worker},
		}}
	}
	for _, s := range []*corev1.Secret{
		secret("cds-worker-w-alive", "w-alive", now.Add(-time.Hour)),
		secret("cds-worker-w-gone", "w-gone", now.Add(-time.Hour)),
		// just created, its MicroVM is being created
		secret("cds-worker-w-spawning", "w-spawning", now),
	} {
		_, err := kube.CoreV1().Secrets("cds-workers").Create(context.TODO(), s, metav1.CreateOptions{})
		require.NoError(t, err)
	}

	require.NoError(t, h.deleteOrphanSecrets(context.TODO()))

	list, err := kube.CoreV1().Secrets("cds-workers").List(context.TODO(), metav1.ListOptions{})
	require.NoError(t, err)
	var remaining []string
	for _, s := range list.Items {
		remaining = append(remaining, s.Name)
	}
	require.ElementsMatch(t, []string{"cds-worker-w-alive", "cds-worker-w-spawning"}, remaining)
}

func TestCheckConfiguration(t *testing.T) {
	h := New()
	cfg := HatcheryConfiguration{Namespace: "cds-workers"}
	cfg.Name = "my-hatchery"
	cfg.API.HTTP.URL = "https://cds-api.example.org"
	cfg.API.Token = "v1-token"
	err := h.CheckConfiguration(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "tokenV2 is mandatory")

	cfg.API.TokenV2 = "v2-token"
	require.NoError(t, h.CheckConfiguration(cfg))
}
