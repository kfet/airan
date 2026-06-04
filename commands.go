package airan

import (
	"fmt"
	"io"
	"strings"
)

// cmdBackends implements `airan backends`:
//
//	airan backends                 list backends + availability
//	airan backends add NAME CMD…   define/replace a custom backend
//	airan backends remove NAME     delete a custom backend
func cmdBackends(args []string, getenv func(string) string, look LookFunc, out io.Writer) error {
	if len(args) == 0 {
		return backendsList(getenv, look, out)
	}
	switch args[0] {
	case "add":
		return backendsAdd(args[1:], getenv, out)
	case "remove":
		return backendsRemove(args[1:], getenv, out)
	default:
		return ErrUsage
	}
}

// backendsList prints every backend (built-in and custom), marking the
// configured default, tagging custom ones, and reporting whether each
// backend's command is found on $PATH.
func backendsList(getenv func(string) string, look LookFunc, out io.Writer) error {
	cfg, err := loadConfig(getenv)
	if err != nil {
		return err
	}
	for _, name := range backendNames(cfg) {
		ad, _ := lookupBackend(name, cfg)
		avail := "missing"
		if look != nil {
			if _, err := look(ad.cmd); err == nil {
				avail = "available"
			}
		}
		var tags []string
		if _, custom := cfg.backends[name]; custom {
			tags = append(tags, "custom")
		}
		if name == cfg.defaultBackend {
			tags = append(tags, "default")
		}
		suffix := ""
		if len(tags) > 0 {
			suffix = "  (" + strings.Join(tags, ", ") + ")"
		}
		fmt.Fprintf(out, "%-12s %-10s%s\n", name, avail, suffix)
	}
	return nil
}

// backendsAdd defines or replaces a custom backend in the config file.
// The command line must contain the {{prompt}} placeholder, which is
// substituted with the agent file at dispatch time.
func backendsAdd(args []string, getenv func(string) string, out io.Writer) error {
	if len(args) < 2 {
		return ErrUsage
	}
	name := args[0]
	if name == "" || strings.ContainsAny(name, " \t.:#") {
		return fmt.Errorf("airan: invalid backend name %q", name)
	}
	cmdline := args[1:]
	hasPrompt := false
	for _, a := range cmdline {
		if a == promptToken {
			hasPrompt = true
		}
	}
	if !hasPrompt {
		return fmt.Errorf("airan: backend command must include the %s placeholder", promptToken)
	}
	p, err := saveConfigKey(getenv, "backend."+name, strings.Join(cmdline, " "))
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "backend %q added\nconfig: %s\n", name, p)
	return nil
}

// backendsRemove deletes a custom backend from the config file.
func backendsRemove(args []string, getenv func(string) string, out io.Writer) error {
	if len(args) != 1 {
		return ErrUsage
	}
	name := args[0]
	p, removed, err := deleteConfigKey(getenv, "backend."+name)
	if err != nil {
		return err
	}
	if !removed {
		return fmt.Errorf("airan: no custom backend %q", name)
	}
	fmt.Fprintf(out, "backend %q removed\nconfig: %s\n", name, p)
	return nil
}

// cmdConfig implements `airan config`:
//
//	airan config          show the config path, default, and custom backends
//	airan config NAME     set NAME as the default backend
func cmdConfig(args []string, getenv func(string) string, out io.Writer) error {
	switch len(args) {
	case 0:
		return configShow(getenv, out)
	case 1:
		return configSet(args[0], getenv, out)
	default:
		return ErrUsage
	}
}

func configShow(getenv func(string) string, out io.Writer) error {
	p, err := configPath(getenv)
	if err != nil {
		return err
	}
	cfg, err := loadConfig(getenv)
	if err != nil {
		return err
	}
	def := cfg.defaultBackend
	if def == "" {
		def = "(none)"
	}
	fmt.Fprintf(out, "config:          %s\ndefault backend: %s\n", p, def)
	if len(cfg.backends) > 0 {
		fmt.Fprintln(out, "custom backends:")
		for _, name := range sortedNames(cfg.backends) {
			ad := cfg.backends[name]
			cmdline := strings.Join(append([]string{ad.cmd}, ad.args...), " ")
			fmt.Fprintf(out, "  %s: %s\n", name, cmdline)
		}
	}
	return nil
}

func configSet(name string, getenv func(string) string, out io.Writer) error {
	cfg, err := loadConfig(getenv)
	if err != nil {
		return err
	}
	if _, ok := lookupBackend(name, cfg); !ok {
		return fmt.Errorf("airan: unknown backend %q (known: %s)", name, strings.Join(backendNames(cfg), ", "))
	}
	p, err := saveDefault(getenv, name)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "default backend set to %q\nconfig: %s\n", name, p)
	return nil
}
