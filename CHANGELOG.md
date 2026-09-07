# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Changed

- The root `install.sh` is now **generated** from `install.sh.json` by the
  canonical template in `github.com/kfet/distkit/installsh` v0.1.4 instead
  of being hand-written, so the whole kfet family shares one installer.
  `make install.sh` regenerates it; `make check-installsh` (wired into
  `make all`, and therefore CI) fails the build when the checked-in copy
  has drifted. distkit is a dev-only `go run` tool — `airan` itself stays
  stdlib-only.

  What this gains over the old script: sha256 verification against the
  release `checksums.txt` (the old one installed unverified), `wget` as a
  fallback when `curl` is absent, the `armv8l` 32-bit ARM spelling
  alongside `armv6l`/`armv7l`, `GITHUB_TOKEN` support for private repos
  and spent rate limits, `VERSION`/`REPO`/`OS`/`ARCH` overrides, atomic
  install via a sibling temp file (so replacing a running binary cannot
  fail with ETXTBSY), `sudo` escalation when the destination is not
  writable, a version smoke test and next-step hints.

  `PREFIX=… ./install.sh` keeps working as before. The destination
  variable is now `BIN_DIR` and, when neither is set, defaults to
  `/usr/local/bin` if it is writable and `~/.local/bin` otherwise —
  previously it was always `~/.local/bin`.

## [0.1.3] - 2026-07-30

### Added

- `--prepend TEXT` composes a prompt at dispatch time, inserting TEXT
  ahead of the file body. The text lands **after** the frontmatter block,
  so the composed prompt stays a well-formed frontmatter document and
  backend resolution is unchanged. Repeatable; blocks keep their order.
  This lets a caller add an instruction without rewriting the file or
  materialising a temporary one.

- `airan help` prints the full synopsis, and `airan version` prints the
  version (embedded from `VERSION`, so the binary and the tag cannot
  disagree). Both accept the usual flag spellings — `-h`/`--help` and
  `-V`/`--version`.

### Fixed

- A non-file first argument is no longer treated as a path. `airan help`
  and `airan --version` previously failed with
  `open --version: no such file or directory`. Unrecognised `-`-prefixed
  arguments now produce a usage error, and `--` marks a literal path.

## [0.1.2] - 2026-07-28

### Changed

- Releases are now built by **GoReleaser** (`.goreleaser.yaml`) instead
  of a hand-rolled cross-compile loop in the workflow. Same eleven
  OS/arch assets under the same names, so `install.sh` is unaffected.
  GoReleaser also regenerates `Formula/airan.rb` on the `kfet/homebrew-ai`
  tap on every tagged release, so `brew install kfet/ai/airan` tracks
  the latest version automatically.

- staticcheck is now a pinned go.mod `tool` dependency and runs via
  `go tool staticcheck` — `make all` no longer silently skips it when
  the binary is absent, and nothing needs installing by hand. This
  raises the go.mod floor to **1.25** (what the `tool` directive and
  staticcheck require); CI's matrix floor moves 1.21 → 1.25.

## [0.1.1] - 2026-07-27

### Changed

- Homebrew install instructions now point at the `kfet/ai` tap:
  `brew install kfet/ai/airan` (was `kfet/tap/airan`). Updated in
  `README.md`, `install.sh`, and `docs/DESIGN.md`.

## [0.1.0] - 2026-06-03

### Added

- `make install` target — builds and installs `airan` into `$PREFIX`
  (default `~/.local`).

- Initial implementation of `airan` — the `env` for AI coding agents.
- `airan FILE` reads an agent file, resolves a backend, and execs the
  matching agent CLI with the **whole file** (frontmatter included) as
  the prompt.
- Backend resolution precedence: frontmatter `backend:` key,
  `AIRAN_BACKEND`, then the configured default backend.
- **Custom backends.** Declare your own adapters in the config file as
  `backend.NAME: CMD ARGS… {{prompt}}` lines, or manage them with
  `airan backends add NAME CMD…` / `airan backends remove NAME`. A custom
  backend shadows a built-in of the same name.
- `airan backends` — list all backends (built-in + custom), marking the
  default and tagging custom ones, and reporting whether each backend's
  command is found on `$PATH` (availability discovery).
- `airan config` — show the config path, current default backend, and
  any custom backends; `airan config NAME` sets the default. State lives
  in the XDG-standard file `$XDG_CONFIG_HOME/airan/config` (else
  `~/.config/airan/config`). Writes preserve comments and unrelated lines.
- Built-in backend adapters: `claude` (`claude -p`), `fir` (`fir -p`),
  `aider` (`aider --message`).
- Library API: `Resolve`, `Run`, `Spec`, `ExecFunc`, `LookFunc`, and the
  sentinel errors `ErrUsage` and `ErrNoBackend`.
- Pre-built release binaries: a GitHub Actions workflow cross-compiles a
  matrix of OS/arch targets on tag push, and `install.sh` downloads the
  matching binary (no Go toolchain required); `make build-all` builds the
  matrix locally.
- `docs/DESIGN.md` — full design and rationale, including the no-params
  decision and the `env -S` shebang gotcha it sidesteps.
