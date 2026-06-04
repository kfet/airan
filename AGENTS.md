Guidance for AI agents working on `airan`.

Keep this doc short and to the point, very capable AIs will be reading it soon.

## Scope

`airan` is the `env` for AI coding agents: a **small, focused** shebang
dispatcher. It reads an agent file, resolves a backend, and execs the
matching agent CLI with the whole file as the prompt.

no args passthrough, by design, see `docs/DESIGN.md`

stdlib only

the whole file is the prompt, do not strip frontmatter

backends are built-in (the in-code `registry`) plus user-defined custom
backends from the config file (`backend.NAME: CMD … {{prompt}}`); config
writes preserve comments + unrelated lines

`Run` injects both an `ExecFunc` (exec) and a `LookFunc` ($PATH lookup,
for availability discovery) — keep them injected so the lib stays testable

## Workflow

wrap-up with `make all` to run all checks

keep CHANGELOG.md updated, we follow the standard keepachangelog.com

update all relevant docs on each change

## Coverage exemptions

.covignore coverage exemptions must be intentional and well considered
