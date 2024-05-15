package command

import (
	"bytes"
	"testing"

	. "github.com/arikkfir/justest"
)

func Test_initializeFlagSet(t *testing.T) {
	type Flag struct {
		Name     string
		Usage    string
		Value    any
		DefValue string
	}
	type testCase struct {
		cmd             Command
		expectedFlags   []Flag
		expectedFailure *string
	}
	testCases := map[string]testCase{
		"nil Config": {cmd: Command{Config: nil}},
		"Config is a pointer to a struct": {
			cmd: Command{Config: &RootConfig{S0: "v0"}},
			expectedFlags: []Flag{
				{Name: "s0", DefValue: "v0", Usage: "String field", Value: nil},
				{Name: "b0", DefValue: "false", Usage: "Bool field", Value: nil},
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			tc := tc
			t.Run(name, func(t *testing.T) {
				if tc.expectedFailure != nil {
					defer func() { With(t).Verify(recover()).Will(Say(*tc.expectedFailure)).OrFail() }()
				}
				With(t).Verify(tc.cmd.initializeFlagSet()).Will(Succeed()).OrFail()
				for _, expectedFlag := range tc.expectedFlags {
					actualFlag := tc.cmd.flagSet.Lookup(expectedFlag.Name)
					With(t).Verify(actualFlag).Will(Not(BeNil())).OrFail()
					With(t).Verify(actualFlag.Usage).Will(EqualTo(expectedFlag.Usage)).OrFail()
					With(t).Verify(actualFlag.DefValue).Will(EqualTo(expectedFlag.DefValue)).OrFail()
				}
			})
		})
	}
	t.Run("Config must be a pointer to a struct", func(t *testing.T) {
		With(t).
			Verify((&Command{Config: RootConfig{S0: "v0"}}).initializeFlagSet()).
			Will(Fail(`must not be a struct, but a pointer to a struct`)).
			OrFail()
		With(t).
			Verify((&Command{Config: 1}).initializeFlagSet()).
			Will(Fail(`is not a pointer: 1`)).
			OrFail()
		With(t).
			Verify((&Command{Config: &[]int{123}[0]}).initializeFlagSet()).
			Will(Fail(`is not a pointer to struct`)).
			OrFail()
	})
}

func Test_printCommandUsage(t *testing.T) {
	t.Parallel()

	rootCmd := New(nil, Spec{
		Name:             "root",
		ShortDescription: "Root command",
		LongDescription:  "This command is the\nroot command.",
		Config:           &RootConfig{},
	})
	sub1Cmd := New(rootCmd, Spec{
		Name:             "sub1",
		ShortDescription: "Sub command 1",
		LongDescription:  "This command is the\nfirst sub command.",
		Config:           &Sub1Config{},
	})
	sub2Cmd := New(rootCmd, Spec{
		Name:             "sub2",
		ShortDescription: "Sub command 2",
		LongDescription:  "This command is the\nsecond sub command.",
		Config:           &Sub2Config{},
	})
	sub3Cmd := New(sub2Cmd, Spec{
		Name:             "sub3",
		ShortDescription: "Sub command 3",
		LongDescription:  "This command is the\nthird sub command.",
		Config:           &Sub3Config{},
	})

	type testCase struct {
		cmd           *Command
		expectedUsage string
	}

	testCases := map[string]testCase{
		rootCmd.Name: {
			cmd: rootCmd,
			expectedUsage: `
root: Root command

This command is the
root command.

Usage:
	root [--b0] [--help] --s0=VAL

Flags:
	--b0        Bool field (default is false)
	--help      Show help about how to use this command (default is false)
	--s0        String field

Available sub-commands:
	sub1      Sub command 1
	sub2      Sub command 2

`,
		},
		sub1Cmd.Name: {
			cmd: sub1Cmd,
			expectedUsage: `
root sub1: Sub command 1

This command is the
first sub command.

Usage:
	root sub1 [--b0] [--b1] [--help] --s0=VAL --s1=VALUE

Flags:
	--b0        Bool field (default is false)
	--b1        Bool field (default is false)
	--help      Show help about how to use this command (default is false)
	--s0        String field
	--s1        String field

`,
		},
		sub2Cmd.Name: {
			cmd: sub2Cmd,
			expectedUsage: `
root sub2: Sub command 2

This command is the
second sub command.

Usage:
	root sub2 [--b0] [--b1] [--b2] [--help] --s0=VAL --s1=VALUE [--s2=VALUE]

Flags:
	--b0        Bool field (default is false)
	--b1        Bool field (default is false)
	--b2        Bool field (default is false)
	--help      Show help about how to use this command (default is false)
	--s0        String field
	--s1        String field
	--s2        String field

Available sub-commands:
	sub3      Sub command 3

`,
		},
		sub3Cmd.Name: {
			cmd: sub3Cmd,
			expectedUsage: `
root sub2 sub3: Sub command 3

This command is the
third sub command.

Usage:
	root sub2 sub3 [--b0] [--b1] [--b2] [--b3] [--help] --s0=VAL --s1=VALUE [--s2=VALUE] [--s3=VALUE] [ARGS]

Flags:
	--b0        Bool field (default is false)
	--b1        Bool field (default is false)
	--b2        Bool field (default is false)
	--b3        Bool field (default is false)
	--help      Show help about how to use this command (default is false)
	--s0        String field
	--s1        String field
	--s2        String field
	--s3        String field

`,
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			usageBuf := &bytes.Buffer{}

			With(t).Verify(tc.cmd.initializeFlagSet()).Will(Succeed()).OrFail()
			tc.cmd.printCommandUsage(usageBuf, false)
			With(t).Verify(usageBuf.String()).Will(EqualTo(tc.expectedUsage[1:])).OrFail()
		})
	}
}
