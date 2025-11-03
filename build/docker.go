package build

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/docker/docker/pkg/stdcopy"
	"github.com/magefile/mage/sh"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

const buildImage = "gophertribe/gobuild:1.25-bookworm"

// DockerBuildOpts represents options for Docker builds
type DockerBuildOpts struct {
	Deps     []string
	DepsPath string
	NoCache  bool
	Arch     string
	Image    string
}

// GoBuildOpts represents options for Go builds
type GoBuildOpts struct {
	EnableCgo     bool
	InjectVersion bool
	Version       string
	ConfigPackage string
	Tags          []string
	Arch          string
	OS            string
}

// Docker runs a mage target in a Docker container
func Docker(ctx context.Context, command string, commandArgs []string, opts DockerBuildOpts) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("could not initialize docker client: %w", err)
	}

	info, err := cli.Info(ctx)
	if err != nil {
		return fmt.Errorf("could not get docker info: %w", err)
	}
	slog.Info("docker info", "arch", info.Architecture, "os", info.OperatingSystem, "version", info.ServerVersion)

	img := buildImage
	if opts.Image != "" {
		img = opts.Image
	}

	reader, err := cli.ImagePull(ctx, img, image.PullOptions{
		Platform: "linux/amd64",
	})
	if err != nil {
		return fmt.Errorf("could not pull build image: %w", err)
	}
	_, _ = io.Copy(os.Stdout, reader)

	pwd := strings.TrimSuffix(GetProjectPath(), "/")
	workspace := strings.TrimSuffix(getWorkspacePath(), "/")

	// if we run inside a workspace we have to adjust the path
	dir, project := filepath.Split(pwd)
	projectDir := fmt.Sprintf("/src/%s", project)
	if workspace != "" {
		pwd = workspace
		projectDir = fmt.Sprintf("/src/%s", filepath.Base(dir))
		project = filepath.Join(filepath.Base(dir), project)
	}
	workingDir := fmt.Sprintf("/src/%s", project)

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not establish user home: %w", err)
	}

	binds := []string{
		fmt.Sprintf("%s:%s", pwd, projectDir),
	}

	netrc := filepath.Join(home, ".netrc")
	_, err = os.Stat(filepath.Join(home, ".gobuild_netrc"))
	if err == nil {
		netrc = filepath.Join(home, ".gobuild_netrc")
	}
	binds = append(binds, fmt.Sprintf("%s:/root/.netrc", netrc))

	if !opts.NoCache {
		// set go build cache inside the container
		binds = append(binds, fmt.Sprintf("%s/.build/cache:/root/.cache/go-build", pwd))
		binds = append(binds, fmt.Sprintf("%s/.build/mod:/go/pkg/mod", pwd))
	}

	for _, dep := range opts.Deps {
		binds = append(binds, fmt.Sprintf("%s/%s:/src/%s", opts.DepsPath, dep, dep))
	}

	cmd := append([]string{command}, commandArgs...)
	slog.Info("running build container", "binds", binds, "cmd", cmd, "workingDir", workingDir)

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:      img,
		WorkingDir: workingDir,
		Cmd:        cmd,
	}, &container.HostConfig{
		Binds:      binds,
		AutoRemove: true,
	}, nil, &v1.Platform{
		Architecture: "amd64",
		OS:           "linux",
	}, "")
	if err != nil {
		return fmt.Errorf("could not create build container: %w", err)
	}

	err = cli.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("could not start build container: %w", err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err = <-errCh:
		err = fmt.Errorf("container wait error: %w", err)
	case status := <-statusCh:
		if status.Error != nil {
			err = fmt.Errorf("container exit error: %s", status.Error.Message)
		} else if status.StatusCode != 0 {
			err = fmt.Errorf("container exit code: %d", status.StatusCode)
		}
	}

	defer func() {
		_ = cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{
			Force: true,
		})
	}()

	out, errlog := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if errlog != nil {
		slog.Error("could not init container log reader", "error", errlog)
	}
	if out != nil {
		_, _ = stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	}
	return err
}

// DockerBuildImage builds a Docker image
func DockerBuildImage(dockerfile, image, version string) error {
	tag := fmt.Sprintf("%s:latest", image)
	err := sh.Run("docker", "build", "-t", tag, "-f", dockerfile, ".")
	if err != nil {
		return fmt.Errorf("error building docker container: %w", err)
	}
	err = sh.Run("docker", "tag", tag, fmt.Sprintf("%s:%s", image, version))
	if err != nil {
		return fmt.Errorf("error tagging docker container: %w", err)
	}
	return nil
}

// DockerTagImage tags a Docker image
func DockerTagImage(baseImage, releaseImage, version string) error {
	baseTag := fmt.Sprintf("%s:latest", baseImage)
	releaseLatest := fmt.Sprintf("%s:latest", releaseImage)
	releaseVersion := fmt.Sprintf("%s:%s", releaseImage, version)

	err := sh.Run("docker", "tag", baseImage, releaseLatest)
	if err != nil {
		return fmt.Errorf("error tagging docker container: %w", err)
	}

	err = sh.Run("docker", "tag", baseTag, releaseVersion)
	if err != nil {
		return fmt.Errorf("error tagging docker container: %w", err)
	}
	return nil
}

// DockerPublishImage publishes a Docker image
func DockerPublishImage(releaseImage, version string) error {
	releaseLatest := fmt.Sprintf("%s:latest", releaseImage)
	releaseVersion := fmt.Sprintf("%s:%s", releaseImage, version)

	err := sh.Run("docker", "push", releaseVersion)
	if err != nil {
		return fmt.Errorf("error pushing docker container: %w", err)
	}

	err = sh.Run("docker", "push", releaseLatest)
	if err != nil {
		return fmt.Errorf("error pushing docker container: %w", err)
	}
	return nil
}

// getWorkspacePath returns the Go workspace path
func getWorkspacePath() string {
	out, _ := exec.Command("go", "env", "GOWORK").CombinedOutput()
	val := strings.TrimSpace(string(out))
	if val == "" {
		return ""
	}
	dir, _ := filepath.Split(val)
	return dir
}

// GetProjectPath returns the Go project path
func GetProjectPath() string {
	out, _ := exec.Command("go", "env", "GOMOD").CombinedOutput()
	val := strings.TrimSpace(string(out))
	if val == "" {
		return ""
	}
	dir, _ := filepath.Split(val)
	return dir
}

// GetOutboundIP returns the outbound IP address
func GetOutboundIP() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("route", "get", "default").CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("failed to get route: %w", err)
		}
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.Contains(line, "interface:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					iface := parts[1]
					out, err := exec.Command("ifconfig", iface).CombinedOutput()
					if err != nil {
						return "", fmt.Errorf("failed to get interface config: %w", err)
					}
					lines := strings.Split(string(out), "\n")
					for _, line := range lines {
						if strings.Contains(line, "inet ") && !strings.Contains(line, "127.0.0.1") {
							parts := strings.Fields(line)
							if len(parts) >= 2 {
								return parts[1], nil
							}
						}
					}
				}
			}
		}
	}
	return "127.0.0.1", nil
}
