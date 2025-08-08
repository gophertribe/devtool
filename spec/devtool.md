# Scaffolding CLI and Build Library Specification

## Objectives

- Provide a repeatable scaffolding workflow for Go repositories:
  - Generate a `cmd/dev` cobra-based CLI wired to build/test/lint/package/deploy.
  - Install a standardized `Makefile` with CI-friendly targets.
  - Optionally set up frontend build and embed flows.
- Ship a reusable Go library (`github.com/gophertribe/devtool`) with helpers for build, test, lint, dockerized cross-compilation, templating, and project ops.
- Offer template-driven generation with variables, conditionals, hooks, dry-run, and safe overwrites.
- Focus on internal workflows; keep surface area simple and pragmatic.

## Non-goals (V1)

- Plugin system or marketplace.
- Merge-based file conflict resolution (default is overwrite with preview).
- Full CI system provisioning beyond emitting optional templates (CI targeted for V2).

---

## Users and Primary Flows

- Starter: initialize a repo with `cmd/dev`, `Makefile`, and standard layout via one command.
- Maintainer: add new CLI tools/apps consistently and build locally or via Docker.
- Contributor: apply templates for common artifacts (commands, packages, adapters, AI configs).

---

## High-Level Architecture

- CLI binary: `devtool` (cobra)
- Generated project CLI: `cmd/dev` (cobra)
- Library packages under `github.com/gophertribe/devtool/...`:
  - `build`, `dockerbuild`, `lint`, `test`, `frontend`, `templating`, `fsops`, `logging`, `env`, `gitrepo`
- Template assets:
  - Built-in templates embedded in the `devtool` binary
  - Local repository templates under `template/`
  - Remote templates via git URL with optional `#ref`

---

## Repository Layout (after init)

```
repo/
  Makefile
  go.mod / go.sum
  .gitignore
  CHANGELOG.md          # Keep a Changelog format, initialized with Unreleased section
  cmd/
    dev/
      main.go
      cmd/              # subcommands
  internal/
    dev/                # business logic behind cmd/dev commands
  web/                  # optional frontend apps
  template/             # optional project-local templates
  .devtool.yaml         # tool configuration
```

---

## CLI Specification (devtool)

- devtool init
  - Scaffold `cmd/dev`, `Makefile`, `.devtool.yaml`.
  - Flags: `--module`, `--name`, `--git`, `--force`, `--no-frontend`, `--defaults`, `--dry-run`.

- devtool template list
  - List templates from builtin, local `template/`, and optional remote sources.
  - Flags: `--remote <git-url[#ref]>`, `--json`.

- devtool template show <template>
  - Preview template files, variables, and metadata.

- devtool generate <template> [target]
  - Render template into the target directory.
  - Flags: `--var key=value` (repeatable), `--defaults`, `--force`, `--dry-run`, `--from <git-url[#ref]>`.
  - Conflict strategy (V1): overwrite by default with optional diff preview; no merge.

- devtool doctor
  - Validate toolchain presence and versions (Go, Docker, npm if frontend, linters) and print a concise report.
  - Validate secrets prerequisites for `pass` (GPG key presence, `pass` binary, store accessibility).
  - Validate secrets prerequisites for `pass` (GPG key presence, `pass` binary, and store accessibility).

- devtool upgrade
  - Upgrade embedded templates/library references; optionally re-apply selected templates using overwrite rules.

Optional convenience commands (thin wrappers over library): `build`, `lint`, `test`, `frontend ...`.

### Secrets Management (pass-based)

- devtool secrets get <name>
  - Read a secret by logical name or explicit `pass` path; prints to stdout or as `.env` when `--format env`.
- devtool secrets export --env .env.ai [--keys KEY1,KEY2]
  - Export selected secrets from `pass` into a local env file; never commits; updates `.gitignore` if needed.
- devtool secrets doctor
  - Diagnose `pass`/GPG setup and mapping coverage from config.

### Secrets management commands (pass-based)

- devtool secrets get <name>
  - Read a secret by logical name or explicit `pass` path; prints to stdout or as `.env` when `--format env`.
- devtool secrets export --env .env.ai [--keys KEY1,KEY2]
  - Export selected secrets from `pass` into a local env file; never commits; updates `.gitignore` if needed.
- devtool secrets doctor
  - Diagnose `pass`/GPG setup and mapping coverage from config.

---

## CLI Specification (generated cmd/dev)

