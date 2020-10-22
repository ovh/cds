package cdn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/redis"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/engine/websocket"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) getItemLogsStreamHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		c, err := websocket.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			service.WriteError(ctx, w, r, sdk.NewErrorWithStack(err, sdk.ErrWebsocketUpgrade))
			return nil
		}
		defer c.Close()

		jwt := ctx.Value(service.ContextJWT).(*jwt.Token)
		claims := jwt.Claims.(*sdk.AuthSessionJWTClaims)
		sessionID := claims.StandardClaims.Id

		wsClient := websocket.NewClient(c)
		wsClientData := &websocketClientData{sessionID: sessionID}
		s.WSServer.AddClient(wsClient, wsClientData)
		defer s.WSServer.RemoveClient(wsClient.UUID())

		wsClient.OnMessage(func(m []byte) {
			if err := wsClientData.UpdateFilter(m); err != nil {
				log.WarningWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
				return
			}
			// Trigger one update at routine startup
			wsClientData.TriggerUpdate()
		})

		ctx, cancel := context.WithCancel(s.Router.Background)
		ctx = context.WithValue(ctx, service.ContextSessionID, sessionID)
		defer cancel()

		s.GoRoutines.Exec(ctx, "getItemLogsStreamHandler."+wsClient.UUID(), func(ctx context.Context) {
			log.Debug("getItemLogsStreamHandler> start routine for client %s (session %s)", wsClient.UUID(), s.sessionID(ctx))

			// Create a ticker to periodically send logs if needed
			sendTicker := time.NewTicker(time.Millisecond * 100)
			defer sendTicker.Stop()
			for {
				select {
				case <-ctx.Done():
					log.Debug("getItemLogsStreamHandler> stop routine for stream client %s", wsClient.UUID())
					return
				case <-sendTicker.C:
					if !wsClientData.ConsumeTrigger() {
						continue
					}
					if err := s.sendLogsToWSClient(ctx, wsClient, wsClientData); err != nil {
						log.Warning(ctx, "getItemLogsStreamHandler> can't send to client %s it will be removed: %+v", wsClient.UUID(), err)
						return
					}
				}
			}
		})

		if err := wsClient.Listen(ctx, s.GoRoutines); err != nil {
			return err
		}

		log.Debug("getItemLogsStreamHandler> stop listenning for client %s", wsClient.UUID())
		return nil
	}
}

func (s *Service) sendLogsToWSClient(ctx context.Context, wsClient websocket.Client, wsClientData *websocketClientData) error {
	wsClientData.mutexData.Lock()
	defer wsClientData.mutexData.Unlock()

	if wsClientData.itemFilter == nil {
		return nil
	}

	if wsClientData.itemUnit == nil {
		it, err := item.LoadByAPIRefHashAndType(ctx, s.Mapper, s.mustDBWithCtx(ctx), wsClientData.itemFilter.APIRef, wsClientData.itemFilter.ItemType)
		if err != nil {
			// Catch not found error as the item can be created after the client stream subscription
			if sdk.ErrorIs(err, sdk.ErrNotFound) {
				log.Debug("sendLogsToWSClient> can't found item with type %s and ref %s for client %s: %+v", wsClientData.itemFilter.ItemType, wsClientData.itemFilter.APIRef, wsClient.UUID(), err)
				return nil
			}
			return nil
		}

		if err := s.itemAccessCheck(ctx, *it); err != nil {
			return sdk.WrapError(err, "client %s can't access logs for workflow %s/%s", wsClient.UUID(), it.APIRef.ProjectKey, it.APIRef.WorkflowName)
		}

		iu, err := storage.LoadItemUnitByUnit(ctx, s.Mapper, s.mustDBWithCtx(ctx), s.Units.Buffer.ID(), it.ID)
		if err != nil {
			return err
		}

		wsClientData.itemUnit = iu
	}

	log.Debug("getItemLogsStreamHandler> send log to client %s from %d", wsClient.UUID(), wsClientData.scoreNextLineToSend)

	rc, err := s.Units.Buffer.NewAdvancedReader(ctx, *wsClientData.itemUnit, sdk.CDNReaderFormatJSON, wsClientData.scoreNextLineToSend, 100, 0)
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
		return sdk.WrapError(err, "cannot unmarshal lines from buffer %v", string(buf.Bytes()))
	}

	log.Debug("getItemLogsStreamHandler> iterate over %d lines to send for client %s", len(lines), wsClient.UUID())
	oldNextLineToSend := wsClientData.scoreNextLineToSend
	for i := range lines {
		if wsClientData.scoreNextLineToSend > 0 && wsClientData.scoreNextLineToSend != lines[i].Number {
			break
		}
		if err := wsClient.Send(lines[i]); err != nil {
			return err
		}
		if wsClientData.scoreNextLineToSend < 0 {
			wsClientData.scoreNextLineToSend = lines[i].Number + 1
		} else {
			wsClientData.scoreNextLineToSend++
		}
	}

	// If all the lines were sent, we can trigger another update, if only one line was send do not trigger an update wait for next event from broker
	if len(lines) > 1 && (oldNextLineToSend > 0 || wsClientData.scoreNextLineToSend-oldNextLineToSend == int64(len(lines))) {
		wsClientData.TriggerUpdate()
	}

	return nil
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
		limit := service.FormUInt(r, "limit")
		sort := service.FormInt64(r, "sort") // < 0 for latest logs first, >= 0 for older logs first

		_, linesCount, rc, _, err := s.getItemLogValue(ctx, itemType, apiRef, sdk.CDNReaderFormatJSON, offset, limit, sort)
		if err != nil {
			return err
		}
		if rc == nil {
			return sdk.WrapError(sdk.ErrNotFound, "no storage found that contains given item %s", apiRef)
		}

		w.Header().Add("X-Total-Count", fmt.Sprintf("%d", linesCount))

		return service.Write(w, rc, http.StatusOK, "application/json")
	}
}
