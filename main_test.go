package main

import (
	"context"
	"errors"
	"testing"

	clix "github.com/gloo-foo/cli"
	"github.com/spf13/afero"
	urf "github.com/urfave/cli/v3"
)

// parse runs args through a bare command carrying fresh wrapper flags and
// returns the parsed accessor, so flag-dependent helpers are tested against real
// parsed flags without leaking IsSet state between cases.
func parse(t *testing.T, args ...string) *urf.Command {
	t.Helper()
	var got *urf.Command
	app := &urf.Command{
		Name:   name,
		Flags:  flags(),
		Action: func(_ context.Context, c *urf.Command) error { got = c; return nil },
	}
	if err := app.Run(context.Background(), args); err != nil {
		t.Fatalf("parse: %v", err)
	}
	return got
}

func TestOperands(t *testing.T) {
	cases := []struct {
		name       string
		wantSuffix nameSuffix
		args       []string
		wantNames  int
	}{
		{"single", "", []string{name, "/a/b.txt"}, 1},
		{"single-suffix", ".txt", []string{name, "/a/b.txt", ".txt"}, 1},
		{"multiple", "", []string{name, "-a", "/a/b", "/c/d"}, 2},
		{"suffix-implies-multiple", ".go", []string{name, "-s", ".go", "a.go", "b.go"}, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			names, suffix, err := operands(parse(t, tc.args...))
			if err != nil {
				t.Fatalf("operands err=%v", err)
			}
			if len(names) != tc.wantNames || suffix != tc.wantSuffix {
				t.Fatalf("names=%v suffix=%q, want %d names / %q", names, suffix, tc.wantNames, tc.wantSuffix)
			}
		})
	}
}

func TestOperandsErrors(t *testing.T) {
	cases := []struct {
		want error
		name string
		args []string
	}{
		{ErrMissingOperand, "single-none", []string{name}},
		{ErrExtraOperand, "single-extra", []string{name, "a", "b", "c"}},
		{ErrMissingOperand, "multiple-none", []string{name, "-a"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := operands(parse(t, tc.args...)); !errors.Is(err, tc.want) {
				t.Fatalf("err=%v, want %v", err, tc.want)
			}
		})
	}
}

func TestSuffixOptions(t *testing.T) {
	if got := len(suffixOptions("")); got != 0 {
		t.Fatalf("suffixOptions(\"\") len=%d, want 0", got)
	}
	if got := len(suffixOptions(".txt")); got != 1 {
		t.Fatalf("suffixOptions(.txt) len=%d, want 1", got)
	}
}

func TestErrorMessage(t *testing.T) {
	if ErrMissingOperand.Error() != string(ErrMissingOperand) {
		t.Fatalf("Error()=%q, want %q", ErrMissingOperand.Error(), string(ErrMissingOperand))
	}
}

func TestBuild(t *testing.T) {
	src, filter, err := build(clix.Invocation{Args: parse(t, name, "/a/b.txt", ".txt"), Fs: afero.NewMemMapFs()})
	if err != nil || src == nil || filter == nil {
		t.Fatalf("build: src=%v filter=%v err=%v", src, filter, err)
	}
}

func TestBuildError(t *testing.T) {
	src, filter, err := build(clix.Invocation{Args: parse(t, name), Fs: afero.NewMemMapFs()})
	if !errors.Is(err, ErrMissingOperand) {
		t.Fatalf("err=%v, want ErrMissingOperand", err)
	}
	if src != nil || filter != nil {
		t.Fatalf("src=%v filter=%v, want both nil on error", src, filter)
	}
}

func Test_main(t *testing.T) {
	orig := runMain
	t.Cleanup(func() { runMain = orig })
	var gotName clix.Name
	runMain = func(s clix.Spec, _ clix.Version) { gotName = s.Name }
	main()
	if gotName != name {
		t.Fatalf("main used spec %q, want %s", gotName, name)
	}
}
