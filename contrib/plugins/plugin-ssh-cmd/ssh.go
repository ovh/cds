package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/ovh/cds/sdk/plugin"
)

//SSHCmdPlugin implements plugin interface
type SSHCmdPlugin struct {
	plugin.Common
}

//Name returns the plugin name
func (s SSHCmdPlugin) Name() string {
	return "plugin-ssh-cmd"
}

//Description returns the plugin description
func (s SSHCmdPlugin) Description() string {
	return "This plugin helps you to run cmd on remote server over ssh."
}

//Author returns the plugin author's name
func (s SSHCmdPlugin) Author() string {
	return "François SAMIN <francois.samin@corp.ovh.com>"
}

//Parameters return parameters description
func (s SSHCmdPlugin) Parameters() plugin.Parameters {
	params := plugin.NewParameters()
	params.Add("username", plugin.StringParameter, "Username", "{{.cds.env.username}}")
	params.Add("privateKey", plugin.StringParameter, "SSH RSA private key", "{{.cds.app.key}}")
	params.Add("hostnames", plugin.StringParameter, "Hostnames (comma separated values)", "{{.cds.env.hostnames}}")
	params.Add("command", plugin.TextParameter, "Command", "echo \"Hello CDS !\"")
	params.Add("timeout", plugin.StringParameter, "Timeout (seconds)", "5")
	params.Add("commandTimeout", plugin.StringParameter, "Command Timeout (seconds)", "60")
	return params
}

//Run execute the action
func (s SSHCmdPlugin) Run(a plugin.IJob) plugin.Result {
	//Parse parameters
	buf := []byte(a.Arguments().Get("privateKey"))
	user := a.Arguments().Get("username")
	hostnames := a.Arguments().Get("hostnames")
	cmd := a.Arguments().Get("command")
	timeoutS := a.Arguments().Get("timeout")
	cmdTimeoutS := a.Arguments().Get("commandTimeout")

	timeout, err := strconv.ParseFloat(timeoutS, 64)
	if err != nil {
		plugin.SendLog(a, "Error parsing timeout value %s : %s", timeoutS, err)
		return plugin.Fail
	}

	cmdTimeout, err := strconv.ParseFloat(cmdTimeoutS, 64)
	if err != nil {
		plugin.SendLog(a, "Error parsing commandTimeout value %s : %s", cmdTimeoutS, err)
		return plugin.Fail
	}

	//Parsing key
	key, err := ssh.ParsePrivateKey(buf)
	if err != nil {
		plugin.SendLog(a, "PLUGIN", "Error parsing private key : %s", err)
		return plugin.Fail
	}

	//Prepare auth
	auth := []ssh.AuthMethod{ssh.PublicKeys(key)}
	sshConfig := &ssh.ClientConfig{
		User:    user,
		Auth:    auth,
		Timeout: time.Duration(timeout) * time.Second,
	}

	//For all hosts
	errs := map[string]error{}
	outs := map[string][]byte{}
	for _, h := range strings.Split(hostnames, ",") {
		host := strings.TrimSpace(h)
		//Dialing server
		client, err := ssh.Dial("tcp", host, sshConfig)
		if err != nil {
			plugin.SendLog(a, "Error dialing %s : %s\n", host, err)
			errs[host] = err
			continue
		}
		//Start session
		session, err := client.NewSession()
		if err != nil {
			plugin.SendLog(a, "Error connecting %s@%s : %s\n", user, host, err)
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
			out, err := session.CombinedOutput(cmd)
			if err != nil {
				errChan <- err
				return
			}
			outChan <- out
		}()

		//Read the channel
		select {
		case <-timeout:
			if err != nil {
				err := fmt.Errorf("Command timeout")
				plugin.SendLog(a, "Error executing command on %s : %s\n", host, err)
				errs[host] = err
			}
		case err := <-errChan:
			if err != nil {
				plugin.SendLog(a, "Error executing command on %s : %s\n", host, err)
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
			plugin.SendLog(a, "Error closing connection on %s : %s\n", host, err)
			errs[host] = err
			continue
		}
	}

	//Print outputs
	for h, out := range outs {
		if len(out) == 0 {
			plugin.SendLog(a, "Results %s : Not output ---\n", h)
			continue
		}
		plugin.SendLog(a, "Results %s : \n---BEGIN---\n%s\n---END---\n", h, string(out))
	}

	//Check errors
	for _, e := range errs {
		if e != nil {
			return plugin.Fail
		}
	}

	return plugin.Success
}

func main() {
	plugin.Main(&SSHCmdPlugin{})
}
