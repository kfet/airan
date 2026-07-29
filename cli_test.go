package airan

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestVersionIsPopulated(t *testing.T) {
	if Version == "" {
		t.Fatal("Version is empty; the embedded VERSION file should populate it")
	}
	if strings.ContainsAny(Version, " \n\t") {
		t.Fatalf("Version %q should be trimmed", Version)
	}
}

func TestCmdVersion(t *testing.T) {
	var buf bytes.Buffer
	if err := cmdVersion(&buf); err != nil {
		t.Fatalf("cmdVersion: %v", err)
	}
	want := "airan " + Version + "\n"
	if buf.String() != want {
		t.Fatalf("got %q, want %q", buf.String(), want)
	}
}

func TestCmdHelpMentionsEveryCommand(t *testing.T) {
	var buf bytes.Buffer
	if err := cmdHelp(&buf); err != nil {
		t.Fatalf("cmdHelp: %v", err)
	}
	for _, want := range []string{"backends", "config", "help", "version", "--prepend"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("help output does not mention %q", want)
		}
	}
}

// Run must treat help/version as reserved words on every spelling, and
// must not try to open them as files.
func TestRunHelpAndVersionSpellings(t *testing.T) {
	for _, arg := range []string{"help", "-h", "--help", "version", "-V", "--version"} {
		var buf bytes.Buffer
		err := Run([]string{arg}, func(string) string { return "" }, nil, &buf,
			func(string, []string, []string) error {
				t.Fatalf("%s must not exec a backend", arg)
				return nil
			}, nil)
		if err != nil {
			t.Errorf("Run(%q): unexpected error %v", arg, err)
		}
		if buf.Len() == 0 {
			t.Errorf("Run(%q): produced no output", arg)
		}
	}
}

func TestParseDispatchArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    options
		wantErr bool
	}{
		{"bare file", []string{"f.md"}, options{file: "f.md"}, false},
		{"prepend space form", []string{"--prepend", "GO", "f.md"},
			options{file: "f.md", prepend: []string{"GO"}}, false},
		{"prepend equals form", []string{"--prepend=GO", "f.md"},
			options{file: "f.md", prepend: []string{"GO"}}, false},
		{"prepend after file", []string{"f.md", "--prepend", "GO"},
			options{file: "f.md", prepend: []string{"GO"}}, false},
		{"repeated prepend keeps order", []string{"--prepend", "A", "--prepend", "B", "f.md"},
			options{file: "f.md", prepend: []string{"A", "B"}}, false},
		{"double dash literal path", []string{"--", "--weird.md"},
			options{file: "--weird.md"}, false},
		{"no file", []string{"--prepend", "GO"}, options{}, true},
		{"missing prepend value", []string{"f.md", "--prepend"}, options{}, true},
		{"two files", []string{"a.md", "b.md"}, options{}, true},
		{"unknown flag", []string{"--nope", "f.md"}, options{}, true},
		{"empty", nil, options{}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseDispatchArgs(tc.args)
			if tc.wantErr {
				if !errors.Is(err, ErrUsage) {
					t.Fatalf("want ErrUsage, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.file != tc.want.file {
				t.Errorf("file: got %q want %q", got.file, tc.want.file)
			}
			if strings.Join(got.prepend, "|") != strings.Join(tc.want.prepend, "|") {
				t.Errorf("prepend: got %v want %v", got.prepend, tc.want.prepend)
			}
		})
	}
}

// The critical property: inserting after the frontmatter fence keeps the
// composed prompt a well-formed frontmatter document, so backend
// resolution is unchanged.
func TestApplyPrependKeepsFrontmatterIntact(t *testing.T) {
	src := "---\nbackend: claude\n---\nbody line\n"
	got := applyPrepend(src, []string{"DO THE THING"})

	if !strings.HasPrefix(got, "---\nbackend: claude\n---\n") {
		t.Fatalf("frontmatter block was disturbed:\n%q", got)
	}
	if got := frontmatterBackend(got); got != "claude" {
		t.Errorf("backend resolution broke after prepend: got %q", got)
	}
	if !strings.Contains(got, "DO THE THING") {
		t.Error("prepended text missing")
	}
	if strings.Index(got, "DO THE THING") > strings.Index(got, "body line") {
		t.Error("prepended text must come before the body")
	}
}

