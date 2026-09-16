package command

// lsAllFlag enables listing entries whose names start with "." (-a).
type lsAllFlag bool

const (
	LsAll   lsAllFlag = true
	LsNoAll lsAllFlag = false
)

// lsRecursiveFlag walks subdirectories, listing their entries too (-R).
type lsRecursiveFlag bool

const (
	LsRecursive   lsRecursiveFlag = true
	LsNoRecursive lsRecursiveFlag = false
)

// lsLongFormatFlag toggles long-format output (-l): one line per entry,
// "<perm> <size> <name>". Owner/mtime are intentionally omitted because
// afero does not surface either uniformly across backends.
type lsLongFormatFlag bool

const (
	LsLongFormat   lsLongFormatFlag = true
	LsNoLongFormat lsLongFormatFlag = false
)

// flags is the option set folded from an Ls call's option values.
type flags struct {
	fs           LsFs
	isAll        lsAllFlag
	isRecursive  lsRecursiveFlag
	isLongFormat lsLongFormatFlag
}

// with folds one option value into the flag set. Values of any other type are
// ignored: Ls takes no positional arguments beyond its path.
func (f flags) with(o any) flags {
	switch v := o.(type) {
	case LsFs:
		f.fs = v
	case lsAllFlag:
		f.isAll = v
	case lsRecursiveFlag:
		f.isRecursive = v
	case lsLongFormatFlag:
		f.isLongFormat = v
	}
	return f
}

// fold collapses the Ls option values into the flag set.
func fold(opts []any) flags {
	var f flags
	for _, o := range opts {
		f = f.with(o)
	}
	return f
}
