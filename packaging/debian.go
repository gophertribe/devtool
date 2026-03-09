package packaging

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

//go:embed templates/control.tpl
var defaultControlTemplate string

// ControlFields holds the standard Debian control metadata used by the default template.
type ControlFields struct {
	Depends     []string
	Section     string
	Priority    string
	Maintainer  string
	Description string
	Homepage    string
}

// InstallFile describes a staged file and its final package destination.
type InstallFile struct {
	Source      string
	Destination string
	Name        string
}

// ControlData is the default template context exposed to debian/*.tpl files.
type ControlData struct {
	Package         string
	Version         string
	UpstreamVersion string
	DebianRevision  string
	Architecture    string
	Depends         string
	DependsList     []string
	Section         string
	Priority        string
	Maintainer      string
	Description     string
	Homepage        string
	Binaries        []InstallFile
	SystemdUnits    []InstallFile
	Configs         []InstallFile
}

// DebianOptions configures Debian package creation.
type DebianOptions struct {
	Package      string
	Version      string
	Revision     string
	Architecture string

	Binaries    []string
	Systemd     []string
	Configs     map[string]string
	OutputDir   string
	ProjectRoot string

	Control      ControlFields
	TemplateData map[string]any

	SkipPackageInfo bool
}

type normalizedDebianOptions struct {
	packageName     string
	fullVersion     string
	architecture    string
	projectRoot     string
	outputDir       string
	binaries        []InstallFile
	systemdUnits    []InstallFile
	configs         []InstallFile
	templateData    map[string]any
	skipPackageInfo bool
}

// Debian keeps the original positional API for compatibility.
func Debian(packageName, version, arch string, binaries, systemd []string, configs map[string]string, outputDir string, debianRevision string) error {
	return BuildDebian(DebianOptions{
		Package:      packageName,
		Version:      version,
		Revision:     debianRevision,
		Architecture: arch,
		Binaries:     binaries,
		Systemd:      systemd,
		Configs:      configs,
		OutputDir:    outputDir,
	})
}

