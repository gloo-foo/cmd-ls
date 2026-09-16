package command

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	gloo "github.com/gloo-foo/framework"
	"github.com/spf13/afero"
)

// LsFs injects the filesystem Ls reads from. Tests pass an in-memory fs
// (afero.NewMemMapFs()); production callers omit it and get the OS filesystem.
type LsFs struct{ afero.Fs }

// value returns the configured filesystem, defaulting to the OS filesystem.
func (f LsFs) value() afero.Fs {
	if f.Fs == nil {
		return afero.NewOsFs()
	}
	return f.Fs
}

// Ls returns a Source that lists the entries of a directory, one name per item.
// Entries are emitted in the lexical order afero.ReadDir guarantees. By default,
// entries whose names start with "." are hidden.
//
// Options:
//   - LsAll (-a): also list entries whose names start with ".".
//   - LsRecursive (-R): walk subdirectories, emitting paths relative to the root.
//   - LsLongFormat (-l): emit "<perm> <size> <name>" per entry.
//   - LsFs: read from a custom afero.Fs (defaults to the OS filesystem).
func Ls(path gloo.File, opts ...any) gloo.Source[[]byte] {
	f := fold(opts)
	return lsSource{
		fs:    f.fs.value(),
		path:  string(path),
		flags: f,
	}
}

// lsSource lists a directory on its filesystem. It is an immutable value that
// satisfies gloo.Source, carrying the configured listing options.
type lsSource struct {
	fs    afero.Fs
	path  string
	flags flags
}

func (s lsSource) Stream(ctx context.Context) gloo.Stream[[]byte] {
	return gloo.Generate(ctx, func(_ context.Context, send func([]byte) bool, sendErr func(error)) {
		s.list(send, sendErr)
	})
}

// list dispatches to the recursive or flat walk based on the -R flag.
func (s lsSource) list(send sendFunc, sendErr errFunc) {
	if bool(s.flags.isRecursive) {
		s.walkRecursive(send, sendErr)
		return
	}
	s.listFlat(send, sendErr)
}

// sendFunc emits one rendered entry; it reports false when the consumer stopped.
type sendFunc func([]byte) bool

// errFunc reports a terminal listing error to the stream.
type errFunc func(error)

func (s lsSource) listFlat(send sendFunc, sendErr errFunc) {
	entries, err := afero.ReadDir(s.fs, s.path)
	if err != nil {
		sendErr(err)
		return
	}
	for _, e := range entries {
		if s.isHidden(e) {
			continue
		}
		if stopped := !send(s.format(e.Name(), e)); stopped {
			return
		}
	}
}

func (s lsSource) walkRecursive(send sendFunc, sendErr errFunc) {
	sendErr(afero.Walk(s.fs, s.path, s.visit(send)))
}

// visit returns the afero.Walk callback. It skips hidden entries (unless -a),
// the root itself, and emits every other entry by its path relative to the root.
func (s lsSource) visit(send sendFunc) filepath.WalkFunc {
	return func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if s.isHidden(info) {
			return skipHidden(info)
		}
		if rel := s.relativize(path); rel != "" {
			send(s.format(rel, info))
		}
		return nil
	}
}

// isHidden reports whether info is a hidden entry that the current flags exclude.
// With -a nothing is hidden.
func (s lsSource) isHidden(info fs.FileInfo) bool {
	return !bool(s.flags.isAll) && strings.HasPrefix(info.Name(), ".")
}

// skipHidden tells Walk to prune a hidden directory's whole subtree, or to drop
// a single hidden file (nil) without aborting the walk.
func skipHidden(info fs.FileInfo) error {
	if info.IsDir() {
		return filepath.SkipDir
	}
	return nil
}

// relativize renders a walk path relative to the listing root. afero.Walk only
// ever yields the root itself or paths beneath it (root joined with child
// names), so trimming the root prefix and its separator is exact — and, unlike
// filepath.Rel, total: it has no error case to leave uncovered. The root itself
// trims to "" and is dropped by the caller.
func (s lsSource) relativize(path string) string {
	rel := strings.TrimPrefix(path, s.path)
	return strings.TrimPrefix(rel, string(filepath.Separator))
}

// format renders one entry, long form when -l is set, else just the name.
func (s lsSource) format(name string, info fs.FileInfo) []byte {
	if !bool(s.flags.isLongFormat) {
		return []byte(name)
	}
	return fmt.Appendf(nil, "%s %d %s", info.Mode().String(), info.Size(), name)
}
