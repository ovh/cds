package main

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"golang.org/x/crypto/ssh"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

/* Inside contrib/grpcplugins/action
$ make build ssh-cmd
$ make publish ssh-cmd
*/

type sshCmdActionPlugin struct {
	actionplugin.Common
}

func (actPlugin *sshCmdActionPlugin) Manifest(ctx context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "plugin-ssh-cmd",
		Author:      "François SAMIN <francois.samin@corp.ovh.com>",
		Description: "This plugin helps you to run cmd on remote server over ssh.",
		Version:     sdk.VERSION,
	}, nil
}

func (actPlugin *sshCmdActionPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	//Parse parameters
	buf := []byte(q.GetOptions()["privateKey"])
	user := q.GetOptions()["username"]
	hostnames := q.GetOptions()["hostnames"]
	cmd := q.GetOptions()["command"]
	timeoutS := q.GetOptions()["timeout"]
	cmdTimeoutS := q.GetOptions()["commandTimeout"]

	timeout, err := strconv.ParseFloat(timeoutS, 64)
	if err != nil {
		return actionplugin.Fail("Error parsing timeout value %s : %s", timeoutS, err)
	}

	cmdTimeout, err := strconv.ParseFloat(cmdTimeoutS, 64)
	if err != nil {
		return actionplugin.Fail("Error parsing commandTimeout value %s : %s", cmdTimeoutS, err)
	}

	//Parsing key
	key, err := ssh.ParsePrivateKey(buf)
	if err != nil {
		return actionplugin.Fail("PLUGIN", "Error parsing private key : %s", err)
	}

	//Prepare auth
	auth := []ssh.AuthMethod{ssh.PublicKeys(key)}
	sshConfig := &ssh.ClientConfig{
		User:    user,
		Auth:    auth,
		Timeout: time.Duration(timeout) * time.Second,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	//For all hosts
	errs := map[string]error{}
	outs := map[string][]byte{}
	for _, h := range strings.Split(hostnames, ",") {
		host := strings.TrimSpace(h)
		//Dialing server
		client, err := ssh.Dial("tcp", host, sshConfig)
		if err != nil {
			fmt.Printf("Error dialing %s : %s\n", host, err)
			errs[host] = err
			continue
		}
		//Start session
		session, err := client.NewSession()
		if err != nil {
			fmt.Printf("Error connecting %s@%s : %s\n", user, host, err)
			errs[host] = err
			continue
		}

		//Get the output
		errChan := make(chan error, 1)
		outChan := make(chan []byte, 1)

		//Timeout
		timeout := make(chan bool, 1)
		go func() {
			time.Sleep(time.Duration(cmdTimeout) * time.Second)
			timeout <- true
		}()

		//Run the Command
		go func() {
			out, errC := session.CombinedOutput(cmd)
			if errC != nil {
				errChan <- errC
				return
			}
			outChan <- out
		}()

		//Read the channel
		select {
		case <-timeout:
			if err != nil {
				err := fmt.Errorf("Command timeout")
				fmt.Printf("Error executing command on %s : %s\n", host, err)
				errs[host] = err
			}
		case err := <-errChan:
			if err != nil {
				fmt.Printf("Error executing command on %s : %s\n", host, err)
				errs[host] = err
			}
		case out := <-outChan:
			outs[host] = out
		}

		//Close the channels
		close(timeout)
		close(errChan)
		close(outChan)

		//Close client
		if err := client.Close(); err != nil {
			fmt.Printf("Error closing connection on %s : %s\n", host, err)
			errs[host] = err
			continue
		}
	}

	//Print outputs
	for h, out := range outs {
		if len(out) == 0 {
			fmt.Printf("Results %s : Not output ---\n", h)
			continue
		}
		fmt.Printf("Results %s : \n---BEGIN---\n%s\n---END---\n", h, string(out))
	}

	//Check errors
	for _, e := range errs {
		if e != nil {
			return actionplugin.Fail("")
		}
	}

	return &actionplugin.ActionResult{
		Status: sdk.StatusSuccess,
	}, nil
}

func main() {
	actPlugin := sshCmdActionPlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
}
