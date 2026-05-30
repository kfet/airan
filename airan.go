package airan

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

// envBackend is the environment variable consulted when the agent file
// carries no explicit "backend:" in its frontmatter.
const envBackend = "AIRAN_BACKEND"

// promptToken is the placeholder slot in an adapter's argument template
// that gets replaced with the full agent file contents (the prompt).
const promptToken = "{{prompt}}"

// Sentinel errors. Callers may test for these with errors.Is.
var (
	// ErrUsage is returned when airan is invoked with anything other
	// than exactly one argument (the agent file).
	ErrUsage = errors.New("airan: usage: airan FILE")

	// ErrNoBackend is returned when no backend can be resolved from the
	// file's frontmatter or the AIRAN_BACKEND environment variable.
	ErrNoBackend = errors.New("airan: no backend: set 'backend:' in the file's frontmatter or $" + envBackend)
)

// adapter maps the canonical airan contract onto a concrete CLI. args
// is an argument template: the element equal to promptToken is replaced
// with the prompt at resolution time; every other element is literal.
type adapter struct {
	cmd  string
	args []string
}

// registry is the built-in set of backend adapters. Adding a backend is
// a one-line entry here for now; a declarative (TOML) adapter format is
// a future extension.
var registry = map[string]adapter{
	"claude": {"claude", []string{"-p", promptToken}},
	"fir":    {"fir", []string{"-p", promptToken}},
	"aider":  {"aider", []string{"--message", promptToken}},
}

// Spec is a fully resolved backend invocation, ready to exec. Cmd is the
// command to look up on $PATH; Args are its arguments (not including
// argv[0]).
type Spec struct {
	Cmd  string
	Args []string
}

// ExecFunc replaces the current process with cmd (resolved on $PATH),
// passing args (argv[1:]) and the given environment. On success it does
// not return; it returns an error only if the exec itself fails.
type ExecFunc func(cmd string, args []string, environ []string) error

// Resolve reads the agent file at path, determines its backend, and
// returns the concrete Spec to exec. The entire file — shebang and
// frontmatter included — is passed through as the prompt.
func Resolve(path string, getenv func(string) string) (Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Spec{}, err
	}
	content := string(data)

	backend := frontmatterBackend(content)
	if backend == "" {
		backend = getenv(envBackend)
	}
	if backend == "" {
		return Spec{}, ErrNoBackend
	}

	ad, ok := registry[backend]
	if !ok {
		return Spec{}, fmt.Errorf("airan: unknown backend %q (known: %s)", backend, strings.Join(knownBackends(), ", "))
	}

	args := make([]string, len(ad.args))
	for i, a := range ad.args {
		if a == promptToken {
			args[i] = content
			continue
		}
		args[i] = a
	}
	return Spec{Cmd: ad.cmd, Args: args}, nil
}

// Run is the program entry point. args is the argument list after the
// program name (i.e. os.Args[1:]); it must contain exactly the agent
// file path. On success Run does not return — exec replaces the process.
func Run(args []string, getenv func(string) string, environ []string, exec ExecFunc) error {
	if len(args) != 1 {
		return ErrUsage
	}
	spec, err := Resolve(args[0], getenv)
	if err != nil {
		return err
	}
	return exec(spec.Cmd, spec.Args, environ)
}

// knownBackends returns the registered backend names, sorted.
func knownBackends() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// frontmatterBackend extracts the "backend:" value from the file's
// leading frontmatter block, or "" if there is none. An optional
// shebang line and leading blank lines precede the opening "---" fence;
// parsing stops at the closing "---". Inline "# comments" are stripped.
func frontmatterBackend(content string) string {
	lines := strings.Split(content, "\n")

	i := 0
	if i < len(lines) && strings.HasPrefix(lines[i], "#!") {
		i++
	}
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	if i >= len(lines) || strings.TrimSpace(lines[i]) != "---" {
		return ""
	}

	for i++; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return ""
		}
		if v, ok := parseBackendLine(lines[i]); ok {
			return v
		}
	}
	return ""
}

// parseBackendLine returns the value of a "backend: NAME" frontmatter
// line (comments and surrounding whitespace stripped), and whether the
// line was such a directive with a non-empty value.
func parseBackendLine(line string) (string, bool) {
	const key = "backend:"
	t := strings.TrimSpace(line)
	if !strings.HasPrefix(t, key) {
		return "", false
	}
	v := strings.TrimSpace(t[len(key):])
	if idx := strings.IndexByte(v, '#'); idx >= 0 {
		v = strings.TrimSpace(v[:idx])
	}
	if v == "" {
		return "", false
	}
	return v, true
}
