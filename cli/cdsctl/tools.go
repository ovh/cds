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
	tagDescription   = "description"
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

type yamlSchemaInstaller interface {
	Install(schemas map[string]string) error
}

type yamlSchemaVSCodeInstaller struct{}

func (y yamlSchemaVSCodeInstaller) Install(schemas map[string]string) error {
	fmt.Println("Installing yaml-syntax for VSCode.")

	fmt.Println("You will need to execute the following command:")
	fmt.Println(cli.Cyan("code --install-extension %s", pluginVSCodeName))

	type settings struct {
		Schemas map[string]string `json:"yaml.schemas"`
	}

	s := settings{Schemas: schemas}
	buf, _ := json.MarshalIndent(s, "", "\t")

	fmt.Println("You need to add the following part in your VSCode settings.json file:")
	fmt.Println(cli.Cyan(string(buf)))

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

	structs := map[string]interface{}{
		//"*.yml": exportentities.Workflow{},
		"*.pip.yml": exportentities.PipelineV1{},
		"*.app.yml": exportentities.Application{},
		"*.env.yml": exportentities.Environment{},
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

	results := make(map[string]string, len(structs))
	for key, s := range structs {
		sch := jsonschema.Reflect(s)
		typ := reflect.TypeOf(s)
		setDescriptionForStruct(sch, typ)

		buf, _ := json.MarshalIndent(sch, "", "\t")
		path := fmt.Sprintf("%s/%s.schema.json", targetFolder, typ.Name())
		if err := ioutil.WriteFile(path, buf, 0775); err != nil {
			return fmt.Errorf("Cannot write file at %s: %s", path, err)
		}
		fmt.Printf("File %s successfully written.\n", path)

		results["file://"+path] = key
	}

	return installer.Install(results)
}

func setDescriptionForStruct(sch *jsonschema.Schema, typ reflect.Type) {
	if typ.Kind() == reflect.Slice || typ.Kind() == reflect.Map {
		typ = typ.Elem()
	}

	if _, ok := sch.Definitions[typ.Name()]; !ok {
		return
	}

	// set description for all current type properties
	for name, property := range sch.Definitions[typ.Name()].Properties {
		field, ok := typ.FieldByNameFunc(func(n string) bool {
			return strings.ToLower(n) == name
		})
		if ok {
			property.Description = field.Tag.Get(tagDescription)
			// recusively set description for properties type
			setDescriptionForStruct(sch, field.Type)
		}
	}
}
