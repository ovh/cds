package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"

	"github.com/alecthomas/jsonschema"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/exportentities"
)

const (
	targetFolderName = ".cds-schema"
	pluginVSCodeName = "redhat.vscode-yaml"
)

var toolsCmd = cli.Command{
	Name:  "tools",
	Short: "Some tooling for CDS",
}

func tools() *cobra.Command {
	return cli.NewCommand(toolsCmd, nil, []*cobra.Command{
		cli.NewCommand(toolsYamlSchema, toolsYamlSchemaRun, nil, withAllCommandModifiers()...),
	})
}

var toolsYamlSchema = cli.Command{
	Name:    "yaml-schema",
	Short:   "Generate and install CDS yaml schema for given IDE",
	Example: "cdsctl tools yaml-schema vscode",
	Args: []cli.Arg{
		{Name: "ide-name"},
	},
}

type yamlSchemaPath struct {
	Workflow    string
	Pipeline    string
	Application string
	Environment string
}

type yamlSchemaInstaller interface {
	Install(schemas yamlSchemaPath) error
}

type yamlSchemaVSCodeInstaller struct{}

func (y yamlSchemaVSCodeInstaller) Install(schemas yamlSchemaPath) error {
	fmt.Println("Installing yaml-syntax for VSCode.")

	fmt.Println("You will need to execute the following command:")
	fmt.Println(cli.Cyan("code --install-extension %s", pluginVSCodeName))

	// manually constructs a json to preserve rules order
	paths := []string{schemas.Workflow, schemas.Application, schemas.Environment, schemas.Pipeline}
	globPatterns := []string{"*.cds*.yml", "*.cds*.app.yml", "*.cds*.env.yml", "*.cds*.pip.yml"}
	var schs []string
	for i := range paths {
		schs = append(schs, fmt.Sprintf("\n\t\t\"%s\": \"%s\"", paths[i], globPatterns[i]))
	}
	res := fmt.Sprintf("{\n\t\"yaml.schemas\": {%s\n\t}\n}", strings.Join(schs, ","))

	fmt.Println("You need to add the following part in your VSCode settings.json file:")
	fmt.Println(cli.Cyan(res))

	return nil
}

func toolsYamlSchemaRun(v cli.Values) error {
	var installer yamlSchemaInstaller

	switch v.GetString("ide-name") {
	case "vscode":
		installer = &yamlSchemaVSCodeInstaller{}
	default:
		return fmt.Errorf("Invalid given IDE name")
	}

	types := []reflect.Type{
		reflect.TypeOf(exportentities.Workflow{}),
		reflect.TypeOf(exportentities.PipelineV1{}),
		reflect.TypeOf(exportentities.Application{}),
		reflect.TypeOf(exportentities.Environment{}),
	}

	home, err := os.UserHomeDir()
	targetFolder := home + "/" + targetFolderName
	if err != nil {
		return fmt.Errorf("Cannot get user home directory info: %s", err)
	}
	if err := os.RemoveAll(targetFolder); err != nil {
		return fmt.Errorf("Cannot remove folder %s: %s", targetFolder, err)
	}
	if err := os.MkdirAll(targetFolder, 0775); err != nil {
		return fmt.Errorf("Cannot create folder %s: %s", targetFolder, err)
	}

	results := make([]string, len(types))
	for i := range types {
		r := jsonschema.Reflector{
			AllowAdditionalProperties:  true,
			RequiredFromJSONSchemaTags: true,
		}
		sch := r.ReflectFromType(types[i])

		buf, _ := json.MarshalIndent(sch, "", "\t")
		path := fmt.Sprintf("%s/%s.schema.json", targetFolder, types[i].Name())
		if err := ioutil.WriteFile(path, buf, 0775); err != nil {
			return fmt.Errorf("Cannot write file at %s: %s", path, err)
		}
		fmt.Printf("File %s successfully written.\n", path)

		results[i] = "file://" + path
	}

	return installer.Install(yamlSchemaPath{
		Workflow:    results[0],
		Pipeline:    results[1],
		Application: results[2],
		Environment: results[3],
	})
}
