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
	ID      int64  `json:"id" cli:"id"`
	Project string `json:"project"`
	Name    string `json:"name" cli:"name"`
	Tag     string `json:"tag"`

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

func CreateTarFromPaths(cwd string, paths []string) (io.Reader, error) {
	// Create a buffer to write our archive to.
	buf := new(bytes.Buffer)

	// Create a new tar archive.
	tw := tar.NewWriter(buf)

	for _, path := range paths {
		fstat, err := os.Stat(path)
		if err != nil {
			return nil, WrapError(err, "CreateTarFromPaths> Cannot stat path %s", path)
		}

		if fstat.IsDir() {
			if err := iterDirectory(cwd, path, tw); err != nil {
				tw.Close()
				return nil, err
			}
		} else {
			if err := tarWrite(cwd, path, tw, fstat); err != nil {
				tw.Close()
				return nil, WrapError(err, "CreateTarFromPaths> Cannot tar write %s", path)
			}
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

func tarWrite(cwd, path string, tw *tar.Writer, fi os.FileInfo) error {
	filR, err := os.Open(path)
	if err != nil {
		return WrapError(err, "tarWrite> cannot open path")
	}
	defer filR.Close()

	filename, err := filepath.Rel(cwd, path)
	if err != nil {
		return WrapError(err, "tarWrite> cannot find relative path")
	}

	hdr := &tar.Header{
		Name:     filename,
		Mode:     0600,
		Size:     fi.Size(),
		Typeflag: tar.TypeReg,
	}

	if fi.IsDir() {
		hdr.Typeflag = tar.TypeDir
	}

	if fi.Mode()&os.ModeSymlink != 0 {
		symlink, err := filepath.EvalSymlinks(path)
		if err != nil {
			return WrapError(err, "tarWrite> cannot get resolve path %s", path)
		}

		fil, err := os.Lstat(symlink)
		if err != nil {
			return WrapError(err, "tarWrite> cannot get resolve(lstat) link")
		}

		hdr.Size = fil.Size()
	}

	if err := tw.WriteHeader(hdr); err != nil {
		return WrapError(err, "tarWrite> cannot write header")
	}

	if _, err := io.Copy(tw, filR); err != nil {
		return err
	}
	return nil
}

func iterDirectory(cwd, dirPath string, tw *tar.Writer) error {
	dir, err := os.Open(dirPath)
	if err != nil {
		return WrapError(err, "iterDirectory> cannot open path %s", dirPath)
	}
	defer dir.Close()
	fis, err := dir.Readdir(0)
	if err != nil {
		return WrapError(err, "iterDirectory> cannot readdir %s", dirPath)
	}
	for _, fi := range fis {
		curPath := dirPath + "/" + fi.Name()
		if fi.IsDir() {
			if err := iterDirectory(cwd, curPath, tw); err != nil {
				return err
			}
		} else {
			if err := tarWrite(cwd, curPath, tw, fi); err != nil {
				return WrapError(err, "iterDirectory> cannot tar write (%s)", curPath)
			}
		}
	}

	return nil
}
