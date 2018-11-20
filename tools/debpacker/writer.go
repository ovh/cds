package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	zglob "github.com/mattn/go-zglob"
	"github.com/pkg/errors"
)

type Writer interface {
	CreateDirectory(path string, perm os.FileMode) error
	CreateFile(path string, content []byte, perm os.FileMode) error
	CopyFiles(path string, perm os.FileMode, sources ...string) error
}

type fileSystemWriter struct{}

func (f fileSystemWriter) CreateDirectory(path string, perm os.FileMode) error {
	return errors.Wrapf(os.MkdirAll(path, os.FileMode(0755)), "Cannot mkdir all at %s", path)
}

func (f fileSystemWriter) CreateFile(path string, content []byte, perm os.FileMode) error {
	fi, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, perm)
	if err != nil {
		return errors.Wrapf(err, "Cannot open file at %s", path)
	}
	defer fi.Close()

	_, err = fi.Write(content)
	return errors.Wrapf(err, "Cannot write file at %s", path)
}

func (f fileSystemWriter) CopyFiles(path string, perm os.FileMode, sources ...string) error {
	for _, source := range sources {
		dest := path

		// check if source file contains an out dir, if true create it
		split := strings.Split(source, ":")
		if len(split) > 1 {
			source = split[0]
			dest = filepath.Join(dest, split[1])
			if err := f.CreateDirectory(dest, os.FileMode(0755)); err != nil {
				return err
			}
		}

		matches, err := zglob.Glob(source)
		if err != nil && err.Error() != "file does not exist" {
			return errors.Wrapf(err, "Error glob for path %s", source)
		}

		for _, m := range matches {
			fi, err := os.Stat(m)
			if err != nil {
				return errors.Wrapf(err, "Cannot stat file or directory at %s", m)
			}

			var list []string
			if fi.IsDir() {
				if err := filepath.Walk(m, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if !info.IsDir() {
						list = append(list, path)
					}
					return nil
				}); err != nil {
					return errors.Wrapf(err, "Cannot walk on directory at %s", m)
				}
			} else {
				list = []string{m}
			}

			for _, l := range list {
				originFile, err := os.Open(l)
				if err != nil {
					return errors.Wrapf(err, "Cannot open file at %s", l)
				}
				defer originFile.Close()

				var destPath string
				if l == m {
					destPath = dest
				} else {
					destPath = filepath.Join(dest, strings.TrimPrefix(filepath.Dir(l), filepath.Dir(m)))
					if err := f.CreateDirectory(destPath, os.FileMode(0755)); err != nil {
						return err
					}
				}

				destFileName := filepath.Join(destPath, filepath.Base(originFile.Name()))
				destFile, err := os.OpenFile(destFileName, os.O_CREATE|os.O_RDWR, perm)
				if err != nil {
					return errors.Wrapf(err, "Cannot open file at %s", destFileName)
				}
				defer destFile.Close()

				if _, err := io.Copy(destFile, originFile); err != nil {
					return errors.Wrap(err, "Cannot copy file content")
				}
			}
		}
	}

	return nil
}
