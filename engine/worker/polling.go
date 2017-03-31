package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/viper"

	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

var firstViewQueue = true

func queuePolling() {
	checkQueue(bookedJobID)
	if firstViewQueue {
		// if worker did not found booked job ID is first iteration
		// reset booked job to take another action
		bookedJobID = 0
	}
}

func checkQueue(bookedJobID int64) {
	defer sdk.SetWorkerStatus(sdk.StatusWaiting)

	queue, err := sdk.GetBuildQueue()
	if err != nil {
		log.Warning("checkQueue> Cannot get build queue: %s", err)
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
