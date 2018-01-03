package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/facebookgo/httpcontrol"
	"github.com/gambol99/go-marathon"
	"github.com/xeipuuv/gojsonschema"

	"github.com/ovh/cds/sdk/plugin"
)

//MarathonPlugin is our marathon plugin to manage app deployment
type MarathonPlugin struct {
	plugin.Common
}

//Name return plugin name. It must me the same as the binary file
func (m MarathonPlugin) Name() string {
	return "plugin-marathon"
}

//Description explains the purpose of the plugin
func (m MarathonPlugin) Description() string {
	return `This action helps you to deploy on Mesos/Marathon. Provide a marathon.json file to configure deployment.

Your marathon.json file can be templated with cds variables "{{.cds.variables}}". Enable "waitForDeployment" option to ensure deployment is successful.`
}

//Author of the plugin
func (m MarathonPlugin) Author() string {
	return "François SAMIN <francois.samin@corp.ovh.com>"
}

//Parameters return parameters description
func (m MarathonPlugin) Parameters() plugin.Parameters {
	params := plugin.NewParameters()

	params.Add("url", plugin.StringParameter,
		"Marathon URL http://127.0.0.1:8081,http://127.0.0.1:8082,http://127.0.0.1:8083",
		"{{.cds.env.marathonHost}}")
	params.Add("user", plugin.StringParameter,
		"Marathon User (please use project, application or environment variables)",
		"{{.cds.env.marathonUser}}")

	params.Add("password", plugin.StringParameter,
		"Marathon Password (please use project, application or environment variables)",
		"{{.cds.env.marathonPassword}}")

	params.Add("configuration", plugin.StringParameter,
		"Marathon application configuration file (json format)",
		"marathon.json")

	params.Add("waitForDeployment", plugin.BooleanParameter,
		`Wait for instances deployment.
If set, CDS will wait for all instances to be deployed until timeout is over. All instances deployment must be done to get a successful result.
If not set, CDS will consider a successful result if marathon accepts the provided configuration.`,
		"true")

	params.Add("insecureSkipVerify", plugin.BooleanParameter,
		`Skip SSL Verify if you want to use self-signed certificate`,
		"false")

	params.Add("timeout", plugin.NumberParameter,
		`Marathon deployment timeout (seconds). Used only if "waitForDeployment" is true. `,
		"120")

	return params
}

