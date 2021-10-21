package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/engine/worker/internal"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func cmdKey() *cobra.Command {
	cmdKeyRoot := &cobra.Command{
		Use:  "key",
		Long: "Inside a step script you can install/uninstall a ssh key generated in CDS in your ssh environment",
	}
	cmdKeyRoot.AddCommand(cmdKeyInstall())

	return cmdKeyRoot
}

var (
	cmdInstallEnvGIT bool
	cmdInstallEnv    bool
	cmdInstallToFile string
)

func cmdKeyInstall() *cobra.Command {
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

So that, you can use custom git commands the previous installed SSH key.

`,
		Example: "worker key install proj-test",
		Run:     keyInstallCmd(),
	}
	c.Flags().BoolVar(&cmdInstallEnv, "env", false, "display shell command for export $PKEY variable. See documentation.")
	c.Flags().BoolVar(&cmdInstallEnvGIT, "env-git", false, "display shell command for advanced usage with git. See documentation.")
	c.Flags().StringVar(&cmdInstallToFile, "file", "", "write key to destination file. See documentation.")

	return c
}

func keyInstallCmd() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		portS := os.Getenv(internal.WorkerServerPort)
		if portS == "" {
			sdk.Exit("Error: worker key install > %s not found, are you running inside a CDS worker job?\n", internal.WorkerServerPort)
		}

		port, errPort := strconv.Atoi(portS)
		if errPort != nil {
			sdk.Exit("Error: worker key install > Cannot parse '%s' as a port number : %s\n", portS, errPort)
		}

		if len(args) < 1 || len(args) > 1 {
			sdk.Exit("Error: worker key install > Wrong usage: Example : worker key install proj-key\n")
		}

		var method = "POST"
		var uri = fmt.Sprintf("http://127.0.0.1:%d/key/%s/install", port, url.PathEscape(args[0]))
		var body io.Reader

		if cmdInstallToFile != "" {
			filename, err := filepath.Abs(cmdInstallToFile)
			if err != nil {
				sdk.Exit("Error: worker key install > cannot post worker key install (Request): %s\n", err)
			}
			var mapBody = map[string]string{
				"file": filename,
			}
			buffer, _ := json.Marshal(mapBody)
			body = bytes.NewReader(buffer)
		}

		req, errRequest := http.NewRequest(method, uri, body)
		if errRequest != nil {
			sdk.Exit("Error: worker key install > cannot post worker key install (Request): %s\n", errRequest)
		}

		client := http.DefaultClient
		client.Timeout = time.Minute

		resp, errDo := client.Do(req)
		if errDo != nil {
			sdk.Exit("Error: worker key install > cannot post worker key install (Do): %s\n", errDo)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 300 {
			body, err := io.ReadAll(resp.Body)
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

		bodyBtes, err := io.ReadAll(resp.Body)
		if err != nil {
			sdk.Exit("Error: worker key install> HTTP body read error %v\n", err)
		}

		defer resp.Body.Close() // nolint

		var keyResp workerruntime.KeyResponse
		if err := sdk.JSONUnmarshal(bodyBtes, &keyResp); err != nil {
			sdk.Exit("Error: worker key install> cannot unmarshall key response: %s", string(bodyBtes))
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
