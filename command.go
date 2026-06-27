package command

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"

	gloo "github.com/gloo-foo/framework"
)

// LsFs injects the filesystem Ls reads from. Tests pass an in-memory fs
// (afero.NewMemMapFs()); production callers omit it and get the OS filesystem.
type LsFs struct{ afero.Fs }

// Ls returns a Source that lists the entries of a directory, one name per item.
// Entries are emitted in the lexical order afero.ReadDir guarantees. By default,
// entries whose names start with "." are hidden.
//
// Options:
//   - LsAll (-a): also list entries whose names start with ".".
//   - LsRecursive (-R): walk subdirectories, emitting paths relative to the root.
//   - LsLongFormat (-l): emit "<perm> <size> <name>" per entry.
//   - LsFs: read from a custom afero.Fs (defaults to the OS filesystem).
func Ls(path string, opts ...any) gloo.Source[[]byte] {
	filesystem, switches := partition(opts)
	return &lsSource{
		fs:    filesystem,
		path:  path,
		flags: gloo.NewParameters[gloo.File, flags](switches...).Flags,
	}
}

// defaultFs is the filesystem Ls reads when no LsFs is injected. It is a
// function (not an inline literal) so the static interface type afero.Fs is
// explicit without a redundant conversion: afero.NewOsFs returns the concrete
// *afero.OsFs, but partition must hold the interface to also accept an injected
// fs.
func defaultFs() afero.Fs { return afero.NewOsFs() }

// partition splits opts into the injected filesystem (or the OS default) and the
// remaining flag switches handed to NewParameters.
func partition(opts []any) (afero.Fs, []any) {
	filesystem := defaultFs()
	rest := make([]any, 0, len(opts))
	for _, o := range opts {
		if injected, ok := o.(LsFs); ok {
			filesystem = injected.Fs
			continue
		}
		rest = append(rest, o)
	}
	return filesystem, rest
}

// lsSource lists a directory on its filesystem. Pointer receiver: it satisfies
// gloo.Source and carries the configured listing options as immutable state.
type lsSource struct {
	fs    afero.Fs
	path  string
	flags flags
}

func (s *lsSource) Stream(ctx context.Context) gloo.Stream[[]byte] {
	return gloo.Generate(ctx, func(_ context.Context, send func([]byte) bool, sendErr func(error)) {
		s.list(send, sendErr)
	})
}

// list dispatches to the recursive or flat walk based on the -R flag.
func (s *lsSource) list(send sendFunc, sendErr errFunc) {
	if bool(s.flags.recursive) {
		s.walkRecursive(send, sendErr)
		return
	}
	s.listFlat(send, sendErr)
}

// sendFunc emits one rendered entry; it reports false when the consumer stopped.
type sendFunc func([]byte) bool

// errFunc reports a terminal listing error to the stream.
type errFunc func(error)

func (s *lsSource) listFlat(send sendFunc, sendErr errFunc) {
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

func (s *lsSource) walkRecursive(send sendFunc, sendErr errFunc) {
	sendErr(afero.Walk(s.fs, s.path, s.visit(send)))
}

// visit returns the afero.Walk callback. It skips hidden entries (unless -a),
// the root itself, and emits every other entry by its path relative to the root.
func (s *lsSource) visit(send sendFunc) filepath.WalkFunc {
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
func (s *lsSource) isHidden(info fs.FileInfo) bool {
	return !bool(s.flags.all) && strings.HasPrefix(info.Name(), ".")
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
func (s *lsSource) relativize(path string) string {
	rel := strings.TrimPrefix(path, s.path)
	return strings.TrimPrefix(rel, string(filepath.Separator))
}

// format renders one entry, long form when -l is set, else just the name.
func (s *lsSource) format(name string, info fs.FileInfo) []byte {
	if !bool(s.flags.longFormat) {
		return []byte(name)
	}
	return fmt.Appendf(nil, "%s %d %s", info.Mode().String(), info.Size(), name)
}
