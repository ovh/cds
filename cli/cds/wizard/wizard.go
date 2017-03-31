package wizard

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/howeyc/gopass"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

// Cmd dashboard
var Cmd = &cobra.Command{
	Use:     "wizard",
	Short:   "Interactive application configuration wizard",
	Long:    "",
	Aliases: []string{"wiz"},
	Run: func(cmd *cobra.Command, args []string) {
		runWizard()
	},
}

func runWizard() {

	fmt.Printf("Application name: ")
	appName := readline()

	project := choseProject(appName)
	buildTmpl := choseBuildTemplate()
	buildTmpl = choseBuildTemplateParameters(buildTmpl)

	applyTemplate(project.Key, appName, buildTmpl)
}

func applyTemplate(projectKey string, name string, build sdk.Template) {
	app, err := sdk.ApplyApplicationTemplate(projectKey, name, build)
	if err != nil {
		sdk.Exit("Error: Cannot apply template (%s)\n", err)
	}

	fmt.Printf("\n")
	fmt.Printf("Application %s/%s created\n", projectKey, app.Name)

	var pip sdk.Pipeline
	for i := range app.Pipelines {
		if app.Pipelines[i].Pipeline.Type == sdk.BuildPipeline {
			pip = app.Pipelines[i].Pipeline
			break
		}
	}
	if pip.ID > 0 {
		fmt.Printf("\n")
		fmt.Printf("Do you want to run %s/%s/%s now ? [Y/n]: ", projectKey, app.Name, pip.Name)
		run := readline()
		if run == "Y" || run == "y" || run == "yes" || run == "" {
			fmt.Printf("Running %s/%s/%s:\n", projectKey, app.Name, pip.Name)
			ch, err := sdk.RunPipeline(projectKey, app.Name, pip.Name, "NoEnv", true, sdk.RunRequest{}, false)
			if err != nil {
				sdk.Exit("Error: Cannot start build (%s)\n", err)
			}

			w := tabwriter.NewWriter(os.Stdout, 27, 1, 2, ' ', 0)
			titles := []string{"DATE", "ACTION", "LOG"}
			fmt.Fprintln(w, strings.Join(titles, "\t"))

			for l := range ch {
				fmt.Fprintf(w, "%s\t%d\t%s",
					[]byte(l.LastModified.String())[:19],
					l.StepOrder,
					l.Val,
				)
				w.Flush()

				// Exit 1 if pipeline fail
				if l.Id == 0 && strings.Contains(l.Val, "status: Fail") {
					sdk.Exit("")
				}
			}
		}
	}
}

func choseDeploymentTemplate() sdk.Template {
	var t sdk.Template

	fmt.Printf("\nDo you need a deployment pipeline ? [Y/n]: ")
	a := readline()
	if a == "n" || a == "no" {
		return t
	}

	for {
		tmpls, err := sdk.GetDeploymentTemplates()
		if err != nil {
			sdk.Exit("Error: cannot retrieve deployment templates (%s)\n", err)
		}
		fmt.Printf("List of existing deployment templates:\n")
		for i, t := range tmpls {
			fmt.Printf("-[%d] %-15s (%s)\n", i+1, t.Name, t.Description)
		}
		fmt.Printf("which deployment template do you want to apply [1-%d]: ", len(tmpls))
		tmplIDS := readline()
		tmplID, err := strconv.Atoi(tmplIDS)
		if err != nil {
			fmt.Printf("Error: %s is not a valid number\n", tmplIDS)
			continue
		}
		if tmplID > len(tmpls) || tmplID <= 0 {
			fmt.Printf("Error: %s is not a valid number", tmplIDS)
			continue
		}
		return tmpls[tmplID-1]
	}

}

func choseDeploymentTemplateParameters(tmpl sdk.Template) sdk.Template {
	if tmpl.ID == 0 || len(tmpl.Params) == 0 {
		return tmpl
	}

	fmt.Printf("\nTemplate %s parameters:\n", tmpl.Name)
	for i, p := range tmpl.Params {
		if p.Type == sdk.SecretVariable {
			fmt.Printf("- %-15s (%s) [no echo]: ", p.Name, p.Description)
			pass, err := gopass.GetPasswd()
			if err != nil {
				fmt.Printf("\nError: cannot read password (%s)\n", err)
			}
			tmpl.Params[i].Value = string(pass)
		} else {
			fmt.Printf("- %-15s (%s) [%s]: ", p.Name, p.Description, p.Type)
			tmpl.Params[i].Value = readline()
		}
	}

	return tmpl
}

