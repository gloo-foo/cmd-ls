package ls_test

import (
	gloo "github.com/gloo-foo/framework/patterns"
	"github.com/spf13/afero"

	command "github.com/gloo-foo/cmd-ls"
)

func ExampleLs_longFormat() {
	// ls -l .
	fs := afero.NewMemMapFs()
	_ = afero.WriteFile(fs, "a.txt", []byte("alpha"), 0o644)
	_ = afero.WriteFile(fs, "b.txt", []byte("beta four bytes"), 0o644)

	gloo.MustRun(
		command.Ls(".", command.LsFs{Fs: fs}, command.LsLongFormat),
	)
	// Output:
	// -rw-r--r-- 5 a.txt
	// -rw-r--r-- 15 b.txt
}
