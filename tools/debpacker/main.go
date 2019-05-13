package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/urfave/cli"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/sdk"
)

func main() {
	app := cli.NewApp()
	app.Name = "debpacker"
	app.Usage = "Package CDS Binary as debian package"
	app.Version = sdk.VERSION
	app.Commands = []cli.Command{
		{
			Name: "init",
			Action: func(c *cli.Context) error {
				p := New(nil, Config{}, "")
				b, _ := yaml.Marshal(p.Config())
				return ioutil.WriteFile(".debpacker.yml", b, os.FileMode(0644))
			},
		},
		{
			Name:   "clean",
			Action: func(c *cli.Context) error { return os.RemoveAll(c.String("target")) },
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "target",
					Value: "./target",
					Usage: "Target output directory",
				},
			},
		},
		{
			Name: "make",
			Action: func(c *cli.Context) error {
				b, err := ioutil.ReadFile(c.String("config"))
				if err != nil {
					return err
				}

				var config Config
				if err := yaml.Unmarshal(b, &config); err != nil {
					return err
				}

				target := c.String("target")
				if ok, _ := isDirEmpty(target); !ok {
					if !c.Bool("force") {
						fmt.Printf("Error: directory %s is not empty. Aborting\n", target)
						fmt.Printf("Run %s clean --target %s\n", os.Args[0], target)
						os.Exit(1)
					}

					fmt.Println("removing directory", target)
					if err := os.RemoveAll(target); err != nil {
						return err
					}
				}

				p := New(&fileSystemWriter{}, config, target)

				if err := p.Prepare(); err != nil {
					return err
				}

				return p.Build()
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "config",
					Value: ".debpacker.yml",
					Usage: "deppacker config file",
				},
				cli.StringFlag{
					Name:  "target",
					Value: "./target",
					Usage: "Target output directory",
				},
				cli.BoolFlag{
					Name:  "force",
					Usage: "Force override of existing target folder",
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
}

func isDirEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return true, err
	}
	defer f.Close()

	// read in ONLY one file
	_, err = f.Readdir(1)

	// and if the file is EOF... well, the dir is empty.
	if err == io.EOF {
		return true, nil
	}
	return false, err
}
