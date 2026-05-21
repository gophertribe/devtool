package build

import (
	"reflect"
	"testing"
)

func TestBuildImageRef(t *testing.T) {
	tests := []struct {
		name             string
		goMinor          string
		codename, flavor string
		want             string
	}{
		{"all defaults", "", "", "", "forgejo.gophertribe.com/gophertribe/gobuild:1.25-bookworm"},
		{"base flavor explicit", "1.24", "bookworm", "base", "forgejo.gophertribe.com/gophertribe/gobuild:1.24-bookworm"},
		{"buster wails", "1.24", "buster", "wails", "forgejo.gophertribe.com/gophertribe/gobuild:1.24-buster-wails"},
		{"bookworm audio", "1.25", "bookworm", "audio", "forgejo.gophertribe.com/gophertribe/gobuild:1.25-bookworm-audio"},
		{"trixie audio", "1.24", "trixie", "audio", "forgejo.gophertribe.com/gophertribe/gobuild:1.24-trixie-audio"},
		{"trixie base", "1.25", "trixie", "base", "forgejo.gophertribe.com/gophertribe/gobuild:1.25-trixie"},
		{"unknown flavor is treated as base", "1.25", "bookworm", "weird", "forgejo.gophertribe.com/gophertribe/gobuild:1.25-bookworm"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildImageRef(tt.goMinor, tt.codename, tt.flavor); got != tt.want {
				t.Fatalf("BuildImageRef(%q,%q,%q) = %q, want %q", tt.goMinor, tt.codename, tt.flavor, got, tt.want)
			}
		})
	}
}

func TestResolveBuildImage(t *testing.T) {
	t.Run("explicit opts.Image wins", func(t *testing.T) {
		t.Setenv(envBuildImageOverride, "env.example/foo:bar")
		got := resolveBuildImage(DockerBuildOpts{Image: "explicit/img:tag"})
		if got != "explicit/img:tag" {
			t.Fatalf("got %q, want %q", got, "explicit/img:tag")
		}
	})

	t.Run("env override beats composition", func(t *testing.T) {
		t.Setenv(envBuildImageOverride, "env.example/foo:bar")
		got := resolveBuildImage(DockerBuildOpts{GoMinor: "1.24", Codename: "bookworm"})
		if got != "env.example/foo:bar" {
			t.Fatalf("got %q, want %q", got, "env.example/foo:bar")
		}
	})

	t.Run("composition fallback", func(t *testing.T) {
		t.Setenv(envBuildImageOverride, "")
		got := resolveBuildImage(DockerBuildOpts{GoMinor: "1.24", Codename: "buster", Flavor: "wails"})
		want := "forgejo.gophertribe.com/gophertribe/gobuild:1.24-buster-wails"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("all defaults", func(t *testing.T) {
		t.Setenv(envBuildImageOverride, "")
		got := resolveBuildImage(DockerBuildOpts{})
		want := "forgejo.gophertribe.com/gophertribe/gobuild:1.25-bookworm"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
}

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
