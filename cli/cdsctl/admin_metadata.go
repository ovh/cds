package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"

	"github.com/spf13/cobra"
)

var adminMetadataCmd = cli.Command{
	Name:  "metadata",
	Short: "Manage CDS Metadata",
	Long: `Metadata a key/value stored on project / application / workflow.

This allows CDS administrators and/or users to make some statistics and charts in a proper tool.
	`,
}
var adminMetadataProjectCmd = cli.Command{
	Name:  "project",
	Short: "Manage CDS Project Metadata",
}
var adminMetadataApplicationCmd = cli.Command{
	Name:  "application",
	Short: "Manage CDS Application Metadata",
}
var adminMetadataWorkflowCmd = cli.Command{
	Name:  "workflow",
	Short: "Manage CDS Workflow Metadata",
}
var adminMetadataProjectExportCmd = cli.Command{
	Name:  "export",
	Short: "export CDS Project Metadata",
	Flags: []cli.Flag{
		{
			Name:    "export-file",
			Usage:   "Filename of file created",
			Default: "export_metadata_projects.csv",
		},
	},
}
var adminMetadataApplicationExportCmd = cli.Command{
	Name:  "export",
	Short: "export CDS Application Metadata",
	Flags: []cli.Flag{
		{
			Name:    "export-file",
			Usage:   "Filename of file created",
			Default: "export_metadata_applications.csv",
		},
	},
}
var adminMetadataWorkflowExportCmd = cli.Command{
	Name:  "export",
	Short: "export CDS Workflow Metadata",
	Flags: []cli.Flag{
		{
			Name:    "export-file",
			Usage:   "Filename of file created",
			Default: "export_metadata_workflows.csv",
		},
	},
}
var adminMetadataProjectImportCmd = cli.Command{
	Name:  "import",
	Short: "import CDS Project Metadata",
	Args: []cli.Arg{
		{Name: "path"},
	},
	Long: `Metadata are represented with key:value

Example of a csv file for a CDS Project
	
	project_key;project_name;last_modified;ou1;ou2
	YOUR_PROJECT_KEY;Your Project Name;2020-01-01T00:00:00;OU_1_VALUE;OU_2_VALUE

You can enter as many metadata as desired, the key name is on the first line of the csv file.
`,
}
var adminMetadataApplicationImportCmd = cli.Command{
	Name:  "import",
	Short: "import CDS Application Metadata",
	Args: []cli.Arg{
		{Name: "path"},
	},
	Long: `Metadata are represented with key:value

Example of a csv file for a CDS Application
	
	project_key;application_name;last_modified;vcs_repofullname;ou1;ou2
	YOUR_PROJECT_KEY;Your Application Name;2020-01-01T00:00:00;repo_of_application;OU_1_VALUE;OU_2_VALUE

You can enter as many metadata as desired, the key name is on the first line of the csv file.
`,
}

func adminMetadata() *cobra.Command {
	return cli.NewCommand(adminMetadataCmd, nil, []*cobra.Command{
		cli.NewCommand(adminMetadataProjectCmd, nil, []*cobra.Command{
			cli.NewCommand(adminMetadataProjectExportCmd, adminMetadataProjectExportRun, nil),
			cli.NewCommand(adminMetadataProjectImportCmd, adminMetadataProjectImportRun, nil),
		}),
		cli.NewCommand(adminMetadataApplicationCmd, nil, []*cobra.Command{
			cli.NewCommand(adminMetadataApplicationExportCmd, adminMetadataApplicationExportRun, nil),
			cli.NewCommand(adminMetadataApplicationImportCmd, adminMetadataApplicationImportRun, nil),
		}),
		cli.NewCommand(adminMetadataWorkflowCmd, nil, []*cobra.Command{
			cli.NewCommand(adminMetadataWorkflowExportCmd, adminMetadataWorkflowExportRun, nil),
		}),
	})
}

type lineMetadata struct {
	key             string
	name            string
	lastModified    time.Time
	additionalInfos sdk.Metadata
	metadata        sdk.Metadata
}

func adminMetadataProjectExportRun(c cli.Values) error {
	var currentDisplay = new(cli.Display)
	currentDisplay.Printf("Gettings projects list...")
	currentDisplay.Do(context.Background())

	projects, err := client.ProjectList(false, false)
	if err != nil {
		return err
	}

	lines := make([]lineMetadata, len(projects))
	for i, p := range projects {
		lines[i] = lineMetadata{
			key:          p.Key,
			name:         p.Name,
			lastModified: p.LastModified,
			metadata:     p.Metadata,
		}
	}

	titles := []string{"project_key", "project_name", "last_modified"}
	adminMetadataExport(titles, nil, lines, c.GetString("export-file"), currentDisplay)
	return nil
}

