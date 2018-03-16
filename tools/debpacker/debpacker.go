package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// DebPacker represents a packagin configuration
type DebPacker struct {
	outputDirectory      string
	PackageName          string               `yaml:"package-name"`
	Architecture         string               `yaml:"architecture"`
	BinaryFile           string               `yaml:"binary-file"`
	ConfigurationFiles   []string             `yaml:"configuration-file,omitempty"`
	Version              string               `yaml:"version"`
	Description          string               `yaml:"description"`
	Maintainer           string               `yaml:"maintainer"`
	Dependencies         []string             `yaml:"dependencies,omitempty"`
	SystemdServiceConfig SystemdServiceConfig `yaml:"systemd-configuration"`
}

// SystemdServiceConfig will generate a .service file which will be located in /lib/systemd/system/
type SystemdServiceConfig struct {
	After         string            `yaml:"after,omitempty"` //default network.target
	User          string            `yaml:"user,omitempty"`
	ExecStart     string            `yaml:"-"`
	ExecStartArgs []string          `yaml:"args"`
	ExecStop      string            `yaml:"stop-command,omitempty"` //default /bin/kill -15 $MAINPID
	Restart       string            `yaml:"restart,omitempty"`      //default always
	WantedBy      string            `yaml:"wanted-by,omitempty"`    //default multi-user.target
	Environments  map[string]string `yaml:"environments"`
}

// Init the configuration
func (p *DebPacker) Init() {
	if p.SystemdServiceConfig.User == "" {
		p.SystemdServiceConfig.User = p.PackageName
	}

	if p.SystemdServiceConfig.After == "" {
		p.SystemdServiceConfig.After = "network.target"
	}

	if p.SystemdServiceConfig.ExecStop == "" {
		p.SystemdServiceConfig.ExecStop = "/bin/kill $MAINPID"
	}

	if p.SystemdServiceConfig.Restart == "" {
		p.SystemdServiceConfig.Restart = "always"
	}

	if p.SystemdServiceConfig.WantedBy == "" {
		p.SystemdServiceConfig.WantedBy = "multi-user.target"
	}

	if p.SystemdServiceConfig.ExecStart == "" {
		p.SystemdServiceConfig.ExecStart = "/usr/bin/" + filepath.Base(p.BinaryFile)
	}

	p.outputDirectory = filepath.Join(p.outputDirectory, p.PackageName)
}

// Clean the target directory
func (p *DebPacker) Clean() error {
	fmt.Println("cleaning directory", p.outputDirectory)
	return os.RemoveAll(p.outputDirectory)
}

// Prepare the debian package config file
func (p DebPacker) Prepare() error {
	path := filepath.Join(p.outputDirectory, "DEBIAN")
	fmt.Println("creating directory", path)
	if err := os.MkdirAll(path, os.FileMode(0755)); err != nil {
		return err
	}

	if err := p.buildControlFile(); err != nil {
		return err
	}

	if err := p.copyBinaryFile(); err != nil {
		return err
	}

	if err := p.copyConfigurationFiles(); err != nil {
		return err
	}

	if err := p.writeSystemdServiceFile(); err != nil {
		return err
	}

	if err := p.writePostinstFile(); err != nil {
		return err
	}

	return nil
}

// Build the debian package with dpkg
func (p DebPacker) Build() (err error) {
	c := exec.Command("dpkg-deb", "--build", p.PackageName)
	c.Dir = filepath.Dir(p.outputDirectory)

	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)

	c.Stdout = bufOut
	c.Stderr = bufErr
	fmt.Println("starting dpkg-deb from", c.Dir)
	err = c.Run()

	fmt.Println(bufErr.String())
	fmt.Println(bufOut.String())

	fmt.Printf("your package is ready: %s.deb\n", filepath.Join(filepath.Dir(p.outputDirectory), p.PackageName))

	return err
}

func (p DebPacker) buildControlFile() error {
	path := filepath.Join(p.outputDirectory, "DEBIAN")
	if err := render(controlTmpl, filepath.Join(path, "control"), p, os.FileMode(0644)); err != nil {
		return err
	}
	return nil
}

func (p DebPacker) copyBinaryFile() error {
	path := filepath.Join(p.outputDirectory, "usr", "bin")
	fmt.Println("creating directory", path)
	if err := os.MkdirAll(path, os.FileMode(0755)); err != nil {
		return err
	}

	originFile, err := os.Open(p.BinaryFile)
	if err != nil {
		return err
	}
	defer originFile.Close()

	destFileName := filepath.Join(path, filepath.Base(originFile.Name()))
	destFile, err := os.OpenFile(destFileName, os.O_CREATE|os.O_RDWR, os.FileMode(0644))
	if err != nil {
		return err
	}
	defer destFile.Close()

	fmt.Printf("copying file %s to %s\n", filepath.Base(originFile.Name()), destFileName)
	if _, err := io.Copy(destFile, originFile); err != nil {
		return err
	}

	return nil
}

func (p DebPacker) copyConfigurationFiles() error {
	if len(p.ConfigurationFiles) == 0 {
		fmt.Println("skipping configuration files")
		return nil
	}

	path := filepath.Join(p.outputDirectory, "etc", p.PackageName)
	fmt.Println("creating directory", path)
	if err := os.MkdirAll(path, os.FileMode(0755)); err != nil {
		return err
	}

	for _, c := range p.ConfigurationFiles {
		originFile, err := os.Open(c)
		if err != nil {
			return err
		}
		defer originFile.Close()

		destFileName := filepath.Join(path, filepath.Base(originFile.Name()))
		destFile, err := os.OpenFile(destFileName, os.O_CREATE|os.O_RDWR, os.FileMode(0644))
		if err != nil {
			return err
		}
		defer destFile.Close()

		fmt.Printf("copying file %s to %s\n", filepath.Base(originFile.Name()), destFileName)
		if _, err := io.Copy(destFile, originFile); err != nil {
			return err
		}
	}

	return nil
}

func (p DebPacker) writePostinstFile() error {
	path := filepath.Join(p.outputDirectory, "DEBIAN", "postinst")

	if err := render(postinstTmpl, filepath.Join(path), p, os.FileMode(0755)); err != nil {
		return err
	}

	return nil
}

func (p DebPacker) writeSystemdServiceFile() error {
	path := filepath.Join(p.outputDirectory, "/lib/systemd/system/")
	fmt.Println("creating directory", path)
	if err := os.MkdirAll(path, os.FileMode(0755)); err != nil {
		return err
	}

	if err := render(systemdServiceTmpl, filepath.Join(path, p.PackageName+".service"), p, os.FileMode(0644)); err != nil {
		return err
	}
	return nil
}
