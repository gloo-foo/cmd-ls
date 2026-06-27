package alias_test

import (
	"context"
	"slices"
	"testing"

	"github.com/spf13/afero"

	ls "github.com/gloo-foo/cmd-ls/alias"
	gloo "github.com/gloo-foo/framework"
)

// The alias package re-exports the constructor and flag constants under
// unprefixed names. A mis-wired re-export (say, All bound to the disabled
// constant, or LongFormat bound to NoLongFormat) compiles cleanly, so only
// behavior can prove the wiring. Each test exercises one re-export against a
// fixed in-memory directory and asserts the exact lines ls must produce.

// fixture builds a deterministic directory: two visible files, one hidden file,
// and a subdirectory holding one file. afero.ReadDir/Walk both sort by name, so
// the expected output order is fixed.
func fixture(t *testing.T) afero.Fs {
	t.Helper()
	fs := afero.NewMemMapFs()
	for path, body := range map[string]string{
		"/dir/alpha.txt":       "aa",
		"/dir/bravo.txt":       "bbbb",
		"/dir/.hidden":         "x",
		"/dir/sub/charlie.txt": "ccc",
	} {
		if err := afero.WriteFile(fs, path, []byte(body), 0o644); err != nil {
			t.Fatalf("seed %s: %v", path, err)
		}
	}
	return fs
}

func list(t *testing.T, opts ...any) []string {
	t.Helper()
	src := ls.Ls("/dir", opts...)
	items, err := gloo.Collect(context.Background(), src.Stream(context.Background()))
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	got := make([]string, len(items))
	for i, b := range items {
		got[i] = string(b)
	}
	return got
}

func assertLines(t *testing.T, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestAlias_DefaultHidesHiddenSortedByName(t *testing.T) {
	// No flags: hidden ".hidden" omitted, subdir name listed, lexical order.
	got := list(t, ls.Fs{Fs: fixture(t)})
	assertLines(t, got, []string{"alpha.txt", "bravo.txt", "sub"})
}

func TestAlias_AllShowsHiddenEntries(t *testing.T) {
	// -a includes ".hidden"; "." sorts before letters.
	got := list(t, ls.Fs{Fs: fixture(t)}, ls.All)
	assertLines(t, got, []string{".hidden", "alpha.txt", "bravo.txt", "sub"})
}

func TestAlias_NoAllMatchesDefault(t *testing.T) {
	// The disabled form must behave exactly like passing no flag.
	got := list(t, ls.Fs{Fs: fixture(t)}, ls.NoAll)
	assertLines(t, got, []string{"alpha.txt", "bravo.txt", "sub"})
}

func TestAlias_RecursiveWalksSubdirsHidingHidden(t *testing.T) {
	// -R descends into sub/, emitting paths relative to the root; ".hidden"
	// stays hidden.
	got := list(t, ls.Fs{Fs: fixture(t)}, ls.Recursive)
	assertLines(t, got, []string{"alpha.txt", "bravo.txt", "sub", "sub/charlie.txt"})
}

func TestAlias_NoRecursiveMatchesDefault(t *testing.T) {
	got := list(t, ls.Fs{Fs: fixture(t)}, ls.NoRecursive)
	assertLines(t, got, []string{"alpha.txt", "bravo.txt", "sub"})
}

func TestAlias_RecursiveAllShowsHidden(t *testing.T) {
	got := list(t, ls.Fs{Fs: fixture(t)}, ls.Recursive, ls.All)
	assertLines(t, got, []string{".hidden", "alpha.txt", "bravo.txt", "sub", "sub/charlie.txt"})
}

func TestAlias_LongFormatRendersPermSizeName(t *testing.T) {
	// -l prefixes mode and byte size. Use a flat fixture of regular files only:
	// file mode and size are stable across afero backends, unlike the synthetic
	// metadata MemMapFs assigns to implicitly-created parent directories.
	fs := afero.NewMemMapFs()
	if err := afero.WriteFile(fs, "/flat/alpha.txt", []byte("aa"), 0o644); err != nil {
		t.Fatalf("seed alpha: %v", err)
	}
	if err := afero.WriteFile(fs, "/flat/bravo.txt", []byte("bbbb"), 0o600); err != nil {
		t.Fatalf("seed bravo: %v", err)
	}
	src := ls.Ls("/flat", ls.Fs{Fs: fs}, ls.LongFormat)
	items, err := gloo.Collect(context.Background(), src.Stream(context.Background()))
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	got := make([]string, len(items))
	for i, b := range items {
		got[i] = string(b)
	}
	assertLines(t, got, []string{
		"-rw-r--r-- 2 alpha.txt",
		"-rw------- 4 bravo.txt",
	})
}

func TestAlias_NoLongFormatMatchesDefault(t *testing.T) {
	got := list(t, ls.Fs{Fs: fixture(t)}, ls.NoLongFormat)
	assertLines(t, got, []string{"alpha.txt", "bravo.txt", "sub"})
}
