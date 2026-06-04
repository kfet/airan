package airan

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// writeTemp writes content to a fresh temp file and returns its path.
func writeTemp(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "agent")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	return p
}

// noEnv is a getenv that always returns "".
func noEnv(string) string { return "" }

// noLook is a LookFunc that reports every command as missing.
func noLook(string) (string, error) { return "", errors.New("not found") }

// okLook is a LookFunc that reports every command as available.
func okLook(cmd string) (string, error) { return "/usr/bin/" + cmd, nil }

// envFrom builds a getenv backed by a map.
func envFrom(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestResolve_FrontmatterBackend(t *testing.T) {
	content := "#!/usr/bin/env airan\n---\nbackend: claude\n---\nDo the thing.\n"
	spec, err := Resolve(writeTemp(t, content), noEnv)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if spec.Cmd != "claude" {
		t.Errorf("Cmd = %q, want claude", spec.Cmd)
	}
	want := []string{"-p", content}
	if !reflect.DeepEqual(spec.Args, want) {
		t.Errorf("Args = %#v, want %#v", spec.Args, want)
	}
}

func TestResolve_EnvBackendFallback(t *testing.T) {
	content := "just a prompt, no frontmatter\n"
	getenv := func(k string) string {
		if k == envBackend {
			return "aider"
		}
		return ""
	}
	spec, err := Resolve(writeTemp(t, content), getenv)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if spec.Cmd != "aider" {
		t.Errorf("Cmd = %q, want aider", spec.Cmd)
	}
	want := []string{"--message", content}
	if !reflect.DeepEqual(spec.Args, want) {
		t.Errorf("Args = %#v, want %#v", spec.Args, want)
	}
}

func TestResolve_FrontmatterWinsOverEnv(t *testing.T) {
	content := "---\nbackend: fir\n---\nbody\n"
	getenv := func(string) string { return "claude" }
	spec, err := Resolve(writeTemp(t, content), getenv)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if spec.Cmd != "fir" {
		t.Errorf("Cmd = %q, want fir (frontmatter wins)", spec.Cmd)
	}
}

func TestResolve_NoBackend(t *testing.T) {
	_, err := Resolve(writeTemp(t, "no frontmatter here\n"), noEnv)
	if !errors.Is(err, ErrNoBackend) {
		t.Errorf("err = %v, want ErrNoBackend", err)
	}
}

func TestResolve_UnknownBackend(t *testing.T) {
	_, err := Resolve(writeTemp(t, "---\nbackend: nope\n---\n"), noEnv)
	if err == nil {
		t.Fatal("want error for unknown backend")
	}
	if errors.Is(err, ErrNoBackend) {
		t.Errorf("err = %v, want unknown-backend error not ErrNoBackend", err)
	}
	// Error should list known backends (sorted).
	if got := err.Error(); got == "" || !contains(got, "aider, claude, fir") {
		t.Errorf("err = %q, want it to list known backends", got)
	}
}

func TestResolve_ReadError(t *testing.T) {
	_, err := Resolve(filepath.Join(t.TempDir(), "does-not-exist"), noEnv)
	if err == nil {
		t.Fatal("want error reading missing file")
	}
	if errors.Is(err, ErrNoBackend) {
		t.Errorf("err = %v, want a read error", err)
	}
}

func TestRun_Success(t *testing.T) {
	content := "---\nbackend: claude\n---\nhello\n"
	path := writeTemp(t, content)

	var gotCmd string
	var gotArgs, gotEnv []string
	exec := func(cmd string, args []string, environ []string) error {
		gotCmd, gotArgs, gotEnv = cmd, args, environ
		return nil
	}

	env := []string{"FOO=bar"}
	if err := Run([]string{path}, noEnv, env, io.Discard, exec, noLook); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if gotCmd != "claude" {
		t.Errorf("cmd = %q, want claude", gotCmd)
	}
	if !reflect.DeepEqual(gotArgs, []string{"-p", content}) {
		t.Errorf("args = %#v", gotArgs)
	}
	if !reflect.DeepEqual(gotEnv, env) {
		t.Errorf("env = %#v, want %#v", gotEnv, env)
	}
}

func TestRun_UsageError(t *testing.T) {
	exec := func(string, []string, []string) error {
		t.Fatal("exec should not be called on usage error")
		return nil
	}
	for _, args := range [][]string{{}, {"a", "b"}} {
		if err := Run(args, noEnv, nil, io.Discard, exec, noLook); !errors.Is(err, ErrUsage) {
			t.Errorf("Run(%v) err = %v, want ErrUsage", args, err)
		}
	}
}

func TestRun_ResolveErrorPropagates(t *testing.T) {
	exec := func(string, []string, []string) error {
		t.Fatal("exec should not be called when resolve fails")
		return nil
	}
	path := writeTemp(t, "no backend\n")
	if err := Run([]string{path}, noEnv, nil, io.Discard, exec, noLook); !errors.Is(err, ErrNoBackend) {
		t.Errorf("err = %v, want ErrNoBackend", err)
	}
}

func TestRun_ExecErrorPropagates(t *testing.T) {
	want := errors.New("boom")
	exec := func(string, []string, []string) error { return want }
	path := writeTemp(t, "---\nbackend: fir\n---\n")
	if err := Run([]string{path}, noEnv, nil, io.Discard, exec, noLook); !errors.Is(err, want) {
		t.Errorf("err = %v, want %v", err, want)
	}
}

func TestFrontmatterBackend(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{"shebang then frontmatter", "#!/usr/bin/env airan\n---\nbackend: claude\n---\nx", "claude"},
		{"leading blank lines", "\n\n---\nbackend: fir\n---\n", "fir"},
		{"inline comment stripped", "---\nbackend: aider   # use aider\n---\n", "aider"},
		{"other keys before backend", "---\nmodel: opus\nbackend: claude\n---\n", "claude"},
		{"no frontmatter fence", "just a prompt\nbackend: claude\n", ""},
		{"empty file", "", ""},
		{"only shebang", "#!/usr/bin/env airan\n", ""},
		{"closing fence before backend", "---\nmodel: opus\n---\nbackend: claude\n", ""},
		{"unclosed frontmatter no backend", "---\nmodel: opus\nmode: print\n", ""},
		{"empty backend value", "---\nbackend:\n---\n", ""},
		{"comment-only value", "---\nbackend:   # nothing\n---\n", ""},
		{"not a backend line", "---\nbackends: claude\n---\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := frontmatterBackend(tc.content); got != tc.want {
				t.Errorf("frontmatterBackend = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestKnownBackends_Sorted(t *testing.T) {
	got := knownBackends()
	want := []string{"aider", "claude", "fir"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("knownBackends = %#v, want %#v", got, want)
	}
}

// contains reports whether s contains sub (avoids importing strings just
// for one call in a test assertion path).
func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
