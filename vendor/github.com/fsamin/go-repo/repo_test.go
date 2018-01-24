package repo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFetchURL(t *testing.T) {
	r, err := New(".")
	assert.NoError(t, err)

	u, err := r.FetchURL()
	assert.NoError(t, err)

	t.Logf("url: %v", u)

	n, err := r.Name()
	assert.NoError(t, err)

	t.Logf("name: %v", n)

}

func Test_trimURL(t *testing.T) {
	type args struct {
		fetchURL string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "git@github.com:ovh/cds.git",
			args:    args{"git@github.com:ovh/cds.git"},
			want:    "ovh/cds",
			wantErr: false,
		},
		{
			name:    "ssh://git@my.gitserver.net:7999/ovh/cds.git",
			args:    args{"ssh://git@my.gitserver.net:7999/ovh/cds.git"},
			want:    "ovh/cds",
			wantErr: false,
		},
		{
			name:    "https://github.com/ovh/cds",
			args:    args{"https://github.com/ovh/cds"},
			want:    "ovh/cds",
			wantErr: false,
		},
		{
			name:    "https://francois.samin@stash.ovh.net/scm/ovh/cds.git",
			args:    args{"https://francois.samin@my.gitserver.net/scm/ovh/cds.git"},
			want:    "ovh/cds",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := trimURL(tt.args.fetchURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("trimURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("trimURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLocalConfigGet(t *testing.T) {
	r, err := New(".")
	assert.NoError(t, err)

	assert.NoError(t, r.LocalConfigSet("foo", "bar", "value"))

	val, err := r.LocalConfigGet("foo", "bar")
	assert.NoError(t, err)
	assert.Equal(t, "value", val)
}

func TestLatestCommit(t *testing.T) {
	r, err := New(".")
	assert.NoError(t, err)

	c, err := r.LatestCommit()
	t.Logf("%+v", c)
	assert.NoError(t, err)
}
