package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/vcs"
)

type keyResponse struct {
	PKey string `json:"pkey"`
}

func cmdKey(w *currentWorker) *cobra.Command {
	cmdKeyRoot := &cobra.Command{
		Use: "key",
		Long: `

    `,
		Short: "Inside a step script you can install/uninstall a ssh key generated in CDS in your ssh environment",
	}
	cmdKeyRoot.AddCommand(cmdKeyInstall(w))

	return cmdKeyRoot
}

func cmdKeyInstall(w *currentWorker) *cobra.Command {
	c := &cobra.Command{
		Use:     "install",
		Aliases: []string{"i", "add"},
		Short:   "worker key install <key-name>",
		Long: `
Inside a step script you can install a ssh key generated in CDS in your ssh environment and return the PKEY variable

So if you want to update your PKEY variable, which is the variable with the path to the ssh private key you just can write ` + "PKEY=`worker key install proj-mykey`" + `
		`,
		Example: "worker key install proj-test",
		Run:     keyInstallCmd(w),
	}
	return c
}

func keyInstallCmd(w *currentWorker) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		portS := os.Getenv(WorkerServerPort)
		if portS == "" {
			sdk.Exit("Error: worker key install > %s not found, are you running inside a CDS worker job?\n", WorkerServerPort)
		}

		port, errPort := strconv.Atoi(portS)
		if errPort != nil {
			sdk.Exit("Error: worker key install > Cannot parse '%s' as a port number : %s\n", portS, errPort)
		}

		if len(args) < 1 {
			sdk.Exit("Error: worker key install > Wrong usage: Example : worker key install proj-key\n")
		}

		req, errRequest := http.NewRequest(
			"POST",
			fmt.Sprintf("http://127.0.0.1:%d/key/%s/install", port, url.PathEscape(args[0])),
			bytes.NewReader(nil),
		)
		if errRequest != nil {
			sdk.Exit("Error: worker key install > cannot post worker key install (Request): %s\n", errRequest)
		}

		resp, errDo := http.DefaultClient.Do(req)
		if errDo != nil {
			sdk.Exit("Error: worker key install > cannot post worker key install (Do): %s\n", errDo)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 300 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				sdk.Exit("Error: worker key install> HTTP error %v\n", err)
			}
			cdsError := sdk.DecodeError(body)
			sdk.Exit("Error: worker key install> http code %d : %v\n", resp.StatusCode, cdsError)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			sdk.Exit("Error: worker key install> HTTP body read error %v\n", err)
		}

		var keyResp keyResponse
		if err := json.Unmarshal(body, &keyResp); err != nil {
			sdk.Exit("Error: worker key install> cannot unmarshall key response")
		}

		fmt.Println(keyResp.PKey)
	}
}

func (wk *currentWorker) keyInstallHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	keyName := vars["key"]
	var key *sdk.Variable

	if wk.currentJob.secrets == nil {
		err := sdk.Error{
			Message: "Cannot find any keys for your job",
			Status:  http.StatusBadRequest,
		}
		log.Error("%v", err)
		writeJSON(w, err, err.Status)
		return
	}

	for _, k := range wk.currentJob.secrets {
		if k.Name == ("cds.key." + keyName + ".priv") {
			key = &k
			break
		}
	}

	if key == nil {
		err := sdk.Error{
			Message: fmt.Sprintf("Key %s not found", keyName),
			Status:  http.StatusNotFound,
		}
		log.Error("%v", err)
		writeJSON(w, err, err.Status)
		return
	}

	wk.currentJob.pkey = path.Join(keysDirectory, key.Name)

	if err := vcs.CleanSSHKeys(keysDirectory, nil); err != nil {
		errClean := sdk.Error{
			Message: fmt.Sprintf("Cannot clean ssh keys : %v", err),
			Status:  http.StatusInternalServerError,
		}
		log.Error("%v", errClean)
		writeJSON(w, errClean, errClean.Status)
		return
	}

	if err := vcs.SetupSSHKey(wk.currentJob.secrets, keysDirectory, key); err != nil {
		errSetup := sdk.Error{
			Message: fmt.Sprintf("Cannot setup ssh key %s : %v", keyName, err),
			Status:  http.StatusInternalServerError,
		}
		log.Error("%v", errSetup)
		writeJSON(w, errSetup, errSetup.Status)
		return
	}

	writeJSON(w, keyResponse{PKey: wk.currentJob.pkey}, http.StatusOK)
	return
}
