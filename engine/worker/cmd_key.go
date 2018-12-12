package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/vcs"
)

type keyResponse struct {
	PKey string `json:"pkey"`
	Type string `json:"type"`
}

func cmdKey(w *currentWorker) *cobra.Command {
	cmdKeyRoot := &cobra.Command{
		Use:  "key",
		Long: "Inside a step script you can install/uninstall a ssh key generated in CDS in your ssh environment",
	}
	cmdKeyRoot.AddCommand(cmdKeyInstall(w))

	return cmdKeyRoot
}

var (
	cmdInstallEnvGIT bool
	cmdInstallEnv    bool
	cmdInstallToFile string
)

func cmdKeyInstall(w *currentWorker) *cobra.Command {
	c := &cobra.Command{
		Use:     "install",
		Aliases: []string{"i", "add"},
		Short:   "worker key install [--env-git] [--env] [--file destination-file] <key-name>",
		Long: `
Inside a step script you can install a SSH/PGP key generated in CDS in your ssh environment and return the PKEY variable (only for SSH)

So if you want to update your PKEY variable, which is the variable with the path to the SSH private key you just can write ` + "PKEY=$(worker key install proj-mykey)`" + ` (only for SSH)

You can use the ` + "`--env`" + ` flag to export the PKEY variable:

` + "```" + `
$ eval $(worker key install --env proj-mykey)
echo $PKEY # variable $PKEY will contains the path of the SSH private key
` + "```" + `

You can use the ` + "`--file`" + `  flag to write the private key to a specific path
` + "```" + `
$ worker key install --file .ssh/id_rsa proj-mykey
` + "```" + `

For most advanced usage with git and SSH, you can run ` + "`eval $(worker key install --env-git proj-mykey)`" + `.

The ` + "`--env-git`" + ` flag will display:

` + "```" + `
$ worker key install --env-git proj-mykey
echo "ssh -i /tmp/5/0/2569/655/bd925028e70aea34/cds.key.proj-mykey.priv -o StrictHostKeyChecking=no \$@" > /tmp/5/0/2569/655/bd925028e70aea34/cds.key.proj-mykey.priv.gitssh.sh;
chmod +x /tmp/5/0/2569/655/bd925028e70aea34/cds.key.proj-mykey.priv.gitssh.sh;
export GIT_SSH="/tmp/5/0/2569/655/bd925028e70aea34/cds.key.proj-mykey.priv.gitssh.sh";
export PKEY="/tmp/5/0/2569/655/bd925028e70aea34/cds.key.proj-mykey.priv";
` + "```" + `

So that, you can use custom git commands the the previous installed SSH key.

`,
		Example: "worker key install proj-test",
		Run:     keyInstallCmd(w),
	}
	c.Flags().BoolVar(&cmdInstallEnvGIT, "env", false, "display shell command for export $PKEY variable. See documentation.")
	c.Flags().BoolVar(&cmdInstallEnv, "env-git", false, "display shell command for advanced usage with git. See documentation.")
	c.Flags().StringVar(&cmdInstallToFile, "file", "", "write key to destination file. See documentation.")

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

		if cmdInstallToFile != "" {
			filename, err := filepath.Abs(cmdInstallToFile)
			if err != nil {
				sdk.Exit("Error: worker key install > cannot post worker key install (Request): %s\n", errRequest)
			}
			q := req.URL.Query()
			q.Add("file", filename)
			req.URL.RawQuery = q.Encode()
		}

		client := http.DefaultClient
		client.Timeout = time.Minute

		resp, errDo := client.Do(req)
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
			if cdsError != nil {
				sdk.Exit("Error: worker key install> error: %v\n", cdsError)
			} else {
				sdk.Exit(string(body))
			}
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			sdk.Exit("Error: worker key install> HTTP body read error %v\n", err)
		}

		var keyResp keyResponse
		if err := json.Unmarshal(body, &keyResp); err != nil {
			sdk.Exit("Error: worker key install> cannot unmarshall key response")
		}

		switch keyResp.Type {
		case sdk.KeyTypeSSH:
			switch {
			case cmdInstallEnvGIT:
				fmt.Printf("echo \"ssh -i %s -o StrictHostKeyChecking=no \\$@\" > %s.gitssh.sh;\n", keyResp.PKey, keyResp.PKey)
				fmt.Printf("chmod +x %s.gitssh.sh;\n", keyResp.PKey)
				fmt.Printf("export GIT_SSH=\"%s.gitssh.sh\";\n", keyResp.PKey)
				fmt.Printf("export PKEY=\"%s\";\n", keyResp.PKey)
			case cmdInstallEnv:
				fmt.Printf("export PKEY=\"%s\";\n", keyResp.PKey)
			case cmdInstallToFile != "":
				fmt.Printf("# Key installed to %s\n", cmdInstallToFile)
			default:
				fmt.Println(keyResp.PKey)
			}
		case sdk.KeyTypePGP:
			fmt.Println("Your PGP key is imported with success")
		}
	}
}

