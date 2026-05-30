# airan

[![test](https://github.com/kfet/airan/actions/workflows/test.yml/badge.svg)](https://github.com/kfet/airan/actions/workflows/test.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/kfet/airan.svg)](https://pkg.go.dev/github.com/kfet/airan)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

The `env` for AI coding agents — a tiny shebang dispatcher. Write a
prompt spec once; swap the backend agent with one line.

```
#!/usr/bin/env airan
---
backend: claude
---
Refactor src/parser into async/await. Don't touch the public API.
```

```sh
chmod +x build.agent
./build.agent          # runs the prompt through `claude -p`
```

Change `backend: claude` to `fir` or `aider`, or set
`AIRAN_BACKEND=aider`, and the same file runs through a different agent —
zero other edits.

## How it works

`airan FILE` reads the file, resolves a backend, and **execs** the
matching agent CLI with the **whole file** (frontmatter included) as the
prompt. The frontmatter is read for routing but never stripped, so the
agent sees its own constraints.

Backend resolution, in precedence order:

1. The frontmatter `backend:` key (wins).
2. The `AIRAN_BACKEND` environment variable.
3. The configured default backend (`airan config NAME`).

Built-in backends: `claude` (`claude -p`), `fir` (`fir -p`), `aider`
(`aider --message`).

## Commands

```sh
airan FILE          # dispatch the agent file (the primary use)
airan backends      # list built-in backends, marking the default
airan config        # show config path + current default backend
airan config NAME   # set NAME as the default backend
```

Config lives in one XDG-standard file —
`$XDG_CONFIG_HOME/airan/config`, else `~/.config/airan/config`.

See [docs/DESIGN.md](docs/DESIGN.md) for the full design and rationale.

## Install

**macOS (Homebrew):**

```sh
brew install kfet/tap/airan
```

**Any Unix:**

```sh
./install.sh                 # from a clone (PREFIX overridable)
```

**Go:**

```sh
go install github.com/kfet/airan/cmd/airan@latest
```

## Develop

```sh
make all     # gofmt + vet + staticcheck + race tests + 100% coverage gate + build
```

## License

MIT — see [LICENSE](LICENSE).
