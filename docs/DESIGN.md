# airan — design

`airan` is the `env` for AI coding agents: a tiny shebang dispatcher
that resolves a generic *agent file* to a concrete agent CLI and execs
it.

```
#!/usr/bin/env airan
---
backend: claude
---
Refactor src/parser into async/await. Don't touch the public API.
```

```sh
chmod +x build.agent
./build.agent
```

`#!/usr/bin/env python` works because `env` is a *dispatcher* that
resolves `python` to a concrete binary via `$PATH`. `airan` is the same
indirection, but the thing resolved is an **agent backend** and the file
run is a **prompt spec** instead of source code. Stable contract on the
left; pluggable adapters on the right.

## Design principles

1. **No flags.** `airan` has no option flags. Its surface is one
   positional argument — the agent file — plus two reserved verbs
   (`backends`, `config`). Every knob is a verb or a config value, never
   a `--flag`. This keeps the shebang line flag-free, which also
   sidesteps the `env -S` trap (see below).
2. **The whole file is the prompt.** `airan` reads the frontmatter for
   *routing* but never strips it. The agent seeing its own constraints
   (`backend`, `model`, "don't touch the public API") is a feature, not
   noise. Non-destructive parse — the backend receives exactly what is
   on disk, shebang line included.
3. **Stdlib only, 100% covered.** No third-party runtime deps. The
   library package is fully unit-tested behind an injected exec
   function; the only un-coverable code (the `execve` itself) lives in
   the `cmd/airan` entry-point shim, excluded via `.covignore`.

## Commands

```
airan FILE          dispatch the agent file (the primary use)
airan backends      list the built-in backends, marking the default
airan config        show the config path and current default backend
airan config NAME   set NAME as the default backend
```

`backends` and `config` are reserved words. An agent file dispatched via
shebang always arrives as a *path* (with a separator) — e.g.
`airan ./build.agent` or the absolute path the kernel resolves from
`$PATH` — so it can never collide with the bare verbs.

## The agent file

Any executable text file. Optional shebang, optional `---`-fenced
frontmatter, then a free-form prompt body:

```
#!/usr/bin/env airan
---
backend: claude          # which agent CLI to dispatch to
model: opus              # (advisory — ignored by v0, see below)
---
<prompt body>
```

The frontmatter block is the text between the first pair of `---` fences
(an optional shebang and blank lines may precede the opening fence).
`airan` reads exactly one key from it — `backend:` — and ignores the
rest. Other keys (`model`, `tools`, `approval`, …) are reserved for
future versions and are deliberately **not** acted on yet; they remain
visible to the agent as part of the prompt.

## Backend resolution

The backend name is resolved in precedence order:

1. **Frontmatter `backend:`** — explicit, per-file. Wins.
2. **`AIRAN_BACKEND` env var** — run a whole repo through one agent with
   zero file edits: `AIRAN_BACKEND=aider ./build.agent`.
3. **Configured default** — the `backend:` value in the XDG config file
   (see below), set via `airan config NAME`.

If none yields a name, `airan` exits with an error. There is no
hard-coded default — the bottom of the chain is whatever *you* configure.

## Configuration

State lives in a single XDG-standard file:

```
$XDG_CONFIG_HOME/airan/config      # if $XDG_CONFIG_HOME is set (and absolute)
$HOME/.config/airan/config         # otherwise
```

Per the XDG Base Directory spec, a non-absolute `$XDG_CONFIG_HOME` is
ignored in favour of the `$HOME` fallback. The file uses the same
`backend: NAME` line syntax as agent-file frontmatter:

```
# airan config
backend: claude
```

It is read for the default backend and written by `airan config NAME`
(creating the directory as needed). `airan config` with no argument
prints the resolved path and the current default; `airan backends` marks
the default in its listing. Today the file holds exactly one key
(`backend:`); more may follow as the canonical contract grows.

## Adapters

A backend name maps to an **adapter**: the recipe for turning the
canonical contract into that CLI's real invocation. In v0 the registry
is a small in-code table:

| backend  | command | invocation                |
|----------|---------|---------------------------|
| `claude` | claude  | `claude -p <prompt>`      |
| `fir`    | fir     | `fir -p <prompt>`         |
| `aider`  | aider   | `aider --message <prompt>`|

The adapter's argument template carries a single placeholder slot that
is replaced with the prompt (the whole file). The resolved command then
**replaces** the `airan` process via `execve`, so the agent owns the
terminal, signals, and exit code directly — `airan` adds no runtime
overhead once it has dispatched.

Adding a backend today is a one-line table entry. A declarative
(TOML) adapter format — so users can add backends without recompiling —
is the natural next extension once the canonical fields settle.

### Canonical fields (future)

The contract the adapter layer will eventually normalise: `prompt`,
`model`, `files`/`context`, `mode` (interactive vs one-shot),
`approval`/`permissions`, `cwd`, `output_format`. A backend that doesn't
support a field degrades gracefully (no-op or warning). v0 implements
only `prompt`.

## The `-S` shebang gotcha (why no flags)

Principle 1 ("no parameters") is also what sidesteps a classic trap.
A shebang with arguments:

```
#!/usr/bin/env airan --mode print
```

does **not** word-split the way a shell would. The kernel splits the
line into at most two pieces — the interpreter path and *everything
after the first space as one argument* — so `env` searches `$PATH` for a
program literally named `airan --mode print` and fails with `No such
file or directory`. `env -S` ("split") fixes it, but only exists in GNU
coreutils ≥ 8.30 and modern BSD `env` — not in some Alpine/busybox
setups.

By keeping the shebang flag-free (`#!/usr/bin/env airan`) and pushing
all configuration into the frontmatter, `airan` never needs `-S` and
stays portable across every `env`.

## Distribution

- **macOS:** Homebrew. `brew install kfet/tap/airan` (a `airan.rb`
  formula in the tap, building from the tagged source with the Go
  toolchain).
- **Any Unix:** `install.sh` — a POSIX shell installer that builds the
  binary from source via the Go toolchain and drops it on `$PATH`
  (`~/.local/bin` by default, override with `PREFIX`). Run straight from
  a clone, or piped from a release URL.
- **Go users:** `go install github.com/kfet/airan/cmd/airan@latest`.

## Repository model

Cloned from the sibling `kfet/{covgate,pinexec}` libraries: Go,
stdlib-only, a quiet-runner `Makefile` whose default target is gofmt +
`go vet` + staticcheck + race/shuffle tests + a **100% coverage gate**
(via `covgate`) + build. CI runs `make all` on the go.mod floor (1.21)
and latest stable. `AGENTS.md` documents constraints for agents; every
user-visible change gets a `CHANGELOG.md` entry.

## Out of scope for v0

Option flags of any kind · args passthrough · stdin piping · `--dry-run` ·
multi-backend fanout · pinning/lockfiles · per-project config · the
`model` and `tools` frontmatter keys · TOML adapters. All are plausible
extensions; none ship until the core proves itself.
