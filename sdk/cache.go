package sdk

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
)

// Cache define a file needed to be save for cache
type Cache struct {
	ID      int64  `json:"id" cli:"id"`
	Project string `json:"project"`
	Name    string `json:"name" cli:"name"`
	Tag     string `json:"tag"`

	DownloadHash     string   `json:"download_hash" cli:"download_hash"`
	Size             int64    `json:"size,omitempty" cli:"size"`
	Perm             uint32   `json:"perm,omitempty"`
	MD5sum           string   `json:"md5sum,omitempty" cli:"md5sum"`
	ObjectPath       string   `json:"object_path,omitempty"`
	TempURL          string   `json:"temp_url,omitempty"`
	TempURLSecretKey string   `json:"-"`
	Files            []string `json:"files"`
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

func CreateTarFromPaths(paths []string) (io.Reader, error) {
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
			if err := iterDirectory(path, tw); err != nil {
				return nil, err
			}
		} else {
			if err := tarWrite(path, tw, fstat); err != nil {
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

func tarWrite(path string, tw *tar.Writer, fi os.FileInfo) error {
	filR, err := os.Open(path)
	if err != nil {
		return WrapError(err, "tarWrite> cannot open path")
	}
	defer filR.Close()

	stat, err := filR.Stat()
	if err != nil {
		return WrapError(err, "tarWrite> cannot stat file")
	}

	hdr := &tar.Header{
		Name: path,
		Mode: 0600,
		Size: stat.Size(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return WrapError(err, "tarWrite> cannot write header")
	}

	if n, err := io.Copy(tw, filR); err != nil {
		return err
	} else if n == 0 {
		return fmt.Errorf("nothing to write for %s", stat.Name())
	}
	return nil
}

func iterDirectory(dirPath string, tw *tar.Writer) error {
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
			if err := iterDirectory(curPath, tw); err != nil {
				return err
			}
		} else {
			fmt.Printf("adding... %s\n", curPath)
			if err := tarWrite(curPath, tw, fi); err != nil {
				return WrapError(err, "iterDirectory> cannot tar write")
			}
		}
	}

	return nil
}
