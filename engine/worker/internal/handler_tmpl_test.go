package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func Test_tmplHandler(t *testing.T) {
	var wk = new(CurrentWorker)
	fs := afero.NewOsFs()
	basedir := "test-" + test.GetTestName(t) + "-" + sdk.RandomString(10) + "-" + fmt.Sprintf("%d", time.Now().Unix())
	require.NoError(t, fs.MkdirAll(basedir, os.FileMode(0755)))

	if err := wk.Init("test-worker", "test-hatchery", "http://lolcat.host", "xxx-my-token", "", true, afero.NewBasePathFs(fs, basedir)); err != nil {
		t.Fatalf("worker init failed: %v", err)
	}
	wk.currentJob.params = []sdk.Parameter{
		{
			Name:  "cds.stuff",
			Value: "stuff",
		},
	}
	wk.currentJob.secrets = []sdk.Variable{
		{
			Name:  "cds.stuff.secret",
			Value: "secret stuff",
		},
	}

	f, err := fs.OpenFile("input", os.O_CREATE|os.O_RDWR, os.FileMode(0644))
	require.NoError(t, err)

	f.WriteString("{{.cds.stuff}}\n{{.cds.stuff.secret}}")
	f.Close()

	in := workerruntime.TmplPath{
		Path:        f.Name(),
		Destination: "output",
	}

	btes, _ := json.Marshal(in)

	handler := tmplHandler(context.TODO(), wk)
	w := httptest.NewRecorder()
	r, err := http.NewRequest("POST", "http://lolcat.host/", bytes.NewReader(btes))
	require.NoError(t, err)
	handler(w, r)

	t.Logf("result: %d : %v", w.Code, string(w.Body.Bytes()))

	output, err := fs.Open("output")
	require.NoError(t, err)

	btes, err = ioutil.ReadAll(output)
	require.NoError(t, err)

	t.Logf("output content: %v", string(btes))
	require.Equal(t, "stuff\nsecret stuff", string(btes))
}

func Test_tmplHandlerInWrongDir(t *testing.T) {
	var wk = new(CurrentWorker)
	fs := afero.NewOsFs()
	basedir := "test-" + test.GetTestName(t) + "-" + sdk.RandomString(10) + "-" + fmt.Sprintf("%d", time.Now().Unix())
	require.NoError(t, fs.MkdirAll(basedir, os.FileMode(0755)))

	if err := wk.Init("test-worker", "test-hatchery", "http://lolcat.host", "xxx-my-token", "", true, afero.NewBasePathFs(fs, basedir)); err != nil {
		t.Fatalf("worker init failed: %v", err)
	}
	wk.currentJob.params = []sdk.Parameter{
		{
			Name:  "cds.stuff",
			Value: "stuff",
		},
	}
	wk.currentJob.secrets = []sdk.Variable{
		{
			Name:  "cds.stuff.secret",
			Value: "secret stuff",
		},
	}

	f, err := fs.OpenFile("input", os.O_CREATE|os.O_RDWR, os.FileMode(0644))
	require.NoError(t, err)

	f.WriteString("{{.cds.stuff}}\n{{.cds.stuff.secret}}")
	f.Close()

	in := workerruntime.TmplPath{
		Path:        f.Name(),
		Destination: "adir/output",
	}

	btes, _ := json.Marshal(in)

	handler := tmplHandler(context.TODO(), wk)
	w := httptest.NewRecorder()
	r, err := http.NewRequest("POST", "http://lolcat.host/", bytes.NewReader(btes))
	require.NoError(t, err)
	handler(w, r)

	body := w.Body.String()
	t.Logf("result: %d : %v", w.Code, body)
	require.Equal(t, `{"id":74,"message":"wrong request","from":"open adir/output: no such file or directory"}`, body)

}
