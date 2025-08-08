package frontend

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
)

var ErrBuildFailed = errors.New("build failed")

const BuildDir = "web/build"

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
