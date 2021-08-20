package exportentities

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/sdk"
)

// GetFormat return a format.
func GetFormat(f string) (Format, error) {
	s := strings.ToLower(f)
	s = strings.TrimSpace(s)
	switch s {
	case "yaml", "yml":
		return FormatYAML, nil
	case "json":
		return FormatJSON, nil
	default:
		return UnknownFormat, sdk.NewErrorFrom(sdk.ErrWrongRequest, "format is not supported")
	}
}

// GetFormatFromPath return a format.
func GetFormatFromPath(f string) (Format, error) {
	s := strings.ToLower(f)
	s = strings.TrimSpace(s)
	s = filepath.Ext(s)
	switch s {
	case ".yaml", ".yml":
		return FormatYAML, nil
	case ".json":
		return FormatJSON, nil
	default:
		return UnknownFormat, sdk.NewErrorFrom(sdk.ErrWrongRequest, "format is not supported")
	}
}

// GetFormatFromContentType return a format.
func GetFormatFromContentType(ct string) (Format, error) {
	switch ct {
	case "application/x-yaml", "text/x-yaml":
		return FormatYAML, nil
	case "application/json":
		return FormatJSON, nil
	default:
		return UnknownFormat, sdk.NewErrorFrom(sdk.ErrWrongRequest, "format is not supported")
	}
}

// Unmarshal supports JSON, YAML
func Unmarshal(btes []byte, f Format, i interface{}) error {
	switch f {
	case FormatJSON:
		if err := sdk.JSONUnmarshal(btes, i); err != nil {
			return sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "cannot unmarshal given data as json"))
		}
		return nil
	case FormatYAML:
		if err := yaml.Unmarshal(btes, i); err != nil {
			return sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "cannot unmarshal given data as yaml"))
		}
		return nil
	}
	return sdk.NewErrorFrom(sdk.ErrWrongRequest, "format is not supported")
}

// UnmarshalStrict supports JSON, YAML
func UnmarshalStrict(btes []byte, f Format, i interface{}) error {
	var err error
	switch f {
	case FormatJSON:
		err = sdk.JSONUnmarshal(btes, i)
	case FormatYAML:
		err = yaml.UnmarshalStrict(btes, i)
	default:
		err = sdk.NewErrorFrom(sdk.ErrWrongRequest, "format is not supported")
	}
	return sdk.WithStack(err)
}

//Marshal supports JSON, YAML
func Marshal(i interface{}, f Format) ([]byte, error) {
	var btes []byte
	var err error
	switch f {
	case FormatJSON:
		btes, err = json.Marshal(i)
	case FormatYAML:
		btes, err = yaml.Marshal(i)
	}
	return btes, sdk.WithStack(err)
}

//OpenFile opens a file
func OpenFile(filename string) (io.ReadCloser, error) {
	r, err := os.Open(filename)
	return r, err
}

// OpenURL opens an URL
func OpenURL(u string) (io.ReadCloser, error) {
	netClient := &http.Client{
		Timeout: time.Second * 10,
	}
	response, err := netClient.Get(u)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	return response.Body, err
}

// OpenPath opens an URL or a file
func OpenPath(path string) (io.ReadCloser, Format, error) {
	format, err := GetFormatFromPath(path)
	if err != nil {
		return nil, format, err
	}

	var contentFile io.ReadCloser
	if sdk.IsURL(path) {
		var err error
		contentFile, err = OpenURL(path)
		if err != nil {
			return nil, format, err
		}
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, format, sdk.WithStack(err)
		}
		contentFile = f
	}

	return contentFile, format, nil
}
