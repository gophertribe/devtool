package console

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/gophertribe/devtool/console"
	"github.com/gophertribe/devtool/frontend"
	"github.com/gophertribe/devtool/httputil"
	"github.com/lmittmann/tint"
	"github.com/spf13/afero"

	"github.com/evanw/esbuild/pkg/api"
)

const (
	entrypoint = "web/console/src/index.js"
	staticDir  = "web/console/public"
)

func Build(mode string, publicPath string) error {
	tmpDir, err := os.MkdirTemp("", "hdis_console")
	if err != nil {
		return fmt.Errorf("could not create temporary dir: %w", err)
	}
	opts := api.BuildOptions{
		Color:       api.ColorAlways,
		EntryPoints: []string{entrypoint},
		EntryNames:  "[dir]/[name]-[hash]",
		Bundle:      true,
		Loader: map[string]api.Loader{
			".jsx":   api.LoaderJSX,
			".js":    api.LoaderJSX,
			".woff":  api.LoaderBinary,
			".woff2": api.LoaderBinary,
			".png":   api.LoaderFile,
			".jpg":   api.LoaderFile,
		},
		Engines: []api.Engine{
			{Name: api.EngineChrome, Version: "58"},
			{Name: api.EngineFirefox, Version: "57"},
			{Name: api.EngineSafari, Version: "11"},
		},
		Write:      true,
		Outdir:     tmpDir,
		PublicPath: publicPath,
	}
	switch mode {
	case "prod":
		opts.MinifyWhitespace = true
		opts.MinifyIdentifiers = true
		opts.MinifySyntax = true
	default:
		opts.Sourcemap = api.SourceMapLinked
	}
	res := api.Build(opts)
	if len(res.Errors) > 0 {
		for _, err := range res.Errors {
			if err.Location == nil {
				slog.Error(err.Text)
				continue
			}
			fmt.Printf("%s %s\n%s\n%s", console.Red(fmt.Sprintf("%s:%d/%d", err.Location.File, err.Location.Line, err.Location.Column)), err.Text, err.Location.LineText, err.Location.Suggestion)

		}
		return frontend.ErrBuildFailed
	}
	indexParams := struct {
		JS        []string
		CSS       []string
		PublicURL string
	}{
		PublicURL: publicPath,
	}
	for _, f := range res.OutputFiles {
		switch {
		case strings.HasSuffix(f.Path, ".js"):
			_, file := filepath.Split(f.Path)
			indexParams.JS = append(indexParams.JS, file)
		case strings.HasSuffix(f.Path, ".css"):
			_, file := filepath.Split(f.Path)
			indexParams.CSS = append(indexParams.CSS, file)
		}
	}
	err = frontend.CopyDir(staticDir, tmpDir)
	if err != nil {
		return fmt.Errorf("could not copy static files: %w", err)
	}

	err = frontend.BuildIndex(tmpDir+"/console.html", indexParams)
	if err != nil {
		return fmt.Errorf("could not build index file: %w", err)
	}
	err = os.RemoveAll(frontend.BuildDir)
	if err != nil {
		return fmt.Errorf("could not clear build dir: %w", err)
	}
	err = os.Mkdir(frontend.BuildDir, 0755)
	if err != nil {
		return fmt.Errorf("could not recreate build dir: %w", err)
	}
	err = frontend.CopyDir(tmpDir, frontend.BuildDir)
	if err != nil {
		return fmt.Errorf("could not copy build results to %s: %w", frontend.BuildDir, err)
	}
	// clear temporary folder
	_ = os.RemoveAll(tmpDir)
	return nil
}

func Proxy(ctx context.Context, mode, remote, publicPath string) error {

	slog.SetDefault(slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		AddSource: true,
		Level:     slog.LevelDebug,
	})))

	err := Build(mode, publicPath)
	if err != nil {
		return err
	}
	ip, err := frontend.GetOutboundIP()
	if err != nil {
		return fmt.Errorf("could not get outbound ip: %w", err)
	}

	frontend.Local(publicPath)

	mux := http.NewServeMux()

	// TODO: hooks for reverse proxy
	mux.Handle("GET /console/{subpath...}", http.StripPrefix("/console/", frontend.ServeHtml("console.html")))
	//mux.Handle("/api/", display.NewReverseProxy("/api", fmt.Sprintf("http://%s", remote)))

	tlsCertPath := filepath.Join(frontend.BuildDir, "cert.pem")
	tlsKeyPath := filepath.Join(frontend.BuildDir, "key.pem")
	tlsConfig, err := httputil.BuildTLSConfig("localhost", tlsCertPath, tlsKeyPath, afero.NewOsFs())
	if err != nil {
		return fmt.Errorf("could not create tls config: %w", err)
	}

	server := http.Server{
		Addr:           ":8087",
		Handler:        mux,
		ReadTimeout:    360 * time.Second,
		WriteTimeout:   360 * time.Second,
		MaxHeaderBytes: 1 << 20,
		TLSConfig:      tlsConfig,
	}

	go func() {
		slog.Info("proxy running", "ip", ip, "port", 8087)
		err = server.ListenAndServeTLS(tlsCertPath, tlsKeyPath)
		if err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return
			}
			slog.Error("http server error", "error", err)
		}
	}()
	out := make(chan os.Signal, 1)
	signal.Notify(out, os.Interrupt)
	<-out
	_ = server.Shutdown(ctx)
	return nil

}