func adminMetadataProjectImportRun(c cli.Values) error {
	path := c.GetString("path")

	updateFunc := func(key, name string, metadata map[string]string) error {
		prj, err := client.ProjectGet(key)
		if err != nil {
			return err
		}
		prj.Metadata = metadata
		if err := client.ProjectUpdate(key, prj); err != nil {
			return err
		}
		fmt.Printf("project %s updated\n", prj.Key)
		return nil
	}

	// 3 columns to ignore: "project_key", "project_name", "last_modified", "nb_workflows"
	return processMetadata(path, 3, updateFunc)
}

func adminMetadataApplicationExportRun(c cli.Values) error {
	var currentDisplay = new(cli.Display)
	currentDisplay.Printf("Gettings projects list...")
	currentDisplay.Do(context.Background())

	projects, err := client.ProjectList(false, false)
	if err != nil {
		return err
	}

	lines := []lineMetadata{}
	for i, p := range projects {
		currentDisplay.Printf("%d/%d - fetching applications on project %s...", i, len(projects), p.Key)
		applications, err := client.ApplicationList(p.Key)
		if err != nil {
			return err
		}

		for _, a := range applications {
			m := sdk.Metadata{}
			// take all metadata from projects
			for k, v := range p.Metadata {
				m[k] = v
			}
			// then add application metadata
			for k, v := range a.Metadata {
				if _, alreadyExists := m[k]; !alreadyExists {
					m[k] = v
				}
			}

			lines = append(lines, lineMetadata{
				key:          a.ProjectKey,
				name:         a.Name,
				lastModified: a.LastModified,
				metadata:     m,
				additionalInfos: sdk.Metadata{
					"vcs_repofullname": a.RepositoryFullname,
				},
			})
		}
	}
	titles := []string{"project_key", "application_name", "last_modified"}
	titlesAdd := []string{"vcs_repofullname"}
	adminMetadataExport(titles, titlesAdd, lines, c.GetString("export-file"), currentDisplay)
	return nil
}

func adminMetadataApplicationImportRun(c cli.Values) error {
	path := c.GetString("path")

	updateFunc := func(key string, name string, metadata map[string]string) error {
		// get metadata from project
		prj, err := client.ProjectGet(key)
		if err != nil {
			return err
		}
		// remove from application metadata list, the metadata from project
		// this avoid to have the same metadata name on app and on project
		fileteredMetadata := sdk.Metadata{}
		for k, v := range metadata {
			if _, exist := prj.Metadata[k]; !exist {
				fileteredMetadata[k] = v
			}
		}
		app, err := client.ApplicationGet(key, name)
		if err != nil {
			return err
		}
		app.Metadata = fileteredMetadata
		if err := client.ApplicationUpdate(key, name, app); err != nil {
			fmt.Printf("ERROR on application %s/%s: %v\n", key, app.Name, err)
		} else {
			fmt.Printf("application %s/%s updated with metadata:%v \n", key, app.Name, app.Metadata)
		}

		return nil
	}

	return processMetadata(path, 4, updateFunc)
}

func adminMetadataWorkflowExportRun(c cli.Values) error {
	projects, err := client.ProjectList(false, false)
	if err != nil {
		return err
	}

	var currentDisplay = new(cli.Display)
	currentDisplay.Printf("Gettings projects list...")
	currentDisplay.Do(context.Background())

	modsWfs := []cdsclient.RequestModifier{}
	modsWfs = append(modsWfs, func(r *http.Request) {
		q := r.URL.Query()
		q.Set("minimal", "true")
		r.URL.RawQuery = q.Encode()
	})

	modsProjects := []cdsclient.RequestModifier{}
	modsProjects = append(modsProjects, func(r *http.Request) {
		q := r.URL.Query()
		q.Set("withWorkflowNames", "true")
		r.URL.RawQuery = q.Encode()
	})

	lines := []lineMetadata{}
	for i, p := range projects {
		currentDisplay.Printf("%d/%d - fetching project %s...", i, len(projects), p.Key)
		proj, err := client.ProjectGet(p.Key, modsProjects...)
		if err != nil {
			return err
		}

		for j, name := range proj.WorkflowNames {
			currentDisplay.Printf("%d/%d - %d/%d - fetching workflow %s/%s...", i, len(projects), j, len(proj.WorkflowNames), proj.Key, name.Name)
			w, err := client.WorkflowGet(proj.Key, name.Name, modsWfs...)
			if err != nil {
				return fmt.Errorf("Error while getting %s/%s", proj.Key, name.Name)
			}

			m := sdk.Metadata{}
			// take all metadata from projects
			for k, v := range p.Metadata {
				m[k] = v
			}
			// then add workflow metadata
			for k, v := range w.Metadata {
				if _, alreadyExists := m[k]; !alreadyExists {
					m[k] = v
				}
			}

			lines = append(lines, lineMetadata{
				key:          w.ProjectKey,
				name:         w.Name,
				lastModified: w.LastModified,
				metadata:     m,
			})
		}
	}
	titles := []string{"project_key", "workflow_name", "last_modified"}
	adminMetadataExport(titles, nil, lines, c.GetString("export-file"), currentDisplay)
	return nil
}

