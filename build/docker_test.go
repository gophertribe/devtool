package build

import (
	"reflect"
	"testing"
)

func TestDockerBuildImageArgs(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		got := dockerBuildImageArgs(DockerImageBuildOpts{}, "acme/app:latest")
		want := []string{"buildx", "build", "--load", "-t", "acme/app:latest", "-f", "Dockerfile", "."}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("dockerBuildImageArgs() = %#v, want %#v", got, want)
		}
	})

	t.Run("platform and custom context", func(t *testing.T) {
		got := dockerBuildImageArgs(DockerImageBuildOpts{
			Dockerfile: "docker/app.Dockerfile",
			ContextDir: "./deploy",
			Platform:   "linux/amd64",
		}, "acme/app:latest")
		want := []string{"buildx", "build", "--load", "--platform", "linux/amd64", "-t", "acme/app:latest", "-f", "docker/app.Dockerfile", "./deploy"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("dockerBuildImageArgs() = %#v, want %#v", got, want)
		}
	})
}

func TestDockerImageTags(t *testing.T) {
	latest, version, err := dockerImageTags(DockerImageBuildOpts{
		Image:   "acme/app",
		Version: "1.2.3",
	})
	if err != nil {
		t.Fatalf("dockerImageTags() error = %v", err)
	}
	if latest != "acme/app:latest" {
		t.Fatalf("latest tag = %q, want %q", latest, "acme/app:latest")
	}
	if version != "acme/app:1.2.3" {
		t.Fatalf("version tag = %q, want %q", version, "acme/app:1.2.3")
	}
}

func TestDockerImageTagsRequiresValues(t *testing.T) {
	tests := []DockerImageBuildOpts{
		{Version: "1.2.3"},
		{Image: "acme/app"},
	}

	for _, opts := range tests {
		if _, _, err := dockerImageTags(opts); err == nil {
			t.Fatalf("dockerImageTags(%+v) error = nil, want non-nil", opts)
		}
	}
}