- Commands delegate to the library for consistent behavior:
  - dev build [app] [--os darwin --arch arm64 --docker]
  - dev test [--unit|--integration]
  - dev lint
  - dev frontend <setup|build|embed>
- Global: `--debug`, `--version`.

---

## Configuration: .devtool.yaml

```yaml
schemaVersion: 1
module: github.com/yourorg/yourrepo
projectName: yourrepo
templates:
  sources:
    - builtin
    - local: template/
    # - remote: git@github.com:yourorg/dev-templates.git#main
build:
  useDockerByDefault: true
  goVersion: "1.22"
  platforms:
    - darwin/arm64
    - linux/amd64
docker:
  image: golang:1.22-bookworm
  mountCache: true
frontend:
  enabled: true
  appDir: web/app
  embedTarget: internal/webassets
lint:
  enable: true
  tool: golangci-lint
test:
  enable: true
  tool: gotestsum
ai:
  providers:
    - claude
    - openai
    - moonshot
    - openrouter
secrets:
  provider: pass
  pass:
    storeDir: ~/.password-store
    gpgKeys:
      - you@example.com
  mappings:
    OPENAI_API_KEY: ai/openai/api_key
    ANTHROPIC_API_KEY: ai/anthropic/api_key
    OPENROUTER_API_KEY: ai/openrouter/api_key
```

---

## Template System

- Sources: builtin (embedded), local `template/`, remote git (`url[#ref]`).
- Syntax: Go `text/template` for file contents and paths (`cmd/{{ .Name }}/main.go`).
- Variables: provided by flags, interactive prompts, and template defaults in `meta.yaml`.
- Hooks: pre/post hooks are supported but disabled by default; require explicit `--allow-hooks`.
- Conflicts: V1 default is overwrite. Diff preview is shown in dry-run or when `--debug` is enabled. No merge.
- Discovery: `devtool template list` merges sources with namespaced IDs (`builtin:`, `local:`, `remote:`).
- Versioning: templates declare `compatVersion` and `requires` constraints.

Secret interpolation in templates (V1):

- Supported only when `--allow-secrets` is provided:
  - `{{ secret "OPENAI_API_KEY" }}` resolves via `secrets.mappings` and `pass`.
  - `{{ secretPath "ai/openai/api_key" }}` addresses a `pass` path directly.
- Disabled by default to avoid leakage.

Template `meta.yaml` example:

```yaml
name: cli-tool
compatVersion: "1"
description: Scaffolds a new cobra CLI under cmd/<name> with internal/<name>
vars:
  name:
    prompt: "CLI tool name"
    default: "tool"
  description:
    prompt: "Description"
    default: ""
hooks:
  post:
    - "go mod tidy"
requires:
  go: ">=1.21"
```

### AI Templates (in scope for V1)

Provide a set of templates to standardize AI coding configurations:

- Claude (Anthropic)
  - `CLAUDE.md` with coding rules and conventions.
  - `.env.ai.example` including `ANTHROPIC_API_KEY` and common flags.
  - Optional `providers.yaml` entry with endpoint and model defaults.

- OpenAI
  - `.env.ai.example` including `OPENAI_API_KEY`.
  - `providers.yaml` example with API base (`https://api.openai.com/v1`) and model defaults (e.g., `gpt-4o`, `gpt-4.1`).

- Moonshot
  - `.env.ai.example` including `MOONSHOT_API_KEY` and base URL if needed.
  - `providers.yaml` entry with recommended models and endpoints.

- OpenRouter
  - `.env.ai.example` including `OPENROUTER_API_KEY` and base URL `https://openrouter.ai/api/v1`.
  - `providers.yaml` entry for model selection and routing guidance.

- MDC configuration (if used in the workflow)
  - Template `mdc.config.yaml` for model/provider selection, rate limits, and tool affordances.

Notes:

- All provider templates are optional and can be generated independently (e.g., `devtool generate ai/openrouter`).
- Avoid committing secrets; only generate `.env.ai.example` and `.gitignore` updates.

---

## Makefile Integration

- Provide standard targets: `build`, `build-%`, `test`, `lint`, `fmt`, `vet`, `clean`, `release`.
- Add `changelog` to generate or update `CHANGELOG.md` using git history and, if available, `git-cliff` or `git-chglog` (fallback to a simple `git log`).
- Frontend: `frontend-build`, `frontend-embed` when enabled.
- Cross-compile via `GOOS/GOARCH` matrix; `DOCKER=1` triggers dockerized builds.
- Variables: `MODULE`, `GO_VERSION`, `OUTPUT_DIR`, `FRONTEND_DIR`, `EMBED_PKG`.
- CI-friendly: all targets must be non-interactive when `CI=1` or `--defaults` is used.

