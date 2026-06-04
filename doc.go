// Package airan is the env for AI coding agents: a tiny shebang
// dispatcher that resolves a generic agent file into a concrete agent
// CLI invocation and execs it.
//
// An agent file is any executable text file whose first line is a
// shebang pointing at airan:
//
//	#!/usr/bin/env airan
//	---
//	backend: claude
//	---
//	Refactor src/parser into async/await. Don't touch the public API.
//
// The block between the first pair of "---" fences is YAML-ish
// frontmatter; everything else is the prompt. airan never strips the
// frontmatter — the whole file (shebang included) is handed to the
// backend as the prompt, so the agent sees its own constraints.
//
// # No flags
//
// airan takes no flags. Beyond dispatching an agent file it exposes two
// reserved subcommands:
//
//	airan FILE                    dispatch the agent file (the primary use)
//	airan backends                list backends (built-in + custom) + availability
//	airan backends add NAME CMD…  define/replace a custom backend
//	airan backends remove NAME    delete a custom backend
//	airan config                  show the config path, default, and custom backends
//	airan config NAME             set NAME as the default backend
//
// # Backend resolution
//
// The backend is resolved in precedence order:
//
//  1. The frontmatter "backend:" key (explicit, per-file — wins).
//  2. The AIRAN_BACKEND environment variable (whole-repo override).
//  3. The default backend in the XDG config file
//     ($XDG_CONFIG_HOME/airan/config, else $HOME/.config/airan/config).
//
// The resolved backend name is looked up first among any user-defined
// custom backends in the config file, then in the built-in registry of
// adapters (claude, fir, aider, …); a custom backend shadows a built-in
// of the same name. The adapter maps the canonical contract onto that
// CLI's real flags. The prompt — the entire file — is substituted for
// the placeholder slot in the adapter's argument template, and the
// resulting command replaces the airan process via execve.
//
// # Custom backends
//
// Custom backends are declared in the config file as
// "backend.NAME: CMD ARGS…" lines, where the args contain the {{prompt}}
// placeholder, e.g.:
//
//	backend.mycli: mycli --message {{prompt}}
//
// They can be managed with "airan backends add/remove" and are listed,
// with $PATH availability, by "airan backends".
package airan
