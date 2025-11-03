package frontend

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/gophertribe/devtool/console"
	"github.com/gophertribe/devtool/httputil"
	"github.com/spf13/afero"
)

var ErrBuildFailed = errors.New("build failed")

type BuildOptions struct {
	// TLSCertPath is the path to the TLS certificate file (auto-generated if not provided)
	TLSCertPath string
	// TLSKeyPath is the path to the TLS key file (auto-generated if not provided)
	TLSKeyPath string
	// Mode is the build mode, either "prod" or "dev"
	Mode string
	// BuildDir is the target directory of the build
	BuildDir string
	// PublicPath is the public path to serve the frontend from
	PublicPath string
	// Entrypoint is the javascript entrypoint file for esbuild
	Entrypoint string
	// StaticDir is the directory to copy static files from
	StaticDir string
	// IndexFile is the name of the html index file that will be used as template to inject the javascript and css
	IndexFile string
}

func Build(opts BuildOptions) error {
	tmpDir, err := os.MkdirTemp("", "dev_frontend")
	if err != nil {
		return fmt.Errorf("could not create temporary dir: %w", err)
	}
	buildOpts := api.BuildOptions{
		Color:       api.ColorAlways,
		EntryPoints: []string{opts.Entrypoint},
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
		PublicPath: opts.PublicPath,
	}
	switch opts.Mode {
	case "prod":
		buildOpts.MinifyWhitespace = true
		buildOpts.MinifyIdentifiers = true
		buildOpts.MinifySyntax = true
	default:
		buildOpts.Sourcemap = api.SourceMapLinked
	}
	res := api.Build(buildOpts)
	if len(res.Errors) > 0 {
		for _, err := range res.Errors {
			if err.Location == nil {
				slog.Error(err.Text)
				continue
			}
			fmt.Printf("%s %s\n%s\n%s", console.Red(fmt.Sprintf("%s:%d/%d", err.Location.File, err.Location.Line, err.Location.Column)), err.Text, err.Location.LineText, err.Location.Suggestion)

		}
		return ErrBuildFailed
	}
	indexParams := struct {
		JS        []string
		CSS       []string
		PublicURL string
	}{
		PublicURL: opts.PublicPath,
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
	err = CopyDir(opts.StaticDir, tmpDir)
	if err != nil {
		return fmt.Errorf("could not copy static files: %w", err)
	}

	err = BuildIndex(tmpDir+"/console.html", indexParams)
	if err != nil {
		return fmt.Errorf("could not build index file: %w", err)
	}
	err = os.RemoveAll(opts.BuildDir)
	if err != nil {
		return fmt.Errorf("could not clear build dir: %w", err)
	}
	err = os.Mkdir(opts.BuildDir, 0755)
	if err != nil {
		return fmt.Errorf("could not recreate build dir: %w", err)
	}
	err = CopyDir(tmpDir, opts.BuildDir)
	if err != nil {
		return fmt.Errorf("could not copy build results to %s: %w", opts.BuildDir, err)
	}
	// clear temporary folder
	_ = os.RemoveAll(tmpDir)
	return nil
}

func Proxy(ctx context.Context, opts BuildOptions, enableTLS bool, port int) error {
	err := Build(opts)
	if err != nil {
		return err
	}
	ip, err := GetOutboundIP()
	if err != nil {
		return fmt.Errorf("could not get outbound ip: %w", err)
	}

	Local(opts.PublicPath)

	mux := http.NewServeMux()

	// TODO: hooks for reverse proxy
	mux.Handle("GET /console/{subpath...}", http.StripPrefix("/console/", ServeHtml(opts.IndexFile)))
	//mux.Handle("/api/", display.NewReverseProxy("/api", fmt.Sprintf("http://%s", remote)))

	server := http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		Handler:        mux,
		ReadTimeout:    360 * time.Second,
		WriteTimeout:   360 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	tlsCertPath := filepath.Join(opts.BuildDir, "cert.pem")
	tlsKeyPath := filepath.Join(opts.BuildDir, "key.pem")
	if enableTLS {
		// BuildTLSConfig will auto-generate certificates if they don't exist
		server.TLSConfig, err = httputil.BuildTLSConfig("localhost", "", "", afero.NewOsFs())
		if err != nil {
			return fmt.Errorf("could not create tls config: %w", err)
		}
	}

	go func() {
		slog.Info("proxy running", "ip", ip, "port", port)
		if enableTLS {
			err = server.ListenAndServeTLS(tlsCertPath, tlsKeyPath)
		} else {
			err = server.ListenAndServe()
		}
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

func BuildIndex(indexFile string, indexParams any) error {
	tpl, err := template.ParseFiles(indexFile)
	if err != nil {
		return fmt.Errorf("could not parse index file template: %w", err)
	}
	err = os.Remove(indexFile)
	if err != nil {
		return fmt.Errorf("could not remove template file: %w", err)
	}
	index, err := os.OpenFile(indexFile, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("could not open target index file: %w", err)
	}
	err = tpl.Execute(index, indexParams)
	if err != nil {
		return fmt.Errorf("could not write index file: %w", err)
	}
	return nil
}

func CopyDir(source, destination string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath := strings.Replace(path, source, "", 1)
		if relPath == "" {
			return nil
		}
		if info.IsDir() {
			return os.Mkdir(filepath.Join(destination, relPath), 0755)
		}
		src := filepath.Join(source, relPath)
		dst := filepath.Join(destination, relPath)
		input, err := os.OpenFile(src, os.O_RDONLY, 0755)
		if err != nil {
			return fmt.Errorf("could not open source file %s: %w", src, err)
		}
		output, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			return fmt.Errorf("could not open target file %s: %w", dst, err)
		}
		_, err = io.Copy(output, input)
		_ = input.Close()
		_ = output.Close()
		if err != nil {
			return fmt.Errorf("could not copy file %s: %w", src, err)
		}
		return nil
	})
}

func GetOutboundIP() (net.IP, error) {
	conn, err := net.Dial("udp", "1.1.1.1:80")
	if err != nil {
		return nil, fmt.Errorf("could not establish dns connection: %w", err)
	}
	defer func() { _ = conn.Close() }()
	return conn.LocalAddr().(*net.UDPAddr).IP, nil
}
