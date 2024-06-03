package command

import (
	"math"
	"reflect"
	"strconv"
	"testing"

	. "github.com/arikkfir/justest"
)

func TestFlagDefIsRequired(t *testing.T) {
	t.Parallel()

	type testCase struct {
		fd               *flagDef
		expectedRequired bool
	}

	testCases := map[string]testCase{
		"nil":    {fd: &flagDef{flagInfo: flagInfo{Name: "my-flag"}}, expectedRequired: false},
		"*true":  {fd: &flagDef{flagInfo: flagInfo{Name: "my-flag", Required: &[]bool{true}[0]}}, expectedRequired: true},
		"*false": {fd: &flagDef{flagInfo: flagInfo{Name: "my-flag", Required: &[]bool{false}[0]}}, expectedRequired: false},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			With(t).Verify(tc.fd.isRequired()).Will(EqualTo(tc.expectedRequired)).OrFail()
		})
	}
}

func TestFlagDefGetValueName(t *testing.T) {
	t.Parallel()

	type testCase struct {
		fd                *flagDef
		expectedValueName string
	}

	testCases := map[string]testCase{
		"does not have value":           {fd: &flagDef{flagInfo: flagInfo{Name: "my-flag", HasValue: false}}, expectedValueName: ""},
		"has value & has value name":    {fd: &flagDef{flagInfo: flagInfo{Name: "my-flag", HasValue: true, ValueName: &[]string{"VVV"}[0]}}, expectedValueName: "VVV"},
		"has value & has no value name": {fd: &flagDef{flagInfo: flagInfo{Name: "my-flag", HasValue: true}}, expectedValueName: "VALUE"},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			With(t).Verify(tc.fd.getValueName()).Will(EqualTo(tc.expectedValueName)).OrFail()
		})
	}
}

func TestFlagDefSetValue(t *testing.T) {
	t.Parallel()
	type Target struct {
		B    bool
		I    int
		I8   int8
		I16  int16
		I32  int32
		I64  int64
		UI   uint
		UI8  uint8
		UI16 uint16
		UI32 uint32
		UI64 uint64
		F32  float32
		F64  float64
		S    string
	}
	type testCase struct {
		target         *Target
		targetsFactory func(tc *testCase) []reflect.Value
		value          string
		expectedTarget Target
		expectedError  string
	}
	testCases := map[string]testCase{
		"valid bool": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("B")}
			},
			value:          "true",
			expectedTarget: Target{B: true},
		},
		"invalid bool": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("B")}
			},
			value:         "bad bool",
			expectedError: `^invalid value 'bad bool' for flag 'my-flag': invalid syntax$`,
		},
		"valid int": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("I")}
			},
			value:          strconv.FormatInt(math.MaxInt, 10),
			expectedTarget: Target{I: math.MaxInt},
		},
		"invalid int": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("I")}
			},
			value:         "abc",
			expectedError: `^invalid value 'abc' for flag 'my-flag': invalid syntax$`,
		},
		"valid int8": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("I8")}
			},
			value:          strconv.FormatInt(math.MaxInt8, 10),
			expectedTarget: Target{I8: math.MaxInt8},
		},
		"invalid int8": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("I8")}
			},
			value:         "abc",
			expectedError: `^invalid value 'abc' for flag 'my-flag': invalid syntax$`,
		},
		"valid int16": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("I16")}
			},
			value:          strconv.FormatInt(math.MaxInt16, 10),
			expectedTarget: Target{I16: math.MaxInt16},
		},
		"invalid int16": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("I16")}
			},
			value:         "abc",
			expectedError: `^invalid value 'abc' for flag 'my-flag': invalid syntax$`,
		},
		"valid int32": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("I32")}
			},
			value:          strconv.FormatInt(math.MaxInt32, 10),
			expectedTarget: Target{I32: math.MaxInt32},
		},
		"invalid int32": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("I32")}
			},
			value:         "abc",
			expectedError: `^invalid value 'abc' for flag 'my-flag': invalid syntax$`,
		},
		"valid int64": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("I64")}
			},
			value:          strconv.FormatInt(math.MaxInt64, 10),
			expectedTarget: Target{I64: math.MaxInt64},
		},
		"invalid int64": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("I64")}
			},
			value:         "abc",
			expectedError: `^invalid value 'abc' for flag 'my-flag': invalid syntax$`,
		},
		"valid uint": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("UI")}
			},
			value:          strconv.FormatUint(math.MaxUint, 10),
			expectedTarget: Target{UI: math.MaxUint},
		},
		"invalid uint": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("UI")}
			},
			value:         "abc",
			expectedError: `^invalid value 'abc' for flag 'my-flag': invalid syntax$`,
		},
		"valid uint8": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("UI8")}
			},
			value:          strconv.FormatUint(math.MaxUint8, 10),
			expectedTarget: Target{UI8: math.MaxUint8},
		},
		"invalid uint8": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("UI8")}
			},
			value:         "abc",
			expectedError: `^invalid value 'abc' for flag 'my-flag': invalid syntax$`,
		},
		"valid uint16": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("UI16")}
			},
			value:          strconv.FormatUint(math.MaxUint16, 10),
			expectedTarget: Target{UI16: math.MaxUint16},
		},
		"invalid uint16": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("UI16")}
			},
			value:         "abc",
			expectedError: `^invalid value 'abc' for flag 'my-flag': invalid syntax$`,
		},
		"valid uint32": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("UI32")}
			},
			value:          strconv.FormatUint(math.MaxUint32, 10),
			expectedTarget: Target{UI32: math.MaxUint32},
		},
		"invalid uint32": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("UI32")}
			},
			value:         "abc",
			expectedError: `^invalid value 'abc' for flag 'my-flag': invalid syntax$`,
		},
		"valid uint64": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("UI64")}
			},
			value:          strconv.FormatUint(math.MaxUint64, 10),
			expectedTarget: Target{UI64: math.MaxUint64},
		},
		"invalid uint64": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("UI64")}
			},
			value:         "abc",
			expectedError: `^invalid value 'abc' for flag 'my-flag': invalid syntax$`,
		},
		"valid float32": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("F32")}
			},
			value:          strconv.FormatFloat(math.MaxFloat32, 'g', -1, 64),
			expectedTarget: Target{F32: math.MaxFloat32},
		},
		"invalid float32": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("F32")}
			},
			value:         "abc",
			expectedError: `^invalid value 'abc' for flag 'my-flag': invalid syntax$`,
		},
		"valid float64": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("F64")}
			},
			value:          strconv.FormatFloat(math.MaxFloat64, 'g', -1, 64),
			expectedTarget: Target{F64: math.MaxFloat64},
		},
		"invalid float64": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("F64")}
			},
			value:         "abc",
			expectedError: `^invalid value 'abc' for flag 'my-flag': invalid syntax$`,
		},
		"string": {
			target: &Target{},
			targetsFactory: func(tc *testCase) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(tc.target).Elem().FieldByName("S")}
			},
			value:          "abc",
			expectedTarget: Target{S: "abc"},
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fd := &flagDef{flagInfo: flagInfo{Name: "my-flag"}, Targets: tc.targetsFactory(&tc)}
			err := fd.setValue(tc.value)
			if tc.expectedError != "" {
				With(t).Verify(err).Will(Fail(tc.expectedError)).OrFail()
			} else {
				With(t).Verify(err).Will(BeNil()).OrFail()
				With(t).Verify(*tc.target).Will(EqualTo(tc.expectedTarget)).OrFail()
			}
		})
	}
}
