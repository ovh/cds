package internal

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"

	"github.com/rockbears/log"
	"github.com/spf13/afero"

	"github.com/ovh/cds/sdk"
)

func setupDirectory(ctx context.Context, fs afero.Fs, jobName string, suffixes ...string) (string, error) {
	// Generate a hash of job name as workspace folder, this folder's name should not be too long as some tools are limiting path size.
	data := []byte(jobName)
	hashedName := fmt.Sprintf("%x", md5.Sum(data))
	paths := append([]string{hashedName}, suffixes...)
	dir := path.Join(paths...)

	if _, err := fs.Stat(dir); os.IsExist(err) {
		log.Info(ctx, "cleaning working directory %s", dir)
		_ = fs.RemoveAll(dir)
	}

	if err := fs.MkdirAll(dir, os.FileMode(0700)); err != nil {
		return dir, sdk.WithStack(err)
	}

	log.Debug(ctx, "directory %s is ready", dir)
	return dir, nil
}

func (w *CurrentWorker) setupWorkingDirectory(ctx context.Context, jobName string) (afero.File, string, error) {
	wd, err := setupDirectory(ctx, w.basedir, jobName, "run")
	if err != nil {
		return nil, "", err
	}

	wdFile, err := setupWorkingDirectory(ctx, w.basedir, wd)
	if err != nil {
		log.Debug(ctx, "setupWorkingDirectory error:%s", err)
		return nil, "", err
	}

	wdAbs, err := filepath.Abs(wdFile.Name())
	if err != nil {
		log.Debug(ctx, "setupWorkingDirectory error:%s", err)
		return nil, "", err
	}

	switch x := w.basedir.(type) {
	case *afero.BasePathFs:
		wdAbs, err = x.RealPath(wdFile.Name())
		if err != nil {
			return nil, "", err
		}

		wdAbs, err = filepath.Abs(wdAbs)
		if err != nil {
			log.Debug(ctx, "setupWorkingDirectory error:%s", err)
			return nil, "", err
		}
	}

	return wdFile, wdAbs, nil
}

func (w *CurrentWorker) setupKeysDirectory(ctx context.Context, jobName string) (afero.File, string, error) {
	keysDirectory, err := setupDirectory(ctx, w.basedir, jobName, "keys")
	if err != nil {
		return nil, "", err
	}

	fs := w.basedir
	if err := fs.MkdirAll(keysDirectory, 0700); err != nil {
		return nil, "", err
	}

	kdFile, err := w.basedir.Open(keysDirectory)
	if err != nil {
		return nil, "", err
	}

	kdAbs, err := filepath.Abs(kdFile.Name())
	if err != nil {
		return nil, "", err
	}

	switch x := w.basedir.(type) {
	case *afero.BasePathFs:
		kdAbs, err = x.RealPath(kdFile.Name())
		if err != nil {
			return nil, "", err
		}

		kdAbs, err = filepath.Abs(kdAbs)
		if err != nil {
			return nil, "", err
		}
	}
	return kdFile, kdAbs, nil
}

func (w *CurrentWorker) setupHooksDirectory(ctx context.Context, jobName string) (afero.File, string, error) {
	hooksDirectory, err := setupDirectory(ctx, w.basedir, jobName, "hooks")
	if err != nil {
		return nil, "", err
	}

	fs := w.basedir
	if err := fs.MkdirAll(hooksDirectory, 0700); err != nil {
		return nil, "", err
	}

	hdFile, err := w.basedir.Open(hooksDirectory)
	if err != nil {
		return nil, "", err
	}

	hdAbs, err := filepath.Abs(hdFile.Name())
	if err != nil {
		return nil, "", err
	}

	switch x := w.basedir.(type) {
	case *afero.BasePathFs:
		hdAbs, err = x.RealPath(hdFile.Name())
		if err != nil {
			return nil, "", err
		}

		hdAbs, err = filepath.Abs(hdAbs)
		if err != nil {
			return nil, "", err
		}
	}

	return hdFile, hdAbs, nil
}

func (w *CurrentWorker) setupTmpDirectory(ctx context.Context, jobName string) (afero.File, string, error) {
	tmpDirectory, err := setupDirectory(ctx, w.basedir, jobName, "tmp")
	if err != nil {
		return nil, "", err
	}

	fs := w.basedir
	if err := fs.MkdirAll(tmpDirectory, 0700); err != nil {
		return nil, "", err
	}

	tdFile, err := w.basedir.Open(tmpDirectory)
	if err != nil {
		return nil, "", err
	}

	tdAbs, err := filepath.Abs(tdFile.Name())
	if err != nil {
		return nil, "", err
	}

	switch x := w.basedir.(type) {
	case *afero.BasePathFs:
		tdAbs, err = x.RealPath(tdFile.Name())
		if err != nil {
			return nil, "", err
		}

		tdAbs, err = filepath.Abs(tdAbs)
		if err != nil {
			return nil, "", err
		}
	}

	return tdFile, tdAbs, nil
}

// creates a working directory in $HOME/PROJECT/APP/PIP/BN
func setupWorkingDirectory(ctx context.Context, fs afero.Fs, wd string) (afero.File, error) {
	log.Debug(ctx, "creating directory %s in Filesystem %s", wd, fs.Name())
	if err := fs.MkdirAll(wd, 0755); err != nil {
		return nil, err
	}

	u, err := user.Current()
	if err != nil {
		log.Error(ctx, "Error while getting current user %v", err)
	} else if u != nil && u.HomeDir != "" {
		if err := os.Setenv("HOME_CDS_PLUGINS", u.HomeDir); err != nil {
			log.Error(ctx, "Error while setting home_plugin %v", err)
		}
	}

	var absWD string
	if x, ok := fs.(*afero.BasePathFs); ok {
		absWD, _ = x.RealPath(wd)
	} else {
		absWD = wd
	}
	if err := os.Setenv("HOME", absWD); err != nil {
		return nil, err
	}

	fi, err := fs.Open(wd)
	if err != nil {
		return nil, err
	}
	return fi, nil
}

func teardownDirectory(fs afero.Fs, dir string) error {
	return fs.RemoveAll(dir)
}