// BuildDebian creates a Debian package using files from the project root and debian templates.
func BuildDebian(opts DebianOptions) error {
	normalized, err := normalizeDebianOptions(opts)
	if err != nil {
		return err
	}

	if _, err := exec.LookPath("dpkg-deb"); err != nil {
		return fmt.Errorf("dpkg-deb not found: %w", err)
	}

	if err := os.MkdirAll(normalized.outputDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	buildRoot := filepath.Join(normalized.projectRoot, "build", "deb")
	if err := os.MkdirAll(buildRoot, 0o755); err != nil {
		return fmt.Errorf("create build root: %w", err)
	}

	stageDir, err := os.MkdirTemp(buildRoot, normalized.packageName+"-")
	if err != nil {
		return fmt.Errorf("create staging directory: %w", err)
	}
	defer os.RemoveAll(stageDir)

	slog.Info("building Debian package", "package", normalized.packageName, "version", normalized.fullVersion, "arch", normalized.architecture)

	if err := createPackageStructure(stageDir, normalized); err != nil {
		return fmt.Errorf("create package structure: %w", err)
	}

	packageName := fmt.Sprintf("%s_%s_%s.deb", normalized.packageName, normalized.fullVersion, normalized.architecture)
	packagePath := filepath.Join(normalized.outputDir, packageName)
	if err := buildPackage(stageDir, packagePath); err != nil {
		return fmt.Errorf("build package: %w", err)
	}

	slog.Info("package built successfully", "path", packagePath)

	if !normalized.skipPackageInfo {
		if err := showPackageInfo(packagePath); err != nil {
			slog.Warn("failed to show package info", "error", err)
		}
	}

	return nil
}

func normalizeDebianOptions(opts DebianOptions) (normalizedDebianOptions, error) {
	if strings.TrimSpace(opts.Package) == "" {
		return normalizedDebianOptions{}, errors.New("package name is required")
	}
	if strings.TrimSpace(opts.Version) == "" {
		return normalizedDebianOptions{}, errors.New("version is required")
	}
	if strings.TrimSpace(opts.Architecture) == "" {
		return normalizedDebianOptions{}, errors.New("architecture is required")
	}

	arch := strings.TrimSpace(opts.Architecture)
	if !isValidDebianArchitecture(arch) {
		return normalizedDebianOptions{}, fmt.Errorf("invalid Debian architecture: %s", arch)
	}

	projectRoot := opts.ProjectRoot
	if projectRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return normalizedDebianOptions{}, fmt.Errorf("get current directory: %w", err)
		}
		projectRoot = cwd
	}
	projectRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return normalizedDebianOptions{}, fmt.Errorf("resolve project root: %w", err)
	}

	revision := strings.TrimSpace(opts.Revision)
	if revision == "" {
		revision = "1"
	}

	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = filepath.Join(projectRoot, "build", "deb")
	} else if !filepath.IsAbs(outputDir) {
		outputDir = filepath.Join(projectRoot, outputDir)
	}
	outputDir, err = filepath.Abs(outputDir)
	if err != nil {
		return normalizedDebianOptions{}, fmt.Errorf("resolve output directory: %w", err)
	}

	binaries, err := resolveInstallFiles(projectRoot, opts.Binaries, func(src string) string {
		return path.Join("/usr/bin", filepath.Base(src))
	})
	if err != nil {
		return normalizedDebianOptions{}, err
	}

	systemdUnits, err := resolveInstallFiles(projectRoot, opts.Systemd, func(src string) string {
		return path.Join("/lib/systemd/system", filepath.Base(src))
	})
	if err != nil {
		return normalizedDebianOptions{}, err
	}

	configs, err := resolveConfigFiles(projectRoot, opts.Configs)
	if err != nil {
		return normalizedDebianOptions{}, err
	}

	control := opts.Control
	if control.Section == "" {
		control.Section = "misc"
	}
	if control.Priority == "" {
		control.Priority = "optional"
	}
	if strings.TrimSpace(control.Maintainer) == "" {
		control.Maintainer = defaultMaintainer()
	}
	if strings.TrimSpace(control.Description) == "" {
		control.Description = opts.Package
	}
	control.Description = formatDebianDescription(control.Description)
	control.Depends = compactSorted(control.Depends)

	data := ControlData{
		Package:         opts.Package,
		Version:         opts.Version + "-" + revision,
		UpstreamVersion: opts.Version,
		DebianRevision:  revision,
		Architecture:    arch,
		Depends:         strings.Join(control.Depends, ", "),
		DependsList:     append([]string(nil), control.Depends...),
		Section:         control.Section,
		Priority:        control.Priority,
		Maintainer:      control.Maintainer,
		Description:     control.Description,
		Homepage:        strings.TrimSpace(control.Homepage),
		Binaries:        binaries,
		SystemdUnits:    systemdUnits,
		Configs:         configs,
	}

	templateData := map[string]any{
		"Package":         data.Package,
		"Version":         data.Version,
		"UpstreamVersion": data.UpstreamVersion,
		"DebianRevision":  data.DebianRevision,
		"Architecture":    data.Architecture,
		"Depends":         data.Depends,
		"DependsList":     data.DependsList,
		"Section":         data.Section,
		"Priority":        data.Priority,
		"Maintainer":      data.Maintainer,
		"Description":     data.Description,
		"Homepage":        data.Homepage,
		"Binaries":        data.Binaries,
		"SystemdUnits":    data.SystemdUnits,
		"Configs":         data.Configs,
	}
	for key, value := range opts.TemplateData {
		templateData[key] = value
	}

	return normalizedDebianOptions{
		packageName:     opts.Package,
		fullVersion:     data.Version,
		architecture:    arch,
		projectRoot:     projectRoot,
		outputDir:       outputDir,
		binaries:        binaries,
		systemdUnits:    systemdUnits,
		configs:         configs,
		templateData:    templateData,
		skipPackageInfo: opts.SkipPackageInfo,
	}, nil
}

