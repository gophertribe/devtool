package frontend

import (
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"regexp"

	"github.com/gophertribe/devtool/httputil"
	"github.com/spf13/afero"
)

var files fs.FS

func Local(path string) {
	files = afero.NewIOFS(afero.NewBasePathFs(afero.NewOsFs(), path))
}

var css = regexp.MustCompile(`\.css$`)

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
				httputil.RenderJSON(w, http.StatusInternalServerError, httputil.HandlerError{Error: err.Error()})
				return
			}
			file, err = fs.Open(indexPath)
			if err != nil {
				httputil.RenderJSON(w, http.StatusInternalServerError, httputil.HandlerError{Error: err.Error()})
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
