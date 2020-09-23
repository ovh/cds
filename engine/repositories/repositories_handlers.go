package repositories

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func muxVar(r *http.Request, s string) string {
	vars := mux.Vars(r)
	return vars[s]
}

func (s *Service) postOperationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		op := new(sdk.Operation)
		files := make(map[string][]byte)
		ct := r.Header.Get("Content-Type")
		if strings.Contains(strings.ToLower(ct), "multipart") {
			var err error
			op, files, err = readOperationMultipart(r)
			if err != nil {
				return err
			}
		} else {
			if err := service.UnmarshalBody(r, op); err != nil {
				return err
			}
		}

		requestID := ctx.Value(log.ContextLoggingRequestIDKey)
		log.Info(ctx, "setting request_id:%s on operation:%s", requestID, op.UUID)
		op.RequestID, _ = requestID.(string)

		uuid := sdk.UUID()
		op.UUID = uuid
		now := time.Now()
		op.Date = &now
		if len(files) != 0 {
			op.LoadFiles = sdk.OperationLoadFiles{
				Results: files,
			}
		}

		op.Status = sdk.OperationStatusPending
		if err := s.dao.saveOperation(op); err != nil {
			return err
		}

		if err := s.dao.pushOperation(op); err != nil {
			return err
		}

		return service.WriteJSON(w, op, http.StatusAccepted)
	}
}

func readOperationMultipart(r *http.Request) (*sdk.Operation, map[string][]byte, error) {
	op := new(sdk.Operation)
	files := make(map[string][]byte)

	//parse the multipart form in the request
	if err := r.ParseMultipartForm(100000); err != nil {
		return nil, nil, sdk.WithStack(err)
	}
	for f := range r.MultipartForm.File {
		file, header, err := r.FormFile(f)
		if err != nil {
			file.Close()
			return nil, nil, sdk.WithStack(err)
		}
		fileContentType := header.Header.Get("Content-Type")
		switch fileContentType {
		case "application/tar":
			tr := tar.NewReader(file)
			// Iterate through the files in the archive.
			for {
				hdr, err := tr.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					return nil, nil, sdk.WrapError(err, "error while reading the tar archive")
				}

				fileBuf := new(bytes.Buffer)
				if _, err := fileBuf.ReadFrom(tr); err != nil {
					return nil, nil, sdk.WrapError(err, "error while reading buffer from tar archive")
				}
				files[hdr.Name] = fileBuf.Bytes()
			}
		}
		file.Close()
	}

	if err := json.Unmarshal([]byte(r.FormValue("dataJSON")), &op); err != nil {
		return nil, nil, sdk.WithStack(err)
	}
	return op, files, nil
}

func (s *Service) getOperationsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		uuid := muxVar(r, "uuid")
		op := s.dao.loadOperation(ctx, uuid)
		op.RepositoryStrategy.SSHKeyContent = sdk.PasswordPlaceholder
		op.RepositoryStrategy.Password = sdk.PasswordPlaceholder

		// Handle old representation of operation error
		if op.DeprecatedError != "" && op.Error == nil {
			op.Error = &sdk.OperationError{
				ID:      sdk.ErrUnknownError.ID,
				Message: op.DeprecatedError,
				Status:  sdk.ErrUnknownError.Status,
			}
		}

		return service.WriteJSON(w, op, http.StatusOK)
	}
}

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (s *Service) Status(ctx context.Context) *sdk.MonitoringStatus {
	m := s.NewMonitoringStatus()
	return m
}

func (s *Service) getStatusHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var status = http.StatusOK
		return service.WriteJSON(w, s.Status(ctx), status)
	}
}
