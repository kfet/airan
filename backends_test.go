package airan

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// writeConfig writes raw config-file content under a fresh XDG dir and
// returns a getenv pointed at it plus the resolved config path.
func writeConfig(t *testing.T, content string) (func(string) string, string) {
	t.Helper()
	dir := t.TempDir()
	cdir := filepath.Join(dir, "airan")
	if err := os.MkdirAll(cdir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(cdir, "config")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return envFrom(map[string]string{"XDG_CONFIG_HOME": dir}), p
}

func TestParseConfigLine(t *testing.T) {
	cases := []struct {
		line     string
		key, val string
		ok       bool
	}{
		{"backend: claude", "backend", "claude", true},
		{"backend.x: mycli -p {{prompt}}", "backend.x", "mycli -p {{prompt}}", true},
		{"  backend: fir  # comment", "backend", "fir", true},
		{"# just a comment", "", "", false},
		{"", "", "", false},
		{"   ", "", "", false},
		{"no colon here", "", "", false},
		{": no key", "", "", false},
		{"key:", "key", "", true},
	}
	for _, tc := range cases {
		k, v, ok := parseConfigLine(tc.line)
		if k != tc.key || v != tc.val || ok != tc.ok {
			t.Errorf("parseConfigLine(%q) = (%q,%q,%v), want (%q,%q,%v)", tc.line, k, v, ok, tc.key, tc.val, tc.ok)
		}
	}
}

func TestParseAdapter(t *testing.T) {
	if _, ok := parseAdapter("   "); ok {
		t.Error("empty value should not parse")
	}
	ad, ok := parseAdapter("mycli --flag {{prompt}}")
	if !ok || ad.cmd != "mycli" || !reflect.DeepEqual(ad.args, []string{"--flag", "{{prompt}}"}) {
		t.Errorf("got (%+v, %v)", ad, ok)
	}
}

func TestLoadConfig_CustomBackends(t *testing.T) {
	getenv, _ := writeConfig(t, "# airan config\nbackend: fir\nbackend.mycli: mycli -p {{prompt}}\nbackend.bad:\n")
	cfg, err := loadConfig(getenv)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.defaultBackend != "fir" {
		t.Errorf("default = %q", cfg.defaultBackend)
	}
	ad, ok := cfg.backends["mycli"]
	if !ok || ad.cmd != "mycli" {
		t.Errorf("mycli backend = (%+v, %v)", ad, ok)
	}
	if _, ok := cfg.backends["bad"]; ok {
		t.Error("empty-value backend should be skipped")
	}
}

func TestLoadConfig_NoDir(t *testing.T) {
	cfg, err := loadConfig(noEnv)
	if err != nil || cfg.defaultBackend != "" || len(cfg.backends) != 0 {
		t.Errorf("got (%+v, %v)", cfg, err)
	}
}

func TestResolve_CustomBackend(t *testing.T) {
	getenv, _ := writeConfig(t, "backend.mycli: mycli --go {{prompt}}\n")
	content := "---\nbackend: mycli\n---\nbody\n"
	spec, err := Resolve(writeTemp(t, content), getenv)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if spec.Cmd != "mycli" {
		t.Errorf("Cmd = %q, want mycli", spec.Cmd)
	}
	if !reflect.DeepEqual(spec.Args, []string{"--go", content}) {
		t.Errorf("Args = %#v", spec.Args)
	}
}

func TestResolve_CustomOverridesBuiltin(t *testing.T) {
	getenv, _ := writeConfig(t, "backend.claude: my-claude {{prompt}}\n")
	spec, err := Resolve(writeTemp(t, "---\nbackend: claude\n---\n"), getenv)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Cmd != "my-claude" {
		t.Errorf("Cmd = %q, want my-claude (custom overrides built-in)", spec.Cmd)
	}
}

func TestBackendNames_Merged(t *testing.T) {
	cfg := config{backends: map[string]adapter{"zeta": {}, "claude": {}}}
	got := backendNames(cfg)
	want := []string{"aider", "claude", "fir", "zeta"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("backendNames = %#v, want %#v", got, want)
	}
}

func TestBackendsList_Availability(t *testing.T) {
	getenv, _ := writeConfig(t, "backend: fir\nbackend.mycli: mycli {{prompt}}\n")
	onlyFir := func(cmd string) (string, error) {
		if cmd == "fir" {
			return "/usr/bin/fir", nil
		}
		return "", errors.New("nope")
	}
	var buf bytes.Buffer
	if err := backendsList(getenv, onlyFir, &buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"fir", "available", "missing", "mycli", "(custom)", "(default)"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestBackendsList_NilLook(t *testing.T) {
	getenv, _ := writeConfig(t, "backend: fir\n")
	var buf bytes.Buffer
	if err := backendsList(getenv, nil, &buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "missing") {
		t.Errorf("nil look should report missing:\n%s", buf.String())
	}
}

func TestBackendsList_LoadError(t *testing.T) {
	getenv, dir := tempXDG(t, "")
	if err := os.MkdirAll(filepath.Join(dir, "airan", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := backendsList(getenv, okLook, &bytes.Buffer{}); err == nil {
		t.Error("want load error")
	}
}

func TestBackendsAdd(t *testing.T) {
	t.Run("roundtrip", func(t *testing.T) {
		getenv, _ := tempXDG(t, "")
		var buf bytes.Buffer
		if err := backendsAdd([]string{"mycli", "mycli", "-p", promptToken}, getenv, &buf); err != nil {
			t.Fatal(err)
		}
		cfg, _ := loadConfig(getenv)
		ad, ok := cfg.backends["mycli"]
		if !ok || ad.cmd != "mycli" || !reflect.DeepEqual(ad.args, []string{"-p", promptToken}) {
			t.Errorf("stored = (%+v, %v)", ad, ok)
		}
		if !strings.Contains(buf.String(), "added") {
			t.Errorf("missing confirmation:\n%s", buf.String())
		}
	})
	t.Run("too few args", func(t *testing.T) {
		if err := backendsAdd([]string{"x"}, noEnv, &bytes.Buffer{}); !errors.Is(err, ErrUsage) {
			t.Errorf("err = %v, want ErrUsage", err)
		}
	})
	t.Run("invalid name", func(t *testing.T) {
		err := backendsAdd([]string{"bad.name", "cli", promptToken}, noEnv, &bytes.Buffer{})
		if err == nil || errors.Is(err, ErrUsage) {
			t.Errorf("err = %v, want invalid-name error", err)
		}
	})
	t.Run("missing prompt token", func(t *testing.T) {
		err := backendsAdd([]string{"x", "cli", "--no-token"}, noEnv, &bytes.Buffer{})
		if err == nil || errors.Is(err, ErrUsage) {
			t.Errorf("err = %v, want missing-placeholder error", err)
		}
	})
	t.Run("save error", func(t *testing.T) {
		dir := t.TempDir()
		blocker := filepath.Join(dir, "blocker")
		if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		getenv := envFrom(map[string]string{"XDG_CONFIG_HOME": blocker})
		if err := backendsAdd([]string{"x", "cli", promptToken}, getenv, &bytes.Buffer{}); err == nil {
			t.Error("want save error")
		}
	})
}

func TestBackendsRemove(t *testing.T) {
	t.Run("roundtrip", func(t *testing.T) {
		getenv, _ := writeConfig(t, "backend: fir\nbackend.mycli: mycli {{prompt}}\n")
		var buf bytes.Buffer
		if err := backendsRemove([]string{"mycli"}, getenv, &buf); err != nil {
			t.Fatal(err)
		}
		cfg, _ := loadConfig(getenv)
		if _, ok := cfg.backends["mycli"]; ok {
			t.Error("backend not removed")
		}
		if cfg.defaultBackend != "fir" {
			t.Errorf("default backend clobbered: %q", cfg.defaultBackend)
		}
		if !strings.Contains(buf.String(), "removed") {
			t.Errorf("missing confirmation:\n%s", buf.String())
		}
	})
	t.Run("wrong arg count", func(t *testing.T) {
		if err := backendsRemove(nil, noEnv, &bytes.Buffer{}); !errors.Is(err, ErrUsage) {
			t.Errorf("err = %v, want ErrUsage", err)
		}
	})
	t.Run("not found", func(t *testing.T) {
		getenv, _ := tempXDG(t, "")
		err := backendsRemove([]string{"nope"}, getenv, &bytes.Buffer{})
		if err == nil || errors.Is(err, ErrUsage) {
			t.Errorf("err = %v, want not-found error", err)
		}
	})
	t.Run("delete error", func(t *testing.T) {
		if err := backendsRemove([]string{"x"}, noEnv, &bytes.Buffer{}); err == nil {
			t.Error("want config-path error")
		}
	})
}

func TestCmdBackends_Dispatch(t *testing.T) {
	t.Run("add", func(t *testing.T) {
		getenv, _ := tempXDG(t, "")
		if err := cmdBackends([]string{"add", "x", "cli", promptToken}, getenv, noLook, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("remove unknown verb", func(t *testing.T) {
		if err := cmdBackends([]string{"frobnicate"}, noEnv, noLook, &bytes.Buffer{}); !errors.Is(err, ErrUsage) {
			t.Errorf("err = %v, want ErrUsage", err)
		}
	})
	t.Run("remove", func(t *testing.T) {
		getenv, _ := writeConfig(t, "backend.x: cli {{prompt}}\n")
		if err := cmdBackends([]string{"remove", "x"}, getenv, noLook, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
	})
}

func TestConfigSet_CustomBackend(t *testing.T) {
	getenv, _ := writeConfig(t, "backend.mycli: mycli {{prompt}}\n")
	if err := configSet("mycli", getenv, &bytes.Buffer{}); err != nil {
		t.Fatalf("configSet custom: %v", err)
	}
	if v, _ := loadDefault(getenv); v != "mycli" {
		t.Errorf("default = %q, want mycli", v)
	}
}

func TestConfigSet_LoadError(t *testing.T) {
	getenv, dir := tempXDG(t, "")
	if err := os.MkdirAll(filepath.Join(dir, "airan", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := configSet("fir", getenv, &bytes.Buffer{}); err == nil {
		t.Error("want load error")
	}
}

func TestConfigSet_SaveError(t *testing.T) {
	getenv, p := writeConfig(t, "backend: aider\n")
	if err := os.Chmod(p, 0o444); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(p, 0o644)
	if err := configSet("fir", getenv, &bytes.Buffer{}); err == nil {
		t.Error("want save error")
	}
}

func TestSaveConfigKey_WriteError(t *testing.T) {
	getenv, p := writeConfig(t, "backend: fir\n")
	if err := os.Chmod(p, 0o444); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(p, 0o644)
	if _, err := saveConfigKey(getenv, "backend", "aider"); err == nil {
		t.Error("want write error")
	}
}

func TestReadConfigLines_Empty(t *testing.T) {
	getenv, p := writeConfig(t, "\n\n")
	lines, err := readConfigLines(p)
	if err != nil || lines != nil {
		t.Errorf("got (%#v, %v), want (nil, nil)", lines, err)
	}
	_ = getenv
}

func TestConfigShow_CustomBackends(t *testing.T) {
	getenv, _ := writeConfig(t, "backend: fir\nbackend.mycli: mycli -p {{prompt}}\n")
	var buf bytes.Buffer
	if err := configShow(getenv, &buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "custom backends:") || !strings.Contains(out, "mycli: mycli -p {{prompt}}") {
		t.Errorf("missing custom backend listing:\n%s", out)
	}
}

func TestConfigShow_LoadError(t *testing.T) {
	getenv, dir := tempXDG(t, "")
	if err := os.MkdirAll(filepath.Join(dir, "airan", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := configShow(getenv, &bytes.Buffer{}); err == nil {
		t.Error("want load error")
	}
}

func TestSaveConfigKey_PreservesComments(t *testing.T) {
	getenv, p := writeConfig(t, "# my header\nbackend: fir\nbackend.x: cli {{prompt}}\n")
	if _, err := saveConfigKey(getenv, "backend", "aider"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(p)
	s := string(data)
	if !strings.Contains(s, "# my header") || !strings.Contains(s, "backend: aider") || !strings.Contains(s, "backend.x: cli") {
		t.Errorf("comments/lines not preserved:\n%s", s)
	}
}

func TestSaveConfigKey_ReadError(t *testing.T) {
	getenv, dir := tempXDG(t, "")
	if err := os.MkdirAll(filepath.Join(dir, "airan", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := saveConfigKey(getenv, "backend", "fir"); err == nil {
		t.Error("want read error")
	}
}

func TestDeleteConfigKey(t *testing.T) {
	t.Run("not present leaves file", func(t *testing.T) {
		getenv, _ := writeConfig(t, "backend: fir\n")
		_, removed, err := deleteConfigKey(getenv, "backend.absent")
		if err != nil || removed {
			t.Errorf("got (removed=%v, %v)", removed, err)
		}
	})
	t.Run("path error", func(t *testing.T) {
		if _, _, err := deleteConfigKey(noEnv, "backend.x"); err == nil {
			t.Error("want config-path error")
		}
	})
	t.Run("read error", func(t *testing.T) {
		getenv, dir := tempXDG(t, "")
		if err := os.MkdirAll(filepath.Join(dir, "airan", "config"), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, _, err := deleteConfigKey(getenv, "backend.x"); err == nil {
			t.Error("want read error")
		}
	})
	t.Run("write error", func(t *testing.T) {
		getenv, p := writeConfig(t, "backend.x: cli {{prompt}}\n")
		if err := os.Chmod(p, 0o444); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(p, 0o644)
		if _, _, err := deleteConfigKey(getenv, "backend.x"); err == nil {
			t.Error("want write error")
		}
	})
}

func TestRun_BackendsAdd(t *testing.T) {
	getenv, _ := tempXDG(t, "")
	exec := func(string, []string, []string) error {
		t.Fatal("exec must not run for subcommand")
		return nil
	}
	if err := Run([]string{"backends", "add", "x", "cli", promptToken}, getenv, nil, &bytes.Buffer{}, exec, noLook); err != nil {
		t.Fatal(err)
	}
	cfg, _ := loadConfig(getenv)
	if _, ok := cfg.backends["x"]; !ok {
		t.Error("backend not added via Run")
	}
}
