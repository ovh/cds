package swagger

import (
	"errors"
	"reflect"
)

type Parameter struct {
	ParamType     string `json:"in"` // path,query,body,header,form
	Name          string `json:"name"`
	Description   string `json:"description"`
	Required      bool   `json:"required"`
	AllowMultiple bool   `json:"-"` // then it's an array

	Type             string   `json:"type,omitempty"`   // integer
	Format           string   `json:"format,omitempty"` // int64
	Enum             []string `json:"enum,omitempty"`
	CollectionFormat string   `json:"collectionFormat,omitempty"` // csv/ssv/tsv/pipe/multi, defaults to csv on swagger spec
	RefId            string   `json:"$ref,omitempty"`
	Minimum          int      `json:"minimum,omitempty"`
	Maximum          int      `json:"maximum,omitempty"`
	Default          string   `json:"default,omitempty"`

	Items  map[string]string `json:"items,omitempty"`
	Schema *Schema           `json:"schema,omitempty"`
}

func NewParameter(paramType string, name string, description string, required bool, allowMultiple bool, dataType, format, refId string) (param Parameter) {

	param.ParamType = paramType
	param.Name = name
	param.Description = description
	param.Required = required
	param.AllowMultiple = allowMultiple
	param.Format = format
	param.Type = dataType
	param.RefId = refId

	if allowMultiple {
		param.Type = "array"
		param.Items = make(map[string]string)
		param.Items["type"] = dataType

		if paramType == "query" {
			param.CollectionFormat = "multi"
		}
	}

	if paramType == "body" {
		schema := Schema{}
		schema.Type = dataType
		param.Schema = &schema
	}

	return
}

func (p Parameter) setTypeAndFormat(kind reflect.Kind) error {
	switch kind {
	case reflect.Int64:
		p.Type = "integer"
		p.Format = "int64"
		return nil
	case reflect.Int, reflect.Int32, reflect.Uint32:
		p.Type = "integer"
		p.Format = "int32"
		return nil
	case reflect.Float64:
		p.Type = "number"
		p.Format = "float"
		return nil
	case reflect.Bool:
		p.Type = "boolean"
		return nil
	case reflect.String:
		p.Type = "string"
		return nil
	}
	return errors.New("unhandled type patch me")
}
