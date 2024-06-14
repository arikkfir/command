package command

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	. "github.com/arikkfir/justest"
	"github.com/go-loremipsum/loremipsum"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestNew(t *testing.T) {
	t.Parallel()
	type testCase struct {
		commandFactory           func(T, *testCase) (*Command, error)
		expectedName             string
		expectedShortDescription string
		expectedLongDescription  string
		expectedError            string
		expectedFlagSet          *flagSet
	}
	testCases := map[string]testCase{
		"empty name": {
			commandFactory: func(t T, tc *testCase) (*Command, error) {
				return New("", "short desc", "long desc", nil, nil, nil)
			},
			expectedError: `^invalid command: empty name$`,
		},
		"empty short description": {
			commandFactory: func(t T, tc *testCase) (*Command, error) {
				return New("cmd", "", "long desc", nil, nil, nil)
			},
			expectedError: `^invalid command: empty short description$`,
		},
		"no flags": {
			commandFactory: func(t T, tc *testCase) (*Command, error) {
				return New("cmd", "desc", "long desc", nil, nil, nil)
			},
			expectedName:             "cmd",
			expectedShortDescription: "desc",
			expectedLongDescription:  "long desc",
		},
		"with flags": {
			commandFactory: func(t T, tc *testCase) (*Command, error) {
				return New(
					"cmd",
					"desc",
					"long desc",
					&struct {
						Action
						MyFlag string `flag:"true"`
					}{},
					nil,
					nil,
				)
			},
			expectedFlagSet: &flagSet{
				flags: []*flagDef{
					{
						flagInfo: flagInfo{
							Name:     "my-flag",
							HasValue: true,
						},
						Targets: []reflect.Value{},
					},
				},
			},
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cmd, err := tc.commandFactory(t, &tc)
			if tc.expectedError != "" {
				With(t).Verify(err).Will(Fail(tc.expectedError)).OrFail()
			} else {
				With(t).Verify(err).Will(BeNil()).OrFail()
				if tc.expectedFlagSet != nil {
					With(t).
						Verify(cmd.flags.flags).
						Will(EqualTo(
							tc.expectedFlagSet.flags,
							cmpopts.IgnoreFields(flagDef{}, "Targets"),
							cmp.AllowUnexported(flagDef{})),
						).
						OrFail()
				}
			}
		})
	}
}

func TestAddSubCommand(t *testing.T) {
	t.Parallel()

	root, err := New("root", "desc", "description", nil, nil, nil)
	With(t).Verify(err).Will(BeNil()).OrFail()

	sub1, err := New("sub1", "sub1 desc", "sub1 description", nil, nil, nil)
	With(t).Verify(err).Will(BeNil()).OrFail()

	sub2, err := New("sub2", "sub2 desc", "sub2 description", nil, nil, nil)
	With(t).Verify(err).Will(BeNil()).OrFail()

	With(t).Verify(root.AddSubCommand(sub1)).Will(BeNil()).OrFail()
	With(t).Verify(root.AddSubCommand(sub2)).Will(BeNil()).OrFail()
	With(t).Verify(root.subCommands[0], root.subCommands[1]).Will(EqualTo(sub1, sub2, cmpopts.EquateComparable(&Command{}))).OrFail()
	With(t).Verify(sub1.parent).Will(EqualTo(root, cmpopts.EquateComparable(&Command{}))).OrFail()
	With(t).Verify(sub2.parent).Will(EqualTo(root, cmpopts.EquateComparable(&Command{}))).OrFail()
}

