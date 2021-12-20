package sdk

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

// This code is heavility inspired by standard golang exec package

// ErrExecutableNotFound is the error resulting if a path search failed to find an executable file.
var ErrExecutableNotFound = errors.New("executable file not found in $PATH")

func findExecutable(fs afero.Fs, file string) error {
	d, err := fs.Stat(file)
	if err != nil {
		return errors.WithStack(err)
	}

	m := d.Mode()

	isWindows := strings.EqualFold(runtime.GOOS, "windows")
	isExecutable := m&0111 != 0

	// Don't check exec permission for Windows.
	// File mode will only return if the file is readonly or not.
	// https://pkg.go.dev/os#Chmod
	if !m.IsDir() && (isWindows || isExecutable) {
		return nil
	}
	return errors.Wrapf(os.ErrPermission, "invalid permission %q", m)
}

// LookPath searches for an executable named file in the
// directories named by the PATH environment variable.
// If file contains a slash, it is tried directly and the PATH is not consulted.
// The result may be an absolute path or a path relative to the current directory.
func LookPath(fs afero.Fs, file string) (string, error) {
	// NOTE(rsc): I wish we could use the Plan 9 behavior here
	// (only bypass the path if file begins with / or ./ or ../)
	// but that would not match all the Unix shells.

	if strings.Contains(file, "/") {
		if err := findExecutable(fs, file); err != nil {
			return "", err
		}
		return file, nil
	}
	path := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(path) {
		if dir == "" {
			// Unix shell semantics: path element "" means "."
			dir = "."
		}
		path := filepath.Join(dir, file)
		if err := findExecutable(fs, path); err == nil {
			return path, nil
		}
	}
	return "", ErrExecutableNotFound
}
