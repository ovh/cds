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
	cmdKeyRoot.AddCommand(cmdKeyInstall(w)) //, cmdKeyUninstall(w))

	return cmdKeyRoot
}

func cmdKeyInstall(w *currentWorker) *cobra.Command {
	c := &cobra.Command{
		Use:     "install",
		Aliases: []string{"i", "add"},
		Short:   "worker key install <key-name>",
		Long: `
Inside a step script you can install a ssh key generated in CDS in your ssh environment
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
			sdk.Exit("worker key install > %s not found, are you running inside a CDS worker job?\n", WorkerServerPort)
		}

		port, errPort := strconv.Atoi(portS)
		if errPort != nil {
			sdk.Exit("worker key install > Cannot parse '%s' as a port number : %s\n", portS, errPort)
		}

		if len(args) < 1 {
			sdk.Exit("worker key install > Wrong usage: Example : worker key install proj-key\n")
		}

		req, errRequest := http.NewRequest(
			"POST",
			fmt.Sprintf("http://127.0.0.1:%d/key/%s/install", port, url.PathEscape(args[0])),
			bytes.NewReader(nil),
		)
		if errRequest != nil {
			sdk.Exit("worker key install > cannot post worker key install (Request): %s\n", errRequest)
		}

		resp, errDo := http.DefaultClient.Do(req)
		if errDo != nil {
			sdk.Exit("worker key install > cannot post worker key install (Do): %s\n", errDo)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 300 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				sdk.Exit("key install HTTP error %v\n", err)
			}
			cdsError := sdk.DecodeError(body)
			sdk.Exit("Error: http code %d : %v\n", resp.StatusCode, cdsError)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			sdk.Exit("key install HTTP body read error %v\n", err)
		}

		var keyResp keyResponse
		if err := json.Unmarshal(body, &keyResp); err != nil {
			sdk.Exit("key install> cannot unmarshall key response")
		}

		os.Setenv("PKEY", keyResp.PKey)
		fmt.Printf("export PKEY=%s\n", keyResp.PKey)

	}
}

func (wk *currentWorker) keyInstallHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	keyName := vars["key"]
	var key *sdk.Variable

	if wk.currentJob.secrets == nil {
		err := sdk.Error{
			Message: "worker key install > Cannot find any keys for your job",
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
			Message: fmt.Sprintf("worker key install > Key %s not found", keyName),
			Status:  http.StatusNotFound,
		}
		log.Error("%v", err)
		writeJSON(w, err, err.Status)
		return
	}

	wk.currentJob.pkey = path.Join(keysDirectory, key.Name)
	os.Setenv("PKEY", wk.currentJob.pkey)
	log.Info("PKEY %s", wk.currentJob.pkey)

	if err := vcs.SetupSSHKey(wk.currentJob.secrets, keysDirectory, key); err != nil {
		errSetup := sdk.Error{
			Message: fmt.Sprintf("worker key install > Cannot setup ssh key %s : %v", keyName, err),
			Status:  http.StatusInternalServerError,
		}
		log.Error("%v", errSetup)
		writeJSON(w, errSetup, errSetup.Status)
		return
	}

	writeJSON(w, keyResponse{PKey: wk.currentJob.pkey}, http.StatusOK)
	return
}

// func cmdKeyUninstall(w *currentWorker) *cobra.Command {
// 	c := &cobra.Command{
// 		Use:     "uninstall",
// 		Aliases: []string{"remove", "rm", "delete", "u"},
// 		Short:   "worker key install <key-name>",
// 		Long: `Inside a step script you can install a ssh key generated in CDS in your ssh environment
// 		`,
// 		Run: keyUninstallCmd(w),
// 	}
// 	return c
// }
//
// func keyUninstallCmd(w *currentWorker) func(cmd *cobra.Command, args []string) {
// 	return func(cmd *cobra.Command, args []string) {
// 		portS := os.Getenv(WorkerServerPort)
// 		if portS == "" {
// 			sdk.Exit("worker cache pull > %s not found, are you running inside a CDS worker job?\n", WorkerServerPort)
// 		}
//
// 		port, errPort := strconv.Atoi(portS)
// 		if errPort != nil {
// 			sdk.Exit("worker cache pull > cannot parse '%s' as a port number: %s", portS, errPort)
// 		}
//
// 		if len(args) < 1 {
// 			sdk.Exit("worker cache pull > Wrong usage: Example : worker cache pull myTagValue")
// 		}
//
// 		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
// 		if err != nil {
// 			sdk.Exit("worker cache pull > cannot get current path: %s\n", err)
// 		}
//
// 		fmt.Printf("Worker cache pull in progress... (tag: %s)\n", args[0])
// 		req, errRequest := http.NewRequest(
// 			"GET",
// 			fmt.Sprintf("http://127.0.0.1:%d/cache/%s/pull?path=%s", port, base64.RawURLEncoding.EncodeToString([]byte(args[0])), url.QueryEscape(dir)),
// 			nil,
// 		)
// 		if errRequest != nil {
// 			sdk.Exit("worker cache pull > cannot post worker cache pull with tag %s (Request): %s\n", args[0], errRequest)
// 		}
//
// 		client := http.DefaultClient
// 		client.Timeout = 10 * time.Minute
//
// 		resp, errDo := client.Do(req)
// 		if errDo != nil {
// 			sdk.Exit("worker cache pull > cannot post worker cache pull (Do): %s\n", errDo)
// 		}
//
// 		if resp.StatusCode >= 300 {
// 			body, err := ioutil.ReadAll(resp.Body)
// 			if err != nil {
// 				sdk.Exit("cache pull HTTP error %v\n", err)
// 			}
// 			cdsError := sdk.DecodeError(body)
// 			sdk.Exit("Error: %v\n", cdsError)
// 		}
//
// 		fmt.Printf("Worker cache pull with success (tag: %s)\n", args[0])
// 	}
// }
//
// func (wk *currentWorker) keyUninstallHandler(w http.ResponseWriter, r *http.Request) {
// 	vars := mux.Vars(r)
//
// 	path := r.FormValue("path")
//
// 	if wk.currentJob.wJob == nil {
// 		errW := fmt.Errorf("worker cache pull > Cannot find workflow job info")
// 		writeError(w, r, errW)
// 		return
// 	}
// 	params := wk.currentJob.wJob.Parameters
// 	projectKey := sdk.ParameterValue(params, "cds.project")
// 	bts, err := wk.client.WorkflowCachePull(projectKey, vars["ref"])
// 	if err != nil {
// 		err = sdk.Error{
// 			Message: "worker cache pull > Cannot pull cache : " + err.Error(),
// 			Status:  http.StatusInternalServerError,
// 		}
// 		writeError(w, r, err)
// 		return
// 	}
//
// 	tr := tar.NewReader(bts)
// 	for {
// 		hdr, err := tr.Next()
// 		if err == io.EOF {
// 			break
// 		}
// 		if err != nil {
// 			err = sdk.Error{
// 				Message: "worker cache pull > Unable to read tar file : " + err.Error(),
// 				Status:  http.StatusBadRequest,
// 			}
// 			writeError(w, r, err)
// 			return
// 		}
//
// 		if hdr == nil {
// 			continue
// 		}
//
// 		target := filepath.Join(path, hdr.Name)
// 		if _, errS := os.Stat(filepath.Dir(target)); errS != nil {
// 			if errM := os.MkdirAll(filepath.Dir(target), 0755); errM != nil {
// 				errM = sdk.Error{
// 					Message: "worker cache pull > Cannot create directory : " + errM.Error(),
// 					Status:  http.StatusInternalServerError,
// 				}
// 				writeError(w, r, errM)
// 				return
// 			}
// 		}
//
// 		f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(hdr.Mode))
// 		if err != nil {
// 			err = sdk.Error{
// 				Message: "worker cache pull > Cannot create file: " + err.Error(),
// 				Status:  http.StatusInternalServerError,
// 			}
// 			writeError(w, r, err)
// 			return
// 		}
//
// 		// copy over contents
// 		if _, err := io.Copy(f, tr); err != nil {
// 			f.Close()
// 			err = sdk.Error{
// 				Message: "worker cache pull > Cannot copy content file : " + err.Error(),
// 				Status:  http.StatusInternalServerError,
// 			}
// 			writeError(w, r, err)
// 			return
// 		}
// 		f.Close()
// 	}
// }
