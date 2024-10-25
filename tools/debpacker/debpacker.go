package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pkg/errors"
)

// New returns a new packer.
func New(w Writer, c Config, out string) *Packer {
	if c.SystemdServiceConfig.User == "" {
		c.SystemdServiceConfig.User = c.PackageName
	}

	if c.SystemdServiceConfig.After == "" {
		c.SystemdServiceConfig.After = "network.target"
	}

	if c.SystemdServiceConfig.ExecStop == "" {
		c.SystemdServiceConfig.ExecStop = "/bin/kill $MAINPID"
	}

	if c.SystemdServiceConfig.Restart == "" {
		c.SystemdServiceConfig.Restart = "always"
	}

	if c.SystemdServiceConfig.WantedBy == "" {
		c.SystemdServiceConfig.WantedBy = "multi-user.target"
	}

	if c.SystemdServiceConfig.ExecStart == "" {
		if c.BinaryFile != "" {
			c.SystemdServiceConfig.ExecStart = "/usr/bin/" + filepath.Base(c.BinaryFile)
		}
		if c.Command != "" {
			c.SystemdServiceConfig.ExecStart = c.Command
		}
	}

	return &Packer{
		writer:          w,
		config:          c,
		outputDirectory: filepath.Join(out, c.PackageName),
	}
}

// Packer struct.
type Packer struct {
	writer          Writer
	config          Config
	outputDirectory string
}

// Config returns packer's config.
func (p Packer) Config() Config { return p.config }

// Config struct.
type Config struct {
	PackageName        string   `yaml:"package-name"`
	Architecture       string   `yaml:"architecture"`
	BinaryFile         string   `yaml:"binary-file"`
	Command            string   `yaml:"command,omitempty"`
	ConfigurationFiles []string `yaml:"configuration-files,omitempty"`
	CopyFiles          []string `yaml:"copy-files,omitempty"`
	ExtractFiles       []struct {
		Archive     string `yaml:"archive,omitempty"`
		Destination string `yaml:"destination,omitempty"`
	} `yaml:"extract-files,omitempty"`
	Mkdirs               []string             `yaml:"mkdirs,omitempty"`
	Version              string               `yaml:"version"`
	Description          string               `yaml:"description"`
	Maintainer           string               `yaml:"maintainer"`
	Dependencies         []string             `yaml:"dependencies,omitempty"`
	SystemdServiceConfig SystemdServiceConfig `yaml:"systemd-configuration"`
	NeedsService         bool                 `yaml:"-"`
}

// SystemdServiceConfig will generate a .service file which will be located in /lib/systemd/system/.
type SystemdServiceConfig struct {
	After            string            `yaml:"after,omitempty"` //default network.target
	User             string            `yaml:"user,omitempty"`
	ExecStart        string            `yaml:"-"`
	ExecStartArgs    []string          `yaml:"args"`
	ExecStop         string            `yaml:"stop-command,omitempty"` //default /bin/kill -15 $MAINPID
	Restart          string            `yaml:"restart,omitempty"`      //default always
	WantedBy         string            `yaml:"wanted-by,omitempty"`    //default multi-user.target
	Environments     map[string]string `yaml:"environments"`
	WorkingDirectory string            `yaml:"working-directory,omitempty"`
	PostInstallCmd   string            `yaml:"post-install-cmd,omitempty"`
	AutoStart        bool              `yaml:"auto-start,omitempty"`
}

// Prepare the debian package config file.
func (p Packer) Prepare() error {
	path := filepath.Join(p.outputDirectory, "DEBIAN")
	if err := p.writer.CreateDirectory(path, os.FileMode(0755)); err != nil {
		return err
	}

	if err := p.buildControlFile(); err != nil {
		return err
	}

	if p.config.BinaryFile != "" {
		p.config.NeedsService = true
		if err := p.copyBinaryFile(); err != nil {
			return err
		}
	}

	if err := p.copyConfigurationFiles(); err != nil {
		return err
	}

	if err := p.mkDirs(); err != nil {
		return err
	}

	if err := p.copyOtherFiles(); err != nil {
		return err
	}

	if err := p.extractOtherFiles(); err != nil {
		return err
	}

	if p.config.NeedsService {
		if err := p.writeSystemdServiceFile(); err != nil {
			return err
		}
	}

	if err := p.writePrermFile(); err != nil {
		return err
	}

	if err := p.writePostrmFile(); err != nil {
		return err
	}

	if err := p.writePreinstFile(); err != nil {
		return err
	}

	return p.writePostinstFile()
}

