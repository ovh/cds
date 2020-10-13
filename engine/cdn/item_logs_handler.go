package cdn

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/redis"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/engine/websocket"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) markItemToDeleteHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if !s.Cfg.EnableLogProcessing {
			return nil
		}
		var req sdk.CDNMarkDelete
		if err := service.UnmarshalBody(r, &req); err != nil {
			return err
		}

		if req.WorkflowID > 0 && req.RunID > 0 {
			return sdk.WrapError(sdk.ErrWrongRequest, "invalid data")
		}
		tx, err := s.mustDBWithCtx(ctx).Begin()
		if err != nil {
			return sdk.WrapError(err, "unable to start transaction")
		}
		defer tx.Rollback() //nolint

		if req.WorkflowID > 0 {
			if err := item.MarkToDeleteByWorkflowID(tx, req.WorkflowID); err != nil {
				return err
			}
		} else {
			if err := item.MarkToDeleteByRunIDs(tx, req.RunID); err != nil {
				return err
			}
		}
		return sdk.WrapError(tx.Commit(), "unable to commit transaction")
	}
}

func (s *Service) getItemDownloadHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		itemType := sdk.CDNItemType(vars["type"])
		apiRef := vars["apiRef"]

		var opts downloadOpts
		// User can give a refresh delay in seconds, Refresh header value will be set if item is not complete
		opts.Log.Refresh = service.FormInt64(r, "refresh")
		opts.Log.Sort = service.FormInt64(r, "sort") // < 0 for latest logs first, >= 0 for older logs first

		return s.downloadItem(ctx, itemType, apiRef, w, opts)
	}
}

func (s *Service) getItemLogsStreamHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		itemType := sdk.CDNItemType(vars["type"])
		if !itemType.IsLog() {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid item log type")
		}
		apiRef := vars["apiRef"]

		offset := service.FormInt64(r, "offset")

		it, err := item.LoadByAPIRefHashAndType(ctx, s.Mapper, s.mustDBWithCtx(ctx), apiRef, itemType)
		if err != nil {
			return err
		}
		iu, err := storage.LoadItemUnitByUnit(ctx, s.Mapper, s.mustDBWithCtx(ctx), s.Units.Buffer.ID(), it.ID)
		if err != nil {
			return err
		}

		c, err := websocket.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			service.WriteError(ctx, w, r, sdk.NewErrorWithStack(err, sdk.ErrWebsocketUpgrade))
			return nil
		}
		defer c.Close()

		wsClient := websocket.NewClient(c)
		wsClientData := &websocketClientData{
			itemID:              it.ID,
			chanItemUpdate:      make(chan struct{}),
			scoreNextLineToSend: offset,
		}
		s.WSServer.AddClient(wsClient, wsClientData)
		defer s.WSServer.RemoveClient(wsClient.UUID())

		s.GoRoutines.Exec(s.Router.Background, "getItemLogsStreamHandler."+wsClient.UUID(), func(ctx context.Context) {
			log.Debug("getItemLogsStreamHandler> start routine for client %s", wsClient.UUID())

			send := func() error {
				log.Debug("getItemLogsStreamHandler> send log to client %s from %d", wsClient.UUID(), wsClientData.scoreNextLineToSend)

				rc, err := s.Units.Buffer.NewAdvancedReader(ctx, *iu, sdk.CDNReaderFormatJSON, wsClientData.scoreNextLineToSend, 100, 0)
				if err != nil {
					return err
				}
				defer rc.Close() // nolint
				buf := new(bytes.Buffer)
				if _, err := io.Copy(buf, rc); err != nil {
					return sdk.WrapError(err, "cannot copy data from reader to memory buffer")
				}
				var lines []redis.Line
				if err := json.Unmarshal(buf.Bytes(), &lines); err != nil {
					return sdk.WrapError(err, "cannot unmarshal lines from buffer")
				}

				log.Debug("getItemLogsStreamHandler> iterate over %d lines to send for client %s", len(lines), wsClient.UUID())
				oldNextLineToSend := wsClientData.scoreNextLineToSend
				for i := range lines {
					if wsClientData.scoreNextLineToSend != lines[i].Number {
						break
					}
					if err := wsClient.Send(lines[i]); err != nil {
						return err
					}
					wsClientData.scoreNextLineToSend++
				}

				// If all the lines were sent, we can trigger another update
				if len(lines) > 0 && wsClientData.scoreNextLineToSend-oldNextLineToSend == int64(len(lines)) {
					go func() { wsClientData.chanItemUpdate <- struct{}{} }()
				}

				return nil
			}

			// Trigger one update at routine startup
			go func() { wsClientData.chanItemUpdate <- struct{}{} }()

			for {
				select {
				case <-ctx.Done():
					log.Debug("getItemLogsStreamHandler> stop routine for stream client %s", wsClient.UUID())
					return
				case <-wsClientData.chanItemUpdate:
					if err := send(); err != nil {
						log.Debug("getItemLogsStreamHandler> can't send to client %s it will be removed: %+v", wsClient.UUID(), err)
						return
					}
				}
			}
		})

		return wsClient.Listen(s.Router.Background, s.GoRoutines)
	}
}

func (s *Service) getItemLogsLinesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		itemType := sdk.CDNItemType(vars["type"])
		if !itemType.IsLog() {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid item log type")
		}

		apiRef := vars["apiRef"]

		// offset can be lower than 0 if we want the n last lines
		offset := service.FormInt64(r, "offset")
		count := service.FormUInt(r, "count")
		sort := service.FormInt64(r, "sort") // < 0 for latest logs first, >= 0 for older logs first

		_, rc, _, err := s.getItemLogValue(ctx, itemType, apiRef, sdk.CDNReaderFormatJSON, offset, count, sort)
		if err != nil {
			return err
		}
		if rc == nil {
			return sdk.WrapError(sdk.ErrNotFound, "no storage found that contains given item %s", apiRef)
		}

		return service.Write(w, rc, http.StatusOK, "application/json")
	}
}

func (s *Service) getSizeByProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["projectKey"]

		// get size used by a project key
		size, err := item.ComputeSizeByProjectKey(s.mustDBWithCtx(ctx), projectKey)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, size, http.StatusOK)
	}
}
