package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getWorkerModelPatternsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		modelPatterns, err := workermodel.LoadPatterns(ctx, api.mustDB())
		if err != nil {
			return err
		}
		return service.WriteJSON(w, modelPatterns, http.StatusOK)
	}
}

func (api *API) getWorkerModelPatternHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		patternName := vars["name"]
		patternType := vars["type"]

		modelPattern, err := workermodel.LoadPatternByNameAndType(ctx, api.mustDB(), patternType, patternName)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, modelPattern, http.StatusOK)
	}
}

func (api *API) postAddWorkerModelPatternHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if !isAdmin(ctx) {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		trackSudo(ctx, w)

		var modelPattern sdk.ModelPattern
		if err := service.UnmarshalBody(r, &modelPattern); err != nil {
			return err
		}

		if !sdk.NamePatternRegex.MatchString(modelPattern.Name) {
			return sdk.WithStack(sdk.ErrInvalidName)
		}

		if modelPattern.Model.Cmd == "" {
			return sdk.WithStack(sdk.ErrInvalidPatternModel)
		}

		if modelPattern.Type == sdk.Docker && modelPattern.Model.Shell == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "cannot add a worker model pattern for %s without shell command", sdk.Docker)
		}

		var typeFound bool
		for _, availableType := range sdk.AvailableWorkerModelType {
			if availableType == modelPattern.Type {
				typeFound = true
				break
			}
		}
		if !typeFound {
			return sdk.WithStack(sdk.ErrInvalidPatternModel)
		}

		// Insert model pattern in db
		if err := workermodel.InsertPattern(api.mustDB(), &modelPattern); err != nil {
			return sdk.WrapError(err, "cannot add worker model pattern")
		}

		return service.WriteJSON(w, modelPattern, http.StatusOK)
	}
}

func (api *API) putWorkerModelPatternHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if !isAdmin(ctx) {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		trackSudo(ctx, w)

		vars := mux.Vars(r)
		patternName := vars["name"]
		patternType := vars["type"]

		var modelPattern sdk.ModelPattern
		if err := service.UnmarshalBody(r, &modelPattern); err != nil {
			return err
		}

		if !sdk.NamePatternRegex.MatchString(modelPattern.Name) {
			return sdk.WithStack(sdk.ErrInvalidName)
		}

		if modelPattern.Model.Cmd == "" {
			return sdk.WithStack(sdk.ErrInvalidPatternModel)
		}

		if modelPattern.Type == sdk.Docker && modelPattern.Model.Shell == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "cannot update a worker model pattern for %s without shell command", sdk.Docker)
		}

		var typeFound bool
		for _, availableType := range sdk.AvailableWorkerModelType {
			if availableType == modelPattern.Type {
				typeFound = true
				break
			}
		}
		if !typeFound {
			return sdk.WithStack(sdk.ErrInvalidPatternModel)
		}

		oldWmp, err := workermodel.LoadPatternByNameAndType(ctx, api.mustDB(), patternType, patternName)
		if err != nil {
			return err
		}
		modelPattern.ID = oldWmp.ID

		if err := workermodel.UpdatePattern(api.mustDB(), &modelPattern); err != nil {
			return sdk.WrapError(err, "cannot update worker model pattern")
		}

		return service.WriteJSON(w, modelPattern, http.StatusOK)
	}
}

func (api *API) deleteWorkerModelPatternHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if !isAdmin(ctx) {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		trackSudo(ctx, w)

		vars := mux.Vars(r)
		patternName := vars["name"]
		patternType := vars["type"]

		wmp, err := workermodel.LoadPatternByNameAndType(ctx, api.mustDB(), patternType, patternName)
		if err != nil {
			return err
		}

		if err := workermodel.DeletePatternByID(api.mustDB(), wmp.ID); err != nil {
			return sdk.WrapError(err, "cannot delete worker model (%s/%s) : %v", patternType, patternName, err)
		}

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}
