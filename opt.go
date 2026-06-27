package command

// lsAllFlag enables listing entries whose names start with "." (-a).
type lsAllFlag bool

const (
	LsAll   lsAllFlag = true
	LsNoAll lsAllFlag = false
)

func (f lsAllFlag) Configure(flags *flags) { flags.all = f }

// lsRecursiveFlag walks subdirectories, listing their entries too (-R).
type lsRecursiveFlag bool

const (
	LsRecursive   lsRecursiveFlag = true
	LsNoRecursive lsRecursiveFlag = false
)

func (f lsRecursiveFlag) Configure(flags *flags) { flags.recursive = f }

// lsLongFormatFlag toggles long-format output (-l): one line per entry,
// "<perm> <size> <name>". Owner/mtime are intentionally omitted because
// afero does not surface either uniformly across backends.
type lsLongFormatFlag bool

const (
	LsLongFormat   lsLongFormatFlag = true
	LsNoLongFormat lsLongFormatFlag = false
)

func (f lsLongFormatFlag) Configure(flags *flags) { flags.longFormat = f }

// flags is the configured option set for one Ls construction.
type flags struct {
	all        lsAllFlag
	recursive  lsRecursiveFlag
	longFormat lsLongFormatFlag
}
