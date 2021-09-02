package sdk

import (
	"strings"
	"testing"
)

func TestInitSupportedOSArch(t *testing.T) {
	type args struct {
		supportedOSArchConf []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []string
	}{
		{
			name:    "with wrong value",
			args:    args{supportedOSArchConf: []string{"foo/bar"}},
			wantErr: true,
		},
		{
			name:    "with empty value",
			args:    args{supportedOSArchConf: []string{}},
			wantErr: false,
			want:    []string{"darwin/amd64", "darwin/arm64", "linux/amd64", "linux/ppc64le"}, // and more
		},
		{
			name:    "with good value",
			args:    args{supportedOSArchConf: []string{"darwin/amd64", "darwin/arm64", "linux/amd64"}},
			wantErr: false,
			want:    []string{"darwin/amd64", "darwin/arm64", "linux/amd64"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := InitSupportedOSArch(tt.args.supportedOSArchConf); (err != nil) != tt.wantErr {
				t.Errorf("InitSupportedOSArch() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.want != nil {
				for _, v := range tt.want {
					if !IsInArray(v, supportedOSArch) {
						t.Errorf("InitSupportedOSArch() error = %v not in %v", v, tt.want)
					}
				}
				if len(AllDownloadableResources()) < len(tt.want) {
					t.Errorf("AllDownloadableResources() AllDownloadableResources does not contains all expected value. %v not in %v", len(AllDownloadableResources()), len(tt.want))
				}
			}
		})
	}
}

func TestDownloadURLFromGithub(t *testing.T) {
	type args struct {
		filename string
		version  string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "latest",
			args:    args{filename: "ui.tar.gz", version: "latest"},
			want:    "",
			wantErr: false,
		},
		{
			name:    "0.49.0",
			args:    args{filename: "ui.tar.gz", version: "0.49.0"},
			want:    "https://github.com/ovh/cds/releases/download/0.49.0/ui.tar.gz",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DownloadURLFromGithub(tt.args.filename, tt.args.version)
			if !strings.HasPrefix(got, "https://github.com/ovh/cds/releases/download/") {
				t.Errorf("DownloadURLFromGithub() invalid url found on github %v", got)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("DownloadURLFromGithub() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want != "" && got != tt.want {
				t.Errorf("DownloadURLFromGithub() = %v, want %v", got, tt.want)
			}
		})
	}
}
