package packaging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreatePackageStructureRendersProjectTemplatesAndUsesDebianPaths(t *testing.T) {
	projectRoot := t.TempDir()
	stageDir := t.TempDir()

	writeTestFile(t, filepath.Join(projectRoot, "bin", "myapp"), []byte("binary"), 0o755)
	writeTestFile(t, filepath.Join(projectRoot, "service", "myapp.service"), []byte("[Unit]\nDescription=My App\n"), 0o644)
	writeTestFile(t, filepath.Join(projectRoot, "config", "myapp.yaml"), []byte("key: value\n"), 0o644)
	writeTestFile(t, filepath.Join(projectRoot, "debian", "control.tpl"), []byte(strings.Join([]string{
		"Package: {{.Package}}",
		"Version: {{.Version}}",
		"Architecture: {{.Architecture}}",
		"Maintainer: {{.Maintainer}}",
		"Description: {{.Description}}",
		"Binary-Count: {{len .Binaries}}",
		"First-Binary: {{(index .Binaries 0).Destination}}",
		"Config-Count: {{len .Configs}}",
	}, "\n")), 0o644)
	writeTestFile(t, filepath.Join(projectRoot, "debian", "postinst.tpl"), []byte("#!/bin/sh\necho {{.Package}} {{.Version}}\n"), 0o644)
	writeTestFile(t, filepath.Join(projectRoot, "debian", "conffiles"), []byte("/var/lib/myapp/state\n\n/etc/myapp/myapp.yaml\n"), 0o644)

	opts, err := normalizeDebianOptions(DebianOptions{
		Package:      "myapp",
		Version:      "1.2.3",
		Revision:     "2",
		Architecture: "amd64",
		ProjectRoot:  projectRoot,
		Binaries:     []string{"bin/myapp"},
		Systemd:      []string{"service/myapp.service"},
		Configs: map[string]string{
			"config/myapp.yaml": "/etc/myapp/myapp.yaml",
		},
		Control: ControlFields{
			Maintainer:  "Jane Example <jane@example.com>",
			Description: "Short summary\nLonger details",
		},
	})
	if err != nil {
		t.Fatalf("normalizeDebianOptions() error = %v", err)
	}

	if err := createPackageStructure(stageDir, opts); err != nil {
		t.Fatalf("createPackageStructure() error = %v", err)
	}

	assertFileExists(t, filepath.Join(stageDir, "usr", "bin", "myapp"))
	assertFileExists(t, filepath.Join(stageDir, "lib", "systemd", "system", "myapp.service"))
	assertFileExists(t, filepath.Join(stageDir, "etc", "myapp", "myapp.yaml"))

	control := readTestFile(t, filepath.Join(stageDir, "DEBIAN", "control"))
	if !strings.Contains(control, "Version: 1.2.3-2") {
		t.Fatalf("control file missing full version:\n%s", control)
	}
	if !strings.Contains(control, "Binary-Count: 1") {
		t.Fatalf("control file missing binary count:\n%s", control)
	}
	if !strings.Contains(control, "First-Binary: /usr/bin/myapp") {
		t.Fatalf("control file missing binary destination:\n%s", control)
	}
	if !strings.Contains(control, "Description: Short summary\n Longer details") {
		t.Fatalf("control file missing formatted multiline description:\n%s", control)
	}

	postinstPath := filepath.Join(stageDir, "DEBIAN", "postinst")
	postinstInfo, err := os.Stat(postinstPath)
	if err != nil {
		t.Fatalf("stat postinst: %v", err)
	}
	if postinstInfo.Mode().Perm() != 0o755 {
		t.Fatalf("postinst permissions = %o, want 755", postinstInfo.Mode().Perm())
	}
	postinst := readTestFile(t, postinstPath)
	if !strings.Contains(postinst, "echo myapp 1.2.3-2") {
		t.Fatalf("postinst template not rendered:\n%s", postinst)
	}

	conffiles := readTestFile(t, filepath.Join(stageDir, "DEBIAN", "conffiles"))
	if got, want := conffiles, "/etc/myapp/myapp.yaml\n/var/lib/myapp/state\n"; got != want {
		t.Fatalf("conffiles = %q, want %q", got, want)
	}
}

func TestCreatePackageStructureFallsBackToEmbeddedControlTemplate(t *testing.T) {
	projectRoot := t.TempDir()
	stageDir := t.TempDir()
	t.Setenv("DEBFULLNAME", "Packager")
	t.Setenv("DEBEMAIL", "packager@example.com")

	writeTestFile(t, filepath.Join(projectRoot, "bin", "tool"), []byte("binary"), 0o755)

	opts, err := normalizeDebianOptions(DebianOptions{
		Package:      "tool",
		Version:      "0.9.0",
		Architecture: "arm64",
		ProjectRoot:  projectRoot,
		Binaries:     []string{"bin/tool"},
	})
	if err != nil {
		t.Fatalf("normalizeDebianOptions() error = %v", err)
	}

	if err := createPackageStructure(stageDir, opts); err != nil {
		t.Fatalf("createPackageStructure() error = %v", err)
	}

	control := readTestFile(t, filepath.Join(stageDir, "DEBIAN", "control"))
	if !strings.Contains(control, "Package: tool") {
		t.Fatalf("control file missing package:\n%s", control)
	}
	if !strings.Contains(control, "Version: 0.9.0-1") {
		t.Fatalf("control file missing default revision:\n%s", control)
	}
	if !strings.Contains(control, "Maintainer: Packager <packager@example.com>") {
		t.Fatalf("control file missing maintainer:\n%s", control)
	}
	if !strings.Contains(control, "Section: misc") || !strings.Contains(control, "Priority: optional") {
		t.Fatalf("control file missing default metadata:\n%s", control)
	}
	if strings.Contains(control, "Depends:") {
		t.Fatalf("control file should omit empty Depends field:\n%s", control)
	}
	if !strings.Contains(control, "Description: tool") {
		t.Fatalf("control file missing default description:\n%s", control)
	}
}

func writeTestFile(t *testing.T, path string, content []byte, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", path, err)
	}
	if err := os.WriteFile(path, content, mode); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	return string(content)
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file %s to exist: %v", path, err)
	}
}
