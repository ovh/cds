package main

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/fsamin/go-dump"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var (
	workflowCmd = cli.Command{
		Name:  "workflow",
		Short: "Manage CDS workflow",
	}

	workflow = cli.NewCommand(workflowCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(workflowListCmd, workflowListRun, nil),
			cli.NewListCommand(workflowHistoryCmd, workflowHistoryRun, nil),
			cli.NewGetCommand(workflowShowCmd, workflowShowRun, nil),
			cli.NewDeleteCommand(workflowDeleteCmd, workflowDeleteRun, nil),
			cli.NewCommand(workflowRunManualCmd, workflowRunManualRun, nil),
			cli.NewCommand(workflowExportCmd, workflowExportRun, nil),
			cli.NewCommand(workflowImportCmd, workflowImportRun, nil),
			cli.NewCommand(workflowPullCmd, workflowPullRun, nil),
			cli.NewCommand(workflowPushCmd, workflowPushRun, nil),
			workflowArtifact,
		})
)

var workflowListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS workflows",
	Args: []cli.Arg{
		{Name: "project-key"},
	},
}

func workflowListRun(v cli.Values) (cli.ListResult, error) {
	w, err := client.WorkflowList(v["project-key"])
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(w), nil
}

var workflowHistoryCmd = cli.Command{
	Name:  "history",
	Short: "History of a CDS workflow",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "workflow-name"},
	},
	OptionalArgs: []cli.Arg{
		{
			Name: "offset",
			IsValid: func(s string) bool {
				match, _ := regexp.MatchString(`[0-9]?`, s)
				return match
			},
			Weight: 1,
		},
		{
			Name: "limit",
			IsValid: func(s string) bool {
				match, _ := regexp.MatchString(`[0-9]?`, s)
				return match
			},
			Weight: 2,
		},
	},
}

func workflowHistoryRun(v cli.Values) (cli.ListResult, error) {
	var offset int64
	if v.GetString("offset") != "" {
		var errn error
		offset, errn = v.GetInt64("offset")
		if errn != nil {
			return nil, errn
		}
	}

	var limit int64
	if v.GetString("limit") != "" {
		var errl error
		limit, errl = v.GetInt64("limit")
		if errl != nil {
			return nil, errl
		}
	}

	w, err := client.WorkflowRunList(v["project-key"], v["workflow-name"], offset, limit)
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(w), nil
}

var workflowShowCmd = cli.Command{
	Name:  "show",
	Short: "Show a CDS workflow",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "workflow-name"},
	},
	OptionalArgs: []cli.Arg{
		{
			Name: "run-number",
			IsValid: func(s string) bool {
				match, _ := regexp.MatchString(`[0-9]?`, s)
				return match
			},
			Weight: 1,
		},
	},
}

func workflowShowRun(v cli.Values) (interface{}, error) {
	var runNumber int64
	if v.GetString("run-number") != "" {
		var errl error
		runNumber, errl = v.GetInt64("run-number")
		if errl != nil {
			return nil, errl
		}
	}

	if runNumber == 0 {
		w, err := client.WorkflowGet(v["project-key"], v["workflow-name"])
		if err != nil {
			return nil, err
		}
		return *w, nil
	}

	w, err := client.WorkflowRunGet(v["project-key"], v["workflow-name"], runNumber)
	if err != nil {
		return nil, err
	}

	var tags []string
	for _, tag := range w.Tags {
		tags = append(tags, fmt.Sprintf("%s:%s", tag.Tag, tag.Value))
	}

	type wtags struct {
		sdk.WorkflowRun
		Payload string `cli:"payload"`
		Tags    string `cli:"tags"`
	}

	var payload []string
	if v, ok := w.WorkflowNodeRuns[w.Workflow.RootID]; ok {
		if len(v) > 0 {
			e := dump.NewDefaultEncoder(new(bytes.Buffer))
			e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
			e.ExtraFields.DetailedMap = false
			e.ExtraFields.DetailedStruct = false
			e.ExtraFields.Len = false
			e.ExtraFields.Type = false
			pl, errm1 := e.ToStringMap(v[0].Payload)
			if errm1 != nil {
				return nil, errm1
			}
			for k, kv := range pl {
				payload = append(payload, fmt.Sprintf("%s:%s", k, kv))
			}
			payload = append(payload)
		}
	}

	wt := &wtags{*w, strings.Join(payload, " "), strings.Join(tags, " ")}
	return *wt, nil
}

var workflowDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a CDS workflow",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "workflow-name"},
	},
}

func workflowDeleteRun(v cli.Values) error {
	err := client.WorkflowDelete(v["project-key"], v["workflow-name"])
	if err != nil && v.GetBool("force") && sdk.ErrorIs(err, sdk.ErrWorkflowNotFound) {
		fmt.Println(err.Error())
		os.Exit(0)
	}
	return err
}

var workflowRunManualCmd = cli.Command{
	Name:  "run",
	Short: "Run a CDS workflow",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "workflow-name"},
	},
	OptionalArgs: []cli.Arg{
		{Name: "payload"},
	},
	Flags: []cli.Flag{
		{
			Name:  "run-number",
			Usage: "Existing Workflow RUN Number",
			IsValid: func(s string) bool {
				match, _ := regexp.MatchString(`[0-9]?`, s)
				return match
			},
			Kind: reflect.String,
		},
		{
			Name:  "node-name",
			Usage: "Node Name to relaunch; Flag run-number is mandatory",
			Kind:  reflect.String,
		},
		{
			Name:      "interactive",
			ShortHand: "i",
			Usage:     "Follow the workflow run in an interactive terminal user interface",
			Kind:      reflect.Bool,
		},
		{
			Name:      "open-web-browser",
			ShortHand: "o",
			Usage:     "Open web browser on the workflow run",
			Kind:      reflect.Bool,
		},
	},
}

func workflowRunManualRun(v cli.Values) error {
	manual := sdk.WorkflowNodeRunManual{}
	if v["payload"] != "" {
		manual.Payload = v["payload"]
	}

	var runNumber, fromNodeID int64

	if v.GetString("run-number") != "" {
		var errp error
		runNumber, errp = strconv.ParseInt(v.GetString("run-number"), 10, 64)
		if errp != nil {
			return fmt.Errorf("run-number invalid: not a integer")
		}
	}

	if v.GetString("node-name") != "" {
		if runNumber <= 0 {
			return fmt.Errorf("You can use flag node-name without flag run-number")
		}
		wr, err := client.WorkflowRunGet(v["project-key"], v["workflow-name"], runNumber)
		if err != nil {
			return err
		}
		for _, wnrs := range wr.WorkflowNodeRuns {
			for _, wnr := range wnrs {
				wn := wr.Workflow.GetNode(wnr.WorkflowNodeID)
				if wn.Name == v.GetString("node-name") {
					fromNodeID = wnr.WorkflowNodeID
					break
				}
			}
		}
	}

	w, err := client.WorkflowRunFromManual(v["project-key"], v["workflow-name"], manual, runNumber, fromNodeID)
	if err != nil {
		return err
	}

	fmt.Printf("Workflow %s #%d has been lauched\n", v["workflow-name"], w.Number)

	var baseURL string
	configUser, err := client.ConfigUser()
	if err != nil {
		return err
	}

	if b, ok := configUser[sdk.ConfigURLUIKey]; ok {
		baseURL = b
	}

	if baseURL == "" {
		fmt.Println("Unable to retrieve workflow URI")
		return nil
	}

	if !v.GetBool("interactive") {
		url := fmt.Sprintf("%s/project/%s/workflow/%s/run/%d", baseURL, v["project-key"], v["workflow-name"], w.Number)
		fmt.Println(url)

		if v.GetBool("open-web-browser") {
			return browser.OpenURL(url)
		}

		return nil
	}

	return workflowRunInteractive(v, w, baseURL)
}

var workflowExportCmd = cli.Command{
	Name:  "export",
	Short: "Export a workflow",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "workflow-name"},
	},
	Flags: []cli.Flag{
		{
			Kind:    reflect.Bool,
			Name:    "with-permissions",
			Usage:   "Export permissions",
			Default: "false",
		},
		{
			Kind:    reflect.String,
			Name:    "format",
			Usage:   "Specify export format (json or yaml)",
			Default: "yaml",
		},
	},
}

