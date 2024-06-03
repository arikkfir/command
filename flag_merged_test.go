package command

import (
	"reflect"
	"slices"
	"testing"

	. "github.com/arikkfir/justest"
)

func TestMergedFlagDefAddFlagDef(t *testing.T) {
	t.Parallel()

	type testCase struct {
		mfd           *mergedFlagDef
		fd            *flagDef
		expectedError string
		verifier      func(t T, tc *testCase)
	}

	testCases := map[string]testCase{
		"valid": {
			mfd: &mergedFlagDef{
				flagInfo: flagInfo{
					Name:         "my-flag",
					EnvVarName:   ptrOf("MY_FLAG"),
					HasValue:     true,
					ValueName:    &[]string{"VVV"}[0],
					Description:  &[]string{"This is the description"}[0],
					Required:     &[]bool{true}[0],
					DefaultValue: "abc",
				},
			},
			fd: &flagDef{
				flagInfo: flagInfo{
					Name:         "my-flag",
					EnvVarName:   ptrOf("MY_FLAG"),
					HasValue:     true,
					ValueName:    &[]string{"VVV"}[0],
					Description:  &[]string{"This is the description"}[0],
					Required:     &[]bool{true}[0],
					DefaultValue: "abc",
				},
			},
			verifier: func(t T, tc *testCase) {
				With(t).Verify(tc.mfd.Name).Will(EqualTo(tc.fd.Name)).OrFail()
				With(t).Verify(tc.mfd.EnvVarName).Will(EqualTo(tc.fd.EnvVarName)).OrFail()
				With(t).Verify(tc.mfd.HasValue).Will(EqualTo(tc.fd.HasValue)).OrFail()
				With(t).Verify(tc.mfd.ValueName).Will(EqualTo(tc.fd.ValueName)).OrFail()
				With(t).Verify(tc.mfd.Description).Will(EqualTo(tc.fd.Description)).OrFail()
				With(t).Verify(tc.mfd.Required).Will(EqualTo(tc.fd.Required)).OrFail()
				With(t).Verify(tc.mfd.DefaultValue).Will(EqualTo(tc.fd.DefaultValue)).OrFail()
			},
		},
		"unexpected name": {
			mfd:           &mergedFlagDef{flagInfo: flagInfo{Name: "my-flag"}},
			fd:            &flagDef{flagInfo: flagInfo{Name: "other-flag"}},
			expectedError: `given flag 'other-flag' has incompatible name - must be 'my-flag'`,
		},
		"unexpected environment variable": {
			mfd:           &mergedFlagDef{flagInfo: flagInfo{Name: "my-flag", EnvVarName: ptrOf("MY_FLAG")}},
			fd:            &flagDef{flagInfo: flagInfo{Name: "my-flag", EnvVarName: ptrOf("BAD_FLAG")}},
			expectedError: `flag 'my-flag' has incompatible environment variable name 'BAD_FLAG' - must be 'MY_FLAG'`,
		},
		"expected flag to have a value": {
			mfd:           &mergedFlagDef{flagInfo: flagInfo{Name: "my-flag", HasValue: true}},
			fd:            &flagDef{flagInfo: flagInfo{Name: "my-flag", HasValue: false}},
			expectedError: `given flag 'my-flag' must have a value, but it does not`,
		},
		"expected flag to not have a value": {
			mfd:           &mergedFlagDef{flagInfo: flagInfo{Name: "my-flag", HasValue: false}},
			fd:            &flagDef{flagInfo: flagInfo{Name: "my-flag", HasValue: true}},
			expectedError: `given flag 'my-flag' must not have a value, but it does`,
		},
		"given value-name overrides nil value name": {
			mfd: &mergedFlagDef{flagInfo: flagInfo{Name: "my-flag"}},
			fd:  &flagDef{flagInfo: flagInfo{Name: "my-flag", ValueName: &[]string{"val"}[0]}},
			verifier: func(t T, tc *testCase) {
				With(t).Verify(tc.mfd.ValueName).Will(EqualTo(tc.fd.ValueName)).OrFail()
			},
		},
		"given value-name equals existing value name does nothing": {
			mfd: &mergedFlagDef{flagInfo: flagInfo{Name: "my-flag", ValueName: &[]string{"val"}[0]}},
			fd:  &flagDef{flagInfo: flagInfo{Name: "my-flag", ValueName: &[]string{"val"}[0]}},
			verifier: func(t T, tc *testCase) {
				With(t).Verify(tc.mfd.ValueName).Will(EqualTo(tc.fd.ValueName)).OrFail()
			},
		},
		"unexpected value name": {
			mfd:           &mergedFlagDef{flagInfo: flagInfo{Name: "my-flag", ValueName: &[]string{"val1"}[0]}},
			fd:            &flagDef{flagInfo: flagInfo{Name: "my-flag", ValueName: &[]string{"val2"}[0]}},
			expectedError: `flag 'my-flag' has incompatible value-name 'val2' - must be 'val1'`,
		},
		"given description overrides nil description": {
			mfd: &mergedFlagDef{flagInfo: flagInfo{Name: "my-flag"}},
			fd:  &flagDef{flagInfo: flagInfo{Name: "my-flag", Description: &[]string{"desc"}[0]}},
			verifier: func(t T, tc *testCase) {
				With(t).Verify(tc.mfd.Description).Will(EqualTo(tc.fd.Description)).OrFail()
			},
		},
		"given description equals existing description does nothing": {
			mfd: &mergedFlagDef{flagInfo: flagInfo{Name: "my-flag", Description: &[]string{"desc"}[0]}},
			fd:  &flagDef{flagInfo: flagInfo{Name: "my-flag", Description: &[]string{"desc"}[0]}},
			verifier: func(t T, tc *testCase) {
				With(t).Verify(tc.mfd.Description).Will(EqualTo(tc.fd.Description)).OrFail()
			},
		},
		"unexpected description": {
			mfd:           &mergedFlagDef{flagInfo: flagInfo{Name: "my-flag", Description: &[]string{"desc1"}[0]}},
			fd:            &flagDef{flagInfo: flagInfo{Name: "my-flag", Description: &[]string{"desc2"}[0]}},
			expectedError: `flag 'my-flag' has incompatible description`,
		},
		"given required overrides nil required": {
			mfd: &mergedFlagDef{flagInfo: flagInfo{Name: "my-flag"}},
			fd:  &flagDef{flagInfo: flagInfo{Name: "my-flag", Required: &[]bool{true}[0]}},
			verifier: func(t T, tc *testCase) {
				With(t).Verify(tc.mfd.Required).Will(EqualTo(tc.fd.Required)).OrFail()
			},
		},
		"given required equals existing required does nothing": {
			mfd: &mergedFlagDef{flagInfo: flagInfo{Name: "my-flag", Required: &[]bool{true}[0]}},
			fd:  &flagDef{flagInfo: flagInfo{Name: "my-flag", Required: &[]bool{true}[0]}},
			verifier: func(t T, tc *testCase) {
				With(t).Verify(tc.mfd.Required).Will(EqualTo(tc.fd.Required)).OrFail()
			},
		},
		"unexpected required": {
			mfd:           &mergedFlagDef{flagInfo: flagInfo{Name: "my-flag", Required: &[]bool{true}[0]}},
			fd:            &flagDef{flagInfo: flagInfo{Name: "my-flag", Required: &[]bool{false}[0]}},
			expectedError: `flag 'my-flag' is incompatibly optional - must be required`,
		},
		"unexpected default value": {
			mfd:           &mergedFlagDef{flagInfo: flagInfo{Name: "my-flag", DefaultValue: "abc"}},
			fd:            &flagDef{flagInfo: flagInfo{Name: "my-flag", DefaultValue: "abcdef"}},
			expectedError: `flag 'my-flag' has incompatible default value 'abcdef' - must be 'abc'`,
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			if tc.expectedError != "" {
				With(t).Verify(tc.mfd.addFlagDef(tc.fd)).Will(Fail(tc.expectedError)).OrFail()
				With(t).Verify(slices.Contains(tc.mfd.flagDefs, tc.fd)).Will(EqualTo(false)).OrFail()
			} else {
				With(t).Verify(tc.mfd.addFlagDef(tc.fd)).Will(Succeed()).OrFail()
				With(t).Verify(slices.Contains(tc.mfd.flagDefs, tc.fd)).Will(EqualTo(true)).OrFail()
			}
			if tc.verifier != nil {
				tc.verifier(t, &tc)
			}
		})
	}
}

