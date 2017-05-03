package main

import (
	"io"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/ovh/cds/sdk/plugin"
)

//DownloadPlugin is a plugin to download a file from an URL
type DownloadPlugin struct {
	plugin.Common
}

//Name return plugin name. It must me the same as the binary file
func (d DownloadPlugin) Name() string {
	return "plugin-download"
}

//Description explains the purpose of the plugin
func (d DownloadPlugin) Description() string {
	return "This is a plugin to download file from URL"
}

//Author of the plugin
func (d DownloadPlugin) Author() string {
	return "Benjamin COENEN <benjamin.coenen@corp.ovh.com>"
}

// Parameters return parameters description
func (d DownloadPlugin) Parameters() plugin.Parameters {
	params := plugin.NewParameters()
	params.Add("url", plugin.StringParameter, "the url of your file", "{{.cds.app.downloadUrl}}")
	params.Add("filepath", plugin.StringParameter, "the destination of your file to be copied", ".")
	params.Add("headers", plugin.TextParameter, `specific headers to add to your request ("headerName"="value" newline separated list)`, "")

	return params
}

// Run execute the action
func (d DownloadPlugin) Run(a plugin.IJob) plugin.Result {
	filepath := a.Arguments().Get("filepath")
	url := a.Arguments().Get("url")
	headers := a.Arguments().Get("headers")

	// Create the file
	file, err := os.Create(filepath)
	if err != nil {
		plugin.SendLog(a, "Error to create the file %s : %s", filepath, err)
		return plugin.Fail
	}
	defer file.Close()

	// Download from URL
	httpClient := &http.Client{
		Timeout: time.Second * 10,
	}
	req, err := http.NewRequest("GET", url, nil)

	req.Header = parseHeaders(headers)

	if err != nil {
		plugin.SendLog(a, "Error to create request with URL %s : %s", url, err)
		return plugin.Fail
	}

	resp, err := httpClient.Do(req)

	if err != nil {
		plugin.SendLog(a, "Error to download the file on URL %s : %s", url, err)
		return plugin.Fail
	}
	defer resp.Body.Close()

	// Copy file in the right directory
	if _, err := io.Copy(file, resp.Body); err != nil {
		plugin.SendLog(a, "Error to copy the file on URL %s : %s", url, err)
		return plugin.Fail
	}

	return plugin.Success
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

func main() {
	plugin.Main(&DownloadPlugin{})
}
