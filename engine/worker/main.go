package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

var (
	// WorkerID is a unique identifier for this worker
	WorkerID string
	// key is the token generated by the user owning the worker
	key      string
	name     string
	api      string
	model    int64
	hatchery int64
	basedir  string
	logChan  chan sdk.Log
	// port of variable exporter HTTP server
	exportport int
	// current actionBuild is here to allow var export
	ab             sdk.ActionBuild
	buildVariables []sdk.Variable
	// Git ssh configuration
	pkey           string
	gitssh         string
	startTimestamp *time.Time
	nbActionsDone  int
	status         struct {
		Name      string    `json:"name"`
		Heartbeat time.Time `json:"heartbeat"`
		Status    string    `json:"status"`
		Model     int64     `json:"model"`
	}
)

var mainCmd = &cobra.Command{
	Use:   "worker",
	Short: "CDS Worker",
	Run: func(cmd *cobra.Command, args []string) {
		viper.SetEnvPrefix("cds")
		viper.AutomaticEnv()

		log.Initialize()

		log.Notice("What a good time to be alive\n")
		var err error

		name, err = os.Hostname()
		if err != nil {
			log.Notice("Cannot retrieve hostname: %s\n", err)
			return
		}

		hatchS := viper.GetString("hatchery")
		hatchery, err = strconv.ParseInt(hatchS, 10, 64)
		if err != nil {
			fmt.Printf("WARNING: Invalid hatchery ID (%s)\n", err)
		}

		api = viper.GetString("api")
		if api == "" {
			fmt.Printf("--api not provided, aborting.\n")
			return
		}

		key = viper.GetString("key")
		if key == "" {
			fmt.Printf("--key not provided, aborting.\n")
			return
		}

		givenName := viper.GetString("name")
		if givenName != "" {
			name = givenName
		}
		status.Name = name

		basedir = viper.GetString("basedir")
		if basedir == "" {
			basedir = os.TempDir()
		}

		model = int64(viper.GetInt("model"))
		status.Model = model

		port, err := server()
		if err != nil {
			sdk.Exit("cannot bind port for worker export: %s\n", err)
		}
		exportport = port

		now := time.Now()
		startTimestamp = &now

		// start logger routine
		logChan = make(chan sdk.Log)
		go logger(logChan)

		go heartbeat()
		queuePolling()
	},
}

func init() {
	flags := mainCmd.Flags()

	flags.String("log-level", "notice", "Log Level : debug, info, notice, warning, critical")
	viper.BindPFlag("log_level", flags.Lookup("log-level"))

	flags.String("api", "", "URL of CDS API")
	viper.BindPFlag("api", flags.Lookup("api"))

	flags.String("key", "", "CDS KEY")
	viper.BindPFlag("key", flags.Lookup("key"))

	flags.Bool("single-use", false, "Exit after executing an action")
	viper.BindPFlag("single_use", flags.Lookup("single-use"))

	flags.String("name", "", "Name of worker")
	viper.BindPFlag("name", flags.Lookup("name"))

	flags.Int("model", 0, "Model of worker")
	viper.BindPFlag("model", flags.Lookup("model"))

	flags.Int("hatchery", 0, "Hatchery spawing worker")
	viper.BindPFlag("hatchery", flags.Lookup("hatchery"))

	flags.String("basedir", "", "Worker working directory")
	viper.BindPFlag("basedir", flags.Lookup("basedir"))

	flags.Int("ttl", 30, "Worker time to live (minutes)")
	viper.BindPFlag("ttl", flags.Lookup("ttl"))

	mainCmd.AddCommand(cmdExport)
	mainCmd.AddCommand(cmdUpload)
}

func main() {
	sdk.SetAgent(sdk.WorkerAgent)

	mainCmd.Execute()
}

// Will be removed when websocket conn is implemented
// for now, poll the /queue
func queuePolling() {
	for {
		if WorkerID == "" {
			log.Notice("[WORKER] Disconnected from CDS engine, trying to register...\n")
			if err := register(api, name, key); err != nil {
				log.Notice("Cannot register: %s\n", err)
				time.Sleep(10 * time.Second)
				continue
			}
		}

		//We we've done nothing until ttl is over, let's exit
		if nbActionsDone == 0 && startTimestamp.Add(time.Duration(viper.GetInt("ttl"))*time.Minute).Before(time.Now()) {
			log.Notice("Time to exit.")
			unregister()
			os.Exit(0)
		}

		checkQueue()
		time.Sleep(5 * time.Second)
	}
}

