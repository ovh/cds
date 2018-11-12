package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

/* Inside contrib/grpcplugins/action
$ make build download
$ make publish download
*/

type downloadActionPlugin struct {
	actionplugin.Common
}

func (actPlugin *downloadActionPlugin) Manifest(ctx context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "plugin-download",
		Author:      "Benjamin COENEN <benjamin.coenen@corp.ovh.com>",
		Description: "This is a plugin to download file from URL",
		Version:     sdk.VERSION,
	}, nil
}

func (actPlugin *downloadActionPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	filepath := q.GetOptions()["filepath"]
	url := q.GetOptions()["url"]
	headers := q.GetOptions()["headers"]

	// Create the file
	file, err := os.Create(filepath)
	if err != nil {
		return fail("Error to create the file %s : %s", filepath, err)
	}
	defer file.Close()

	// Download from URL
	httpClient := &http.Client{
		Timeout: time.Second * 10,
	}
	req, err := http.NewRequest("GET", url, nil)

	req.Header = parseHeaders(headers)

	if err != nil {
		return fail("Error to create request with URL %s : %s", url, err)
	}

	resp, err := httpClient.Do(req)

	if err != nil {
		return fail("Error to download the file on URL %s : %s", url, err)
	}
	defer resp.Body.Close()

	// Copy file in the right directory
	if _, err := io.Copy(file, resp.Body); err != nil {
		return fail("Error to copy the file on URL %s : %s", url, err)
	}

	return &actionplugin.ActionResult{
		Status: sdk.StatusSuccess.String(),
	}, nil
}

func (actPlugin *downloadActionPlugin) WorkerHTTPPort(ctx context.Context, q *actionplugin.WorkerHTTPPortQuery) (*empty.Empty, error) {
	actPlugin.HTTPPort = q.Port
	return &empty.Empty{}, nil
}

func main() {
	actPlugin := downloadActionPlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
	return

}

func fail(format string, args ...interface{}) (*actionplugin.ActionResult, error) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(msg)
	return &actionplugin.ActionResult{
		Details: msg,
		Status:  sdk.StatusFail.String(),
	}, nil
}

func parseHeaders(hParams string) http.Header {
	headers := http.Header{}
	regx := regexp.MustCompile(`"(.+)"="(.+)"`)
	subStrList := regx.FindAllStringSubmatch(hParams, -1)

	for _, subStr := range subStrList {
		if len(subStr) < 3 {
			continue
		}

		headers.Add(subStr[1], subStr[2])
	}

	return headers
}