func TestMergedFlagDefSetValue(t *testing.T) {
	t.Parallel()

	targets := [3]string{}
	mfd := &mergedFlagDef{
		flagInfo: flagInfo{
			Name:     "my-flag",
			HasValue: true,
		},
		flagDefs: []*flagDef{
			{flagInfo: flagInfo{Name: "my-flag", HasValue: true}, Targets: []reflect.Value{reflect.ValueOf(&targets).Elem().Index(0)}},
			{flagInfo: flagInfo{Name: "my-flag", HasValue: true}, Targets: []reflect.Value{reflect.ValueOf(&targets).Elem().Index(1)}},
			{flagInfo: flagInfo{Name: "my-flag", HasValue: true}, Targets: []reflect.Value{reflect.ValueOf(&targets).Elem().Index(2)}},
		},
	}

	With(t).Verify(mfd.setValue("v1")).Will(Succeed()).OrFail()
	With(t).Verify(targets).Will(EqualTo([3]string{"v1", "v1", "v1"})).OrFail()
}

func TestMergedFlagDefIsRequired(t *testing.T) {
	t.Parallel()

	type testCase struct {
		mfd              *mergedFlagDef
		expectedRequired bool
	}

	testCases := map[string]testCase{
		"nil": {
			mfd:              &mergedFlagDef{flagInfo: flagInfo{Name: "my-flag"}},
			expectedRequired: false,
		},
		"*true": {
			mfd:              &mergedFlagDef{flagInfo: flagInfo{Name: "my-flag", Required: &[]bool{true}[0]}},
			expectedRequired: true,
		},
		"*false": {
			mfd:              &mergedFlagDef{flagInfo: flagInfo{Name: "my-flag", Required: &[]bool{false}[0]}},
			expectedRequired: false,
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			With(t).Verify(tc.mfd.isRequired()).Will(EqualTo(tc.expectedRequired)).OrFail()
		})
	}
}

