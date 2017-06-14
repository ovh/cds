package main

import (
	"container/list"
	"time"

	"google.golang.org/grpc"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

type currentWorker struct {
	alive         bool
	apiEndpoint   string
	token         string
	id            string
	modelID       int64
	bookedJobID   int64
	nbActionsDone int
	basedir       string
	logger        struct {
		logChan chan sdk.Log
		llist   *list.List
	}
	exportPort int
	hatchery   struct {
		id   int64
		name string
	}
	grpc struct {
		address string
		conn    *grpc.ClientConn
	}
	currentJob struct {
		pbJob          sdk.PipelineBuildJob
		wJob           *sdk.WorkflowNodeJobRun
		currentStep    int
		buildVariables []sdk.Variable
		pkey           string
		gitsshPath     string
	}
	status struct {
		Name      string    `json:"name"`
		Heartbeat time.Time `json:"heartbeat"`
		Status    string    `json:"status"`
		Model     int64     `json:"model"`
	}
	client cdsclient.Interface
}

var (
	//VERSION is set with -ldflags "-X main.VERSION={{.cds.proj.version}}+{{.cds.version}}"
	VERSION = "snapshot"
)

func main() {
	sdk.SetAgent(sdk.WorkerAgent)

	w := &currentWorker{}
	cmd := cmdMain(w)
	cmd.AddCommand(cmdExport)
	cmd.AddCommand(cmdUpload(w))
	cmd.AddCommand(cmdVersion)
	cmd.AddCommand(cmdRegister(w))
	cmd.Execute()
}
