package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func runNotifAction(a *sdk.Action, actionBuild sdk.ActionBuild) sdk.Result {
	res := sdk.Result{Status: sdk.StatusFail}

	var destination, title, message, messagefile string
	for _, a := range a.Parameters {
		switch a.Name {
		case "destination":
			destination = a.Value
		case "title":
			title = a.Value
		case "message":
			message = a.Value
		case "messagefile":
			messagefile = a.Value
		}
	}

	// Check if all args are provided
	if destination == "" || title == "" {
		sendLog(actionBuild.ID, sdk.ScriptAction, fmt.Sprintf("destination or title not provided, aborting\n"))
		res.Status = sdk.StatusFail
		return res
	}

	if messagefile != "" {

		dat, err := ioutil.ReadFile(messagefile)
		if err == nil {
			message = string(dat)
			sendLog(actionBuild.ID, sdk.ScriptAction, fmt.Sprintf("Replace notif message with content of %s done.\n", messagefile))
		} else {
			sendLog(actionBuild.ID, sdk.ScriptAction, fmt.Sprintf("Error while replacing notif message with content of %s, error: %s\n", messagefile, err))
		}
	}

	notif := &sdk.Notif{
		Event:       sdk.CreateNotifEvent,
		DateNotif:   time.Now().Unix(),
		NotifType:   sdk.BuiltinNotif,
		Status:      sdk.StatusSuccess,
		ActionBuild: &actionBuild,
		Destination: destination,
		Title:       title,
		Message:     message,
	}

	body, err := json.Marshal(notif)
	if err != nil {
		sendLog(actionBuild.ID, sdk.ScriptAction, fmt.Sprintf("Notif, Cannot marshal body: %s\n", err))
		return res
	}

	sendLog(actionBuild.ID, sdk.ScriptAction, "Send notif message to API\n")
	path := fmt.Sprintf("/notif/%d", actionBuild.ID)
	_, _, err = sdk.Request("POST", path, body)
	if err != nil {
		log.Notice("error: cannot send notif: %s\n", err)
		return res
	}

	res.Status = sdk.StatusSuccess
	return res
}