func TestMergedFlagDefIsMissing(t *testing.T) {
	t.Parallel()

	type testCase struct {
		mfd             *mergedFlagDef
		expectedMissing bool
	}

	testCases := map[string]testCase{
		"required & not applied": {
			mfd:             &mergedFlagDef{flagInfo: flagInfo{Name: "my-flag", Required: &[]bool{true}[0]}, applied: false},
			expectedMissing: true,
		},
		"not required & not applied": {
			mfd:             &mergedFlagDef{flagInfo: flagInfo{Name: "my-flag", Required: &[]bool{false}[0]}, applied: false},
			expectedMissing: false,
		},
		"implicitly not required & not applied": {
			mfd:             &mergedFlagDef{flagInfo: flagInfo{Name: "my-flag"}, applied: false},
			expectedMissing: false,
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			With(t).Verify(tc.mfd.isMissing()).Will(EqualTo(tc.expectedMissing)).OrFail()
		})
	}
}

func TestMergedFlagDefGetValueName(t *testing.T) {
	t.Parallel()

	type testCase struct {
		mfd               *mergedFlagDef
		expectedValueName string
	}

	testCases := map[string]testCase{
		"does not have value": {
			mfd:               &mergedFlagDef{flagInfo: flagInfo{Name: "my-flag", HasValue: false}},
			expectedValueName: "",
		},
		"has value & has value name": {
			mfd:               &mergedFlagDef{flagInfo: flagInfo{Name: "my-flag", HasValue: true, ValueName: &[]string{"VVV"}[0]}},
			expectedValueName: "VVV",
		},
		"has value & has no value name": {
			mfd:               &mergedFlagDef{flagInfo: flagInfo{Name: "my-flag", HasValue: true}},
			expectedValueName: "VALUE",
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			With(t).Verify(tc.mfd.getValueName()).Will(EqualTo(tc.expectedValueName)).OrFail()
		})
	}
}
