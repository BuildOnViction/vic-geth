# Project Structure

Here are standard project structures of languages and frameworks, kept for reference.

The file holding the layout attuned to this project is [context.md](../.attunic/context.md).

The files holding the rules for working with this project are [coding-conventions.md](coding-conventions.md) and [development-guideline.md](development-guideline.md).

## General structure

### Layout

Entries a project commonly has at its root, whatever its languages; each layout below lists only what its language or framework adds:

```
├── .agents/            # shared AI agent instructions (coding conventions, PR/task checklists, project structure)
├── .attunic/           # project memory (context, decisions, work in progress)
├── .github/            # GitHub-specific config (workflows, issue/PR templates)
├── .vscode/            # VS Code editor/workspace settings
├── .copier-answers.yml # Copier answers replayed by `copier update`; never edited by hand
├── .editorconfig       # editor formatting rules
├── .gitignore          # git ignore patterns
├── AGENTS.md           # instructions for AI coding agents
├── CLAUDE.md           # points to AGENTS.md
├── CONTRIBUTING.md     # contribution guidelines
├── COPYING             # license text for GPL, LGPL, and AGPL
├── COPYING.LESSER      # LGPL additions, beside COPYING
├── LICENSE             # license text for other licenses (e.g. MIT, Apache-2.0)
├── README.md           # project overview
<!-- Project-specific / Layout -->
```

## Go structure

### Guideline

- Keep `main()` functions thin: parse flags/config, wire dependencies, then delegate to `internal/` packages.
- Code that should never be imported by other projects belongs under `internal/`.
- Only put code under `pkg/` if it's genuinely meant to be a public, importable API; don't use it as a dumping ground.
- Group files within a package by responsibility, not by type.
- Keep test files (`_test.go`) alongside the code they test, in the same package or a `_test` package for black-box tests.
- Mirror the module's existing layout (`internal/`, `pkg/`, or flat) instead of introducing a second convention.
<!-- Project-specific / Go-Guideline -->

### Layout

```
├── api/            # OpenAPI/Swagger specs, JSON schemas, protocol definitions
├── assets/         # repository assets (images, logos, etc)
├── cmd/            # main packages (one subdirectory per binary)
├── config/         # global application configuration
├── deployments/    # deployment configs and templates (docker-compose, kubernetes/helm, terraform)
├── docs/           # design and user documents (beyond godoc)
├── examples/       # examples for applications and/or public libraries
├── init/           # system init (systemd, upstart, sysv) and supervisor (runit, supervisord) configs
├── internal/       # private application/library code, not importable by other modules
├── pkg/            # public library code intended for external use (optional)
├── scripts/        # build/install/analysis scripts (keeps root Makefile small)
├── test/           # additional external test apps and test data
├── tui/            # terminal UI components
├── vendor/         # application dependencies (created by `go mod vendor`)
├── web/            # web app components: static assets, server-side templates, SPAs
├── Dockerfile      # container build definition
├── Makefile        # build/test/lint task automation (optional)
├── go.mod          # Go module definition
├── go.sum          # Go module checksums
├── main.go         # default application entrypoint
<!-- Project-specific / Go-Layout -->
```

### Layer Dependencies

Notes for declaring layers in the `Layer Dependencies` section of `.attunic/context.md`; adjust them to the project before declaring.

- `cmd/` is the outermost layer: entrypoints parse flags/config, wire dependencies, then delegate to `internal/`; it may import everything, and nothing imports it.
- `internal/` holds the layers themselves; one suggestion, innermost first:

| Layer | Typical path | Depends on | Must never contain |
|-------|--------------|------------|--------------------|
| domain | internal/\<app\>/domain | itself | framework types, database, HTTP |
| application | internal/\<app\>/app | domain | database access, HTTP handlers |
| adapters | internal/\<app\>/adapters | application (its ports), domain | business rules |
| transport | internal/\<app\>/transport | application, domain | SQL, external clients |

- `pkg/` is a public boundary for other modules; keep application-internal layer types out of it.
- Check by hand: reading imports or `go list -deps ./...` must never show an inner package importing an outer one.
<!-- Project-specific / Go-Layer Dependencies -->

## Project-specific

<!-- Project-specific -->