func Test_inferCommandAndArgs(t *testing.T) {
	type testCase struct {
		root                *Command
		args                []string
		expectedCommand     string
		expectedFlags       []string
		expectedPositionals []string
	}
	testCases := map[string]testCase{
		"No arguments": {
			root: MustNew(
				"root", "desc", "description", nil, nil, nil,
				MustNew("sub1", "sub1 desc", "sub1 description", nil, nil, nil,
					MustNew("sub2", "sub2 desc", "sub2 description", nil, nil, nil,
						MustNew("sub3", "sub3 desc", "sub3 description", nil, nil, nil),
					),
				),
			),
			args:                []string{},
			expectedCommand:     "root",
			expectedFlags:       nil,
			expectedPositionals: nil,
		},
		"Flags for root command": {
			root: MustNew(
				"root", "desc", "description", nil, nil, nil,
				MustNew("sub1", "sub1 desc", "sub1 description", nil, nil, nil,
					MustNew("sub2", "sub2 desc", "sub2 description", nil, nil, nil),
				),
			),
			args:                strings.Split("-f1 -f2", " "),
			expectedCommand:     "root",
			expectedFlags:       []string{"-f1", "-f2"},
			expectedPositionals: nil,
		},
		"Flags and positionals for root command": {
			root: MustNew(
				"root", "desc", "description", nil, nil, nil,
				MustNew("sub1", "sub1 desc", "sub1 description", nil, nil, nil,
					MustNew("sub2", "sub2 desc", "sub2 description", nil, nil, nil),
				),
			),
			args:                strings.Split("-f1 a -f2 b", " "),
			expectedCommand:     "root",
			expectedFlags:       []string{"-f1", "-f2"},
			expectedPositionals: []string{"a", "b"},
		},
		"Flags and positionals for sub1 command": {
			root: MustNew(
				"root", "desc", "description", nil, nil, nil,
				MustNew("sub1", "sub1 desc", "sub1 description", nil, nil, nil,
					MustNew("sub2", "sub2 desc", "sub2 description", nil, nil, nil),
				),
			),
			args:                strings.Split("-f1 sub1 -f2 a b", " "),
			expectedCommand:     "sub1",
			expectedFlags:       []string{"-f1", "-f2"},
			expectedPositionals: []string{"a", "b"},
		},
		"Flags and positionals for sub2 command": {
			root: MustNew(
				"root", "desc", "description", nil, nil, nil,
				MustNew("sub1", "sub1 desc", "sub1 description", nil, nil, nil,
					MustNew("sub2", "sub2 desc", "sub2 description", nil, nil, nil),
				),
			),
			args:                strings.Split("-f1 sub1 -f2 a b sub2 c", " "),
			expectedCommand:     "sub2",
			expectedFlags:       []string{"-f1", "-f2"},
			expectedPositionals: []string{"a", "b", "c"},
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			flags, positionals, cmd := tc.root.inferCommandAndArgs(tc.args)
			With(t).Verify(flags).Will(EqualTo(tc.expectedFlags)).OrFail()
			With(t).Verify(positionals).Will(EqualTo(tc.expectedPositionals)).OrFail()
			With(t).Verify(cmd.name).Will(EqualTo(tc.expectedCommand)).OrFail()
		})
	}
}

func Test_getFullName(t *testing.T) {
	type testCase struct {
		cmd              *Command
		expectedFullName string
	}
	sub3 := MustNew("sub3", "sub3 desc", "sub3 description", nil, nil, nil)
	sub2 := MustNew("sub2", "sub2 desc", "sub2 description", nil, nil, nil, sub3)
	sub1 := MustNew("sub1", "sub1 desc", "sub1 description", nil, nil, nil, sub2)
	root := MustNew("root", "desc", "description", nil, nil, nil, sub1)
	testCases := map[string]testCase{
		"root": {
			cmd:              root,
			expectedFullName: "root",
		},
		"sub1": {
			cmd:              sub1,
			expectedFullName: "root sub1",
		},
		"sub2": {
			cmd:              sub2,
			expectedFullName: "root sub1 sub2",
		},
		"sub3": {
			cmd:              sub3,
			expectedFullName: "root sub1 sub2 sub3",
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			With(t).Verify(tc.cmd.getFullName()).Will(EqualTo(tc.expectedFullName)).OrFail()
		})
	}
}

func Test_getChain(t *testing.T) {
	type testCase struct {
		cmd           *Command
		expectedChain []string
	}
	sub3 := MustNew("sub3", "sub3 desc", "sub3 description", nil, nil, nil)
	sub2 := MustNew("sub2", "sub2 desc", "sub2 description", nil, nil, nil, sub3)
	sub1 := MustNew("sub1", "sub1 desc", "sub1 description", nil, nil, nil, sub2)
	root := MustNew("root", "desc", "description", nil, nil, nil, sub1)
	testCases := map[string]testCase{
		"root": {
			cmd:           root,
			expectedChain: []string{"root"},
		},
		"sub1": {
			cmd:           sub1,
			expectedChain: []string{"root", "sub1"},
		},
		"sub2": {
			cmd:           sub2,
			expectedChain: []string{"root", "sub1", "sub2"},
		},
		"sub3": {
			cmd:           sub3,
			expectedChain: []string{"root", "sub1", "sub2", "sub3"},
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			var chainNames []string
			for _, cmd := range tc.cmd.getChain() {
				chainNames = append(chainNames, cmd.name)
			}
			With(t).Verify(chainNames).Will(EqualTo(tc.expectedChain)).OrFail()
		})
	}
}

