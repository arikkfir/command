package command

import (
	"bytes"
	"context"
	"regexp"
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
		cmd             *Command
		expectedFlags   []Flag
		expectedFailure *string
	}
	testCases := map[string]testCase{
		"nil Config": {cmd: New(nil, Spec{Config: nil})},
		"Config is a pointer to a struct": {
			cmd: New(nil, Spec{Config: &RootConfig{S0: "v0"}}),
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
			Verify(New(nil, Spec{Config: RootConfig{S0: "v0"}}).initializeFlagSet()).
			Will(Fail(`must not be a struct, but a pointer to a struct`)).
			OrFail()
		With(t).
			Verify(New(nil, Spec{Config: 1}).initializeFlagSet()).
			Will(Fail(`is not a pointer: 1`)).
			OrFail()
		With(t).
			Verify(New(nil, Spec{Config: &[]int{123}[0]}).initializeFlagSet()).
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

func TestExecute(t *testing.T) {
	t.Parallel()

	rootSpec := Spec{Name: "root", ShortDescription: "Root!", LongDescription: "The root command.", Config: &RootConfig{}}
	sub1Spec := Spec{Name: "sub1", ShortDescription: "Sub 1", LongDescription: "The first sub command.", Config: &Sub1Config{}}
	sub2Spec := Spec{Name: "sub2", ShortDescription: "Sub 2", LongDescription: "The second sub command.", Config: &Sub2Config{}}
	sub3Spec := Spec{Name: "sub3", ShortDescription: "Sub 3", LongDescription: "The third sub command.", Config: &Sub3Config{}}

	type testCase struct {
		args                []string
		envVars             []string
		expectedExitCode    int
		expectedPreRunCalls map[string]bool
		expectedRunCalls    map[string]bool
		expectedOutput      string
	}
	testCases := map[string]testCase{
		"": {
			args:                nil,
			envVars:             nil,
			expectedExitCode:    0,
			expectedPreRunCalls: map[string]bool{"root": true},
			expectedRunCalls:    map[string]bool{"root": true},
		},
		"sub1": {
			args:                []string{"sub1"},
			envVars:             nil,
			expectedExitCode:    0,
			expectedPreRunCalls: map[string]bool{"root": true, "sub1": true},
			expectedRunCalls:    map[string]bool{"sub1": true},
		},
		"sub2 sub3": {
			args:                []string{"sub2", "sub3"},
			envVars:             nil,
			expectedExitCode:    0,
			expectedPreRunCalls: map[string]bool{"root": true, "sub2": true, "sub3": true},
			expectedRunCalls:    map[string]bool{"sub3": true},
		},
		"sub2 sub3 --help": {
			args:                []string{"sub2", "sub3", "--help"},
			envVars:             nil,
			expectedExitCode:    0,
			expectedPreRunCalls: map[string]bool{"root": true, "sub2": true, "sub3": true},
			expectedRunCalls:    map[string]bool{},
			expectedOutput: `root sub2 sub3: Sub 3

The third sub command.

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

			preRunCalls := make(map[string]bool)
			runCalls := make(map[string]bool)

			rootSpec := rootSpec
			rootSpec.OnSubCommandRun = func(ctx context.Context, config any, usagePrinter UsagePrinter) error {
				preRunCalls["root"] = true
				return nil
			}
			rootSpec.Run = func(ctx context.Context, config any, usagePrinter UsagePrinter) error {
				runCalls["root"] = true
				return nil
			}
			rootCmd := New(nil, rootSpec)

			sub1Spec := sub1Spec
			sub1Spec.OnSubCommandRun = func(ctx context.Context, config any, usagePrinter UsagePrinter) error {
				preRunCalls["sub1"] = true
				return nil
			}
			sub1Spec.Run = func(ctx context.Context, config any, usagePrinter UsagePrinter) error {
				runCalls["sub1"] = true
				return nil
			}
			sub1Cmd := New(rootCmd, sub1Spec)

			sub2Spec := sub2Spec
			sub2Spec.OnSubCommandRun = func(ctx context.Context, config any, usagePrinter UsagePrinter) error {
				preRunCalls["sub2"] = true
				return nil
			}
			sub2Spec.Run = func(ctx context.Context, config any, usagePrinter UsagePrinter) error {
				runCalls["sub2"] = true
				return nil
			}
			sub2Cmd := New(rootCmd, sub2Spec)

			sub3Spec := sub3Spec
			sub3Spec.OnSubCommandRun = func(ctx context.Context, config any, usagePrinter UsagePrinter) error {
				preRunCalls["sub3"] = true
				return nil
			}
			sub3Spec.Run = func(ctx context.Context, config any, usagePrinter UsagePrinter) error {
				runCalls["sub3"] = true
				return nil
			}
			sub3Cmd := New(sub2Cmd, sub3Spec)
			_, _, _, _ = rootCmd, sub1Cmd, sub2Cmd, sub3Cmd

			b := &bytes.Buffer{}
			exitCode := Execute(context.Background(), b, rootCmd, tc.args, EnvVarsArrayToMap(tc.envVars))
			With(t).Verify(exitCode).Will(EqualTo(tc.expectedExitCode)).OrFail()
			With(t).Verify(preRunCalls).Will(EqualTo(tc.expectedPreRunCalls)).OrFail()
			With(t).Verify(runCalls).Will(EqualTo(tc.expectedRunCalls)).OrFail()
			if tc.expectedOutput != "" {
				With(t).Verify(b).Will(Say(`^` + regexp.QuoteMeta(tc.expectedOutput) + `$`)).OrFail()
			}
		})
	}
}
