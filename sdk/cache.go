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
		symlink, err := filepath.EvalSymlinks(path)
		if err != nil {
			return nil, WrapError(err, "CreateTarFromPaths> cannot get resolve path %s", path)
		}

		fstat, err := os.Lstat(symlink)
		if err != nil {
			return nil, WrapError(err, "CreateTarFromPaths> cannot get resolve(lstat) link")
		}

		if fstat.IsDir() {
			if err := iterDirectory(cwd, symlink, tw); err != nil {
				tw.Close()
				return nil, WrapError(err, "CreateTarFromPaths> cannot iter directory for path %s and symlink %s", path, symlink)
			}
		} else {
			if err := tarWrite(cwd, symlink, tw, fstat); err != nil {
				tw.Close()
				return nil, WrapError(err, "CreateTarFromPaths> Cannot tar write %s and symlink %s", path, symlink)
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

	// Useful to not copy files like socket or device files
	if !fi.IsDir() && !fi.Mode().IsRegular() {
		return nil
	}

	if fi.Mode()&os.ModeSymlink != 0 {
		symlink, errEval := filepath.EvalSymlinks(path)
		if errEval != nil {
			return WrapError(errEval, "tarWrite> cannot get resolve path %s", path)
		}

		fil, errLs := os.Lstat(symlink)
		if errLs != nil {
			return WrapError(errLs, "tarWrite> cannot get resolve(lstat) link")
		}

		hdr.Size = fil.Size()

		// Useful to not copy files like socket or device files
		if !fil.IsDir() && !fil.Mode().IsRegular() {
			return nil
		}
	}

	filR, err := os.Open(path)
	if err != nil {
		return WrapError(err, "tarWrite> cannot open path")
	}
	defer filR.Close()

	if err := tw.WriteHeader(hdr); err != nil {
		return WrapError(err, "tarWrite> cannot write header")
	}

	if _, err := io.Copy(tw, filR); err != nil {
		return WrapError(err, "tarWrite> cannot copy")
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
		symlink, err := filepath.EvalSymlinks(curPath)
		if err != nil {
			return WrapError(err, "tarWrite> cannot get resolve path %s", curPath)
		}

		fil, err := os.Lstat(symlink)
		if err != nil {
			return WrapError(err, "tarWrite> cannot get resolve(lstat) link")
		}

		if fil.IsDir() {
			if err := iterDirectory(cwd, symlink, tw); err != nil {
				return WrapError(err, "iterDirectory> cannot iter on directory")
			}
		} else {
			if err := tarWrite(cwd, symlink, tw, fil); err != nil {
				return WrapError(err, "tarWrite> cannot tar write (%s)", symlink)
			}
		}
	}

	return nil
}