func createPackageStructure(stageDir string, opts normalizedDebianOptions) error {
	dirs := []string{
		filepath.Join(stageDir, "DEBIAN"),
		filepath.Join(stageDir, "usr", "bin"),
		filepath.Join(stageDir, "lib", "systemd", "system"),
		filepath.Join(stageDir, "usr", "share", "doc", opts.packageName),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	if err := copyInstallFiles(stageDir, opts.binaries, 0o755); err != nil {
		return err
	}
	if err := copyInstallFiles(stageDir, opts.systemdUnits, 0o644); err != nil {
		return err
	}
	if err := copyInstallFiles(stageDir, opts.configs, 0o644); err != nil {
		return err
	}

	if err := writeOptionalMetadataFiles(stageDir, opts); err != nil {
		return err
	}
	if err := writeConffiles(stageDir, opts); err != nil {
		return err
	}
	if err := writeControlFile(stageDir, opts); err != nil {
		return err
	}
	if err := copyDocumentation(stageDir, opts); err != nil {
		return err
	}

	return nil
}

func resolveInstallFiles(projectRoot string, sources []string, destination func(string) string) ([]InstallFile, error) {
	files := make([]InstallFile, 0, len(sources))
	for _, src := range sources {
		resolvedSource, err := resolveExistingFile(projectRoot, src)
		if err != nil {
			return nil, err
		}
		dest := destination(src)
		files = append(files, InstallFile{
			Source:      resolvedSource,
			Destination: normalizeInstallPath(dest),
			Name:        filepath.Base(resolvedSource),
		})
	}
	sortInstallFiles(files)
	return files, nil
}

func resolveConfigFiles(projectRoot string, configs map[string]string) ([]InstallFile, error) {
	files := make([]InstallFile, 0, len(configs))
	for src, dest := range configs {
		resolvedSource, err := resolveExistingFile(projectRoot, src)
		if err != nil {
			return nil, err
		}
		normalizedDestination := normalizeInstallPath(dest)
		if normalizedDestination == "/" {
			return nil, fmt.Errorf("invalid config destination: %s", dest)
		}
		files = append(files, InstallFile{
			Source:      resolvedSource,
			Destination: normalizedDestination,
			Name:        path.Base(normalizedDestination),
		})
	}
	sortInstallFiles(files)
	return files, nil
}

func resolveExistingFile(projectRoot, file string) (string, error) {
	if file == "" {
		return "", errors.New("file path is required")
	}
	resolved := file
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(projectRoot, resolved)
	}
	resolved, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", file, err)
	}
	if _, err := os.Stat(resolved); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found at %s", file)
		}
		return "", fmt.Errorf("stat %s: %w", file, err)
	}
	return resolved, nil
}

func writeOptionalMetadataFiles(stageDir string, opts normalizedDebianOptions) error {
	metadataFiles := []struct {
		name string
		mode os.FileMode
	}{
		{name: "preinst", mode: 0o755},
		{name: "postinst", mode: 0o755},
		{name: "prerm", mode: 0o755},
		{name: "postrm", mode: 0o755},
	}

	for _, file := range metadataFiles {
		dest := filepath.Join(stageDir, "DEBIAN", file.name)
		if err := copyOrRenderOptionalFile(opts.projectRoot, filepath.Join("debian", file.name), dest, file.mode, opts.templateData); err != nil {
			if errors.Is(err, errOptionalSourceMissing) {
				continue
			}
			return err
		}
	}

	return nil
}

func writeConffiles(stageDir string, opts normalizedDebianOptions) error {
	entries := make([]string, 0, len(opts.configs))
	for _, config := range opts.configs {
		entries = append(entries, config.Destination)
	}

	staticConffiles, err := readStaticConffiles(opts.projectRoot)
	if err != nil {
		return err
	}
	entries = append(entries, staticConffiles...)
	entries = compactSorted(entries)
	if len(entries) == 0 {
		return nil
	}

	var builder strings.Builder
	for _, entry := range entries {
		builder.WriteString(entry)
		builder.WriteString("\n")
	}

	dest := filepath.Join(stageDir, "DEBIAN", "conffiles")
	if err := os.WriteFile(dest, []byte(builder.String()), 0o644); err != nil {
		return fmt.Errorf("write conffiles: %w", err)
	}
	return nil
}

