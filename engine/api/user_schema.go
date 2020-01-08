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

		var res sdk.SchemaResponse

		ref := jsonschema.Reflector{
			RequiredFromJSONSchemaTags: true,
		}

		sch := ref.ReflectFromType(reflect.TypeOf(exportentities.Workflow{}))
		buf, _ := json.MarshalIndent(sch, "", "\t")
		res.Workflow = string(buf)

		sch = ref.ReflectFromType(reflect.TypeOf(exportentities.PipelineV1{}))

		for i := range as {
			path := fmt.Sprintf("%s/%s", as[i].Group.Name, as[i].Name)
			s := slug.Convert(path)
			sch.Definitions["Step"].Properties[path] = &jsonschema.Type{
				Version:     "http://json-schema.org/draft-04/schema#",
				Ref:         "#/definitions/" + s,
				Description: as[i].Description,
			}

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

		buf, _ = json.MarshalIndent(sch, "", "\t")
		res.Pipeline = string(buf)

		sch = ref.ReflectFromType(reflect.TypeOf(exportentities.Application{}))
		buf, _ = json.MarshalIndent(sch, "", "\t")
		res.Application = string(buf)

		sch = ref.ReflectFromType(reflect.TypeOf(exportentities.Environment{}))
		buf, _ = json.MarshalIndent(sch, "", "\t")
		res.Environment = string(buf)

		return service.WriteJSON(w, res, http.StatusOK)
	}
}
