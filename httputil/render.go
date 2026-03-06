package httputil

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
)

func RenderJSON(w http.ResponseWriter, status int, body any) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, err := w.Write(buf.Bytes())
	if err != nil {
		slog.Error("failed to write response", "error", err)
	}
}

func RenderError(w http.ResponseWriter, status int, msg string, cause error) {
	RenderJSON(w, status, HandlerError{Error: msg, Details: cause.Error()})
}

func DecodeJSON(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}