func readStaticConffiles(projectRoot string) ([]string, error) {
	src := filepath.Join(projectRoot, "debian", "conffiles")
	data, err := os.ReadFile(src)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read conffiles: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	values := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		values = append(values, normalizeInstallPath(line))
	}
	return compactSorted(values), nil
}

func writeControlFile(stageDir string, opts normalizedDebianOptions) error {
	dest := filepath.Join(stageDir, "DEBIAN", "control")
	if err := copyOrRenderOptionalFile(opts.projectRoot, filepath.Join("debian", "control"), dest, 0o644, opts.templateData); err == nil {
		return nil
	} else if !errors.Is(err, errOptionalSourceMissing) {
		return err
	}

	content, err := renderTemplate(defaultControlTemplate, opts.templateData)
	if err != nil {
		return fmt.Errorf("render default control template: %w", err)
	}
	if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write control file: %w", err)
	}
	return nil
}

func copyDocumentation(stageDir string, opts normalizedDebianOptions) error {
	docDir := filepath.Join(stageDir, "usr", "share", "doc", opts.packageName)
	docs := []struct {
		source string
		name   string
		mode   os.FileMode
	}{
		{source: filepath.Join("debian", "README.md"), name: "README.md", mode: 0o644},
		{source: filepath.Join("debian", "copyright"), name: "copyright", mode: 0o644},
		{source: filepath.Join("debian", "changelog"), name: "changelog", mode: 0o644},
	}

	for _, doc := range docs {
		dest := filepath.Join(docDir, doc.name)
		if err := copyOrRenderOptionalFile(opts.projectRoot, doc.source, dest, doc.mode, opts.templateData); err != nil {
			if errors.Is(err, errOptionalSourceMissing) {
				continue
			}
			return err
		}
	}

	projectReadme := filepath.Join(opts.projectRoot, "README.md")
	destReadme := filepath.Join(docDir, "README.md")
	if _, err := os.Stat(destReadme); errors.Is(err, os.ErrNotExist) {
		if _, err := os.Stat(projectReadme); err == nil {
			if err := copyFile(projectReadme, destReadme, 0o644); err != nil {
				return fmt.Errorf("copy README.md: %w", err)
			}
		}
	}

	changelogPath := filepath.Join(docDir, "changelog")
	if _, err := os.Stat(changelogPath); err == nil {
		gzipCmd := exec.Command("gzip", "-9", "-f", changelogPath)
		if err := gzipCmd.Run(); err != nil {
			slog.Warn("failed to compress changelog", "error", err)
		}
	}

	return nil
}

var errOptionalSourceMissing = errors.New("optional source file not found")

type optionalSourceMissingError struct {
	path string
}

func (e optionalSourceMissingError) Error() string {
	return fmt.Sprintf("%s: %v", e.path, errOptionalSourceMissing)
}

func (e optionalSourceMissingError) Unwrap() error {
	return errOptionalSourceMissing
}

func errOptionalFileMissing(path string) error {
	return optionalSourceMissingError{path: path}
}

