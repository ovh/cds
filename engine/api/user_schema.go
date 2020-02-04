package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/alecthomas/jsonschema"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/slug"
)

func (api *API) getUserJSONSchema() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		filter := FormString(r, "filter")

		var res sdk.SchemaResponse

		ref := jsonschema.Reflector{
			RequiredFromJSONSchemaTags: true,
		}

		var sch *jsonschema.Schema
		if filter == "" || filter == "workflow" {
			sch = ref.ReflectFromType(reflect.TypeOf(exportentities.Workflow{}))
			buf, _ := json.Marshal(sch)
			res.Workflow = string(buf)
		}

		if filter == "" || filter == "pipeline" {
			var as []sdk.Action
			var err error
			if isMaintainer(ctx) || isAdmin(ctx) {
				as, err = action.LoadAllByTypes(ctx, api.mustDB(),
					[]string{sdk.DefaultAction},
					action.LoadOptions.WithGroup,
					action.LoadOptions.WithParameters,
				)
			} else {
				as, err = action.LoadAllTypeDefaultByGroupIDs(ctx, api.mustDB(),
					append(getAPIConsumer(ctx).GetGroupIDs(), group.SharedInfraGroup.ID),
					action.LoadOptions.WithGroup,
					action.LoadOptions.WithParameters,
				)
			}
			if err != nil {
				return err
			}

			sch = ref.ReflectFromType(reflect.TypeOf(exportentities.PipelineV1{}))
			for i := range as {
				path := as[i].Name
				if as[i].Group.Name != sdk.SharedInfraGroupName {
					path = fmt.Sprintf("%s/%s", as[i].Group.Name, as[i].Name)
				}
				s := slug.Convert(path)
				sch.Definitions["Step"].Properties[path] = &jsonschema.Type{
					Version:     "http://json-schema.org/draft-04/schema#",
					Ref:         "#/definitions/" + s,
					Description: as[i].Description,
				}
				sch.Definitions["Step"].OneOf = append(sch.Definitions["Step"].OneOf, &jsonschema.Type{
					Required: []string{
						path,
					},
					Title: path,
				})

				sch.Definitions[s] = &jsonschema.Type{
					Properties:           map[string]*jsonschema.Type{},
					AdditionalProperties: sch.Definitions["Step"].AdditionalProperties,
					Type:                 "object",
				}
				for j := range as[i].Parameters {
					p := as[i].Parameters[j]
					switch p.Type {
					case "number":
						sch.Definitions[s].Properties[p.Name] = &jsonschema.Type{
							Type: "integer",
						}
					case "boolean":
						sch.Definitions[s].Properties[p.Name] = &jsonschema.Type{
							Type: "boolean",
						}
					default:
						sch.Definitions[s].Properties[p.Name] = &jsonschema.Type{
							Type: "string",
						}
					}
				}
			}

			buf, _ := json.Marshal(sch)
			res.Pipeline = string(buf)
		}

		if filter == "" || filter == "application" {
			sch = ref.ReflectFromType(reflect.TypeOf(exportentities.Application{}))
			buf, _ := json.Marshal(sch)
			res.Application = string(buf)
		}

		if filter == "" || filter == "environment" {
			sch = ref.ReflectFromType(reflect.TypeOf(exportentities.Environment{}))
			buf, _ := json.Marshal(sch)
			res.Environment = string(buf)
		}

		return service.WriteJSON(w, res, http.StatusOK)
	}
}
