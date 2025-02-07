package hooks

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (s *Service) listRepositoriesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		filter := r.FormValue("filter")
		keys, err := s.Dao.ListRepositories(ctx, filter)
		if err != nil {
			return err
		}
		return service.WriteJSON(w, keys, http.StatusOK)
	}
}

func (s *Service) getRepositoryEventHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		vcsServer := vars["vcsServer"]
		repo, err := url.PathUnescape(vars["repoName"])
		if err != nil {
			return sdk.WithStack(err)
		}
		uuid := vars["uuid"]

		e, err := s.Dao.GetRepositoryEvent(ctx, vcsServer, repo, uuid)
		if err != nil {
			return err
		}
		return service.WriteJSON(w, e, http.StatusOK)
	}
}

func (s *Service) deleteRepositoryEventHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		vcsServer := vars["vcsServer"]
		repo, err := url.PathUnescape(vars["repoName"])
		if err != nil {
			return sdk.WithStack(err)
		}

		events, err := s.Dao.ListRepositoryEvents(ctx, vcsServer, repo)
		if err != nil {
			return err
		}
		for _, e := range events {
			_ = s.Dao.RemoveRepositoryEventFromInProgressList(ctx, e.UUID)
			if err := s.Dao.DeleteRepositoryEvent(ctx, vcsServer, repo, e.UUID); err != nil {
				return err
			}
		}
		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (s *Service) listRepositoryEventHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		vcsServer := vars["vcsServer"]
		repo, err := url.PathUnescape(vars["repoName"])
		if err != nil {
			return sdk.WithStack(err)
		}

		events, err := s.Dao.ListRepositoryEvents(ctx, vcsServer, repo)
		if err != nil {
			return err
		}
		return service.WriteJSON(w, events, http.StatusOK)
	}

}

func (s *Service) deleteRepositoryHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		vcsServer := vars["vcsServer"]
		repo, err := url.PathUnescape(vars["repoName"])
		if err != nil {
			return err
		}

		if err := s.Dao.DeleteAllRepositoryEvent(ctx, vcsServer, repo); err != nil {
			return err
		}

		if err := s.Dao.DeleteRepository(ctx, vcsServer, repo); err != nil {
			return err
		}
		return nil
	}
}

func (s *Service) postRestartRepositoryHookEventHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		vcsServer := vars["vcsServer"]
		repo := vars["repoName"]
		uuid := vars["uuid"]

		e, err := s.Dao.GetRepositoryEvent(ctx, vcsServer, repo, uuid)
		if err != nil {
			return err
		}
		if !e.IsTerminated() {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "hook event is not in a final state")
		}

		e.Status = sdk.HookEventStatusScheduled
		e.DeprecatedUserID = ""
		e.DeprecatedUsername = ""
		e.SignKey = ""
		e.SigningKeyOperation = ""
		e.LastError = ""
		e.NbErrors = 0
		e.ModelUpdated = nil
		e.WorkflowUpdated = nil
		e.WorkflowHooks = nil
		e.Analyses = nil
		e.LastUpdate = time.Now().UnixNano()
		e.Initiator = nil

		if err := s.Dao.SaveRepositoryEvent(ctx, e); err != nil {
			return err
		}
		if err := s.Dao.EnqueueRepositoryEvent(ctx, e); err != nil {
			return sdk.WrapError(err, "unable to enqueue repository event %s", e.GetFullName())
		}

		return nil
	}

}
