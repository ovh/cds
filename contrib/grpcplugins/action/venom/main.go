package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ovh/venom"
	"github.com/ovh/venom/context/default"
	"github.com/ovh/venom/context/redis"
	"github.com/ovh/venom/context/webctx"
	"github.com/ovh/venom/executors/dbfixtures"
	"github.com/ovh/venom/executors/exec"
	"github.com/ovh/venom/executors/http"
	"github.com/ovh/venom/executors/imap"
	"github.com/ovh/venom/executors/ovhapi"
	"github.com/ovh/venom/executors/readfile"
	"github.com/ovh/venom/executors/redis"
	"github.com/ovh/venom/executors/smtp"
	"github.com/ovh/venom/executors/ssh"
	"github.com/ovh/venom/executors/web"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

/*
$ make build
$ make publish
*/

type venomActionPlugin struct {
	actionplugin.Common
}

func (actPlugin *venomActionPlugin) Manifest(ctx context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:   "plugin-venom",
		Author: "Yvonnick Esnault <yvonnick.esnault@corp.ovh.com>",
		Description: `This plugin helps you to run venom. Venom: https://github.com/ovh/venom.

	Add an extra step of type junit on your job to view tests results on CDS UI.`,
		Version: sdk.VERSION,
	}, nil
}

func (actPlugin *venomActionPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	// Parse parameters
	path := q.GetOptions()["path"]
	exclude := q.GetOptions()["exclude"]
	output := q.GetOptions()["output"]
	parallelS := q.GetOptions()["parallel"]
	loglevel := q.GetOptions()["loglevel"]
	vars := q.GetOptions()["vars"]
	varsFromFile := q.GetOptions()["vars-from-file"]

	if path == "" {
		path = "."
	}

	parallel, err := strconv.Atoi(parallelS)
	if err != nil {
		fmt.Printf("VENOM - parallel arg must be an integer\n")
		return &actionplugin.ActionResult{
			Status: sdk.StatusSuccess.String(),
		}, nil
	}

	v := venom.New()
	v.RegisterExecutor(exec.Name, exec.New())
	v.RegisterExecutor(http.Name, http.New())
	v.RegisterExecutor(imap.Name, imap.New())
	v.RegisterExecutor(ovhapi.Name, ovhapi.New())
	v.RegisterExecutor(readfile.Name, readfile.New())
	v.RegisterExecutor(redis.Name, redis.New())
	v.RegisterExecutor(smtp.Name, smtp.New())
	v.RegisterExecutor(ssh.Name, ssh.New())
	v.RegisterExecutor(web.Name, web.New())
	v.RegisterExecutor(dbfixtures.Name, dbfixtures.New())
	v.RegisterTestCaseContext(defaultctx.Name, defaultctx.New())
	v.RegisterTestCaseContext(webctx.Name, webctx.New())
	v.RegisterTestCaseContext(redisctx.Name, redisctx.New())

	v.PrintFunc = func(format string, aa ...interface{}) (n int, err error) {
		fmt.Printf(format, aa)
		return 0, nil
	}

	start := time.Now()
	data := make(map[string]string)
	if vars == "" {
		// no vars -> all .cds... variables can by used in yml
		data = q.GetOptions()
	} else {
		// if vars is not empty
		// vars could be:
		// cds.foo.bar,cds.foo2.bar2
		// cds.foo.bar,cds.foo2.bar2,anotherVars=foo,anotherVars2=bar
		for _, v := range strings.Split(vars, ",") {
			t := strings.Split(v, "=")
			if len(t) > 1 {
				// if value of current var is setted, we take it
				data[t[0]] = t[1]
				fmt.Printf("VENOM - var %s has value %s\n", t[0], t[1])
			} else if len(t) == 1 && strings.HasPrefix(v, "cds.") {
				fmt.Printf("VENOM - try fo find var %s in cds variables\n", v)
				// if var starts with .cds, we try to take value from current CDS variables
				for k := range q.GetOptions() {
					if k == v {
						fmt.Printf("VENOM - var %s is found with value %s\n", v, q.GetOptions()[k])
						data[k] = q.GetOptions()[k]
						break
					}
				}
			}
		}

		//If we use the var list, it means we do pretty hacky stuffs, so let's ignore all cds vars
		v.IgnoreVariables = append(v.IgnoreVariables, "cds", "workflow", "git")
	}

	if varsFromFile != "" {
		varFileMap := make(map[string]string)
		bytes, err := ioutil.ReadFile(varsFromFile)
		if err != nil {
			return fail("VENOM - Error while reading file: %v\n", err)
		}
		switch filepath.Ext(varsFromFile) {
		case ".json":
			err = json.Unmarshal(bytes, &varFileMap)
		case ".yaml":
			err = yaml.Unmarshal(bytes, &varFileMap)
		default:
			return fail("VENOM - unsupported varFile format")
		}

		if err != nil {
			return fail("VENOM - Error on unmarshal file: %v\n", err)
		}

		for key, value := range varFileMap {
			data[key] = value
		}
	}

	v.AddVariables(data)
	v.LogLevel = loglevel
	v.OutputFormat = "xml"
	v.OutputDir = output
	v.Parallel = parallel

	filepath := strings.Split(path, ",")
	filepathExcluded := strings.Split(exclude, ",")

	if len(filepath) == 1 {
		filepath = strings.Split(filepath[0], " ")
	}

	for _, fp := range filepath {
		filepath, err = walkGlobFile(fp)
		if err != nil {
			return fail("VENOM - Error on walk files: %v\n", err)
		}
	}

	if len(filepathExcluded) == 1 {
		filepathExcluded = strings.Split(filepathExcluded[0], " ")
	}

	for _, fp := range filepathExcluded {
		filepathExcluded, err = walkGlobFile(fp)
		if err != nil {
			return fail("VENOM - Error on walk excluded files: %v\n", err)
		}
	}

	tests, err := v.Process(filepath, filepathExcluded)
	if err != nil {
		return fail("VENOM - Fail on venom: %v\n", err)
	}

	elapsed := time.Since(start)
	fmt.Printf("VENOM - Output test results under: %s\n", output)
	if err := v.OutputResult(*tests, elapsed); err != nil {
		return fail("VENOM - Error while uploading test results: %v\n", err)
	}

	return &actionplugin.ActionResult{
		Status: sdk.StatusSuccess.String(),
	}, nil
}

func (actPlugin *venomActionPlugin) WorkerHTTPPort(ctx context.Context, q *actionplugin.WorkerHTTPPortQuery) (*empty.Empty, error) {
	actPlugin.HTTPPort = q.Port
	return &empty.Empty{}, nil
}

func main() {
	actPlugin := venomActionPlugin{}
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

func walkGlobFile(path string) ([]string, error) {
	filenames, err := filepath.Glob(path)
	if err != nil {
		return nil, err
	}
	result := []string{}
	for _, fpath := range filenames {
		err := filepath.Walk(fpath,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					result = append(result, path)
				}
				return nil
			})
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
