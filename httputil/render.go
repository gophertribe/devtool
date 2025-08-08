package httputil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

func RenderJSON(w http.ResponseWriter, status int, body any) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func RenderError(w http.ResponseWriter, status int, msg string, cause error) {
	RenderJSON(w, status, HandlerError{Error: msg, Details: cause.Error()})
}

func DecodeJSON(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}
