package venom

import (
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

func tempDir(t *testing.T) (string, error) {
	dir := os.TempDir()
	name := path.Join(dir, randomString(5))
	if err := os.MkdirAll(name, os.FileMode(0744)); err != nil {
		return "", err
	}
	t.Logf("Creating directory %s", name)
	return name, nil
}

func randomString(n int) string {
	var letter = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}

func Test_getFilesPath(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	log.SetLevel(log.DebugLevel)

	type args struct {
		exclude []string
	}
	tests := []struct {
		init    func(t *testing.T) ([]string, error)
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "Check an empty directory",
			init: func(t *testing.T) ([]string, error) {
				dir, err := tempDir(t)
				return []string{dir}, err
			},
			wantErr: false,
		},
		{
			name: "Check an directory with one yaml file",
			init: func(t *testing.T) ([]string, error) {
				dir, err := tempDir(t)
				if err != nil {
					return nil, err
				}

				d1 := []byte("hello")
				err = ioutil.WriteFile(path.Join(dir, "d1.yml"), d1, 0644)
				return []string{dir}, err
			},
			want:    []string{"d1.yml"},
			wantErr: false,
		},
		{
			name: "Check an directory with one yaml file and a subdirectory with another file",
			init: func(t *testing.T) ([]string, error) {
				dir1, err := tempDir(t)
				if err != nil {
					return nil, err
				}

				d1 := []byte("hello")
				if err = ioutil.WriteFile(path.Join(dir1, "d1.yml"), d1, 0644); err != nil {
					return nil, err
				}

				dir2 := path.Join(dir1, randomString(10))
				t.Logf("Creating directory %s", dir2)

				if err := os.Mkdir(dir2, 0744); err != nil {
					return nil, err
				}

				d2 := []byte("hello")
				if err = ioutil.WriteFile(path.Join(dir2, "d2.yml"), d2, 0644); err != nil {
					return nil, err
				}

				return []string{dir1, dir2}, err
			},
			want:    []string{"d1.yml", "d2.yml"},
			wantErr: false,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			path, err := tt.init(t)
			if err != nil {
				t.Fatal(err)
			}

			got, err := getFilesPath(path, tt.args.exclude)
			if (err != nil) != tt.wantErr {
				t.Errorf("getFilesPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			for _, f := range tt.want {
				var found bool
				for _, g := range got {
					if strings.HasSuffix(g, f) {
						found = true
					}
				}
				if !found {
					t.Errorf("getFilesPath() error want %v got %v", f, got)
				}
			}
		})
	}
}
