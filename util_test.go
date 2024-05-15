package command

import (
	"strings"
	"testing"

	. "github.com/arikkfir/justest"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestFieldNameToFlagName(t *testing.T) {
	t.Parallel()
	testCases := map[string]string{
		"abc":               "abc",
		"Abc":               "abc",
		"ABC":               "abc",
		"BrownFox":          "brown-fox",
		"BrownFOX":          "brown-fox",
		"BrownFoxJUMPSOver": "brown-fox-jumps-over",
		"ATeam":             "a-team",
	}
	for fieldName, expectedFlagName := range testCases {
		fieldName := fieldName
		expectedFlagName := expectedFlagName
		t.Run(fieldName, func(t *testing.T) {
			t.Parallel()
			With(t).Verify(fieldNameToFlagName(fieldName)).Will(EqualTo(expectedFlagName)).OrFail()
		})
	}
}

func TestFieldNameToEnvVarName(t *testing.T) {
	t.Parallel()
	testCases := map[string]string{
		"abc":               "ABC",
		"Abc":               "ABC",
		"ABC":               "ABC",
		"BrownFox":          "BROWN_FOX",
		"BrownFOX":          "BROWN_FOX",
		"BrownFoxJUMPSOver": "BROWN_FOX_JUMPS_OVER",
		"ATeam":             "A_TEAM",
	}
	for fieldName, expectedEnvVarName := range testCases {
		fieldName := fieldName
		expectedFlagName := expectedEnvVarName
		t.Run(fieldName, func(t *testing.T) {
			t.Parallel()
			With(t).Verify(fieldNameToEnvVarName(fieldName)).Will(EqualTo(expectedFlagName)).OrFail()
		})
	}
}

func Test_inferCommandFlagsAndPositionals(t *testing.T) {
	type testCase struct {
		root                *Command
		args                []string
		expectedCommand     *Command
		expectedFlags       []string
		expectedPositionals []string
	}

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
	sub2Cmd := New(sub1Cmd, Spec{
		Name:             "sub2",
		ShortDescription: "Sub command 2",
		LongDescription:  "This command is the\nsecond sub command.",
		Config:           &Sub2Config{},
	})
	New(sub2Cmd, Spec{
		Name:             "sub3",
		ShortDescription: "Sub command 3",
		LongDescription:  "This command is the\nthird sub command.",
		Config:           &Sub3Config{},
	})

	testCases := map[string]testCase{
		"No arguments": {
			root:                rootCmd,
			expectedCommand:     rootCmd,
			expectedFlags:       nil,
			expectedPositionals: nil,
		},
		"Flags for root command": {
			root:                rootCmd,
			args:                strings.Split("-f1 -f2", " "),
			expectedCommand:     rootCmd,
			expectedFlags:       []string{"-f1", "-f2"},
			expectedPositionals: nil,
		},
		"Flags and positionals for root command": {
			root:                rootCmd,
			args:                strings.Split("-f1 a -f2 b", " "),
			expectedCommand:     rootCmd,
			expectedFlags:       []string{"-f1", "-f2"},
			expectedPositionals: []string{"a", "b"},
		},
		"Flags and positionals for sub1 command": {
			root:                rootCmd,
			args:                strings.Split("-f1 sub1 -f2 a b", " "),
			expectedCommand:     sub1Cmd,
			expectedFlags:       []string{"-f1", "-f2"},
			expectedPositionals: []string{"a", "b"},
		},
		"Flags and positionals for sub2 command": {
			root:                rootCmd,
			args:                strings.Split("-f1 sub1 -f2 a b sub2 c", " "),
			expectedCommand:     sub2Cmd,
			expectedFlags:       []string{"-f1", "-f2"},
			expectedPositionals: []string{"a", "b", "c"},
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			cmd, flags, pos := inferCommandFlagsAndPositionals(tc.root, tc.args)
			With(t).Verify(cmd).Will(EqualTo(tc.expectedCommand, cmpopts.IgnoreUnexported(Command{}))).OrFail()
			With(t).Verify(flags).Will(EqualTo(tc.expectedFlags)).OrFail()
			With(t).Verify(pos).Will(EqualTo(tc.expectedPositionals)).OrFail()
		})
	}
}
