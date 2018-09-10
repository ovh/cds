package main

import (
	"context"
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
	"strings"
	"sync"
	"time"

	"github.com/facebookgo/httpcontrol"
	marathon "github.com/gambol99/go-marathon"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/xeipuuv/gojsonschema"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
	"github.com/ovh/cds/sdk/interpolate"
)

/*
$ make build
$ make publish
*/

type marathonActionPlugin struct {
	actionplugin.Common
}

func (actPlugin *marathonActionPlugin) Manifest(ctx context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:   "plugin-marathon",
		Author: "François SAMIN <francois.samin@corp.ovh.com>",
		Description: `This action helps you generates a file using a template file and text/template golang package.

	Check documentation on text/template for more information https://golang.org/pkg/text/template.`,
		Version: sdk.VERSION,
	}, nil
}

func (actPlugin *marathonActionPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	//Get arguments
	URL := q.GetOptions()["url"]
	user := q.GetOptions()["user"]
	password := q.GetOptions()["password"]
	tmplConf := q.GetOptions()["configuration"]
	waitForDeploymentStr := q.GetOptions()["waitForDeployment"]
	insecureSkipVerifyStr := q.GetOptions()["insecureSkipVerify"]
	timeoutStr := q.GetOptions()["timeout"]

	//Parse arguments
	waitForDeployment, err := strconv.ParseBool(waitForDeploymentStr)
	if err != nil {
		return fail("Error parsing waitForDeployment value : %s\n", err.Error())
	}

	insecureSkipVerify := false
	if insecureSkipVerifyStr != "" {
		var errb error
		insecureSkipVerify, errb = strconv.ParseBool(insecureSkipVerifyStr)
		if err != nil {
			return fail("Error parsing insecureSkipVerify value : %s\n", errb.Error())
		}
	}

	if insecureSkipVerify {
		fmt.Printf("You are using insecureSkipVerify flag to true. It is not recommended\n")
	}

	timeout, err := strconv.Atoi(timeoutStr)
	if err != nil {
		return fail("Error parsing timeout value :  %s\n", err.Error())
	}

	//Custom http client with 3 retries
	httpClient := &http.Client{
		Transport: &httpcontrol.Transport{
			RequestTimeout:  time.Minute,
			MaxTries:        3,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerify},
		},
	}

	fmt.Printf("Connecting on %s\n", URL)
	config := marathon.NewDefaultConfig()
	config.URL = URL
	config.HTTPBasicAuthUser = user
	config.HTTPBasicPassword = password
	config.HTTPClient = httpClient

	//Connecting to marathon
	client, err := marathon.NewClient(config)
	if err != nil {
		return fail("Connection failed on %s\n", URL)
	}

	//Run tmpl on configuration file to replace all cds variables
	conf, err := tmplApplicationConfigFile(q, tmplConf)
	if err != nil {
		return fail("Templating Configuration File KO (tmplApplicationConfigFile): %s\n", err.Error())
	}
	defer os.RemoveAll(conf)
	fmt.Printf("Templating Configuration File: OK\n")

	//Validate json file and load application
	appConfig, err := parseApplicationConfigFile(conf)
	if err != nil {
		return fail("Templating Configuration File KO (parseApplicationConfigFile): %s\n", err.Error())
	}
	fmt.Printf("Parsing Configuration File: OK\n")

	//Allways put cds.version labels
	if appConfig.Labels == nil {
		appConfig.Labels = &map[string]string{}
	}

	(*appConfig.Labels)["CDS_VERSION"] = q.GetOptions()["cds.version"]
	(*appConfig.Labels)["CDS_PROJECT"] = q.GetOptions()["cds.project"]
	(*appConfig.Labels)["CDS_APPLICATION"] = q.GetOptions()["cds.application"]
	(*appConfig.Labels)["CDS_ENVIRONMENT"] = q.GetOptions()["cds.environment"]

	cdsWorkflow := q.GetOptions()["cds.workflow"]
	if cdsWorkflow != "" {
		(*appConfig.Labels)["CDS_WORKFLOW"] = cdsWorkflow
	}

	cdsRunNumber := q.GetOptions()["cds.run"]
	if cdsRunNumber != "" {
		(*appConfig.Labels)["CDS_RUN"] = cdsRunNumber
	}

	gitRepository := q.GetOptions()["git.repository"]
	if gitRepository != "" {
		(*appConfig.Labels)["CDS_GIT_REPOSITORY"] = gitRepository
	}

	gitBranch := q.GetOptions()["git.branch"]
	if gitBranch != "" {
		(*appConfig.Labels)["CDS_GIT_BRANCH"] = gitBranch
	}

	gitHash := q.GetOptions()["git.hash"]
	if gitHash != "" {
		(*appConfig.Labels)["CDS_GIT_HASH"] = gitHash
	}

	fmt.Printf("Configuration File %s: OK\n", tmplConf)

	//Searching for application
	val := url.Values{"id": []string{appConfig.ID}}
	applications, err := client.Applications(val)
	if err != nil {
		return fail("Failed to list applications: %s\n", err.Error())
	}

	var appExists bool
	if len(applications.Apps) != 0 {
		appExists = true
	}

	//Update or create application
	if appExists {
		if _, err := client.UpdateApplication(appConfig, true); err != nil {
			return fail("Application %s update failed:%s\n", appConfig.ID, err)
		}
		fmt.Printf("Application updated %s: OK\n", appConfig.ID)
	} else {
		if _, err := client.CreateApplication(appConfig); err != nil {
			return fail("Application %s creation failed:%s\n", appConfig.ID, err)
		}
		fmt.Printf("Application creation %s: OK\n", appConfig.ID)
	}

	//Check deployments
	if waitForDeployment {
		ticker := time.NewTicker(time.Second * 5)
		go func() {
			t0 := time.Now()
			for t := range ticker.C {
				delta := math.Floor(t.Sub(t0).Seconds())
				fmt.Printf("Application %s deployment in progress [%d seconds] please wait...\n", appConfig.ID, int(delta))
			}
		}()

		fmt.Printf("Application %s deployment in progress please wait...\n", appConfig.ID)
		deployments, err := client.ApplicationDeployments(appConfig.ID)
		if err != nil {
			ticker.Stop()
			return fail("Failed to list deployments : %s\n", err.Error())
		}

		if len(deployments) == 0 {
			ticker.Stop()
			return &actionplugin.ActionResult{
				Status: sdk.StatusSuccess.String(),
			}, nil
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
					fmt.Printf("Error on deployment %s: %s\n", id, err.Error())
					ticker.Stop()
					successChan <- false
					wg.Done()
					return
				}

				fmt.Printf("Deployment %s succeeded", id)
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
			return &actionplugin.ActionResult{
				Status: sdk.StatusSuccess.String(),
			}, nil
		}
		return fail("")
	}

	return &actionplugin.ActionResult{
		Status: sdk.StatusSuccess.String(),
	}, nil
}

