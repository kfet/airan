package airan

import (
	"fmt"
	"io"
	"strings"
)

// cmdBackends implements `airan backends`: list the built-in backend
// names, marking the configured default.
func cmdBackends(args []string, getenv func(string) string, out io.Writer) error {
	if len(args) != 0 {
		return ErrUsage
	}
	def, _ := loadDefault(getenv)
	for _, name := range knownBackends() {
		marker := ""
		if name == def {
			marker = "  (default)"
		}
		fmt.Fprintf(out, "%s%s\n", name, marker)
	}
	return nil
}

// cmdConfig implements `airan config`:
//
//	airan config          show the config path and current default
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
	def, err := loadDefault(getenv)
	if err != nil {
		return err
	}
	if def == "" {
		def = "(none)"
	}
	fmt.Fprintf(out, "config:          %s\ndefault backend: %s\n", p, def)
	return nil
}

func configSet(name string, getenv func(string) string, out io.Writer) error {
	if _, ok := registry[name]; !ok {
		return fmt.Errorf("airan: unknown backend %q (known: %s)", name, strings.Join(knownBackends(), ", "))
	}
	p, err := saveDefault(getenv, name)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "default backend set to %q\nconfig: %s\n", name, p)
	return nil
}
