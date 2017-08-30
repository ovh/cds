package vsphere

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/ovh/cds/sdk"
	"github.com/spf13/viper"
	"github.com/vmware/govmomi/guest"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

// SpawnWorker creates a new cloud instances
func (h *HatcheryVSphere) SpawnWorker(model *sdk.Model, job *sdk.PipelineBuildJob, registerOnly bool, logInfo string) (string, error) {
	// To know if i do a linked clone or not, check if vm already exist with this model
	ctx := context.TODO()
	name := model.Name + "-" + strings.Replace(namesgenerator.GetRandomName(0), "_", "-", -1)
	if registerOnly {
		name = "register-" + name
	}

	vm, errV := h.finder.VirtualMachine(ctx, model.Name)
	if errV != nil {
		return "", errV
	}

	devices, errD := vm.Device(ctx)
	if errD != nil {
		return "", errD
	}

	var card *types.VirtualEthernetCard
	for _, device := range devices {
		if c, ok := device.(types.BaseVirtualEthernetCard); ok {
			card = c.GetVirtualEthernetCard()
			break
		}
	}

	if card == nil {
		return "", fmt.Errorf("no network device found.")
	}

	backing, errB := h.network.EthernetCardBackingInfo(ctx)
	if errB != nil {
		return "", errB
	}

	device, errE := object.EthernetCardTypes().CreateEthernetCard("e1000", backing)
	if errE != nil {
		return "", errE
	}
	//set backing info
	card.Backing = device.(types.BaseVirtualEthernetCard).GetVirtualEthernetCard().Backing

	// prepare virtual device config spec for network card
	configSpecs := []types.BaseVirtualDeviceConfigSpec{
		&types.VirtualDeviceConfigSpec{
			Operation: types.VirtualDeviceConfigSpecOperationEdit,
			Device:    card,
		},
	}

	folder, errF := h.finder.FolderOrDefault(ctx, "")
	if errF != nil {
		return "", errF
	}
	relocateSpec := types.VirtualMachineRelocateSpec{
		DeviceChange: configSpecs,
		DiskMoveType: string(types.VirtualMachineRelocateDiskMoveOptionsMoveChildMostDiskBacking),
	}

	datastore, errD := h.finder.DatastoreOrDefault(ctx, h.datastoreString)
	if errD != nil {
		return "", errD
	}
	datastoreref := datastore.Reference()

	afterPO := true
	cloneSpec := &types.VirtualMachineCloneSpec{
		Location: relocateSpec,
		PowerOn:  true,
		Template: false,
		Config: &types.VirtualMachineConfigSpec{
			Annotation: time.Now().String(),
			Tools: &types.ToolsConfigInfo{
				AfterPowerOn: &afterPO,
			},
		},
	}

	// Set the destination datastore
	cloneSpec.Location.Datastore = &datastoreref

	task, errC := vm.Clone(ctx, folder, name, *cloneSpec)
	if errC != nil {
		return "", errC
	}

	info, errW := task.WaitForResult(ctx, nil)
	if errW != nil || info.State == types.TaskInfoStateError {
		return "", errW
	}

	return "", h.reconfigureVM(name, model.ID, job.ID, info.Result.(types.ManagedObjectReference))
}

func (h *HatcheryVSphere) reconfigureVM(name string, modelID, jobID int64, vmInfo types.ManagedObjectReference) error {
	ctx := context.TODO()
	// Retrieve the new VM
	vm := object.NewVirtualMachine(h.client.Client, vmInfo)

	if _, errW := vm.WaitForIP(ctx); errW != nil {
		return errW
	}

	running, errT := vm.IsToolsRunning(ctx)
	if errT != nil {
		return errT
	}
	fmt.Printf("Running %v\n", running)
	opman := guest.NewOperationsManager(h.client.Client, vm.Reference())

	auth := types.NamePasswordAuthentication{
		Username: "root",
		Password: "",
	}

	procman, errPr := opman.ProcessManager(ctx)
	if errPr != nil {
		return errPr
	}
	scr := `
cd $HOME
# Download and start worker with curl
rm -f worker
curl  "{{.API}}/download/worker/$(uname -m)" -o worker --retry 10 --retry-max-time 120 -C - >> /tmp/user_data 2>&1
chmod +x worker
export CDS_SINGLE_USE=1
export CDS_FORCE_EXIT=1
export CDS_API={{.API}}
export CDS_TOKEN={{.Key}}
export CDS_NAME={{.Name}}
export CDS_MODEL={{.Model}}
export CDS_HATCHERY={{.Hatchery}}
export CDS_HATCHERY_NAME={{.HatcheryName}}
export CDS_BOOKED_JOB_ID={{.JobID}}
export CDS_TTL={{.TTL}}
./worker >> /tmp/user_data 2>&1`

	script, errS := h.tmplGuestScript(name, modelID, jobID, scr)
	if errS != nil {
		return errS
	}

	guestspec := types.GuestProgramSpec{
		ProgramPath:      "/usr/bin",
		Arguments:        script,
		WorkingDirectory: "/root",
	}

	if _, errS := procman.StartProgram(ctx, &auth, &guestspec); errS != nil {
		return errS
	}

	return nil
}

func (h *HatcheryVSphere) tmplGuestScript(name string, modelID, jobID int64, scr string) (string, error) {
	graylog := ""
	if viper.GetString("worker_graylog_host") != "" {
		graylog += fmt.Sprintf("export CDS_GRAYLOG_HOST=%s ", viper.GetString("worker_graylog_host"))
	}
	if viper.GetString("worker_graylog_port") != "" {
		graylog += fmt.Sprintf("export CDS_GRAYLOG_PORT=%s ", viper.GetString("worker_graylog_port"))
	}
	if viper.GetString("worker_graylog_extra_key") != "" {
		graylog += fmt.Sprintf("export CDS_GRAYLOG_EXTRA_KEY=%s ", viper.GetString("worker_graylog_extra_key"))
	}
	if viper.GetString("worker_graylog_extra_value") != "" {
		graylog += fmt.Sprintf("export CDS_GRAYLOG_EXTRA_VALUE=%s ", viper.GetString("worker_graylog_extra_value"))
	}

	// grpc := ""
	// if viper.GetString("grpc_api") != "" && model.Communication == sdk.GRPC {
	// 	grpc += fmt.Sprintf("export CDS_GRPC_API=%s ", viper.GetString("grpc_api"))
	// 	grpc += fmt.Sprintf("export CDS_GRPC_INSECURE=%t ", viper.GetBool("grpc_insecure"))
	// }

	tmpl, errt := template.New("udata").Parse(scr)
	if errt != nil {
		return "", errt
	}

	udataParam := struct {
		API          string
		Name         string
		Key          string
		Model        int64
		Hatchery     int64
		HatcheryName string
		JobID        int64
		TTL          int
		Graylog      string
		// Grpc         string
	}{
		API:          viper.GetString("api"),
		Name:         name,
		Key:          viper.GetString("token"),
		Model:        modelID,
		Hatchery:     h.hatch.ID,
		HatcheryName: h.hatch.Name,
		JobID:        jobID,
		TTL:          h.workerTTL,
		Graylog:      graylog,
		// Grpc:         grpc,
	}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, udataParam); err != nil {
		return "", err
	}

	return buffer.String(), nil
}
