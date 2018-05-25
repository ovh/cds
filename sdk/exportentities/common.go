package exportentities

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

//GetFormat return a format
func GetFormat(f string) (Format, error) {
	s := strings.ToLower(f)
	s = strings.TrimSpace(s)
	switch s {
	case "yaml", "yml", ".yaml", ".yml":
		return FormatYAML, nil
	case "json", ".json":
		return FormatJSON, nil
	case "toml", "tml", ".toml", ".tml":
		return FormatTOML, nil
	default:
		return UnknownFormat, ErrUnsupportedFormat
	}
}

//GetContentType returns the content type for a content type
func GetContentType(f Format) string {
	switch f {
	case FormatYAML:
		return "application/x-yaml"
	case FormatJSON:
		return "application/json"
	default:
		return "application/octet-stream"
	}
}

//Unmarshal supports JSON, YAML
func Unmarshal(btes []byte, f Format, i interface{}) error {
	var errMarshal error
	switch f {
	case FormatJSON:
		errMarshal = json.Unmarshal(btes, i)
	case FormatYAML:
		errMarshal = yaml.Unmarshal(btes, i)
	default:
		errMarshal = ErrUnsupportedFormat
	}
	return errMarshal
}

//Marshal supports JSON, YAML
func Marshal(i interface{}, f Format) ([]byte, error) {
	var btes []byte
	var errMarshal error
	switch f {
	case FormatJSON:
		btes, errMarshal = json.Marshal(i)
	case FormatYAML:
		btes, errMarshal = yaml.Marshal(i)
	}
	return btes, errMarshal
}

// ReadFile reads the file and return the content, the format and eventually an error
func ReadFile(filename string) ([]byte, Format, error) {
	format := FormatYAML
	if strings.HasSuffix(filename, ".json") {
		format = FormatJSON
	}

	btes, err := ioutil.ReadFile(filename)
	return btes, format, err
}

//OpenFile opens a file
func OpenFile(filename string) (io.ReadCloser, Format, error) {
	format := FormatYAML
	if strings.HasSuffix(filename, ".json") {
		format = FormatJSON
	}
	r, err := os.Open(filename)
	return r, format, err
}

// ReadURL reads the file given by an URL
func ReadURL(u string, f string) ([]byte, Format, error) {
	format, err := GetFormat(f)
	if err != nil {
		return nil, format, err
	}
	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}

	response, err := netClient.Get(u)
	if err != nil {
		return nil, format, err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, format, err
	}
	defer response.Body.Close()

	return body, format, nil
}

// OpenURL opens an URL
func OpenURL(u string, f string) (io.ReadCloser, Format, error) {
	format, err := GetFormat(f)
	if err != nil {
		return nil, format, err
	}
	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}

	response, err := netClient.Get(u)
	return response.Body, format, err
}
