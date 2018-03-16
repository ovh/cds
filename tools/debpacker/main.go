package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
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
				p := new(DebPacker)
				p.Init()
				b, _ := yaml.Marshal(p)
				return ioutil.WriteFile(".debpacker.yml", b, os.FileMode(0644))
			},
		}, {
			Name: "clean",
			Action: func(c *cli.Context) error {
				p := new(DebPacker)
				p.outputDirectory = c.String("target")
				return p.Clean()
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "target",
					Value: "./target",
					Usage: "Target output directory",
				},
			},
		}, {
			Name: "make",
			Action: func(c *cli.Context) error {
				p := new(DebPacker)
				b, err := ioutil.ReadFile(c.String("config"))
				if err != nil {
					return err
				}

				if err := yaml.Unmarshal(b, p); err != nil {
					return err
				}

				p.outputDirectory = c.String("target")
				if ok, _ := isDirEmpty(p.outputDirectory); !ok {
					fmt.Printf("Error: directory %s is not empty. Aborting\n", p.outputDirectory)
					fmt.Printf("Run %s clean --target %s\n", os.Args[0], p.outputDirectory)
					os.Exit(1)
				}

				p.Init()

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
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
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
