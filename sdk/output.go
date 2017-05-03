package sdk

import (
	"encoding/json"
	"encoding/xml"
	"fmt"

	"gopkg.in/yaml.v2"
)

// Output output result to sdtout, files...
func Output(format string, v interface{}, printFunc func(format string, a ...interface{}) (n int, err error)) error {
	var data []byte
	var err error
	switch format {
	case "json":
		data, err = json.Marshal(v)
		if err != nil {
			return fmt.Errorf("Error: cannot format output json (%s)", err)
		}
	case "yml", "yaml":
		data, err = yaml.Marshal(v)
		if err != nil {
			return fmt.Errorf("Error: cannot format output yaml (%s)", err)
		}
	case "xml":
		dataxml, errm := xml.Marshal(v)
		if errm != nil {
			return fmt.Errorf("Error: cannot format xml output: %s", errm)
		}
		data = append([]byte("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"), dataxml...)
	default:
		return fmt.Errorf("Invalid format %s", format)
	}

	printFunc(string(data))

	return nil
}
