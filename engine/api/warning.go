package api

import (
	"bytes"
	"context"
	"net/http"
	"text/template"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/warning"
	"github.com/ovh/cds/sdk"
)

func (api *API) getWarningsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		al := r.Header.Get("Accept-Language")

		warnings, errW := warning.GetByProject(api.mustDB(), key)
		if errW != nil {
			return sdk.WrapError(errW, "getWarningsHandler> Unable to get warning for project %s", key)
		}

		for i := range warnings {
			w := &warnings[i]
			processWarning(w, al)
		}
		return WriteJSON(w, warnings, http.StatusOK)
	}
}

func processWarning(w *sdk.WarningV2, acceptedlanguage string) error {
	var buffer bytes.Buffer

	var tmplBody string
	switch acceptedlanguage {
	case "fr":
		tmplBody = warning.MessageFrench[w.Type]
	default:
		tmplBody = warning.MessageAmericanEnglish[w.Type]
	}

	// Execute template
	t := template.Must(template.New("warning").Parse(tmplBody))
	if err := t.Execute(&buffer, w.MessageParams); err != nil {
		return err
	}

	// Set message value
	w.Message = buffer.String()
	return nil
}
