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
- `airan backends` — list the built-in backends, marking the default.
- `airan config` — show the config path and current default backend;
  `airan config NAME` sets the default. State lives in the XDG-standard
  file `$XDG_CONFIG_HOME/airan/config` (else `~/.config/airan/config`).
- Built-in backend adapters: `claude` (`claude -p`), `fir` (`fir -p`),
  `aider` (`aider --message`).
- Library API: `Resolve`, `Run`, `Spec`, `ExecFunc`, and the sentinel
  errors `ErrUsage` and `ErrNoBackend`.
- `install.sh` (build-from-source installer for any Unix) and Homebrew
  install path for macOS.
- `docs/DESIGN.md` — full design and rationale, including the no-params
  decision and the `env -S` shebang gotcha it sidesteps.
