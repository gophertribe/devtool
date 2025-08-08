package frontend

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"regexp"

	"github.com/spf13/afero"
)

const ConsolePublicPath = "/console"

var files fs.FS

func Local(path string) {
	files = afero.NewIOFS(afero.NewBasePathFs(afero.NewOsFs(), path))
}

var css = regexp.MustCompile("\\.css$")

func ServeHtml(indexPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slog.Debug("serving html", "path", r.URL.Path, "match", r.PathValue("subpath"))
		fs := http.FS(files)
		if r.URL.Path == "" || r.URL.Path == "/" {
			r.URL.Path = indexPath
		}
		file, err := fs.Open(r.URL.Path)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				jsonResponse(w, http.StatusInternalServerError, Error{Err: err.Error()})
				return
			}
			file, err = fs.Open(indexPath)
			if err != nil {
				jsonResponse(w, http.StatusInternalServerError, Error{Err: err.Error()})
				return
			}
		}
		defer func() { _ = file.Close() }()
		switch {
		case css.MatchString(r.URL.Path):
			w.Header().Set("Content-Type", "text/css")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, file)
	}
}

type Error struct {
	Err     string `json:"error"`
	Details string `json:"details,omitempty"`
}

func jsonResponse(w http.ResponseWriter, code int, resp interface{}) {
	w.Header().Set("Content-Type", "application/json")
	data, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(code)
	_, _ = w.Write(data)
}
