package main

import (
	"os"
	"testing"
)

func Test_downloadSQLTarGz(t *testing.T) {
	type args struct {
		currentVersion string
		artifactName   string
		migrateDir     string
	}
	tmpMigrateDir := os.TempDir()
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "valid version",
			args:    args{currentVersion: "0.37.3+cds.7708", artifactName: "sql.tar.gz", migrateDir: tmpMigrateDir},
			wantErr: false,
		}, {
			name:    "valid version, invalid dir",
			args:    args{currentVersion: "0.37.3+cds.7708", artifactName: "sql.tar.gz", migrateDir: "/tmp/foo"},
			wantErr: true,
		}, {
			name:    "invalid artifact name",
			args:    args{currentVersion: "0.37.3+cds.7708", artifactName: "test-artifact.tar.gz", migrateDir: tmpMigrateDir},
			wantErr: true,
		}, {
			name:    "invalid version without '+'",
			args:    args{currentVersion: "0.37.3", artifactName: "sql.tar.gz", migrateDir: tmpMigrateDir},
			wantErr: true,
		}, {
			name:    "invalid version",
			args:    args{currentVersion: "foo", artifactName: "sql.tar.gz", migrateDir: tmpMigrateDir},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := databaseDownloadSQLTarGz(tt.args.currentVersion, tt.args.artifactName, tt.args.migrateDir); (err != nil) != tt.wantErr {
				t.Errorf("databaseDownloadSQLTarGz() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
