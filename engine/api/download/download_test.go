package download

import (
	"context"
	"os"
	"testing"
)

func TestInit(t *testing.T) {
	if os.Getenv("CI") != "1" {
		t.Skip("Skip download test when not running on CI")
	}
	tmpDir1, _ := os.MkdirTemp(os.TempDir(), "download1")
	tmpDir2, _ := os.MkdirTemp(os.TempDir(), "download2")
	defer os.RemoveAll(tmpDir1)
	defer os.RemoveAll(tmpDir2)

	conf := Conf{
		Directory:           tmpDir1,
		DownloadFromGitHub:  true,
		ForceDownloadGitHub: true,
	}

	if err := Init(context.TODO(), conf); err != nil {
		t.Errorf("Init() error = %v", err)
	}

	conf2 := Conf{
		Directory:          tmpDir2,
		DownloadFromGitHub: false,
	}

	if err := Init(context.TODO(), conf2); err == nil {
		t.Error("Init() should be in error as there is no worker binary downloaded")
	}
}
