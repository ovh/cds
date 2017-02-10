package exportentities

import (
	"bytes"
	"encoding/json"
	"strings"

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
