package sdk

import (
	"archive/tar"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

// Cache define a file needed to be save for cache
type Cache struct {
	ID              int64  `json:"id" cli:"id"`
	Project         string `json:"project"`
	Name            string `json:"name" cli:"name"`
	Tag             string `json:"tag"`
	TmpURL          string `json:"tmp_url"`
	SecretKey       string `json:"secret_key"`
	IntegrationName string `json:"integration_name"`

	Files            []string `json:"files"`
	WorkingDirectory string   `json:"working_directory"`
}

// GetName returns the name the artifact
func (c *Cache) GetName() string {
	return c.Name
}

// GetPath returns the path of the artifact
func (c *Cache) GetPath() string {
	container := fmt.Sprintf("%s-%s", c.Project, c.Tag)
	container = url.QueryEscape(container)
	container = strings.Replace(container, "/", "-", -1)
	return container
}

// TarOptions useful to indicate some options when we want to tar directory or files
type TarOptions struct {
	TrimDirName string
}

// CreateTarFromPaths returns a tar formatted reader of a tar made of several path
func CreateTarFromPaths(fs afero.Fs, cwd string, paths []string, w io.Writer, opts *TarOptions) error {
	// Create a new tar archive.
	tw := tar.NewWriter(w)

	for _, p := range paths {
		// ensure the src actually exists before trying to tar it

		completePath := p
		if !PathIsAbs(p) {
			completePath = filepath.Join(cwd, p)
		}

		if _, err := fs.Stat(completePath); err != nil {
			return fmt.Errorf("unable to tar files - %v", err.Error())
		}

		// walk path
		errWalk := afero.Walk(fs, completePath, func(file string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// create a new dir/file header
			header, err := tar.FileInfoHeader(fi, fi.Name())
			if err != nil {
				return err
			}

			header.Name = strings.TrimPrefix(strings.TrimPrefix(file, cwd), string(filepath.Separator))

			if opts != nil && opts.TrimDirName != "" {
				opts.TrimDirName = strings.TrimPrefix(opts.TrimDirName, string(filepath.Separator))
				header.Name = strings.TrimPrefix(strings.TrimPrefix(header.Name, opts.TrimDirName), string(filepath.Separator))
			}
			if fi.Mode()&os.ModeSymlink != 0 {

				symlink, errEval := filepath.EvalSymlinks(file)
				if errEval != nil {
					return errEval
				}

				abs := filepath.Dir(filepath.Join(cwd, header.Name))

				symlinkRel, errRel := filepath.Rel(abs, symlink)
				if errRel != nil {
					return errRel
				}
				header.Linkname = symlinkRel
			}

			// write the header
			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			if !fi.Mode().IsRegular() {
				return nil
			}

			// open files for taring
			f, err := fs.Open(file)
			if err != nil {
				return err
			}

			// copy file data into tar writer
			if _, err := io.Copy(tw, f); err != nil {
				return err
			}

			_ = f.Close()

			return nil
		})

		if errWalk != nil {
			_ = tw.Close()
			return WrapError(errWalk, "cannot walk file")
		}
	}

	if err := tw.Close(); err != nil {
		return err
	}

	return nil
}
