package command

import (
	"testing"

	. "github.com/arikkfir/justest"
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
