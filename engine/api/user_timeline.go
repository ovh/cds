package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getTimelineFilterHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		u := deprecatedGetUser(ctx)
		filter, err := user.LoadTimelineFilter(api.mustDB(), u)
		if err != nil {
			return sdk.WrapError(err, "getTimelineFilterHandler")
		}
		return service.WriteJSON(w, filter, http.StatusOK)
	}
}

func (api *API) postTimelineFilterHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		u := deprecatedGetUser(ctx)
		var timelineFilter sdk.TimelineFilter
		if err := service.UnmarshalBody(r, &timelineFilter); err != nil {
			return sdk.WrapError(err, "Unable to read body")
		}

		// Try to load
		count, errLoad := user.CountTimelineFilter(api.mustDB(), u)
		if errLoad != nil {
			return sdk.WrapError(errLoad, "Cannot load filter")
		}
		if count == 0 {
			if err := user.InsertTimelineFilter(api.mustDB(), timelineFilter, u); err != nil {
				return sdk.WrapError(err, "Cannot insert filter")
			}
		} else {
			if err := user.UpdateTimelineFilter(api.mustDB(), timelineFilter, u); err != nil {
				return sdk.WrapError(err, "Unable to update filter")
			}
		}
		return nil
	}
}
