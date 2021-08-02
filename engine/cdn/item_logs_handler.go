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
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/redis"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/engine/websocket"
	"github.com/ovh/cds/sdk"
)

func (s *Service) getItemLogsStreamHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		c, err := websocket.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			service.WriteError(ctx, w, r, sdk.NewErrorWithStack(err, sdk.ErrWebsocketUpgrade))
			return nil
		}
		defer c.Close() //nolint

		jwtToken := ctx.Value(service.ContextJWT).(*jwt.Token)
		claims := jwtToken.Claims.(*sdk.AuthSessionJWTClaims)
		sessionID := claims.StandardClaims.Id

		wsClient := websocket.NewClient(c)
		wsClientData := &websocketClientData{sessionID: sessionID}
		s.WSServer.AddClient(wsClient, wsClientData)
		defer s.WSServer.RemoveClient(wsClient.UUID())

		wsClient.OnMessage(func(m []byte) {
			if err := wsClientData.UpdateFilter(m); err != nil {
				ctx = sdk.ContextWithStacktrace(ctx, err)
				log.Warn(ctx, err.Error())
				return
			}
			// Trigger one update at routine startup
			wsClientData.TriggerUpdate()
		})

		ctx, cancel := context.WithCancel(s.Router.Background)
		ctx = context.WithValue(ctx, service.ContextSessionID, sessionID)
		defer cancel()

		s.GoRoutines.Exec(ctx, "getItemLogsStreamHandler."+wsClient.UUID(), func(ctx context.Context) {
			log.Debug(ctx, "getItemLogsStreamHandler> start routine for client %s (session %s)", wsClient.UUID(), s.sessionID(ctx))

			// Create a ticker to periodically send logs if needed
			sendTicker := time.NewTicker(time.Millisecond * 100)
			defer sendTicker.Stop()
			for {
				select {
				case <-ctx.Done():
					log.Debug(ctx, "getItemLogsStreamHandler> stop routine for stream client %s", wsClient.UUID())
					return
				case <-sendTicker.C:
					if !wsClientData.ConsumeTrigger() {
						continue
					}
					if err := s.sendLogsToWSClient(ctx, wsClient, wsClientData); err != nil {
						log.Warn(ctx, "getItemLogsStreamHandler> can't send to client %s it will be removed: %+v", wsClient.UUID(), err)
						return
					}
				}
			}
		})

		if err := wsClient.Listen(ctx, s.GoRoutines); err != nil {
			return err
		}

		log.Debug(ctx, "getItemLogsStreamHandler> stop listenning for client %s", wsClient.UUID())
		return nil
	}
}