func getParamValue(p sdk.TemplateParam) string {
	var val string

	for {
		if p.Type == sdk.SecretVariable {
			fmt.Printf("- %-15s (%s) [no echo]: ", p.Name, p.Description)
			pass, err := gopass.GetPasswd()
			if err != nil {
				fmt.Printf("\nError: cannot read password (%s)\n", err)
			}
			val = string(pass)
		} else {
			fmt.Printf("- %-15s (%s) [%s]: ", p.Name, p.Description, p.Type)
			val = readline()
		}

		switch p.Type {
		case sdk.BooleanVariable:
			if strings.ToLower(val) == "y" ||
				strings.ToLower(val) == "yes" ||
				strings.ToLower(val) == "n" ||
				strings.ToLower(val) == "no" {
				return val
			}
			_, err := strconv.ParseBool(val)
			if err != nil {
				fmt.Printf("Value '%s' is not a boolean\n", val)
				continue
			}
			return val
		}

		return val
	}

}

func choseBuildTemplateParameters(tmpl sdk.Template) sdk.Template {
	if len(tmpl.Params) == 0 {
		return tmpl
	}

	fmt.Printf("\nTemplate %s parameters:\n", tmpl.Name)
	for i, p := range tmpl.Params {
		tmpl.Params[i].Value = getParamValue(p)
	}

	return tmpl
}

func choseBuildTemplate() sdk.Template {
	for {
		tmpls, err := sdk.GetBuildTemplates()
		if err != nil {
			sdk.Exit("Error: cannot retrieve build templates (%s)\n", err)
		}
		fmt.Printf("\nList of existing build templates:\n")
		for i, t := range tmpls {
			fmt.Printf("-[%d] %-15s (%s)\n", i+1, t.Name, t.Description)
		}
		fmt.Printf("which build template do you want to apply [1-%d]: ", len(tmpls))
		tmplIDS := readline()

		tmplID, err := strconv.Atoi(tmplIDS)
		if err != nil {
			fmt.Printf("Error: %s is not a valid number\n", tmplIDS)
			continue
		}
		if tmplID > len(tmpls) || tmplID <= 0 {
			fmt.Printf("Error: %s is not a valid number\n", tmplIDS)
			continue
		}
		return tmpls[tmplID-1]
	}
}

func createProject() sdk.Project {
	for {
		fmt.Printf("\n")
		fmt.Printf("Project name: ")
		name := readline()
		fmt.Printf("Project key: ")
		key := readline()
		fmt.Printf("Project admin group: ")
		group := readline()

		err := sdk.AddProject(name, key, group)
		if err != nil {
			fmt.Printf("Error: Cannot create project (%s)\n", err)
			continue
		}

		p, err := sdk.GetProject(key)
		if err != nil {
			sdk.Exit("Error: Cannot retrieve project (%s)\n", err)
		}

		return p
	}
}

func choseProject(appName string) sdk.Project {
	for {
		fmt.Printf("\nProjects you have access to:\n")
		projects, err := sdk.ListProject()
		if err != nil {
			sdk.Exit("Error: cannot retrieve project list (%s)\n", err)
		}
		for i, p := range projects {
			fmt.Printf("-[%d] %s\n", i+1, p.Key)
		}
		var lowerBound int
		if len(projects) > 0 {
			lowerBound = 1
		}
		fmt.Printf("Creating application %s in [Enter: create][%d-%d]: ", appName, lowerBound, len(projects))
		projectIDS := readline()

		// create a new project ?
		if projectIDS == "" {
			return createProject()
		}

		projectID, err := strconv.Atoi(projectIDS)
		if err != nil {
			fmt.Printf("Error: %s is not a valid number\n", projectIDS)
			continue
		}
		if projectID > len(projects) || projectID <= 0 {
			fmt.Printf("Error: %s is not a valid number", projectIDS)
			continue
		}
		return projects[projectID-1]
	}
}

func readline() string {
	var all string
	var line []byte
	var err error

	hasMoreInLine := true
	bio := bufio.NewReader(os.Stdin)

	for hasMoreInLine {
		line, hasMoreInLine, err = bio.ReadLine()
		if err != nil {
			sdk.Exit("Error: cannot read from stdin (%s)\n", err)
		}
		all += string(line)
	}

	return strings.Replace(all, "\n", "", -1)
}