func workflowExportRun(c cli.Values) error {
	btes, err := client.WorkflowExport(c.GetString("project-key"), c.GetString("workflow-name"), c.GetBool("with-permissions"), c.GetString("format"))
	if err != nil {
		return err
	}
	fmt.Println(string(btes))
	return nil
}

var workflowPullCmd = cli.Command{
	Name:  "pull",
	Short: "Pull a workflow",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "workflow-name"},
	},
	Flags: []cli.Flag{
		{
			Kind:      reflect.String,
			Name:      "output-dir",
			ShortHand: "d",
			Usage:     "Output directory",
			Default:   ".cds",
		},
		{
			Kind:    reflect.Bool,
			Name:    "with-permissions",
			Usage:   "Export permissions",
			Default: "false",
		},
		{
			Kind:    reflect.Bool,
			Name:    "force",
			Usage:   "Force, may override files",
			Default: "false",
		},
		{
			Kind:    reflect.Bool,
			Name:    "quiet",
			Usage:   "If true, do not output filename created",
			Default: "false",
		},
	},
}

func workflowPullRun(c cli.Values) error {
	dir := strings.TrimSpace(c.GetString("output-dir"))
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, os.FileMode(0744)); err != nil {
		return fmt.Errorf("Unable to create directory %s: %v", c.GetString("output-dir"), err)
	}

	tr, err := client.WorkflowPull(c.GetString("project-key"), c.GetString("workflow-name"), c.GetBool("with-permissions"))
	if err != nil {
		return err
	}

	// Iterate through the files in the archive.
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		fname := filepath.Join(dir, hdr.Name)
		if _, err = os.Stat(fname); err == nil || os.IsExist(err) {
			if !c.GetBool("force") {
				if !cli.AskForConfirmation(fmt.Sprintf("This will override %s. Do you want to continue?", fname)) {
					os.Exit(0)
				}
			}
		}

		if verbose {
			fmt.Println("Creating file", fname)
		}
		fi, err := os.Create(fname)
		if err != nil {
			return err
		}
		if _, err := io.Copy(fi, tr); err != nil {
			return err
		}
		if err := fi.Close(); err != nil {
			return err
		}
		if !c.GetBool("quiet") {
			fmt.Println(fname)
		}
	}
	return nil
}

var workflowImportCmd = cli.Command{
	Name:  "import",
	Short: "Import a workflow",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "filename"},
	},
	Flags: []cli.Flag{
		{
			Kind:    reflect.Bool,
			Name:    "force",
			Usage:   "Override workflow if exists",
			Default: "false",
		},
	},
}

func workflowImportRun(c cli.Values) error {
	path := c.GetString("filename")
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var format = "yaml"
	if strings.HasSuffix(path, ".json") {
		format = "json"
	}

	msgs, err := client.WorkflowImport(c.GetString("project-key"), f, format, c.GetBool("force"))
	if err != nil {
		return err
	}

	for _, s := range msgs {
		fmt.Println(s)
	}

	return nil
}

var workflowPushCmd = cli.Command{
	Name:  "push",
	Short: "Push a workflow",
	Args: []cli.Arg{
		{Name: "project-key"},
	},
	VariadicArgs: cli.Arg{
		Name: "yaml-file",
	},
}

func workflowPushRun(c cli.Values) error {
	// Get the file
	files := strings.Split(c.GetString("yaml-file"), ",")

	// Create a buffer to write our archive to.
	buf := new(bytes.Buffer)

	// Create a new tar archive.
	tw := tar.NewWriter(buf)

	// Add some files to the archive.
	for _, file := range files {
		fmt.Println("Reading file ", file)
		filBuf, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}

		hdr := &tar.Header{
			Name: filepath.Base(file),
			Mode: 0600,
			Size: int64(len(filBuf)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if n, err := tw.Write(filBuf); err != nil {
			return err
		} else if n == 0 {
			return fmt.Errorf("nothing to write")
		}
	}
	// Make sure to check the error on Close.
	if err := tw.Close(); err != nil {
		return err
	}

	// Open the tar archive for reading.
	btes := buf.Bytes()
	r := bytes.NewBuffer(btes)

	// Push it !
	msgList, err := client.WorkflowPush(c.GetString("project-key"), r)
	for _, msg := range msgList {
		fmt.Println(msg)
	}

	return err
}