func (s *Service) sendLogsToWSClient(ctx context.Context, wsClient websocket.Client, wsClientData *websocketClientData) error {
	wsClientData.mutexData.Lock()
	defer wsClientData.mutexData.Unlock()

	if wsClientData.itemFilter == nil {
		return nil
	}

	if wsClientData.itemUnitsData == nil {
		wsClientData.itemUnitsData = make(map[string]ItemUnitClientData)
	}

	its, err := item.LoadByJobRunID(ctx, s.Mapper, s.mustDBWithCtx(ctx), wsClientData.itemFilter.JobRunID, []string{string(sdk.CDNTypeItemStepLog), string(sdk.CDNTypeItemServiceLog)})
	if err != nil {
		// Catch not found error as the item can be created after the client stream subscription
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			log.Debug(ctx, "sendLogsToWSClient> can't found item job run %d for client %s: %+v", wsClientData.itemFilter.JobRunID, wsClient.UUID(), err)
			return nil
		}
		return nil
	}

	for _, it := range its {
		if _, has := wsClientData.itemUnitsData[it.ID]; !has {
			if err := s.itemAccessCheck(ctx, it); err != nil {
				var projectKey, workflow string
				logRef, has := it.GetCDNLogApiRef()
				if has {
					projectKey = logRef.ProjectKey
					workflow = logRef.WorkflowName
				}
				return sdk.WrapError(err, "client %s can't access logs for workflow %s/%s", wsClient.UUID(), projectKey, workflow)
			}
			iu, err := storage.LoadItemUnitByUnit(ctx, s.Mapper, s.mustDBWithCtx(ctx), s.Units.LogsBuffer().ID(), it.ID)
			if err != nil {
				return err
			}
			wsClientData.itemUnitsData[it.ID] = ItemUnitClientData{
				itemUnit:            iu,
				scoreNextLineToSend: 0,
			}
		}

	}

	for k, v := range wsClientData.itemUnitsData {
		// avoid to send all messages for ended steps
		if v.scoreNextLineToSend == 0 && v.itemUnit.Item.Status == sdk.CDNStatusItemCompleted {
			continue
		}
		log.Debug(ctx, "getItemLogsStreamHandler> send log to client %s from %d", wsClient.UUID(), v.scoreNextLineToSend)

		rc, err := s.Units.LogsBuffer().NewAdvancedReader(ctx, *v.itemUnit, sdk.CDNReaderFormatJSON, v.scoreNextLineToSend, 100, 0)
		if err != nil {
			return err
		}
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, rc); err != nil {
			_ = rc.Close()
			return sdk.WrapError(err, "cannot copy data from reader to memory buffer")
		}
		var lines []redis.Line
		if err := json.Unmarshal(buf.Bytes(), &lines); err != nil {
			_ = rc.Close()
			return sdk.WrapError(err, "cannot unmarshal lines from buffer %v", string(buf.Bytes()))
		}

		log.Debug(ctx, "getItemLogsStreamHandler> iterate over %d lines to send for client %s", len(lines), wsClient.UUID())
		oldNextLineToSend := v.scoreNextLineToSend
		for i := range lines {
			if v.scoreNextLineToSend > 0 && v.scoreNextLineToSend != lines[i].Number {
				break
			}
			if err := wsClient.Send(lines[i]); err != nil {
				return err
			}
			if v.scoreNextLineToSend < 0 {
				v.scoreNextLineToSend = lines[i].Number + 1
				wsClientData.itemUnitsData[k] = v
			} else {
				v.scoreNextLineToSend++
				wsClientData.itemUnitsData[k] = v
			}
		}

		// If all the lines were sent, we can trigger another update, if only one line was send do not trigger an update wait for next event from broker
		if len(lines) > 1 && (oldNextLineToSend > 0 || v.scoreNextLineToSend-oldNextLineToSend == int64(len(lines))) {
			wsClientData.TriggerUpdate()
		}
	}

	return nil
}

func (s *Service) getItemsAllLogsLinesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		itemType := sdk.CDNItemType(vars["type"])
		if !itemType.IsLog() {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid item log type")
		}
		refsHash := r.URL.Query()["apiRefHash"]

		resp := make([]sdk.CDNLogsLines, 0)

		for _, hash := range refsHash {
			linesCount, err := s.getItemLogLinesCount(ctx, itemType, hash)
			if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return err
			}
			if err != nil && sdk.ErrorIs(err, sdk.ErrNotFound) {
				break
			}
			resp = append(resp, sdk.CDNLogsLines{APIRef: hash, LinesCount: linesCount})
		}
		return service.WriteJSON(w, resp, http.StatusOK)
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

		opts := getItemLogOptions{
			format:      sdk.CDNReaderFormatJSON,
			from:        service.FormInt64(r, "offset"), // offset can be lower than 0 if we want the n last lines
			size:        service.FormUInt(r, "limit"),
			sort:        service.FormInt64(r, "sort"), // < 0 for latest logs first, >= 0 for older logs first
			cacheClean:  service.FormBool(r, "cacheClean"),
			cacheSource: r.FormValue("cacheSource"),
		}

		// Only admin can use the parameter 'cacheRefresh*'
		if opts.cacheClean || opts.cacheSource != "" {
			sessionID := s.sessionID(ctx)
			data, err := s.Client.AuthSessionGet(sessionID)
			if err != nil {
				return err
			}
			if data.Consumer.AuthentifiedUser.Ring != sdk.UserRingAdmin {
				return sdk.WithStack(sdk.ErrUnauthorized)
			}
		}

		_, linesCount, rc, _, err := s.getItemLogValue(ctx, itemType, apiRef, opts)
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