func TestPrintHelp(t *testing.T) {
	t.Parallel()

	type testCase struct {
		commandFactory          func(*testCase) *Command
		expectedHelpOutput      string
		expectedHelpUsageOutput string
	}
	testCases := map[string]testCase{
		"no flags & no positionals": {
			commandFactory: func(*testCase) *Command {
				ligen := loremipsum.NewWithSeed(4321)
				return MustNew("cmd", ligen.Sentence(), ligen.Sentences(2), nil, nil, nil)
			},
			expectedHelpUsageOutput: `
Usage: cmd [--help]
`,
			expectedHelpOutput: `
cmd: Lorem ipsum dolor sit amet consectetur 
    adipiscing elit ac, purus molestie luctus nec 
    neque cursus conubia vehicula rutrum primis 
    laoreet vivamus sed nisl lobortis efficitur 
    ultrices.

Description: Lorem ipsum dolor sit amet 
    consectetur adipiscing elit ac, purus 
    molestie luctus nec. Urna magnis platea risus 
    habitant diam pellentesque per mauris 
    consequat, nec ex dis vehicula convallis 
    habitasse vel molestie auctor suspendisse 
    efficitur rutrum praesent eleifend quisque 
    volutpat curae quis lectus.

Usage:
    cmd [--help]

Flags:
    [--help]  Show this help screen and exit. 
              (default value: false, environment 
              variable: HELP)

`,
		},
		"with flags, args": {
			commandFactory: func(*testCase) *Command {
				ligen := loremipsum.NewWithSeed(4321)
				return MustNew(
					"cmd",
					ligen.Sentence(),
					ligen.Sentences(2),
					&struct {
						Action
						MyFlag string   `desc:"flag description"`
						Args   []string `args:"true"`
					}{},
					nil,
					nil,
				)
			},
			expectedHelpUsageOutput: `
Usage: cmd [--help] 
    [--my-flag=VALUE] 
    [ARGS...]
`,
			expectedHelpOutput: `
cmd: Lorem ipsum dolor sit amet consectetur 
    adipiscing elit ac, purus molestie luctus nec 
    neque cursus conubia vehicula rutrum primis 
    laoreet vivamus sed nisl lobortis efficitur 
    ultrices.

Description: Lorem ipsum dolor sit amet 
    consectetur adipiscing elit ac, purus 
    molestie luctus nec. Urna magnis platea risus 
    habitant diam pellentesque per mauris 
    consequat, nec ex dis vehicula convallis 
    habitasse vel molestie auctor suspendisse 
    efficitur rutrum praesent eleifend quisque 
    volutpat curae quis lectus.

Usage:
    cmd [--help] [--my-flag=VALUE] [ARGS...]

Flags:
    [--help]            Show this help screen and 
                        exit. (default value: 
                        false, environment 
                        variable: HELP)
    [--my-flag=VALUE]   flag description 
                        (environment variable: 
                        MY_FLAG)

`,
		},
		"with sub-commands": {
			commandFactory: func(*testCase) *Command {
				ligen := loremipsum.NewWithSeed(4321)
				return MustNew(
					"cmd",
					ligen.Sentence(),
					ligen.Sentences(2),
					&struct {
						Action
						MyFlag string   `desc:"flag description"`
						Args   []string `args:"true"`
					}{},
					nil,
					nil,
					MustNew(
						"child1",
						ligen.Sentence(),
						ligen.Sentences(2),
						&struct {
							Action
							SubFlag string   `desc:"sub flag description"`
							Args    []string `args:"true"`
						}{},
						nil,
						nil,
					),
				)
			},
			expectedHelpUsageOutput: `
Usage: cmd [--help] 
    [--my-flag=VALUE] 
    [ARGS...]
`,
			expectedHelpOutput: `
cmd: Lorem ipsum dolor sit amet consectetur 
    adipiscing elit ac, purus molestie luctus nec 
    neque cursus conubia vehicula rutrum primis 
    laoreet vivamus sed nisl lobortis efficitur 
    ultrices.

Description: Lorem ipsum dolor sit amet 
    consectetur adipiscing elit ac, purus 
    molestie luctus nec. Urna magnis platea risus 
    habitant diam pellentesque per mauris 
    consequat, nec ex dis vehicula convallis 
    habitasse vel molestie auctor suspendisse 
    efficitur rutrum praesent eleifend quisque 
    volutpat curae quis lectus.

Usage:
    cmd [--help] [--my-flag=VALUE] [ARGS...]

Flags:
    [--help]            Show this help screen and 
                        exit. (default value: 
                        false, environment 
                        variable: HELP)
    [--my-flag=VALUE]   flag description 
                        (environment variable: 
                        MY_FLAG)

Available sub-commands:
    child1    Et dolor viverra nulla ipsum 
              finibus curae conubia gravida 
              elementum litora eleifend class 
              porttitor morbi nisi mus non 
              consequat pharetra convallis 
              bibendum rhoncus etiam.

`,
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := tc.commandFactory(&tc)
			b := &bytes.Buffer{}

			With(t).Verify(cmd.PrintHelp(b, 50)).Will(Succeed()).OrFail()
			With(t).Verify(b.String()).Will(EqualTo(tc.expectedHelpOutput[1:])).OrFail()

			b.Reset()
			With(t).Verify(cmd.PrintUsageLine(b, 30)).Will(Succeed()).OrFail()
			With(t).Verify(b.String()).Will(EqualTo(tc.expectedHelpUsageOutput[1:])).OrFail()
		})
	}
}
