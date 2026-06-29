package ls_test

import (
	gloo "github.com/gloo-foo/framework/patterns"
	"github.com/spf13/afero"

	command "github.com/gloo-foo/cmd-ls"
)

func ExampleLs_basic() {
	// ls .
	// Uses afero.NewMemMapFs for deterministic output.
	fs := afero.NewMemMapFs()
	_ = afero.WriteFile(fs, "a.txt", []byte("alpha"), 0o644)
	_ = afero.WriteFile(fs, "b.txt", []byte("beta"), 0o644)

	gloo.MustRun(
		command.Ls(".", command.LsFs{Fs: fs}),
	)
	// Output:
	// a.txt
	// b.txt
}
