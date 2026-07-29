package airan

import (
	_ "embed"
	"fmt"
	"io"
	"strings"
)

// versionFile is the canonical version string, shared with the release
// tooling so the binary and the tag can never disagree.
//
//go:embed VERSION
var versionFile string

// Version is the airan release version.
var Version = strings.TrimSpace(versionFile)

// usageText is printed by `airan help` and on a usage error.
const usageText = `airan — the env for AI coding agents.

Usage:
  airan [--prepend TEXT] FILE   dispatch the agent file (the primary use)
  airan backends                list backends + $PATH availability
  airan backends add NAME CMD…  define / replace a custom backend
  airan backends remove NAME    delete a custom backend
  airan config                  show config path, default + custom backends
  airan config NAME             set NAME as the default backend
  airan help                    show this help
  airan version                 show the airan version

Options:
  --prepend TEXT   insert TEXT into the prompt ahead of the file body. If the
                   file has frontmatter, TEXT is inserted just after the
                   closing "---" so the composed prompt remains a well-formed
                   frontmatter document and backend resolution is unaffected.
                   Repeat the flag to add several blocks, in order.

Backend resolution, in precedence order:
  1. the frontmatter "backend:" key
  2. $` + envBackend + `
  3. the configured default backend (airan config NAME)

Typically invoked via a shebang:
  #!/usr/bin/env airan
  ---
  backend: claude
  ---
  <prompt body>
`

// cmdHelp implements `airan help`.
func cmdHelp(out io.Writer) error {
	_, err := io.WriteString(out, usageText)
	return err
}

// cmdVersion implements `airan version`.
func cmdVersion(out io.Writer) error {
	_, err := fmt.Fprintf(out, "airan %s\n", Version)
	return err
}

// options holds the parsed command-line options for a dispatch.
type options struct {
	file    string
	prepend []string
}

// parseDispatchArgs parses the argument list for a file dispatch:
// zero or more options followed by exactly one FILE. It returns
// ErrUsage for anything it does not understand, including a missing
// --prepend value or more than one FILE.
func parseDispatchArgs(args []string) (options, error) {
	var opt options
	seenFile := false

	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--prepend":
			if i+1 >= len(args) {
				return options{}, ErrUsage
			}
			i++
			opt.prepend = append(opt.prepend, args[i])
		case strings.HasPrefix(a, "--prepend="):
			opt.prepend = append(opt.prepend, strings.TrimPrefix(a, "--prepend="))
		case a == "--":
			// Everything after "--" is a literal path.
			if i+1 >= len(args) || i+2 != len(args) || seenFile {
				return options{}, ErrUsage
			}
			opt.file = args[i+1]
			seenFile = true
			i = len(args)
		case strings.HasPrefix(a, "-") && a != "-":
			return options{}, ErrUsage
		default:
			if seenFile {
				return options{}, ErrUsage
			}
			opt.file = a
			seenFile = true
		}
	}
	if !seenFile {
		return options{}, ErrUsage
	}
	return opt, nil
}

// applyPrepend inserts each block of text into content ahead of the
// body. When content carries a frontmatter block the text is inserted
// immediately after its closing "---" fence, so the composed prompt is
// still a well-formed frontmatter document — byte 0 stays "---" (or the
// shebang), and frontmatter backend resolution is unaffected. Without
// frontmatter the text simply leads.
func applyPrepend(content string, blocks []string) string {
	if len(blocks) == 0 {
		return content
	}
	text := strings.Join(blocks, "\n\n")

	idx, ok := frontmatterEnd(content)
	if !ok {
		return text + "\n\n" + content
	}
	head, tail := content[:idx], content[idx:]
	tail = strings.TrimLeft(tail, "\n")
	return head + "\n" + text + "\n\n" + tail
}

// frontmatterEnd returns the offset just past the newline that ends the
// closing "---" fence of a leading frontmatter block, and whether such a
// block was found. An optional shebang and leading blank lines may
// precede the opening fence, matching frontmatterBackend.
func frontmatterEnd(content string) (int, bool) {
	lines := strings.SplitAfter(content, "\n")

	i, off := 0, 0
	if i < len(lines) && strings.HasPrefix(lines[i], "#!") {
		off += len(lines[i])
		i++
	}
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		off += len(lines[i])
		i++
	}
	if i >= len(lines) || strings.TrimSpace(lines[i]) != "---" {
		return 0, false
	}
	off += len(lines[i])

	for i++; i < len(lines); i++ {
		off += len(lines[i])
		if strings.TrimSpace(lines[i]) == "---" {
			return off, true
		}
	}
	return 0, false
}
