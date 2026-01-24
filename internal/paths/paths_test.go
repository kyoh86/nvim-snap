package paths

import (
	"path/filepath"
	"testing"
)

func TestResolveCasesRoot(t *testing.T) {
	root := t.TempDir()
	got := ResolveCasesRoot(root, "")
	want := filepath.Join(root, "snapcase")
	if got != want {
		t.Fatalf("ResolveCasesRoot default = %q, want %q", got, want)
	}

	abs := filepath.Join(root, "cases")
	got = ResolveCasesRoot(root, abs)
	if got != abs {
		t.Fatalf("ResolveCasesRoot abs = %q, want %q", got, abs)
	}

	got = ResolveCasesRoot(root, "cases")
	want = filepath.Join(root, "cases")
	if got != want {
		t.Fatalf("ResolveCasesRoot rel = %q, want %q", got, want)
	}
}

func TestResolveResultsRoot(t *testing.T) {
	root := t.TempDir()
	got := ResolveResultsRoot(root, "")
	want := filepath.Join(root, "snapcase", ".result")
	if got != want {
		t.Fatalf("ResolveResultsRoot = %q, want %q", got, want)
	}
}
