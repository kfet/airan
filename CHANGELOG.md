# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added

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
