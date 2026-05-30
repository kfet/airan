# AGENTS.md

Guidance for AI agents working on `airan`.

## Scope

`airan` is the `env` for AI coding agents: a **small, focused** shebang
dispatcher. It reads an agent file, resolves a backend, and execs the
matching agent CLI with the whole file as the prompt. Building blocks:

- `Resolve` / `Spec` — read the file, pick the backend, build the
  concrete invocation (`airan.go`).
- `Run` / `ExecFunc` — the program entry point; dispatches subcommands
  or an agent file, execs via an injected exec function (`airan.go`).
- `registry` — the in-code table of backend adapters (`airan.go`).
- `frontmatterBackend` / `parseBackendLine` — frontmatter parsing
  (`airan.go`).
- `configPath` / `loadDefault` / `saveDefault` — XDG config file
  handling (`config.go`).
- `cmdBackends` / `cmdConfig` — the `backends` and `config` subcommands
  (`commands.go`).
- `cmd/airan/main.go` — the entry-point shim wiring `Run` to the real
  `syscall.Exec`.

`doc.go` is the source of truth for the public API surface; keep it, this
list, and `docs/DESIGN.md` in sync.

**Do not** add option flags (`--foo`), args passthrough, stdin piping,
`--dry-run`, or new frontmatter keys without an explicit decision. The
surface is one positional file argument plus the `backends`/`config`
verbs; configuration is the XDG config file and `AIRAN_BACKEND`. See
`docs/DESIGN.md` § "Out of scope for v0".

## Constraints

- **Stdlib only.** No third-party runtime deps. Ever. If you reach for
  one, stop and ask first. (`covgate` is a build-time tool, not a dep.)
- **Go 1.21+.** Don't use newer language features without a real need;
  bumping the minimum cuts users.
- **No global mutable state.** The `registry` is the one package-level
  table and is treated as read-only.
- **The whole file is the prompt.** Never strip the shebang or
  frontmatter before handing the content to a backend.
- **Exec is injected.** Keep the real `syscall.Exec` in
  `cmd/airan/main.go` so the library stays 100% testable; `Run` takes an
  `ExecFunc`.

## Workflow

- `make all` runs gofmt + go vet + staticcheck (if installed) + race
  tests with a **100% coverage gate** (excluding paths in `.covignore`)
  + build. Must pass before any commit.
- Add a `## [Unreleased]` entry in `CHANGELOG.md` for any user-visible
  change.
- Update `doc.go`, `README.md`, `docs/DESIGN.md`, and `AGENTS.md` when
  the public API or behaviour changes.

## Coverage exemptions

`cmd/airan/main.go` is excluded from the coverage gate via `.covignore` —
it is the entry-point shim whose only real logic is the `syscall.Exec`
process replacement, which can't run under the test process.