func adminMetadataExport(firstTitles, addTitles []string, lines []lineMetadata, filename string, currentDisplay *cli.Display) {
	keysTitle := map[string]string{}
	for _, l := range lines {
		for k := range l.metadata {
			keysTitle[k] = ""
		}
	}

	// sort the title keys
	keys := make([]string, 0, len(keysTitle))
	for k := range keysTitle {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// sort the add title keys
	sort.Strings(addTitles)

	currentDisplay.Printf("Generating file %s...", filename)

	f, err := os.Create(filename)
	if err != nil {
		fmt.Fprintf(os.Stdout, "Error while creating file %s: %v", filename, err)
		return
	}
	defer f.Close()

	// prepare header
	writeLine(f, strings.Join(firstTitles, ";"))
	for _, k := range addTitles {
		writeLine(f, fmt.Sprintf(";%s", k))
	}
	for _, k := range keys {
		writeLine(f, fmt.Sprintf(";%s", k))
	}
	writeLine(f, fmt.Sprintf("\n"))

	for _, l := range lines {
		ptime := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
			l.lastModified.Year(), l.lastModified.Month(), l.lastModified.Day(),
			l.lastModified.Hour(), l.lastModified.Minute(), l.lastModified.Second())

		writeLine(f, fmt.Sprintf("%s;%s;%s", l.key, l.name, ptime))

		for _, k := range addTitles {
			writeLine(f, fmt.Sprintf(";%s", l.additionalInfos[k]))
		}

		// if metadata key exists, print it
		nMetadataWrite := 0
		for _, k := range keys {
			if v, exists := l.metadata[k]; exists {
				nMetadataWrite++
				writeLine(f, fmt.Sprintf(";%s", v))
			} else {
				writeLine(f, ";")
			}
		}
		writeLine(f, fmt.Sprintf("\n"))
		fmt.Printf("")
	}
	currentDisplay.Printf("file %s created...\n", filename)
	// sleep 2s to let display the currentDisplay
	time.Sleep(2 * time.Second)
}

func writeLine(fi *os.File, s string) {
	if _, err := io.WriteString(fi, s); err != nil {
		fmt.Fprintf(os.Stdout, "error while writing file: %v\n", err)
	}
}

func processMetadata(path string, nbColumnsToIgnore int, updateFunc func(key, name string, metadata map[string]string) error) error {
	csvFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer csvFile.Close() //nolint

	reader := csv.NewReader(bufio.NewReader(csvFile))

	metadataKeys := map[int]string{}

	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		columns := strings.Split(line[0], ";")
		pkey := columns[0]
		name := columns[1]

		// first line: title
		if len(metadataKeys) == 0 {
			// project_key, name, last_update. Metadata begins at 3th position
			for index := nbColumnsToIgnore; index < len(columns); index++ {
				metadataKeys[index-nbColumnsToIgnore] = columns[index]
			}
			continue
		}

		metadata := make(map[string]string, len(metadataKeys))

		for index := 0; index < len(metadataKeys); index++ {
			if index > len(metadataKeys) || nbColumnsToIgnore+index >= len(columns) {
				return fmt.Errorf("CSV File invalid. Please check number of columns on %s;%s", pkey, name)
			}
			metadata[metadataKeys[index]] = columns[nbColumnsToIgnore+index]
		}
		if err := updateFunc(pkey, name, metadata); err != nil {
			return err
		}
	}
	return nil
}
