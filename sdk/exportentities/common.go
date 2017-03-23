package exportentities

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

//GetFormat return a format
func GetFormat(f string) (Format, error) {
	s := strings.ToLower(f)
	s = strings.TrimSpace(s)
	switch s {
	case "yaml", "yml":
		return FormatYAML, nil
	case "json":
		return FormatJSON, nil
	case "hcl":
		return FormatHCL, nil
	case "toml", "tml":
		return FormatTOML, nil
	default:
		return UnknownFormat, ErrUnsupportedFormat
	}
}

//Marshal suppoets JSON, YAML and HCL
func Marshal(i interface{}, f Format) ([]byte, error) {
	o, ok := i.(HCLable)
	if f == FormatHCL && !ok {
		return nil, ErrUnsupportedHCLFormat
	}

	var btes []byte
	var errMarshal error
	switch f {
	case FormatJSON:
		btes, errMarshal = json.Marshal(i)
	case FormatYAML:
		btes, errMarshal = yaml.Marshal(i)
	case FormatHCL:
		t, err := o.HCLTemplate()
		if err != nil {
			return nil, err
		}
		buff := new(bytes.Buffer)
		errMarshal = t.Execute(buff, o)
		btes = buff.Bytes()
	}
	return btes, errMarshal
}

// ReadFile reads the file and return the content, the format and eventually an error
func ReadFile(filename string) ([]byte, Format, error) {
	format := FormatYAML
	if strings.HasSuffix(filename, ".json") {
		format = FormatJSON
	} else if strings.HasSuffix(filename, ".hcl") {
		format = FormatHCL
	}

	btes, err := ioutil.ReadFile(filename)
	return btes, format, err
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
