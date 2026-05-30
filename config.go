package airan

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// errNoConfigDir is returned internally when no config directory can be
// determined (neither $XDG_CONFIG_HOME nor $HOME is usable).
var errNoConfigDir = errors.New("airan: cannot determine config dir: set $XDG_CONFIG_HOME or $HOME")

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

// loadDefault reads the default backend from the config file. A missing
// config file (or no resolvable config dir) is not an error — it yields
// "" with a nil error. Only a genuine read failure surfaces.
func loadDefault(getenv func(string) string) (string, error) {
	p, err := configPath(getenv)
	if err != nil {
		return "", nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return configBackend(string(data)), nil
}

// saveDefault writes name as the default backend to the config file,
// creating the config directory if needed. It returns the path written.
func saveDefault(getenv func(string) string, name string) (string, error) {
	p, err := configPath(getenv)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	content := "# airan config\nbackend: " + name + "\n"
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		return "", err
	}
	return p, nil
}

// configBackend extracts the "backend:" value from config file contents,
// or "" if none. The config file uses the same "backend: NAME" line
// syntax as agent-file frontmatter (with "# comments").
func configBackend(content string) string {
	for _, line := range strings.Split(content, "\n") {
		if v, ok := parseBackendLine(line); ok {
			return v
		}
	}
	return ""
}
