package airan

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tempXDG returns a getenv with XDG_CONFIG_HOME pointed at a fresh temp
// dir. If backend != "", a config file presetting that default is
// written. It also returns the resolved config dir.
func tempXDG(t *testing.T, backend string) (func(string) string, string) {
	t.Helper()
	dir := t.TempDir()
	if backend != "" {
		cdir := filepath.Join(dir, "airan")
		if err := os.MkdirAll(cdir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cdir, "config"), []byte("backend: "+backend+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return envFrom(map[string]string{"XDG_CONFIG_HOME": dir}), dir
}

func TestConfigPath(t *testing.T) {
	t.Run("xdg absolute", func(t *testing.T) {
		getenv := envFrom(map[string]string{"XDG_CONFIG_HOME": "/x/cfg"})
		p, err := configPath(getenv)
		if err != nil || p != "/x/cfg/airan/config" {
			t.Errorf("got (%q, %v)", p, err)
		}
	})
	t.Run("xdg relative ignored, HOME fallback", func(t *testing.T) {
		getenv := envFrom(map[string]string{"XDG_CONFIG_HOME": "rel/cfg", "HOME": "/home/u"})
		p, err := configPath(getenv)
		if err != nil || p != "/home/u/.config/airan/config" {
			t.Errorf("got (%q, %v)", p, err)
		}
	})
	t.Run("HOME fallback when xdg empty", func(t *testing.T) {
		getenv := envFrom(map[string]string{"HOME": "/home/u"})
		p, err := configPath(getenv)
		if err != nil || p != "/home/u/.config/airan/config" {
			t.Errorf("got (%q, %v)", p, err)
		}
	})
	t.Run("no config dir", func(t *testing.T) {
		if _, err := configPath(noEnv); !errors.Is(err, errNoConfigDir) {
			t.Errorf("err = %v, want errNoConfigDir", err)
		}
	})
}

func TestLoadDefault(t *testing.T) {
	t.Run("no config dir -> empty, no error", func(t *testing.T) {
		v, err := loadDefault(noEnv)
		if v != "" || err != nil {
			t.Errorf("got (%q, %v)", v, err)
		}
	})
	t.Run("missing file -> empty, no error", func(t *testing.T) {
		getenv, _ := tempXDG(t, "")
		v, err := loadDefault(getenv)
		if v != "" || err != nil {
			t.Errorf("got (%q, %v)", v, err)
		}
	})
	t.Run("present file", func(t *testing.T) {
		getenv, _ := tempXDG(t, "claude")
		v, err := loadDefault(getenv)
		if v != "claude" || err != nil {
			t.Errorf("got (%q, %v)", v, err)
		}
	})
	t.Run("read error (config is a dir)", func(t *testing.T) {
		getenv, dir := tempXDG(t, "")
		if err := os.MkdirAll(filepath.Join(dir, "airan", "config"), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := loadDefault(getenv); err == nil {
			t.Error("want read error")
		}
	})
}

func TestSaveDefault(t *testing.T) {
	t.Run("roundtrip", func(t *testing.T) {
		getenv, dir := tempXDG(t, "")
		p, err := saveDefault(getenv, "fir")
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(dir, "airan", "config"); p != want {
			t.Errorf("path = %q, want %q", p, want)
		}
		got, err := loadDefault(getenv)
		if err != nil || got != "fir" {
			t.Errorf("readback = (%q, %v)", got, err)
		}
	})
	t.Run("no config dir", func(t *testing.T) {
		if _, err := saveDefault(noEnv, "fir"); !errors.Is(err, errNoConfigDir) {
			t.Errorf("err = %v, want errNoConfigDir", err)
		}
	})
	t.Run("mkdir error (xdg under a file)", func(t *testing.T) {
		dir := t.TempDir()
		blocker := filepath.Join(dir, "blocker")
		if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		getenv := envFrom(map[string]string{"XDG_CONFIG_HOME": blocker})
		if _, err := saveDefault(getenv, "fir"); err == nil {
			t.Error("want mkdir error")
		}
	})
	t.Run("write error (target is a dir)", func(t *testing.T) {
		getenv, dir := tempXDG(t, "")
		if err := os.MkdirAll(filepath.Join(dir, "airan", "config"), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := saveDefault(getenv, "fir"); err == nil {
			t.Error("want write error")
		}
	})
}

func TestResolve_ConfigDefault(t *testing.T) {
	getenv, _ := tempXDG(t, "aider")
	spec, err := Resolve(writeTemp(t, "prompt body, no frontmatter\n"), getenv)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if spec.Cmd != "aider" {
		t.Errorf("Cmd = %q, want aider (from config default)", spec.Cmd)
	}
}

func TestResolve_ConfigLoadErrorPropagates(t *testing.T) {
	getenv, dir := tempXDG(t, "")
	if err := os.MkdirAll(filepath.Join(dir, "airan", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := Resolve(writeTemp(t, "no frontmatter\n"), getenv)
	if err == nil || errors.Is(err, ErrNoBackend) {
		t.Errorf("err = %v, want a config read error", err)
	}
}

func TestCmdBackends(t *testing.T) {
	t.Run("marks default", func(t *testing.T) {
		getenv, _ := tempXDG(t, "fir")
		var buf bytes.Buffer
		if err := cmdBackends(nil, getenv, &buf); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		if !strings.Contains(out, "fir  (default)") {
			t.Errorf("output missing default marker:\n%s", out)
		}
		if !strings.Contains(out, "aider\n") || !strings.Contains(out, "claude\n") {
			t.Errorf("output missing backends:\n%s", out)
		}
	})
	t.Run("extra args -> usage", func(t *testing.T) {
		if err := cmdBackends([]string{"x"}, noEnv, &bytes.Buffer{}); !errors.Is(err, ErrUsage) {
			t.Errorf("err = %v, want ErrUsage", err)
		}
	})
}

func TestCmdConfig(t *testing.T) {
	t.Run("show none", func(t *testing.T) {
		getenv, _ := tempXDG(t, "")
		var buf bytes.Buffer
		if err := cmdConfig(nil, getenv, &buf); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), "(none)") {
			t.Errorf("want (none):\n%s", buf.String())
		}
	})
	t.Run("show with default", func(t *testing.T) {
		getenv, _ := tempXDG(t, "claude")
		var buf bytes.Buffer
		if err := cmdConfig(nil, getenv, &buf); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), "claude") {
			t.Errorf("want claude:\n%s", buf.String())
		}
	})
	t.Run("show config-path error", func(t *testing.T) {
		if err := cmdConfig(nil, noEnv, &bytes.Buffer{}); !errors.Is(err, errNoConfigDir) {
			t.Errorf("err = %v, want errNoConfigDir", err)
		}
	})
	t.Run("show load error", func(t *testing.T) {
		getenv, dir := tempXDG(t, "")
		if err := os.MkdirAll(filepath.Join(dir, "airan", "config"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := cmdConfig(nil, getenv, &bytes.Buffer{}); err == nil {
			t.Error("want load error")
		}
	})
	t.Run("set valid", func(t *testing.T) {
		getenv, _ := tempXDG(t, "")
		var buf bytes.Buffer
		if err := cmdConfig([]string{"fir"}, getenv, &buf); err != nil {
			t.Fatal(err)
		}
		if v, _ := loadDefault(getenv); v != "fir" {
			t.Errorf("default = %q, want fir", v)
		}
		if !strings.Contains(buf.String(), "fir") {
			t.Errorf("want confirmation:\n%s", buf.String())
		}
	})
	t.Run("set unknown backend", func(t *testing.T) {
		getenv, _ := tempXDG(t, "")
		err := cmdConfig([]string{"nope"}, getenv, &bytes.Buffer{})
		if err == nil || errors.Is(err, ErrUsage) {
			t.Errorf("err = %v, want unknown-backend error", err)
		}
	})
	t.Run("set save error", func(t *testing.T) {
		dir := t.TempDir()
		blocker := filepath.Join(dir, "blocker")
		if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		getenv := envFrom(map[string]string{"XDG_CONFIG_HOME": blocker})
		if err := cmdConfig([]string{"fir"}, getenv, &bytes.Buffer{}); err == nil {
			t.Error("want save error")
		}
	})
	t.Run("too many args -> usage", func(t *testing.T) {
		if err := cmdConfig([]string{"a", "b"}, noEnv, &bytes.Buffer{}); !errors.Is(err, ErrUsage) {
			t.Errorf("err = %v, want ErrUsage", err)
		}
	})
}

func TestRun_Subcommands(t *testing.T) {
	exec := func(string, []string, []string) error {
		t.Fatal("exec must not be called for a subcommand")
		return nil
	}
	t.Run("backends", func(t *testing.T) {
		getenv, _ := tempXDG(t, "")
		var buf bytes.Buffer
		if err := Run([]string{"backends"}, getenv, nil, &buf, exec); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), "claude") {
			t.Errorf("no backends listed:\n%s", buf.String())
		}
	})
	t.Run("config set then show", func(t *testing.T) {
		getenv, _ := tempXDG(t, "")
		if err := Run([]string{"config", "aider"}, getenv, nil, &bytes.Buffer{}, exec); err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		if err := Run([]string{"config"}, getenv, nil, &buf, exec); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), "aider") {
			t.Errorf("config show missing default:\n%s", buf.String())
		}
	})
}

func TestConfigBackend(t *testing.T) {
	cases := map[string]string{
		"# airan config\nbackend: claude\n": "claude",
		"backend: fir  # comment\n":         "fir",
		"nothing here\n":                    "",
		"":                                  "",
	}
	for content, want := range cases {
		if got := configBackend(content); got != want {
			t.Errorf("configBackend(%q) = %q, want %q", content, got, want)
		}
	}
}

func TestConfigSet_PathReported(t *testing.T) {
	getenv, dir := tempXDG(t, "")
	var buf bytes.Buffer
	if err := configSet("claude", getenv, &buf); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "airan", "config")
	if !strings.Contains(buf.String(), want) {
		t.Errorf("output should report path %q:\n%s", want, buf.String())
	}
}
