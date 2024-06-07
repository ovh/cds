package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) postGenerateWorkflowFromTemplateHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {

			var tmplGen sdk.V2WorkflowTemplateGenerateRequest
			if err := service.UnmarshalBody(req, &tmplGen); err != nil {
				return err
			}

			errs := tmplGen.Template.Lint()
			if len(errs) > 0 {
				errorsS := ""
				for _, e := range errs {
					errorsS = e.Error() + "\n"
				}
				resp := sdk.V2WorkflowTemplateGenerateResponse{
					Error: errorsS,
				}
				return service.WriteJSON(w, resp, http.StatusOK)
			}

			//craft workflow
			work := sdk.V2Workflow{
				Parameters: tmplGen.Params,
			}

			yamlWorkflow, err := tmplGen.Template.Resolve(ctx, &work)
			if err != nil {
				resp := sdk.V2WorkflowTemplateGenerateResponse{
					Error:    fmt.Sprintf("%v", err.Error()),
					Workflow: yamlWorkflow,
				}

				return service.WriteJSON(w, resp, http.StatusOK)
			}
			resp := sdk.V2WorkflowTemplateGenerateResponse{
				Workflow: yamlWorkflow,
			}
			return service.WriteJSON(w, resp, http.StatusOK)
		}
}