// Build the debian package with dpkg.
func (p Packer) Build() (err error) {
	c := exec.Command("dpkg-deb", "--build", p.config.PackageName)
	c.Dir = filepath.Dir(p.outputDirectory)

	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)

	c.Stdout = bufOut
	c.Stderr = bufErr
	fmt.Println("starting dpkg-deb from", c.Dir)
	err = c.Run()

	fmt.Println(bufErr.String())
	fmt.Println(bufOut.String())

	if err == nil {
		fmt.Printf("your package is ready: %s.deb\n", filepath.Join(filepath.Dir(p.outputDirectory), p.config.PackageName))
	}

	return errors.Wrap(err, "Error running dpkg-deb")
}

func (p Packer) buildControlFile() error {
	path := filepath.Join(p.outputDirectory, "DEBIAN", "control")
	return p.render(controlTmpl, path, os.FileMode(0644))
}

func (p Packer) copyBinaryFile() error {
	path := filepath.Join("usr", "bin")
	if err := p.writer.CreateDirectory(filepath.Join(p.outputDirectory, path), os.FileMode(0755)); err != nil {
		return err
	}

	return p.writer.CopyFiles(p.outputDirectory, path, os.FileMode(0744), p.config.BinaryFile)
}

func (p Packer) copyConfigurationFiles() error {
	if len(p.config.ConfigurationFiles) == 0 {
		return nil
	}

	path := filepath.Join("etc", p.config.PackageName)
	if err := p.writer.CreateDirectory(filepath.Join(p.outputDirectory, path), os.FileMode(0755)); err != nil {
		return err
	}

	return p.writer.CopyFiles(p.outputDirectory, path, os.FileMode(0644), p.config.ConfigurationFiles...)
}

func (p Packer) mkDirs() error {
	path := filepath.Join(p.outputDirectory, "var", "lib", p.config.PackageName)
	if err := p.writer.CreateDirectory(path, os.FileMode(0755)); err != nil {
		return err
	}

	for _, d := range p.config.Mkdirs {
		path := filepath.Join(p.outputDirectory, d)
		if !strings.HasPrefix(d, "/") {
			path = filepath.Join(p.outputDirectory, "var", "lib", p.config.PackageName, d)
		}
		if err := p.writer.CreateDirectory(path, os.FileMode(0755)); err != nil {
			return err
		}
	}

	return nil
}

func (p Packer) copyOtherFiles() error {
	if len(p.config.CopyFiles) == 0 {
		return nil
	}

	path := filepath.Join("var", "lib", p.config.PackageName)
	if err := p.writer.CreateDirectory(filepath.Join(p.outputDirectory, path), os.FileMode(0755)); err != nil {
		return err
	}

	return p.writer.CopyFiles(p.outputDirectory, path, os.FileMode(0644), p.config.CopyFiles...)
}

func (p Packer) extractOtherFiles() error {
	if len(p.config.ExtractFiles) == 0 {
		return nil
	}

	path := filepath.Join("var", "lib", p.config.PackageName)
	if err := p.writer.CreateDirectory(filepath.Join(p.outputDirectory, path), os.FileMode(0755)); err != nil {
		return err
	}

	for _, f := range p.config.ExtractFiles {
		path = filepath.Join("var", "lib", p.config.PackageName, f.Destination)
		if err := p.writer.ExtractArchive(p.outputDirectory, path, f.Archive); err != nil {
			return err
		}
	}
	return nil
}

func (p Packer) writeSystemdServiceFile() error {
	path := filepath.Join(p.outputDirectory, "lib", "systemd", "system")
	if err := p.writer.CreateDirectory(path, os.FileMode(0755)); err != nil {
		return err
	}

	return p.render(systemdServiceTmpl, filepath.Join(path, p.config.PackageName+".service"), os.FileMode(0644))
}

func (p Packer) writePreinstFile() error {
	path := filepath.Join(p.outputDirectory, "DEBIAN", "preinst")
	return p.render(preinstTmpl, path, os.FileMode(0755))
}

func (p Packer) writePostinstFile() error {
	path := filepath.Join(p.outputDirectory, "DEBIAN", "postinst")
	return p.render(postinstTmpl, path, os.FileMode(0755))
}

func (p Packer) writePrermFile() error {
	path := filepath.Join(p.outputDirectory, "DEBIAN", "prerm")
	return p.render(prermTmpl, path, os.FileMode(0755))
}

func (p Packer) writePostrmFile() error {
	path := filepath.Join(p.outputDirectory, "DEBIAN", "postrm")
	return p.render(postrmTmpl, path, os.FileMode(0755))
}

func (p Packer) render(tmpl string, path string, perm os.FileMode) error {
	t, err := template.New("tmpl").
		Funcs(template.FuncMap{"StringsJoin": strings.Join}).
		Parse(tmpl)
	if err != nil {
		return errors.Wrap(err, "Cannot parse template")
	}

	buf := new(bytes.Buffer)

	if err := t.Execute(buf, p.config); err != nil {
		return errors.Wrap(err, "Cannot execute template")
	}

	return p.writer.CreateFile(path, buf.Bytes(), perm)
}
