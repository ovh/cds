package sdk

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Cache define a file needed to be save for cache
type Cache struct {
	ID        int64  `json:"id" cli:"id"`
	Project   string `json:"project"`
	Name      string `json:"name" cli:"name"`
	Tag       string `json:"tag"`
	TmpURL    string `json:"tmp_url"`
	SecretKey string `json:"secret_key"`

	Files            []string `json:"files"`
	WorkingDirectory string   `json:"working_directory"`
}

//GetName returns the name the artifact
func (c *Cache) GetName() string {
	return c.Name
}

//GetPath returns the path of the artifact
func (c *Cache) GetPath() string {
	container := fmt.Sprintf("%s-%s", c.Project, c.Tag)
	container = url.QueryEscape(container)
	container = strings.Replace(container, "/", "-", -1)
	return container
}

// CreateTarFromPaths returns a tar formatted reader of a tar made of several path
func CreateTarFromPaths(cwd string, paths []string) (io.Reader, error) {
	// Create a buffer to write our archive to.
	buf := new(bytes.Buffer)

	// Create a new tar archive.
	tw := tar.NewWriter(buf)

	for _, path := range paths {
		// ensure the src actually exists before trying to tar it
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("Unable to tar files - %v", err.Error())
		}
		// walk path
		errWalk := filepath.Walk(path, func(file string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// create a new dir/file header
			header, err := tar.FileInfoHeader(fi, fi.Name())
			if err != nil {
				return err
			}

			header.Name = strings.TrimPrefix(strings.Replace(file, cwd, "", -1), string(filepath.Separator))
			if fi.Mode()&os.ModeSymlink != 0 {
				symlink, errEval := filepath.EvalSymlinks(file)
				if errEval != nil {
					return errEval
				}
				abs, errAbs := filepath.Abs(filepath.Dir(header.Name))
				if errAbs != nil {
					return errAbs
				}
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
			f, err := os.Open(file)
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
			return nil, WrapError(errWalk, "CreateTarFromPaths> Cannot walk file")
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	// Open the tar archive for reading.
	btes := buf.Bytes()
	res := bytes.NewBuffer(btes)

	return res, nil
}
