package git

import (
	"bytes"
	"os"
	"os/user"
	"reflect"
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk/vcs"
)

func TestClone(t *testing.T) {
	type args struct {
		repo   string
		path   string
		auth   *AuthOpts
		opts   *CloneOpts
		output *OutputOpts
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "clone on http is not supported",
			args: args{
				path: "http://github.com/ovh/cds.git",
			},
			wantErr: true,
		},
		{
			name: "clone on ftp is not supported",
			args: args{
				path: "ftp://github.com/ovh/cds.git",
			},
			wantErr: true,
		},
		{
			name: "clone on ftps is not supported",
			args: args{
				path: "ftps://github.com/ovh/cds.git",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		if err := Clone(tt.args.repo, tt.args.path, tt.args.auth, tt.args.opts, tt.args.output); (err != nil) != tt.wantErr {
			t.Errorf("%q. Clone() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func Test_gitCloneOverHTTPS(t *testing.T) {
	type args struct {
		repo   string
		path   string
		auth   *AuthOpts
		opts   *CloneOpts
		output *OutputOpts
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Clone public repo over http without any options",
			args: args{
				repo: "https://github.com/fsamin/go-dump.git",
				path: "/tmp/Test_gitCloneOverHTTPS-1",
			},
			wantErr: false,
		},
		{
			name: "Clone public repo over http options and checkout commit",
			args: args{
				repo: "https://github.com/fsamin/go-dump.git",
				path: "/tmp/Test_gitCloneOverHTTPS-2",
				opts: &CloneOpts{
					CheckoutCommit: "ffa09687b10de28606ad5b7f577f3cebe44cdd56",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		os.RemoveAll(tt.args.path)
		out := new(bytes.Buffer)
		err := new(bytes.Buffer)
		tt.args.output = &OutputOpts{
			Stdout: out,
			Stderr: err,
		}

		if err := Clone(tt.args.repo, tt.args.path, tt.args.auth, tt.args.opts, tt.args.output); (err != nil) != tt.wantErr {
			t.Errorf("%q. gitCloneOverHTTPS() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}

		t.Log(out)
		t.Log(err)
	}
}

func Test_gitCloneOverSSH(t *testing.T) {
	u, err := user.Current()
	test.NoError(t, err)
	homedir := u.HomeDir

	type args struct {
		repo   string
		path   string
		auth   *AuthOpts
		opts   *CloneOpts
		output *OutputOpts
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Clone public repo over http without any options",
			args: args{
				repo: "git@github.com:fsamin/go-dump.git",
				path: "/tmp/Test_gitCloneOverHTTPS-1",
				auth: &AuthOpts{
					PrivateKey: vcs.SSHKey{Filename: homedir + "/.ssh/id_rsa"},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		if tt.args.auth != nil {
			if _, err := os.Stat(tt.args.auth.PrivateKey.Filename); os.IsNotExist(err) {
				t.SkipNow()
			}
		}

		os.RemoveAll(tt.args.path)
		out := new(bytes.Buffer)
		err := new(bytes.Buffer)
		tt.args.output = &OutputOpts{
			Stdout: out,
			Stderr: err,
		}

		if err := Clone(tt.args.repo, tt.args.path, tt.args.auth, tt.args.opts, tt.args.output); (err != nil) != tt.wantErr {
			t.Errorf("%q. gitCloneOverSSH() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}

		t.Log(out)
		t.Log(err)
	}
}

func Test_gitCommand(t *testing.T) {
	type args struct {
		repo string
		path string
		opts *CloneOpts
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Clone public repo over http without any options",
			args: args{
				repo: "https://github.com/ovh/cds.git",
				path: "/tmp/Test_gitCommand-1",
			},
			want: []string{
				"git clone https://github.com/ovh/cds.git /tmp/Test_gitCommand-1",
			},
		},
		{
			name: "Clone public repo over http with options",
			args: args{
				repo: "https://github.com/ovh/cds.git",
				path: "/tmp/Test_gitCommand-2",
				opts: &CloneOpts{
					Branch:    "master",
					Depth:     1,
					Verbose:   true,
					Recursive: true,
				},
			},
			want: []string{
				"git clone --verbose --depth 1 --branch master --no-single-branch --recursive https://github.com/ovh/cds.git /tmp/Test_gitCommand-2",
			},
		},
		{
			name: "Clone public repo over http with options and checkout commit",
			args: args{
				repo: "https://github.com/ovh/cds.git",
				path: "/tmp/Test_gitCommand-3",
				opts: &CloneOpts{
					Branch:         "master",
					Quiet:          true,
					CheckoutCommit: "eb8b87a",
				},
			},
			want: []string{
				"git clone --quiet --branch master https://github.com/ovh/cds.git /tmp/Test_gitCommand-3",
				"git reset --hard eb8b87a",
			},
		},
	}
	for _, tt := range tests {
		os.RemoveAll(tt.args.path)
		if got := prepareGitCloneCommands(tt.args.repo, tt.args.path, tt.args.opts); !reflect.DeepEqual(got.Strings(), tt.want) {
			t.Errorf("%q. gitCloneCommand() = %v, want %v", tt.name, got, tt.want)
		}
	}
}
