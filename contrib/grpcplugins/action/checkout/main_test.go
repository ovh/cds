package main

import (
	"testing"
)

func Test_sanitizeGitURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "hybrid ssh with SCP colon",
			raw:  "ssh://git@github.com:ovh/manager.git",
			want: "ssh://git@github.com/ovh/manager.git",
		},
		{
			name: "hybrid ssh with SCP colon and deep path",
			raw:  "ssh://git@github.com:ovh/cds.git",
			want: "ssh://git@github.com/ovh/cds.git",
		},
		{
			name: "valid ssh with numeric port",
			raw:  "ssh://git@bitbucket:7999/foo/bar.git",
			want: "ssh://git@bitbucket:7999/foo/bar.git",
		},
		{
			name: "valid ssh without port",
			raw:  "ssh://git@github.com/ovh/manager.git",
			want: "ssh://git@github.com/ovh/manager.git",
		},
		{
			name: "SCP-like without scheme",
			raw:  "git@github.com:ovh/manager.git",
			want: "git@github.com:ovh/manager.git",
		},
		{
			name: "https URL",
			raw:  "https://github.com/ovh/manager.git",
			want: "https://github.com/ovh/manager.git",
		},
		{
			name: "empty string",
			raw:  "",
			want: "",
		},
		{
			name: "ssh scheme only host no colon",
			raw:  "ssh://git@github.com",
			want: "ssh://git@github.com",
		},
		{
			name: "hybrid ssh with port-like but alpha",
			raw:  "ssh://git@myhost:myproject/repo.git",
			want: "ssh://git@myhost/myproject/repo.git",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeGitURL(tt.raw)
			if got != tt.want {
				t.Errorf("sanitizeGitURL(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}