---

## Build and Dockerized Cross-Compilation

- Native builds for simple targets; dockerized path for cgo/cross-compile.
- Standard image (configurable) with module and build cache mounts.
- Propagate `CGO_ENABLED`, `GOOS`, `GOARCH`, and compiler env when needed.
- Library `dockerbuild` abstracts container invocation and logging.

---

## Library Modules (Public APIs)

- build
  - `BuildBinary(ctx, BuildOptions) error`
  - `CrossCompile(ctx, Matrix, BuildOptions) error`
- dockerbuild
  - `BuildInContainer(ctx, DockerOptions, func(ctx context.Context) error) error`
- lint
  - `LintGolang(ctx, LintOptions) error`
- test
  - `TestGolang(ctx, TestOptions) error`
- frontend
  - `Setup(ctx, FrontendOptions) error`
  - `Build(ctx, FrontendOptions) error`
  - `Embed(ctx, EmbedOptions) error`
- templating
  - `Render(source, targetFS, vars, hooks, strategy) (Report, error)`
- fsops
  - Safe write with conflict detection, diff report, transactional apply.
- logging
  - Structured slog with colorful handler.
- env
  - Toolchain detection (Go, Docker, npm), version checks, doctor report.
- gitrepo
  - Fetch/caches remote template repos by URL+ref under `~/.cache/devtool/`.

All exported APIs should be explicit and strongly typed; avoid `any`.

---

## Frontend Support (Optional)

- `frontend setup`: install deps in `web/<app>`.
- `frontend build`: run frontend build script.
- `frontend embed`: either copy build assets or generate a small Go package that exposes `embed.FS` with `go:embed`.

---

## Logging and UX

- Global `--debug` for verbose logs; default to concise progress logs.
- Dry-run prints a plan of file operations and diffs.
- Overwrite is the default; interactive merge is not provided in V1.

---

## Security

- Hooks disabled by default; require `--allow-hooks` to execute.
- Remote template fetching uses read-only checkouts; refs must be explicit unless `main`.
- Docker images pinned by tag; allow override via config or env.
- Secrets are never logged; redact known secret keys and values returned by `secret(...)` in templates.

---

## Versioning and Compatibility

- `devtool` is semver versioned.
- Embedded templates include `compatVersion`; upgrades validate compatibility.
- `.devtool.yaml` schema is versioned; `devtool upgrade` can migrate.

---

## Performance and Caching

- Cache remote templates under `~/.cache/devtool/`.
- Reuse Go module/cache in docker builds via bind mounts.
- Hash-based writes avoid re-rendering unchanged files.

---

## Testing Strategy

- Unit tests: templating engine, fsops (overwrite and diff), docker wrapper, build matrix.
- Golden tests for templates (Makefile, cmd/dev, AI provider configs).
- E2E smoke tests: `devtool init` into temp dir; run `make build test`.

---

## Documentation

- `spec/` contains this specification and related design docs.
- `docs/` should include:
  - Quickstart
  - CLI reference (`devtool` and generated `dev`)
  - Template authoring guide
  - Build and cross-compilation guide
  - Frontend embedding guide
  - AI configuration templates (providers and examples)
  - Secrets management with `pass` and env file exports
  - Base `.gitignore` and `CHANGELOG.md` guidance
  - Troubleshooting/doctor

---

## Milestones

- M1: `init` + `Makefile` + `cmd/dev` skeleton + `.devtool.yaml` + base `.gitignore` and `CHANGELOG.md`.
- M2: Template engine (builtin + local), `generate`, overwrite handling, dry-run.
- M3: Library build/test/lint, Makefile wiring, doctor.
- M4: Dockerized cross-compile.
- M5: Frontend build/embed support.
- M6: AI provider templates (Claude, OpenAI, Moonshot, OpenRouter) and docs.
- V2: CI workflows templates (e.g., GitHub Actions) and guided setup.

---

## Open Questions

- Define the exact "mdc" configuration format targeted by the workflow (file name, schema, and intended tools).
- Preferred default model sets per provider for examples (e.g., Claude 3.5 Sonnet, GPT-4o).
- Any additional AI providers to support out-of-the-box (e.g., Cohere, Google AI Studio) in V1 or V2?