func checkQueue() {
	//Set the status to checking to avoid beeing killed while checking queue, actions and requirements
	sdk.SetWorkerStatus(sdk.StatusChecking)
	defer sdk.SetWorkerStatus(sdk.StatusWaiting)

	queue, err := sdk.GetBuildQueue()
	if err != nil {
		log.Notice("checkQueue> Cannot get build queue: %s\n", err)
		time.Sleep(5 * time.Second)
		WorkerID = ""
		return
	}

	log.Notice("checkQueue> %d Actions in queue", len(queue))

	for i := range queue {
		requirementsOK := true
		// Check requirement
		log.Notice("checkQueue> Checking requirements for action [%d] %s", queue[i].ID, queue[i].ActionName)
		for _, r := range queue[i].Requirements {
			ok, err := checkRequirement(r)
			if err != nil {
				postCheckRequirementError(&r, err)
				requirementsOK = false
				continue
			}
			if !ok {
				requirementsOK = false
				continue
			}
		}

		if requirementsOK {
			log.Notice("checkQueue> Taking action %s", queue[i].ID)
			takeAction(queue[i])
		}
	}

	log.Notice("checkQueue> Nothing to do...")
}

func postCheckRequirementError(r *sdk.Requirement, err error) {
	s := fmt.Sprintf("Error checking requirement Name=%s Type=%s Value=%s :%s", r.Name, r.Type, r.Value, err)
	btes := []byte(s)
	sdk.Request("POST", "/queue/requirements/errors", btes)
}

func takeAction(b sdk.ActionBuild) {
	nbActionsDone++

	gitssh = ""
	pkey = ""
	path := fmt.Sprintf("/queue/%d/take", b.ID)
	data, code, err := sdk.Request("POST", path, nil)
	if err != nil {
		log.Notice("takeAction> Cannot take action %d:%s\n", b.PipelineActionID, err)
		return
	}
	if code != http.StatusOK {
		return
	}

	abi := worker.ActionBuildInfo{}
	err = json.Unmarshal([]byte(data), &abi)
	if err != nil {
		log.Notice("takeAction> Cannot unmarshal action: %s\n", err)
		return
	}

	// Reset build variables
	ab = abi.ActionBuild
	buildVariables = nil
	res := run(abi.Action, abi.ActionBuild, abi.Secrets)
	// Give time to buffered logs to be sent
	time.Sleep(3 * time.Second)

	path = fmt.Sprintf("/queue/%d/result", b.ID)
	body, err := json.MarshalIndent(res, " ", " ")
	if err != nil {
		log.Notice("takeAction>Cannot marshal result: %s\n", err)
		return
	}

	code = 300
	var isThereAnyHopeLeft = 50
	for code >= 300 {
		_, code, err = sdk.Request("POST", path, body)
		if err == nil && code == http.StatusNotFound {
			unregister() // well...
			log.Notice("takeAction> Cannot send build result: ActionBuild does not exists anymore\n")
			break
		}
		if err == nil && code < 300 {
			sendLog(b.ID, "SYSTEM", "BuildResult sent.\n")
			fmt.Printf("BuildResult sent.\n")
			break
		}

		if err != nil {
			log.Notice("takeAction> Cannot send build result: %s\n", err)
		} else {
			log.Notice("takeAction> Cannot send build result: HTTP %d\n", code)
		}

		time.Sleep(5 * time.Second)
		isThereAnyHopeLeft--
		if isThereAnyHopeLeft < 0 {
			log.Notice("takeAction> Could not send built result 50 times, giving up\n")
			break
		}
	}

	if viper.GetBool("single_use") {
		// Give time to logs to be flushed
		time.Sleep(2 * time.Second)
		// Unregister from engine
		err := unregister()
		if err != nil {
			log.Warning("takeAction> could not unregister: %s\n", err)
		}
		// then exit
		log.Notice("takeAction> --single_use is on, exiting\n")
		os.Exit(0)
	}

}

func heartbeat() {
	for {
		time.Sleep(10 * time.Second)
		if WorkerID == "" {
			log.Notice("[WORKER] Disconnected from CDS engine, trying to register...\n")
			if err := register(api, name, key); err != nil {
				log.Notice("Cannot register: %s\n", err)
				continue
			}
		}

		_, code, err := sdk.Request("POST", "/worker/refresh", nil)
		if err != nil || code >= 300 {
			log.Notice("heartbeat> cannot refresh beat: %d %s\n", code, err)
			WorkerID = ""
		}
	}
}

func unregister() error {
	uri := "/worker/unregister"
	_, code, err := sdk.Request("POST", uri, nil)
	if err != nil {
		return err
	}
	if code > 300 {
		return fmt.Errorf("HTTP %d", code)
	}

	return nil
}
