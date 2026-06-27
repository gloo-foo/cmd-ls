// Package alias provides unprefixed type aliases for ls command flags.
//
//	import ls "github.com/gloo-foo/cmd-ls/alias"
//	ls.Ls("/dir", ls.All, ls.LongFormat)
package alias

import command "github.com/gloo-foo/cmd-ls"

// Ls re-exports the constructor.
var Ls = command.Ls

// Fs re-exports the filesystem injector (LsFs).
type Fs = command.LsFs

// -a flag: show hidden files (entries starting with ".")
const All = command.LsAll

// default: hide hidden files
const NoAll = command.LsNoAll

// -R flag: recursive listing
const Recursive = command.LsRecursive

// default: non-recursive listing
const NoRecursive = command.LsNoRecursive

// -l flag: long format ("<perm> <size> <name>" per entry)
const LongFormat = command.LsLongFormat

// default: names only
const NoLongFormat = command.LsNoLongFormat