//Run execute the action
func (m MarathonPlugin) Run(a plugin.IJob) plugin.Result {
	//Get arguments
	URL := a.Arguments().Get("url")
	user := a.Arguments().Get("user")
	password := a.Arguments().Get("password")
	tmplConf := a.Arguments().Get("configuration")
	waitForDeploymentStr := a.Arguments().Get("waitForDeployment")
	insecureSkipVerifyStr := a.Arguments().Get("insecureSkipVerify")
	timeoutStr := a.Arguments().Get("timeout")

	//Parse arguments
	waitForDeployment, err := strconv.ParseBool(waitForDeploymentStr)
	if err != nil {
		plugin.SendLog(a, "Error parsing waitForDeployment value : %s\n", err.Error())
		return plugin.Fail
	}

	insecureSkipVerify := false
	if insecureSkipVerifyStr != "" {
		var errb error
		insecureSkipVerify, errb = strconv.ParseBool(insecureSkipVerifyStr)
		if err != nil {
			plugin.SendLog(a, "Error parsing insecureSkipVerify value : %s\n", errb.Error())
			return plugin.Fail
		}
	}

	if insecureSkipVerify {
		plugin.SendLog(a, "You are using insecureSkipVerify flag to true. It is not recommended\n")
	}

	timeout, err := strconv.Atoi(timeoutStr)
	if err != nil {
		plugin.SendLog(a, "Error parsing timeout value :  %s\n", err.Error())
		return plugin.Fail
	}

	//Custom http client with 3 retries
	httpClient := &http.Client{
		Transport: &httpcontrol.Transport{
			RequestTimeout:  time.Minute,
			MaxTries:        3,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerify},
		},
	}

	plugin.SendLog(a, "Connecting on %s\n", URL)
	config := marathon.NewDefaultConfig()
	config.URL = URL
	config.HTTPBasicAuthUser = user
	config.HTTPBasicPassword = password
	config.HTTPClient = httpClient

	//Connecting to marathon
	client, err := marathon.NewClient(config)
	if err != nil {
		plugin.SendLog(a, "Connection failed on %s\n", URL)
		return plugin.Fail
	}

	//Run tmpl on configuration file to replace all cds variables
	conf, err := tmplApplicationConfigFile(a, tmplConf)
	if err != nil {
		plugin.SendLog(a, "Templating Configuration File KO (tmplApplicationConfigFile): %s\n", err.Error())
		return plugin.Fail
	}
	defer os.RemoveAll(conf)
	plugin.SendLog(a, "Templating Configuration File: OK\n")

	//Validate json file and load application
	appConfig, err := parseApplicationConfigFile(a, conf)
	if err != nil {
		plugin.SendLog(a, "Templating Configuration File KO (parseApplicationConfigFile): %s\n", err.Error())
		return plugin.Fail
	}
	plugin.SendLog(a, "Parsing Configuration File: OK\n")

	//Allways put cds.version labels
	if appConfig.Labels == nil {
		appConfig.Labels = &map[string]string{}
	}

	(*appConfig.Labels)["CDS_VERSION"] = a.Arguments().Get("cds.version")
	(*appConfig.Labels)["CDS_PROJECT"] = a.Arguments().Get("cds.project")
	(*appConfig.Labels)["CDS_APPLICATION"] = a.Arguments().Get("cds.application")
	(*appConfig.Labels)["CDS_ENVIRONMENT"] = a.Arguments().Get("cds.environment")

	gitBranch := a.Arguments().Get("git.branch")
	if gitBranch != "" {
		(*appConfig.Labels)["CDS_GIT_BRANCH"] = gitBranch
	}

	gitHash := a.Arguments().Get("git.hash")
	if gitHash != "" {
		(*appConfig.Labels)["CDS_GIT_HASH"] = gitHash
	}

	plugin.SendLog(a, "Configuration File %s: OK\n", tmplConf)

	//Searching for application
	val := url.Values{"id": []string{appConfig.ID}}
	applications, err := client.Applications(val)
	if err != nil {
		plugin.SendLog(a, "Failed to list applications: %s\n", err.Error())
		return plugin.Fail
	}

	var appExists bool
	if len(applications.Apps) != 0 {
		appExists = true
	}

	//Update or create application
	if appExists {
		if _, err := client.UpdateApplication(appConfig, true); err != nil {
			plugin.SendLog(a, "Application %s update failed:%s\n", appConfig.ID, err)
			return plugin.Fail
		}
		plugin.SendLog(a, "Application updated %s: OK\n", appConfig.ID)
	} else {
		if _, err := client.CreateApplication(appConfig); err != nil {
			plugin.SendLog(a, "Application %S creation failed :%s\n", appConfig.ID, err)
			return plugin.Fail
		}
		plugin.SendLog(a, "Application creation %s: OK\n", appConfig.ID)
	}

	//Check deployments
	if waitForDeployment {
		ticker := time.NewTicker(time.Second * 5)
		go func() {
			t0 := time.Now()
			for t := range ticker.C {
				delta := math.Floor(t.Sub(t0).Seconds())
				plugin.SendLog(a, "Application %s deployment in progress [%d seconds] please wait...\n", appConfig.ID, int(delta))
			}
		}()

		plugin.SendLog(a, "Application %s deployment in progress please wait...\n", appConfig.ID)
		deployments, err := client.ApplicationDeployments(appConfig.ID)
		if err != nil {
			plugin.SendLog(a, "Failed to list deployments : %s\n", err.Error())
			ticker.Stop()
			return plugin.Fail
		}

		if len(deployments) == 0 {
			ticker.Stop()
			return plugin.Success
		}

		wg := &sync.WaitGroup{}
		var successChan = make(chan bool, len(deployments))
		for _, deploy := range deployments {
			wg.Add(1)
			go func(id string) {
				go func() {
					time.Sleep((time.Duration(timeout) + 1) * time.Second)
					ticker.Stop()
					successChan <- false
					wg.Done()
				}()

				if err := client.WaitOnDeployment(id, time.Duration(timeout)*time.Second); err != nil {
					plugin.SendLog(a, "Error on deployment %s: %s\n", id, err.Error())
					ticker.Stop()
					successChan <- false
					wg.Done()
					return
				}

				plugin.SendLog(a, "Deployment %s succeeded", id)
				ticker.Stop()
				successChan <- true
				wg.Done()
			}(deploy.DeploymentID)
		}

		wg.Wait()

		var success = true
		for b := range successChan {
			success = success && b
			if len(successChan) == 0 {
				break
			}
		}
		ticker.Stop()
		close(successChan)

		if success {
			return plugin.Success
		}
		return plugin.Fail
	}

	return plugin.Success
}

