package sdk

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

// IsTar returns true if the content is a tar
func IsTar(buf []byte) bool {
	return len(buf) > 261 &&
		buf[257] == 0x75 && buf[258] == 0x73 &&
		buf[259] == 0x74 && buf[260] == 0x61 &&
		buf[261] == 0x72
}

// IsGz returns true if the content is gzipped
func IsGz(buf []byte) bool {
	return len(buf) > 2 &&
		buf[0] == 0x1F && buf[1] == 0x8B && buf[2] == 0x8
}

// UntarGz takes a destination path and a reader; a tar.gz reader loops over the tarfile
// creating the file structure at 'dst' along the way, and writing any files
func UntarGz(fs afero.Fs, dst string, r io.Reader) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	return Untar(fs, dst, gzr)
}

// Untar takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'dst' along the way, and writing any files
func Untar(fs afero.Fs, dst string, r io.Reader) error {
	tr := tar.NewReader(r)

	for {
		header, err := tr.Next()

		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case header == nil:
			continue
		}

		target := filepath.Join(dst, header.Name)

		// check the file type
		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := fs.Stat(target); err != nil {
				if err := fs.MkdirAll(target, 0755); err != nil {
					return err
				}
			}

		case tar.TypeReg:
			f, err := fs.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				return err
			}
			f.Close()
		}
	}
}
