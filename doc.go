// Package airun is the env for AI coding agents: a tiny shebang
// dispatcher that resolves a generic agent file into a concrete agent
// CLI invocation and execs it.
//
// An agent file is any executable text file whose first line is a
// shebang pointing at airun:
//
//	#!/usr/bin/env airun
//	---
//	backend: claude
//	---
//	Refactor src/parser into async/await. Don't touch the public API.
//
// The block between the first pair of "---" fences is YAML-ish
// frontmatter; everything else is the prompt. airun never strips the
// frontmatter — the whole file (shebang included) is handed to the
// backend as the prompt, so the agent sees its own constraints.
//
// # No parameters
//
// airun takes exactly one argument — the agent file — and no flags.
// All configuration lives in the file's frontmatter or the
// environment. The public surface is:
//
//	airun FILE
//
// # Backend resolution
//
// The backend is resolved in precedence order:
//
//  1. The frontmatter "backend:" key (explicit, per-file — wins).
//  2. The AIRUN_BACKEND environment variable (whole-repo override).
//
// The resolved backend name is looked up in a built-in registry of
// adapters (claude, fir, aider, …); the adapter maps the canonical
// contract onto that CLI's real flags. The prompt — the entire file —
// is substituted for the placeholder slot in the adapter's argument
// template, and the resulting command replaces the airun process via
// execve.
package airun