func tmplApplicationConfigFile(a plugin.IJob, filepath string) (string, error) {
	//Read initial marathon.json file
	buff, err := ioutil.ReadFile(filepath)
	if err != nil {
		plugin.SendLog(a, "Configuration file error: %s\n", err)
		return "", err
	}

	// apply cds.var on marathon.json file
	out, errapp := plugin.ApplyArguments(a.Arguments().Data, buff)
	if errapp != nil {
		plugin.SendLog(a, "Apply cds variables error: %s\n", errapp)
		return "", errapp
	}

	// create file
	outfile, errtemp := ioutil.TempFile(os.TempDir(), "marathon.json")
	if errtemp != nil {
		plugin.SendLog(a, "Error writing temporary file: %s\n", errtemp.Error())
		return "", errtemp
	}
	outPath := outfile.Name()

	// write new content in new marathon.json
	_, errw := outfile.Write(out)
	if errw != nil {
		plugin.SendLog(a, "Error writing content to file: %s\n", errw.Error())
		return "", errw
	}
	outfile.Sync()
	outfile.Close()

	return outPath, nil
}

func parseApplicationConfigFile(a plugin.IJob, f string) (*marathon.Application, error) {
	//Read marathon.json
	buff, errf := ioutil.ReadFile(f)
	if errf != nil {
		plugin.SendLog(a, "Configuration file error: %s\n", errf)
		return nil, errf
	}

	//Parse marathon.json
	appConfig := &marathon.Application{}
	if err := json.Unmarshal(buff, appConfig); err != nil {
		plugin.SendLog(a, "Configuration file parse error: %s\n", err)
		return nil, err
	}

	//Validate with official schema : https://mesosphere.github.io/marathon/docs/generated/api.html#v2_apps_post
	wd, erro := os.Getwd()
	if erro != nil {
		plugin.SendLog(a, "Error with working directory : %s\n", erro)
		return nil, erro
	}
	schemaPath, errt := ioutil.TempFile(os.TempDir(), "marathon.schema")
	if errt != nil {
		plugin.SendLog(a, "Error marathon schema (%s) : %s\n", schemaPath.Name(), errt)
		return nil, errt
	}
	defer os.RemoveAll(schemaPath.Name())

	if err := ioutil.WriteFile(schemaPath.Name(), []byte(schema), 0644); err != nil {
		plugin.SendLog(a, "Error marathon schema : %s\n", err)
		return nil, err
	}

	var filePath = f
	if !filepath.IsAbs(f) {
		filePath = path.Join(wd, f)
	}

	schemaLoader := gojsonschema.NewReferenceLoader("file://" + schemaPath.Name())
	documentLoader := gojsonschema.NewReferenceLoader("file://" + filePath)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		plugin.SendLog(a, "Unable to validate document %s\n", err)
		return nil, err
	}
	if !result.Valid() {
		plugin.SendLog(a, "The document is not valid. see following errors\n")
		for _, desc := range result.Errors() {
			plugin.SendLog(a, " - %s", desc.Details())
		}
		return nil, fmt.Errorf("IMarathonPluginnvalid json document")
	}

	return appConfig, nil
}

func main() {
	plugin.Main(&MarathonPlugin{})
}
