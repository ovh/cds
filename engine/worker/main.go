package main

import (
	"container/list"
	"os"

	"google.golang.org/grpc"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/doc"
)

type currentWorker struct {
	apiEndpoint   string
	token         string
	id            string
	model         sdk.Model
	groupID       int64
	bookedPBJobID int64
	bookedWJobID  int64
	nbActionsDone int
	basedir       string
	manualExit    bool
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
		params         []sdk.Parameter
	}
	status struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	client cdsclient.Interface
}

func main() {
	sdk.SetAgent(sdk.WorkerAgent)

	w := &currentWorker{}
	cmd := cmdMain(w)
	cmd.AddCommand(cmdExport)
	cmd.AddCommand(cmdUpload(w))
	cmd.AddCommand(cmdTmpl(w))
	cmd.AddCommand(cmdTag(w))
	cmd.AddCommand(cmdRun(w))
	cmd.AddCommand(cmdUpdate(w))
	cmd.AddCommand(cmdExit(w))
	cmd.AddCommand(cmdVersion)
	cmd.AddCommand(cmdRegister(w))

	// hidden command, generateDocumentation, only used to generate hugo documentation
	// run with ./engine generateDocumentation
	osArgs := os.Args[1:]
	if len(osArgs) == 1 && osArgs[0] == "generateDocumentation" {
		if err := doc.GenerateDocumentation(cmd); err != nil {
			sdk.Exit(err.Error())
		}
		os.Exit(0)
	}

	cmd.Execute()
}
