// Command yup-basename is the CLI wrapper around
// github.com/gloo-foo/cmd-basename.
package main

import (
	"strings"

	clix "github.com/gloo-foo/cli"
	command "github.com/gloo-foo/cmd-basename"
	urf "github.com/urfave/cli/v3"
)

// version is the build version. It defaults to "dev" for local builds and is
// overridden at release time via the linker: -ldflags "-X main.version=<v>".
var version = "dev"

const (
	name         = "basename"
	flagMultiple = "multiple"
	flagSuffix   = "suffix"
)

// Error is the sentinel error type the wrapper emits, so every failure path is
// comparable with errors.Is.
type Error string

func (e Error) Error() string { return string(e) }

const (
	// ErrMissingOperand is returned when no NAME operand is given.
	ErrMissingOperand Error = "missing operand"
	// ErrExtraOperand is returned when a third bare operand follows NAME and
	// SUFFIX without -a/-s (GNU basename rejects it).
	ErrExtraOperand Error = "extra operand"
)

// synopsis is the multi-line --help usage block. urfave/cli indents the whole
// block three spaces, so the lines stay flush-left.
const synopsis = `basename NAME [SUFFIX]
basename OPTION... NAME...

Print NAME with any leading directory components removed.
If specified, also remove a trailing SUFFIX.`

// spec declares the basename wrapper: each NAME operand is a literal path fed as
// an input line, stripped to its final component (with an optional suffix).
var spec = clix.Spec{
	Name:     name,
	Summary:  "strip directory and suffix from filenames",
	Synopsis: synopsis,
	Build:    build,
	Flags:    flags(),
}

// flags builds the wrapper's flag set. It is a constructor rather than a shared
// slice so each urfave/cli command owns distinct flag instances: urfave/cli
// records a flag's "was set" state on the flag value itself and never clears it,
// so reusing one slice across parses would leak IsSet state between them.
func flags() []urf.Flag {
	return []urf.Flag{
		&urf.BoolFlag{
			Name:    flagMultiple,
			Aliases: []string{"a"},
			Usage:   "support multiple NAME arguments, treating each as a NAME",
		},
		&urf.StringFlag{Name: flagSuffix, Aliases: []string{"s"}, Usage: "remove a trailing SUFFIX; implies -a"},
	}
}

// build maps the invocation to basename's pipeline: the NAME operands become the
// input lines, fed through the basename command with any suffix to strip. A
// missing or extra operand is a usage error.
func build(inv clix.Invocation) (clix.Source, clix.Command, error) {
	names, suffix, err := operands(inv.Args)
	if err != nil {
		return nil, nil, err
	}
	lines := strings.Join(names, "\n") + "\n"
	return clix.Stdin(strings.NewReader(lines)), command.Basename(suffixOptions(suffix)...), nil
}

// nameSuffix is the SUFFIX operand stripped from the end of each NAME.
type nameSuffix string

// operands resolves the NAME list and SUFFIX from the parsed flags and
// positional arguments, applying GNU basename's two operand grammars:
//
//   - -a/--multiple (or -s): every positional is a NAME; SUFFIX comes from -s.
//   - otherwise:             NAME [SUFFIX], i.e. exactly one or two positionals.
func operands(c *urf.Command) ([]string, nameSuffix, error) {
	if c.Bool(flagMultiple) || c.IsSet(flagSuffix) {
		return multipleOperands(c)
	}
	return singleOperands(c)
}

func multipleOperands(c *urf.Command) ([]string, nameSuffix, error) {
	if c.NArg() == 0 {
		return nil, "", ErrMissingOperand
	}
	return c.Args().Slice(), nameSuffix(c.String(flagSuffix)), nil
}

func singleOperands(c *urf.Command) ([]string, nameSuffix, error) {
	switch c.NArg() {
	case 1:
		return []string{c.Args().Get(0)}, "", nil
	case 2:
		return []string{c.Args().Get(0)}, nameSuffix(c.Args().Get(1)), nil
	case 0:
		return nil, "", ErrMissingOperand
	default:
		return nil, "", ErrExtraOperand
	}
}

// suffixOptions folds a non-empty SUFFIX into basename's option values.
func suffixOptions(suffix nameSuffix) []any {
	if suffix == "" {
		return nil
	}
	return []any{command.BasenameSuffix(string(suffix))}
}

// runMain is an indirection seam so main's wiring is testable without spawning
// the process; a test swaps it and restores it.
var runMain = clix.Main

func main() { runMain(spec, version) }