func copyOrRenderOptionalFile(projectRoot, sourceBase, destination string, mode os.FileMode, data map[string]any) error {
	templateCandidate := filepath.Join(projectRoot, sourceBase+".tpl")
	if _, err := os.Stat(templateCandidate); err == nil {
		templateData, err := os.ReadFile(templateCandidate)
		if err != nil {
			return fmt.Errorf("read template %s: %w", templateCandidate, err)
		}
		rendered, err := renderTemplate(string(templateData), data)
		if err != nil {
			return fmt.Errorf("render template %s: %w", templateCandidate, err)
		}
		if err := writeFile(destination, []byte(rendered), mode); err != nil {
			return err
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat template %s: %w", templateCandidate, err)
	}

	fileCandidate := filepath.Join(projectRoot, sourceBase)
	if _, err := os.Stat(fileCandidate); err == nil {
		return copyFile(fileCandidate, destination, mode)
	} else if errors.Is(err, os.ErrNotExist) {
		return errOptionalFileMissing(fileCandidate)
	} else {
		return fmt.Errorf("stat file %s: %w", fileCandidate, err)
	}
}

func renderTemplate(source string, data map[string]any) (string, error) {
	tmpl, err := template.New("debian").Funcs(template.FuncMap{
		"join": strings.Join,
		"base": path.Base,
		"dir":  path.Dir,
	}).Parse(source)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func copyInstallFiles(stageDir string, files []InstallFile, mode os.FileMode) error {
	for _, file := range files {
		destination := filepath.Join(stageDir, strings.TrimPrefix(filepath.FromSlash(file.Destination), string(filepath.Separator)))
		if err := copyFile(file.Source, destination, mode); err != nil {
			return fmt.Errorf("copy %s to %s: %w", file.Source, file.Destination, err)
		}
	}
	return nil
}

func writeFile(destination string, content []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", destination, err)
	}
	if err := os.WriteFile(destination, content, mode); err != nil {
		return fmt.Errorf("write %s: %w", destination, err)
	}
	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return writeFile(dst, data, mode)
}

func buildPackage(stageDir, packagePath string) error {
	slog.Info("building .deb package", "output", packagePath)

	cmd := exec.Command("dpkg-deb", "--build", "--root-owner-group", stageDir, packagePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("dpkg-deb failed: %w", err)
	}
	return nil
}

func showPackageInfo(packagePath string) error {
	slog.Info("package information")
	fmt.Println()

	infoCmd := exec.Command("dpkg-deb", "--info", packagePath)
	infoCmd.Stdout = os.Stdout
	infoCmd.Stderr = os.Stderr
	if err := infoCmd.Run(); err != nil {
		return err
	}

	fmt.Println()
	slog.Info("package contents")
	contentsCmd := exec.Command("dpkg-deb", "--contents", packagePath)
	contentsCmd.Stdout = os.Stdout
	contentsCmd.Stderr = os.Stderr
	if err := contentsCmd.Run(); err != nil {
		return err
	}

	return nil
}

func normalizeInstallPath(value string) string {
	cleaned := path.Clean("/" + strings.TrimSpace(filepath.ToSlash(value)))
	if cleaned == "." {
		return "/"
	}
	return cleaned
}

func compactSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	compacted := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		compacted = append(compacted, value)
	}
	sort.Strings(compacted)
	return compacted
}

func sortInstallFiles(files []InstallFile) {
	sort.Slice(files, func(i, j int) bool {
		if files[i].Destination == files[j].Destination {
			return files[i].Source < files[j].Source
		}
		return files[i].Destination < files[j].Destination
	})
}

func formatDebianDescription(value string) string {
	lines := strings.Split(strings.TrimSpace(value), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return "No description provided"
	}

	lines[0] = strings.TrimSpace(lines[0])
	for idx := 1; idx < len(lines); idx++ {
		line := strings.TrimSpace(lines[idx])
		if line == "" {
			lines[idx] = " ."
			continue
		}
		lines[idx] = " " + line
	}
	return strings.Join(lines, "\n")
}

func defaultMaintainer() string {
	name := strings.TrimSpace(os.Getenv("DEBFULLNAME"))
	email := strings.TrimSpace(os.Getenv("DEBEMAIL"))
	switch {
	case name != "" && email != "":
		return fmt.Sprintf("%s <%s>", name, email)
	case email != "":
		return email
	default:
		return "Unknown <unknown@localhost>"
	}
}

func isValidDebianArchitecture(value string) bool {
	if value == "" {
		return false
	}
	for idx, ch := range value {
		switch {
		case ch >= 'a' && ch <= 'z':
		case ch >= '0' && ch <= '9':
		case ch == '-' && idx > 0:
		default:
			return false
		}
	}
	return true
}
