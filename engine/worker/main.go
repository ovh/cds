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
	//VERSION is set with -ldflags "-X main.VERSION={{.cds.proj.version}}+{{.cds.version}}"
	VERSION = "snapshot"
	// WorkerID is a unique identifier for this worker
	WorkerID string
	// key is the token generated by the user owning the worker
	key         string
	name        string
	api         string
	model       int64
	hatchery    int64
	basedir     string
	bookedJobID int64
	logChan     chan sdk.Log
	// port of variable exporter HTTP server
	exportport int
	// current actionBuild is here to allow var export
	pbJob          sdk.PipelineBuildJob
	currentStep    int
	buildVariables []sdk.Variable
	// Git ssh configuration
	pkey           string
	gitsshPath     string
	startTimestamp time.Time
	nbActionsDone  int
	status         struct {
		Name      string    `json:"name"`
		Heartbeat time.Time `json:"heartbeat"`
		Status    string    `json:"status"`
		Model     int64     `json:"model"`
	}
	alive bool
)

var mainCmd = &cobra.Command{
	Use:   "worker",
	Short: "CDS Worker",
	Run: func(cmd *cobra.Command, args []string) {
		viper.SetEnvPrefix("cds")
		viper.AutomaticEnv()

		log.Initialize()

		log.Notice("What a good time to be alive")
		var err error

		name, err = os.Hostname()
		if err != nil {
			log.Notice("Cannot retrieve hostname: %s", err)
			return
		}

		hatchS := viper.GetString("hatchery")
		hatchery, err = strconv.ParseInt(hatchS, 10, 64)
		if err != nil {
			fmt.Printf("WARNING: Invalid hatchery ID (%s)", err)
		}

		api = viper.GetString("api")
		if api == "" {
			fmt.Printf("--api not provided, aborting.")
			return
		}

		key = viper.GetString("key")
		if key == "" {
			fmt.Printf("--key not provided, aborting.")
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

		bookedJobID = viper.GetInt64("booked_job_id")

		model = int64(viper.GetInt("model"))
		status.Model = model

		port, err := server()
		if err != nil {
			sdk.Exit("cannot bind port for worker export: %s", err)
		}
		exportport = port

		startTimestamp = time.Now()

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

	flags.Int("heartbeat", 10, "Worker heartbeat frequency")
	viper.BindPFlag("heartbeat", flags.Lookup("heartbeat"))

	flags.Int64("booked-job-id", 0, "Booked job id")
	viper.BindPFlag("booked_job_id", flags.Lookup("booked-job-id"))

	mainCmd.AddCommand(cmdExport)
	mainCmd.AddCommand(cmdUpload)
	mainCmd.AddCommand(versionCmd)
}

func main() {
	sdk.SetAgent(sdk.WorkerAgent)
	mainCmd.Execute()
}

// Will be removed when websocket conn is implemented
// for now, poll the /queue
func queuePolling() {
	firstViewQueue := true
	for {
		if WorkerID == "" {
			var info string
			if bookedJobID > 0 {
				info = fmt.Sprintf(", I was born to work on job %d", bookedJobID)
			}
			log.Notice("Registering on CDS engine%s", info)
			if err := register(api, name, key); err != nil {
				log.Notice("Cannot register: %s", err)
				time.Sleep(10 * time.Second)
				continue
			}
			alive = true
		}

		//We we've done nothing until ttl is over, let's exit
		if nbActionsDone == 0 && startTimestamp.Add(time.Duration(viper.GetInt("ttl"))*time.Minute).Before(time.Now()) {
			log.Notice("Time to exit.")
			unregister()
		}

		checkQueue(bookedJobID)
		if firstViewQueue {
			// if worker did not found booked job ID is first iteration
			// reset booked job to take another action
			bookedJobID = 0
		}

		firstViewQueue = false
		time.Sleep(4 * time.Second)
	}
}

func checkQueue(bookedJobID int64) {
	defer sdk.SetWorkerStatus(sdk.StatusWaiting)

	queue, err := sdk.GetBuildQueue()
	if err != nil {
		log.Warning("checkQueue> Cannot get build queue: %s", err)
		time.Sleep(5 * time.Second)
		WorkerID = ""
		return
	}

	log.Notice("checkQueue> %d actions in queue", len(queue))

	//Set the status to checking to avoid beeing killed while checking queue, actions and requirements
	sdk.SetWorkerStatus(sdk.StatusChecking)

	for i := range queue {
		if bookedJobID != 0 && queue[i].ID != bookedJobID {
			continue
		}

		requirementsOK := true
		// Check requirement
		log.Notice("checkQueue> Checking requirements for action [%d] %s", queue[i].ID, queue[i].Job.Action.Name)
		for _, r := range queue[i].Job.Action.Requirements {
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
			t := ""
			if queue[i].ID != bookedJobID {
				t = ", this was my booked job"
			}
			log.Notice("checkQueue> Taking job %d%s", queue[i].ID, t)
			takeJob(queue[i], queue[i].ID == bookedJobID)
		}
	}

	if bookedJobID > 0 {
		log.Notice("checkQueue> worker born for work on job %d but job is not found in queue", bookedJobID)
	}

	if !viper.GetBool("single_use") {
		log.Notice("checkQueue> Nothing to do...")
	}
}

func postCheckRequirementError(r *sdk.Requirement, err error) {
	s := fmt.Sprintf("Error checking requirement Name=%s Type=%s Value=%s :%s", r.Name, r.Type, r.Value, err)
	sdk.Request("POST", "/queue/requirements/errors", []byte(s))
}

func takeJob(b sdk.PipelineBuildJob, isBooked bool) {
	in := worker.TakeForm{Time: time.Now()}
	if isBooked {
		in.BookedJobID = b.ID
	}

	bodyTake, errm := json.Marshal(in)
	if errm != nil {
		log.Notice("takeJob> Cannot marshal body: %s", errm)
	}

	nbActionsDone++
	gitsshPath = ""
	pkey = ""
	path := fmt.Sprintf("/queue/%d/take", b.ID)
	data, code, errr := sdk.Request("POST", path, bodyTake)
	if errr != nil {
		log.Notice("takeJob> Cannot take action %d : %s", b.Job.PipelineActionID, errr)
		return
	}
	if code != http.StatusOK {
		return
	}

	pbji := worker.PipelineBuildJobInfo{}
	if err := json.Unmarshal([]byte(data), &pbji); err != nil {
		log.Notice("takeJob> Cannot unmarshal action: %s", err)
		return
	}

	pbJob = pbji.PipelineBuildJob
	// Reset build variables
	buildVariables = nil
	start := time.Now()
	res := run(&pbji)
	res.RemoteTime = time.Now()
	res.Duration = sdk.Round(time.Since(start), time.Second).String()

	// Give time to buffered logs to be sent
	time.Sleep(3 * time.Second)

	path = fmt.Sprintf("/queue/%d/result", b.ID)
	body, errm := json.MarshalIndent(res, " ", " ")
	if errm != nil {
		log.Critical("takeJob> Cannot marshal result: %s", errm)
		unregister()
		return
	}

	code = 300
	var isThereAnyHopeLeft = 10
	for code >= 300 {
		var errre error
		_, code, errre = sdk.Request("POST", path, body)
		if code == http.StatusNotFound {
			log.Notice("takeJob> Cannot send build result: PipelineBuildJob does not exists anymore")
			unregister() // well...
			break
		}
		if errre == nil && code < 300 {
			fmt.Printf("BuildResult sent.")
			break
		}

		if errre != nil {
			log.Warning("takeJob> Cannot send build result: %s", errre)
		} else {
			log.Warning("takeJob> Cannot send build result: HTTP %d", code)
		}

		time.Sleep(5 * time.Second)
		isThereAnyHopeLeft--
		if isThereAnyHopeLeft < 0 {
			log.Notice("takeJob> Could not send built result 10 times, giving up")
			break
		}
	}

	if viper.GetBool("single_use") {
		// Give time to logs to be flushed
		time.Sleep(2 * time.Second)
		// Unregister from engine
		if err := unregister(); err != nil {
			log.Warning("takeJob> could not unregister: %s", err)
		}
	}

}

func heartbeat() {
	for {
		time.Sleep(time.Duration(viper.GetInt("heartbeat")) * time.Second)
		if WorkerID == "" {
			log.Notice("Disconnected from CDS engine, trying to register...")
			if err := register(api, name, key); err != nil {
				log.Notice("Cannot register: %s", err)
				continue
			}
		}

		_, code, err := sdk.Request("POST", "/worker/refresh", nil)
		if err != nil || code >= 300 {
			log.Notice("heartbeat> cannot refresh beat: %d %s", code, err)
			WorkerID = ""
		}
	}
}

func unregister() error {
	_, code, err := sdk.Request("POST", "/worker/unregister", nil)
	if err != nil {
		return err
	}
	if code > 300 {
		return fmt.Errorf("HTTP %d", code)
	}

	if viper.GetBool("single_use") {
		log.Notice("queuePolling> waiting 30min to be killed by hatchery, if not killed, worker will exit")
		time.Sleep(30 * time.Minute)
		os.Exit(0)
	}
	return nil
}