func (wk *currentWorker) keyInstallHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	keyName := vars["key"]
	fileName := r.FormValue("file")
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

	switch key.Type {
	case sdk.KeyTypeSSH:
		if fileName == "" {
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
		} else {
			if err := ioutil.WriteFile(fileName, []byte(key.Value), os.FileMode(0600)); err != nil {
				errSetup := sdk.Error{
					Message: fmt.Sprintf("Cannot setup ssh key %s : %v", keyName, err),
					Status:  http.StatusInternalServerError,
				}
				log.Error("%v", errSetup)
				writeJSON(w, errSetup, errSetup.Status)
			}
		}

		writeJSON(w, keyResponse{PKey: wk.currentJob.pkey, Type: sdk.KeyTypeSSH}, http.StatusOK)

	case sdk.KeyTypePGP:
		gpg2Found := false

		if _, err := exec.LookPath("gpg2"); err == nil {
			gpg2Found = true
		}

		if !gpg2Found {
			if _, err := exec.LookPath("gpg"); err != nil {
				errBinary := sdk.Error{
					Message: fmt.Sprintf("Cannot use gpg in your worker because you haven't gpg or gpg2 binary"),
					Status:  http.StatusBadRequest,
				}
				log.Error("%v", errBinary)
				writeJSON(w, errBinary, errBinary.Status)
				return
			}
		}
		content := []byte(key.Value)
		tmpfile, errTmpFile := ioutil.TempFile("", key.Name)
		if errTmpFile != nil {
			errFile := sdk.Error{
				Message: fmt.Sprintf("Cannot setup pgp key %s : %v", key.Name, errTmpFile),
				Status:  http.StatusInternalServerError,
			}
			log.Error("%v", errFile)
			writeJSON(w, errFile, errFile.Status)
			return
		}
		defer func() {
			_ = os.Remove(tmpfile.Name())
		}()

		if _, err := tmpfile.Write(content); err != nil {
			errW := sdk.Error{
				Message: fmt.Sprintf("Cannot setup pgp key file %s : %v", key.Name, err),
				Status:  http.StatusInternalServerError,
			}
			log.Error("%v", errW)
			writeJSON(w, errW, errW.Status)
			return
		}

		if err := tmpfile.Close(); err != nil {
			errC := sdk.Error{
				Message: fmt.Sprintf("Cannot setup pgp key file %s (close) : %v", key.Name, err),
				Status:  http.StatusInternalServerError,
			}
			log.Error("%v", errC)
			writeJSON(w, errC, errC.Status)
			return
		}

		gpgBin := "gpg"
		if gpg2Found {
			gpgBin = "gpg2"
		}
		cmd := exec.Command(gpgBin, "--import", tmpfile.Name())
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			errR := sdk.Error{
				Message: fmt.Sprintf("Cannot import pgp key %s : %v", key.Name, err),
				Status:  http.StatusInternalServerError,
			}
			log.Error("%v", errR)
			writeJSON(w, errR, errR.Status)
			return
		}
		writeJSON(w, keyResponse{Type: sdk.KeyTypePGP}, http.StatusOK)
	default:
		err := sdk.Error{
			Message: fmt.Sprintf("Type key %s is not implemented", key.Type),
			Status:  http.StatusNotImplemented,
		}
		log.Error("%v", err)
		writeJSON(w, err, err.Status)
	}
}