func TestApplyPrependWithShebang(t *testing.T) {
	src := "#!/usr/bin/env airan\n---\nbackend: fir\n---\nbody\n"
	got := applyPrepend(src, []string{"GO"})

	if !strings.HasPrefix(got, "#!/usr/bin/env airan\n") {
		t.Errorf("shebang must stay on line 1:\n%q", got)
	}
	if b := frontmatterBackend(got); b != "fir" {
		t.Errorf("backend resolution broke: got %q", b)
	}
}

func TestApplyPrependWithoutFrontmatter(t *testing.T) {
	got := applyPrepend("just a body\n", []string{"GO"})
	if !strings.HasPrefix(got, "GO\n") {
		t.Errorf("without frontmatter the text should lead: %q", got)
	}
	if !strings.Contains(got, "just a body") {
		t.Error("original body lost")
	}
}

func TestApplyPrependNoBlocksIsIdentity(t *testing.T) {
	src := "---\nbackend: fir\n---\nbody\n"
	if got := applyPrepend(src, nil); got != src {
		t.Errorf("no blocks should be identity, got %q", got)
	}
}

func TestApplyPrependJoinsBlocksInOrder(t *testing.T) {
	got := applyPrepend("---\nbackend: fir\n---\nbody\n", []string{"FIRST", "SECOND"})
	if strings.Index(got, "FIRST") > strings.Index(got, "SECOND") {
		t.Error("blocks must appear in the order given")
	}
}

// End-to-end: the prompt handed to the backend must contain the
// prepended text.
func TestRunPrependReachesTheBackend(t *testing.T) {
	dir := t.TempDir()
	file := dir + "/skill.md"
	if err := os.WriteFile(file, []byte("---\nbackend: claude\n---\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var gotArgs []string
	err := Run([]string{"--prepend", "APPLY THIS", file},
		func(string) string { return "" }, nil, &bytes.Buffer{},
		func(cmd string, args []string, _ []string) error {
			gotArgs = args
			return nil
		}, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	joined := strings.Join(gotArgs, " ")
	if !strings.Contains(joined, "APPLY THIS") {
		t.Errorf("prepended text never reached the backend: %q", joined)
	}
	if !strings.Contains(joined, "body") {
		t.Errorf("file body never reached the backend: %q", joined)
	}
}

// --- coverage for the remaining error / edge branches ---------------------

// Run must surface a read error rather than pretending the file was empty.
func TestRunMissingFile(t *testing.T) {
	err := Run([]string{t.TempDir() + "/nope.md"},
		func(string) string { return "" }, nil, &bytes.Buffer{},
		func(string, []string, []string) error {
			t.Fatal("must not exec a backend when the file cannot be read")
			return nil
		}, nil)
	if err == nil {
		t.Fatal("want an error for a missing file, got nil")
	}
}

func TestParseDispatchArgsDoubleDashErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"nothing after --", []string{"--"}},
		{"more than one path after --", []string{"--", "a.md", "b.md"}},
		{"file already seen", []string{"a.md", "--", "b.md"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseDispatchArgs(tc.args); !errors.Is(err, ErrUsage) {
				t.Fatalf("want ErrUsage, got %v", err)
			}
		})
	}
}

// Blank lines may precede the opening fence.
func TestApplyPrependSkipsLeadingBlankLines(t *testing.T) {
	src := "\n\n---\nbackend: fir\n---\nbody\n"
	got := applyPrepend(src, []string{"GO"})

	if b := frontmatterBackend(got); b != "fir" {
		t.Errorf("backend resolution broke: got %q", b)
	}
	if strings.Index(got, "GO") < strings.Index(got, "backend: fir") {
		t.Errorf("text must land after the frontmatter block:\n%q", got)
	}
}

// An unterminated frontmatter block is not a frontmatter block, so the
// text simply leads.
func TestApplyPrependUnterminatedFrontmatter(t *testing.T) {
	src := "---\nbackend: fir\nnever closed\n"
	got := applyPrepend(src, []string{"GO"})

	if !strings.HasPrefix(got, "GO\n") {
		t.Errorf("with no closing fence the text should lead:\n%q", got)
	}
	if !strings.Contains(got, "never closed") {
		t.Error("original content lost")
	}
}
