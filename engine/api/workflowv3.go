package api

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/featureflipping"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/workflowv3"
)

func (api *API) postWorkflowV3ValidateHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]

		_, enabled := featureflipping.IsEnabled(ctx, gorpmapping.Mapper, api.mustDB(), sdk.FeatureWorkflowV3, map[string]string{
			"project_key": projectKey,
		})
		if !enabled {
			return sdk.WrapError(sdk.ErrForbidden, "workflow v3 is not enabled for project %s", projectKey)
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return sdk.NewError(sdk.ErrWrongRequest, err)
		}
		defer r.Body.Close()

		var res workflowv3.ValidationResponse

		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(body)
		}
		format, err := exportentities.GetFormatFromContentType(contentType)
		if err != nil {
			res.Error = sdk.ExtractHTTPError(err).Error()
			return service.WriteJSON(w, res, http.StatusOK)
		}

		var workflow workflowv3.Workflow
		if err := exportentities.Unmarshal(body, format, &workflow); err != nil {
			res.Error = sdk.ExtractHTTPError(sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid workflow v3 format: %v", err)).Error()
			return service.WriteJSON(w, res, http.StatusOK)
		}

		res.Workflow = workflow

		// Static validation for workflow
		extDep, err := workflow.Validate()

		res.Valid = err == nil
		if err != nil {
			res.Error = sdk.ExtractHTTPError(sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid workflow v3 format: %v", err)).Error()
		}
		res.ExternalDependencies = extDep

		return service.WriteJSON(w, res, http.StatusOK)
	}
}

type workflowv3ProxyWriter struct {
	header     http.Header
	buf        bytes.Buffer
	statusCode int
}

func (w *workflowv3ProxyWriter) Header() http.Header {
	return w.header
}

func (w *workflowv3ProxyWriter) Write(bs []byte) (int, error) {
	return w.buf.Write(bs)
}

func (w *workflowv3ProxyWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func (api *API) getWorkflowV3Handler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]

		_, enabled := featureflipping.IsEnabled(ctx, gorpmapping.Mapper, api.mustDB(), sdk.FeatureWorkflowV3, map[string]string{
			"project_key": projectKey,
		})
		if !enabled {
			return sdk.WrapError(sdk.ErrForbidden, "workflow v3 is not enabled for project %s", projectKey)
		}

		full := service.FormBool(r, "full")
		format := FormString(r, "format")
		if format == "" {
			format = "yaml"
		}
		f, err := exportentities.GetFormat(format)
		if err != nil {
			return err
		}

		p := workflowv3ProxyWriter{header: make(http.Header)}

		r.Form = url.Values{}
		r.Form.Add("withDeepPipelines", "true")
		if err := api.getWorkflowHandler()(ctx, &p, r); err != nil {
			return err
		}

		var wk sdk.Workflow
		if err := sdk.JSONUnmarshal(p.buf.Bytes(), &wk); err != nil {
			return sdk.WithStack(err)
		}

		res := workflowv3.Convert(wk, full)

		buf, err := exportentities.Marshal(res, f)
		if err != nil {
			return err
		}
		if _, err := w.Write(buf); err != nil {
			return sdk.WithStack(err)
		}

		w.Header().Add("Content-Type", f.ContentType())
		return nil
	}
}

func (api *API) getWorkflowV3RunHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]

		_, enabled := featureflipping.IsEnabled(ctx, gorpmapping.Mapper, api.mustDB(), sdk.FeatureWorkflowV3, map[string]string{
			"project_key": projectKey,
		})
		if !enabled {
			return sdk.WrapError(sdk.ErrForbidden, "workflow v3 is not enabled for project %s", projectKey)
		}

		full := service.FormBool(r, "full")
		format := FormString(r, "format")
		if format == "" {
			format = "yaml"
		}
		f, err := exportentities.GetFormat(format)
		if err != nil {
			return err
		}

		p := workflowv3ProxyWriter{header: make(http.Header)}

		if err := api.getWorkflowRunHandler()(ctx, &p, r); err != nil {
			return err
		}

		var wkr sdk.WorkflowRun
		if err := sdk.JSONUnmarshal(p.buf.Bytes(), &wkr); err != nil {
			return err
		}

		res := workflowv3.ConvertRun(&wkr, full)

		buf, err := exportentities.Marshal(res, f)
		if err != nil {
			return err
		}
		if _, err := w.Write(buf); err != nil {
			return sdk.WithStack(err)
		}

		w.Header().Add("Content-Type", f.ContentType())

		return nil
	}
}
