package airan

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// errNoConfigDir is returned internally when no config directory can be
// determined (neither $XDG_CONFIG_HOME nor $HOME is usable).
var errNoConfigDir = errors.New("airan: cannot determine config dir: set $XDG_CONFIG_HOME or $HOME")

// config is the parsed contents of the XDG config file: the default
// backend plus any user-defined custom backend adapters.
type config struct {
	defaultBackend string
	backends       map[string]adapter
}

// configPath returns the XDG-standard path to airan's config file:
// $XDG_CONFIG_HOME/airan/config, falling back to $HOME/.config/airan/config.
// Per the XDG Base Directory spec, a non-absolute $XDG_CONFIG_HOME is
// ignored.
func configPath(getenv func(string) string) (string, error) {
	base := getenv("XDG_CONFIG_HOME")
	if base == "" || !filepath.IsAbs(base) {
		home := getenv("HOME")
		if home == "" {
			return "", errNoConfigDir
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "airan", "config"), nil
}

// loadConfig reads and parses the config file. A missing config file (or
// no resolvable config dir) is not an error — it yields an empty config
// with a nil error. Only a genuine read failure surfaces.
func loadConfig(getenv func(string) string) (config, error) {
	cfg := config{backends: map[string]adapter{}}
	p, err := configPath(getenv)
	if err != nil {
		return cfg, nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		key, val, ok := parseConfigLine(line)
		if !ok {
			continue
		}
		switch {
		case key == "backend":
			cfg.defaultBackend = val
		case strings.HasPrefix(key, "backend."):
			name := strings.TrimPrefix(key, "backend.")
			if ad, ok := parseAdapter(val); ok {
				cfg.backends[name] = ad
			}
		}
	}
	return cfg, nil
}

// loadDefault reads the default backend from the config file. A missing
// config file is not an error — it yields "" with a nil error.
func loadDefault(getenv func(string) string) (string, error) {
	cfg, err := loadConfig(getenv)
	if err != nil {
		return "", err
	}
	return cfg.defaultBackend, nil
}

// saveDefault writes name as the default backend, preserving any other
// config lines. It returns the path written.
func saveDefault(getenv func(string) string, name string) (string, error) {
	return saveConfigKey(getenv, "backend", name)
}

// saveConfigKey sets "key: value" in the config file, replacing the
// existing line for that key if present and otherwise appending it.
// Comments and unrelated lines are preserved. It returns the path written.
func saveConfigKey(getenv func(string) string, key, value string) (string, error) {
	p, err := configPath(getenv)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	lines, err := readConfigLines(p)
	if err != nil {
		return "", err
	}
	newLine := key + ": " + value
	found := false
	for i, line := range lines {
		if k, _, ok := parseConfigLine(line); ok && k == key {
			lines[i] = newLine
			found = true
			break
		}
	}
	if !found {
		if len(lines) == 0 {
			lines = append(lines, "# airan config")
		}
		lines = append(lines, newLine)
	}
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		return "", err
	}
	return p, nil
}

// deleteConfigKey removes the line defining key, preserving all others.
// removed reports whether such a line existed; if not, the file is left
// untouched.
func deleteConfigKey(getenv func(string) string, key string) (path string, removed bool, err error) {
	p, err := configPath(getenv)
	if err != nil {
		return "", false, err
	}
	lines, err := readConfigLines(p)
	if err != nil {
		return "", false, err
	}
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if k, _, ok := parseConfigLine(line); ok && k == key {
			removed = true
			continue
		}
		kept = append(kept, line)
	}
	if !removed {
		return p, false, nil
	}
	if err := os.WriteFile(p, []byte(strings.Join(kept, "\n")+"\n"), 0o644); err != nil {
		return "", false, err
	}
	return p, true, nil
}

// readConfigLines returns the config file's lines (trailing newline
// stripped), or nil for a missing or empty file.
func readConfigLines(p string) ([]string, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	s := strings.TrimRight(string(data), "\n")
	if s == "" {
		return nil, nil
	}
	return strings.Split(s, "\n"), nil
}

// parseConfigLine parses a "key: value" config line. It returns the key,
// the value (with any trailing "# comment" stripped), and whether the
// line is a directive (non-blank, non-comment, with a colon and key).
func parseConfigLine(line string) (key, value string, ok bool) {
	t := strings.TrimSpace(line)
	if t == "" || strings.HasPrefix(t, "#") {
		return "", "", false
	}
	idx := strings.IndexByte(t, ':')
	if idx < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(t[:idx])
	if key == "" {
		return "", "", false
	}
	value = strings.TrimSpace(t[idx+1:])
	if c := strings.IndexByte(value, '#'); c >= 0 {
		value = strings.TrimSpace(value[:c])
	}
	return key, value, true
}

// parseAdapter turns a whitespace-separated command line into an
// adapter (first field is the command, the rest are literal args). It
// reports false for an empty value.
func parseAdapter(value string) (adapter, bool) {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return adapter{}, false
	}
	return adapter{cmd: fields[0], args: fields[1:]}, true
}

// configBackend extracts the "backend:" value from config file contents,
// or "" if none.
func configBackend(content string) string {
	for _, line := range strings.Split(content, "\n") {
		if k, v, ok := parseConfigLine(line); ok && k == "backend" {
			return v
		}
	}
	return ""
}

// sortedNames returns the keys of m, sorted.
func sortedNames(m map[string]adapter) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
