package command_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/spf13/afero"

	command "github.com/gloo-foo/cmd-ls"
	gloo "github.com/gloo-foo/framework"
	"github.com/gloo-foo/framework/patterns"
)

// collect lists /dir on an in-memory fixture and returns the emitted lines in
// the exact order ls produced them. ls relies on afero.ReadDir/Walk, both of
// which sort by name, so the order is deterministic and asserted verbatim — no
// test-side re-sorting that would mask a sort regression.
func collect(t *testing.T, fs afero.Fs, path string, opts ...any) []string {
	t.Helper()
	all := append([]any{command.LsFs{Fs: fs}}, opts...)
	items, err := gloo.Collect(context.Background(), command.Ls(path, all...).Stream(context.Background()))
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	out := make([]string, len(items))
	for i, b := range items {
		out[i] = string(b)
	}
	return out
}

func assertLines(t *testing.T, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// seed writes the given path→contents pairs into a fresh in-memory fs.
func seed(t *testing.T, files map[string]string) afero.Fs {
	t.Helper()
	fs := afero.NewMemMapFs()
	for path, body := range files {
		if err := afero.WriteFile(fs, path, []byte(body), 0o644); err != nil {
			t.Fatalf("seed %s: %v", path, err)
		}
	}
	return fs
}

func TestLs_DefaultListsVisibleSortedByName(t *testing.T) {
	fs := seed(t, map[string]string{
		"/dir/bravo.txt": "",
		"/dir/alpha.txt": "",
		"/dir/.hidden":   "",
	})
	// Lexical order, hidden entry omitted by default.
	assertLines(t, collect(t, fs, "/dir"), []string{"alpha.txt", "bravo.txt"})
}

func TestLs_EmptyDir(t *testing.T) {
	fs := afero.NewMemMapFs()
	if err := fs.MkdirAll("/empty", 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	assertLines(t, collect(t, fs, "/empty"), []string{})
}

func TestLs_AllShowsHidden(t *testing.T) {
	fs := seed(t, map[string]string{
		"/dir/.hidden":     "",
		"/dir/visible.txt": "",
	})
	// "." sorts before letters, so .hidden leads.
	assertLines(t, collect(t, fs, "/dir", command.LsAll),
		[]string{".hidden", "visible.txt"})
}

func TestLs_NoAllMatchesDefault(t *testing.T) {
	fs := seed(t, map[string]string{
		"/dir/.hidden":     "",
		"/dir/visible.txt": "",
	})
	assertLines(t, collect(t, fs, "/dir", command.LsNoAll), []string{"visible.txt"})
}

func TestLs_RecursiveEmitsRelativePaths(t *testing.T) {
	fs := seed(t, map[string]string{
		"/dir/a.txt":     "",
		"/dir/sub/b.txt": "",
	})
	// Walk yields the root (dropped), sub, then sub/b.txt — depth-first, sorted.
	assertLines(t, collect(t, fs, "/dir", command.LsRecursive),
		[]string{"a.txt", "sub", "sub/b.txt"})
}

func TestLs_RecursiveHidesHiddenSubtree(t *testing.T) {
	fs := seed(t, map[string]string{
		"/dir/a.txt":              "",
		"/dir/.hidden/secret.txt": "",
	})
	// The hidden directory's whole subtree is pruned.
	assertLines(t, collect(t, fs, "/dir", command.LsRecursive), []string{"a.txt"})
}

func TestLs_RecursiveAllIncludesHiddenSubtree(t *testing.T) {
	fs := seed(t, map[string]string{
		"/dir/a.txt":              "",
		"/dir/.hidden/secret.txt": "",
	})
	assertLines(t, collect(t, fs, "/dir", command.LsRecursive, command.LsAll),
		[]string{".hidden", ".hidden/secret.txt", "a.txt"})
}

func TestLs_LongFormatRendersPermSizeName(t *testing.T) {
	fs := seed(t, map[string]string{"/dir/a.txt": "hello"})
	assertLines(t, collect(t, fs, "/dir", command.LsLongFormat),
		[]string{"-rw-r--r-- 5 a.txt"})
}

func TestLs_NoLongFormatMatchesDefault(t *testing.T) {
	fs := seed(t, map[string]string{"/dir/a.txt": "hello"})
	assertLines(t, collect(t, fs, "/dir", command.LsNoLongFormat), []string{"a.txt"})
}

func TestLs_ReadDirErrorPropagates(t *testing.T) {
	fs := afero.NewMemMapFs()
	_, err := gloo.Collect(context.Background(),
		command.Ls("/missing", command.LsFs{Fs: fs}).Stream(context.Background()))
	if err == nil {
		t.Fatal("expected error listing a nonexistent directory")
	}
}

func TestLs_RecursiveWalkErrorPropagates(t *testing.T) {
	fs := afero.NewMemMapFs()
	// afero.Walk first Lstat's the root; a missing root surfaces as a walk error.
	_, err := gloo.Collect(context.Background(),
		command.Ls("/missing", command.LsFs{Fs: fs}, command.LsRecursive).Stream(context.Background()))
	if err == nil {
		t.Fatal("expected error walking a nonexistent directory")
	}
}

func TestLs_StopsWhenConsumerStops(t *testing.T) {
	// A directory larger than the stream buffer (64) makes the source block on a
	// send past the buffer; Take(1) tears it down, exercising the "consumer
	// stopped" branch where send reports false and listing halts early.
	fs := afero.NewMemMapFs()
	for i := range 200 {
		if err := afero.WriteFile(fs, fmt.Sprintf("/dir/f%03d.txt", i), []byte(""), 0o644); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	out := gloo.From(context.Background(),
		command.Ls("/dir", command.LsFs{Fs: fs}), patterns.Take[[]byte](1))
	items, err := gloo.Collect(context.Background(), out)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	assertLines(t, []string{string(items[0])}, []string{"f000.txt"})
}

func TestLs_RecursiveStopsWhenConsumerStops(t *testing.T) {
	fs := afero.NewMemMapFs()
	for i := range 200 {
		if err := afero.WriteFile(fs, fmt.Sprintf("/dir/f%03d.txt", i), []byte(""), 0o644); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	out := gloo.From(context.Background(),
		command.Ls("/dir", command.LsFs{Fs: fs}, command.LsRecursive), patterns.Take[[]byte](1))
	items, err := gloo.Collect(context.Background(), out)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
}

func TestLs_DefaultFilesystemIsOS(t *testing.T) {
	// With no LsFs injected, Ls must read the real OS filesystem. Listing this
	// package's own source directory proves the OS default is wired without
	// depending on any particular file's contents.
	items, err := gloo.Collect(context.Background(), command.Ls(".").Stream(context.Background()))
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	names := make([]string, len(items))
	for i, b := range items {
		names[i] = string(b)
	}
	if !slices.Contains(names, "command.go") {
		t.Fatalf("OS listing of . missing command.go: %v", names)
	}
}