func (actPlugin *marathonActionPlugin) WorkerHTTPPort(ctx context.Context, q *actionplugin.WorkerHTTPPortQuery) (*empty.Empty, error) {
	actPlugin.HTTPPort = q.Port
	return &empty.Empty{}, nil
}

func main() {
	actPlugin := marathonActionPlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
	return

}

func fail(format string, args ...interface{}) (*actionplugin.ActionResult, error) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(msg)
	return &actionplugin.ActionResult{
		Details: msg,
		Status:  sdk.StatusFail.String(),
	}, nil
}

// parseTemplateParameters parses a list of key value pairs separated by new lines
func parseTemplateParameters(s string) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	for _, l := range strings.Split(s, "\n") {
		components := strings.SplitN(l, "=", 2)
		if len(components) != 2 {
			return nil, fmt.Errorf("invalid key value pair form for %q", l)
		}
		params[components[0]] = components[1]
	}

	return params, nil
}

func tmplApplicationConfigFile(q *actionplugin.ActionQuery, filepath string) (string, error) {
	//Read initial marathon.json file
	buff, err := ioutil.ReadFile(filepath)
	if err != nil {
		fmt.Printf("Configuration file error: %s\n", err)
		return "", err
	}

	// apply cds.var on marathon.json file
	out, errapp := interpolate.Do(string(buff), q.GetOptions())
	if errapp != nil {
		fmt.Printf("Apply cds variables error: %s\n", errapp)
		return "", errapp
	}

	// create file
	outfile, errtemp := ioutil.TempFile(os.TempDir(), "marathon.json")
	if errtemp != nil {
		fmt.Printf("Error writing temporary file: %s\n", errtemp.Error())
		return "", errtemp
	}
	outPath := outfile.Name()

	// write new content in new marathon.json
	if _, errw := outfile.Write([]byte(out)); errw != nil {
		fmt.Printf("Error writing content to file: %s\n", errw.Error())
		return "", errw
	}
	outfile.Sync()
	outfile.Close()

	return outPath, nil
}

func parseApplicationConfigFile(f string) (*marathon.Application, error) {
	//Read marathon.json
	buff, errf := ioutil.ReadFile(f)
	if errf != nil {
		fmt.Printf("Configuration file error: %s\n", errf)
		return nil, errf
	}

	//Parse marathon.json
	appConfig := &marathon.Application{}
	if err := json.Unmarshal(buff, appConfig); err != nil {
		fmt.Printf("Configuration file parse error: %s\n", err)
		return nil, err
	}

	//Validate with official schema : https://mesosphere.github.io/marathon/docs/generated/api.html#v2_apps_post
	wd, erro := os.Getwd()
	if erro != nil {
		fmt.Printf("Error with working directory : %s\n", erro)
		return nil, erro
	}
	schemaPath, errt := ioutil.TempFile(os.TempDir(), "marathon.schema")
	if errt != nil {
		fmt.Printf("Error marathon schema (%s) : %s\n", schemaPath.Name(), errt)
		return nil, errt
	}
	defer os.RemoveAll(schemaPath.Name())

	if err := ioutil.WriteFile(schemaPath.Name(), []byte(schema), 0644); err != nil {
		fmt.Printf("Error marathon schema : %s\n", err)
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
		fmt.Printf("Unable to validate document %s\n", err)
		return nil, err
	}
	if result == nil {
		fmt.Printf("Unable to validate document (result validate is nil)\n")
		return nil, fmt.Errorf("Unable to validate document (result validate is nil)")
	}
	if !result.Valid() {
		fmt.Printf("The document is not valid. see following errors\n")
		for _, desc := range result.Errors() {
			fmt.Printf(" - %s", desc.Details())
		}
		return nil, fmt.Errorf("IMarathonPlugin invalid json document")
	}

	return appConfig, nil
}
