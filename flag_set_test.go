package command

import (
	"bytes"
	stdcmp "cmp"
	"reflect"
	"testing"

	. "github.com/arikkfir/justest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestNewFlagSet(t *testing.T) {
	t.Parallel()
	type testCase struct {
		config                     any
		expectedError              string
		expectedFlags              func(tc *testCase) []*flagDef
		expectedPositionalsTargets func(tc *testCase) []*[]string
	}
	testCases := map[string]testCase{
		"nil config":                {},
		"config wih no flags":       {config: &struct{}{}},
		"config with ignored flags": {config: &struct{ MyField string }{MyField: "abc"}},
		"config with a single flag": {
			config: &struct {
				MyField string `name:"my-field" env:"MY_FIELD" value-name:"VVV" desc:"desc" required:"true" inherited:"true" args:"false"`
			}{MyField: "abc"},
			expectedFlags: func(tc *testCase) []*flagDef {
				return []*flagDef{
					{
						flagInfo: flagInfo{
							Name:         "my-field",
							EnvVarName:   ptrOf("MY_FIELD"),
							HasValue:     true,
							ValueName:    ptrOf("VVV"),
							Description:  ptrOf("desc"),
							Required:     ptrOf(true),
							DefaultValue: "abc",
						},
						Inherited: true,
						Targets:   []reflect.Value{reflect.ValueOf(tc.config).Elem().FieldByName("MyField")},
					},
				}
			},
		},
		"non struct pointer config is ignored": {
			config: struct {
				MyField string `name:"my-field" env:"MY_FIELD" value-name:"VVV" desc:"desc" required:"true" inherited:"true" args:"false"`
			}{MyField: "abc"},
		},
		"config with multiple flags": {
			config: &struct {
				MyField1 string `name:"my-field1" env:"MY_FIELD1" value-name:"V1" desc:"desc1" required:"true" inherited:"true" args:"false"`
				MyField2 string `name:"my-field2" env:"MY_FIELD2" value-name:"V2" desc:"desc2" required:"false" inherited:"false" args:"false"`
			}{MyField1: "abc1", MyField2: "abc2"},
			expectedFlags: func(tc *testCase) []*flagDef {
				return []*flagDef{
					{
						flagInfo: flagInfo{
							Name:         "my-field1",
							EnvVarName:   ptrOf("MY_FIELD1"),
							HasValue:     true,
							ValueName:    ptrOf("V1"),
							Description:  ptrOf("desc1"),
							Required:     ptrOf(true),
							DefaultValue: "abc1",
						},
						Inherited: true,
						Targets:   []reflect.Value{reflect.ValueOf(tc.config).Elem().FieldByName("MyField1")},
					},
					{
						flagInfo: flagInfo{
							Name:         "my-field2",
							EnvVarName:   ptrOf("MY_FIELD2"),
							HasValue:     true,
							ValueName:    ptrOf("V2"),
							Description:  ptrOf("desc2"),
							Required:     ptrOf(false),
							DefaultValue: "abc2",
						},
						Targets: []reflect.Value{reflect.ValueOf(tc.config).Elem().FieldByName("MyField2")},
					},
				}
			},
		},
		"bad 'flag' tag": {
			config: &struct {
				MyField string `flag:"bad-value"`
			}{},
			expectedError: `^invalid field 'struct \{ MyField string "flag:\\"bad-value\\"" \}.MyField': invalid tag 'flag=bad-value': invalid syntax$`,
		},
		"field with 'flag=false' tag is ignored": {
			config: &struct {
				MyField string `flag:"false"`
			}{},
		},
		"field with just 'flag=true' tag is picked up": {
			config: &struct {
				MyField string `flag:"true"`
			}{},
			expectedFlags: func(tc *testCase) []*flagDef {
				return []*flagDef{
					{
						flagInfo: flagInfo{
							Name:     "my-field",
							HasValue: true,
						},
						Targets: []reflect.Value{reflect.ValueOf(tc.config).Elem().FieldByName("MyField")},
					},
				}
			},
		},
		"field with empty 'name' tag is rejected": {
			config: &struct {
				MyField string `name:""`
			}{},
			expectedError: `^invalid field 'struct \{ MyField string "name:\\"\\"" \}.MyField': invalid tag 'name=': must not be empty$`,
		},
		"value of 'name' tag is used": {
			config: &struct {
				MyField string `name:"a"`
			}{},
			expectedFlags: func(tc *testCase) []*flagDef {
				return []*flagDef{
					{
						flagInfo: flagInfo{Name: "a", HasValue: true},
						Targets:  []reflect.Value{reflect.ValueOf(tc.config).Elem().FieldByName("MyField")},
					},
				}
			},
		},
		"field with empty 'env' tag is rejected": {
			config: &struct {
				MyField string `env:""`
			}{},
			expectedError: `^invalid field 'struct \{ MyField string "env:\\"\\"" \}.MyField': invalid tag 'env=': must not be empty$`,
		},
		"value of 'env' tag is used and uppercased": {
			config: &struct {
				MyField string `env:"a"`
			}{},
			expectedFlags: func(tc *testCase) []*flagDef {
				return []*flagDef{
					{
						flagInfo: flagInfo{Name: "my-field", EnvVarName: ptrOf("A"), HasValue: true},
						Targets:  []reflect.Value{reflect.ValueOf(tc.config).Elem().FieldByName("MyField")},
					},
				}
			},
		},
		"field with empty 'value-name' tag is rejected": {
			config: &struct {
				MyField string `value-name:""`
			}{},
			expectedError: `^invalid field 'struct \{ MyField string "value-name:\\"\\"" \}.MyField': invalid tag 'value-name=': must not be empty$`,
		},
		"value of 'value-name' tag is used": {
			config: &struct {
				MyField string `value-name:"V"`
			}{},
			expectedFlags: func(tc *testCase) []*flagDef {
				return []*flagDef{
					{
						flagInfo: flagInfo{Name: "my-field", HasValue: true, ValueName: ptrOf("V")},
						Targets:  []reflect.Value{reflect.ValueOf(tc.config).Elem().FieldByName("MyField")},
					},
				}
			},
		},
		"field with empty 'description' tag is allowed": {
			config: &struct {
				MyField string `desc:""`
			}{},
			expectedFlags: func(tc *testCase) []*flagDef {
				return []*flagDef{
					{
						flagInfo: flagInfo{Name: "my-field", HasValue: true, Description: ptrOf("")},
						Targets:  []reflect.Value{reflect.ValueOf(tc.config).Elem().FieldByName("MyField")},
					},
				}
			},
		},
		"value of 'description' tag is used": {
			config: &struct {
				MyField string `desc:"Some Description"`
			}{},
			expectedFlags: func(tc *testCase) []*flagDef {
				return []*flagDef{
					{
						flagInfo: flagInfo{Name: "my-field", HasValue: true, Description: ptrOf("Some Description")},
						Targets:  []reflect.Value{reflect.ValueOf(tc.config).Elem().FieldByName("MyField")},
					},
				}
			},
		},
		"bad 'required' tag": {
			config: &struct {
				MyField string `required:"bad-value"`
			}{},
			expectedError: `^invalid field 'struct \{ MyField string "required:\\"bad-value\\"" \}.MyField': invalid tag 'required=bad-value': invalid syntax$`,
		},
		"field with 'required=false' tag is not required": {
			config: &struct {
				MyField string `required:"false"`
			}{},
			expectedFlags: func(tc *testCase) []*flagDef {
				return []*flagDef{
					{
						flagInfo: flagInfo{Name: "my-field", HasValue: true, Required: ptrOf(false)},
						Targets:  []reflect.Value{reflect.ValueOf(tc.config).Elem().FieldByName("MyField")},
					},
				}
			},
		},
		"field with 'required=true' tag is required": {
			config: &struct {
				MyField string `required:"true"`
			}{},
			expectedFlags: func(tc *testCase) []*flagDef {
				return []*flagDef{
					{
						flagInfo: flagInfo{Name: "my-field", HasValue: true, Required: ptrOf(true)},
						Targets:  []reflect.Value{reflect.ValueOf(tc.config).Elem().FieldByName("MyField")},
					},
				}
			},
		},
		"bad 'inherited' tag": {
			config: &struct {
				MyField string `inherited:"bad-value"`
			}{},
			expectedError: `^invalid field 'struct \{ MyField string "inherited:\\"bad-value\\"" \}.MyField': invalid tag 'inherited=bad-value': invalid syntax$`,
		},
		"field with 'inherited=false' tag is not inherited": {
			config: &struct {
				MyField string `inherited:"false"`
			}{},
			expectedFlags: func(tc *testCase) []*flagDef {
				return []*flagDef{
					{
						flagInfo:  flagInfo{Name: "my-field", HasValue: true},
						Inherited: false,
						Targets:   []reflect.Value{reflect.ValueOf(tc.config).Elem().FieldByName("MyField")},
					},
				}
			},
		},
		"field with 'inherited=true' tag is inherited": {
			config: &struct {
				MyField string `inherited:"true"`
			}{},
			expectedFlags: func(tc *testCase) []*flagDef {
				return []*flagDef{
					{
						flagInfo:  flagInfo{Name: "my-field", HasValue: true},
						Inherited: true,
						Targets:   []reflect.Value{reflect.ValueOf(tc.config).Elem().FieldByName("MyField")},
					},
				}
			},
		},
		"bad 'args' tag": {
			config: &struct {
				MyField string `args:"bad-value"`
			}{},
			expectedError: `^invalid field 'struct \{ MyField string "args:\\"bad-value\\"" \}.MyField': invalid tag 'args=bad-value': invalid syntax$`,
		},
		"field with 'args=false' tag is not marked as args": {
			config: &struct {
				MyField string `name:"f" args:"false"`
			}{},
			expectedFlags: func(tc *testCase) []*flagDef {
				return []*flagDef{
					{
						flagInfo: flagInfo{Name: "f", HasValue: true},
						Targets:  []reflect.Value{reflect.ValueOf(tc.config).Elem().FieldByName("MyField")},
					},
				}
			},
		},
		"field with 'args=true' tag is marked as args": {
			config: &struct {
				MyField []string `args:"true"`
			}{},
			expectedPositionalsTargets: func(tc *testCase) []*[]string {
				typedVal := reflect.ValueOf(tc.config).Elem().FieldByName("MyField").Interface().([]string)
				return []*[]string{&typedVal}
			},
		},
		"field with 'name' and 'args' tags is rejected": {
			config: &struct {
				MyField []string `name:"f" args:"true"`
			}{},
			expectedError: `^invalid field 'struct \{ MyField \[\]string "name:\\"f\\" args:\\"true\\"" \}.MyField': invalid tag 'args=true': cannot be a flag as well$`,
		},
		"field with 'env' and 'args' tags is rejected": {
			config: &struct {
				MyField []string `env:"f" args:"true"`
			}{},
			expectedError: `^invalid field 'struct \{ MyField \[\]string "env:\\"f\\" args:\\"true\\"" \}.MyField': invalid tag 'args=true': cannot be a flag as well$`,
		},
		"field with 'value-name' and 'args' tags is rejected": {
			config: &struct {
				MyField []string `value-name:"f" args:"true"`
			}{},
			expectedError: `^invalid field 'struct \{ MyField \[\]string "value-name:\\"f\\" args:\\"true\\"" \}.MyField': invalid tag 'args=true': cannot be a flag as well$`,
		},
		"field with 'desc' and 'args' tags is rejected": {
			config: &struct {
				MyField []string `desc:"f" args:"true"`
			}{},
			expectedError: `^invalid field 'struct \{ MyField \[\]string "desc:\\"f\\" args:\\"true\\"" \}.MyField': invalid tag 'args=true': cannot be a flag as well$`,
		},
		"field with 'required' and 'args' tags is rejected": {
			config: &struct {
				MyField []string `required:"true" args:"true"`
			}{},
			expectedError: `^invalid field 'struct \{ MyField \[\]string "required:\\"true\\" args:\\"true\\"" \}.MyField': invalid tag 'args=true': cannot be a flag as well$`,
		},
		"field with 'inherited' and 'args' tags is rejected": {
			config: &struct {
				MyField []string `inherited:"true" args:"true"`
			}{},
			expectedError: `^invalid field 'struct \{ MyField \[\]string "inherited:\\"true\\" args:\\"true\\"" \}.MyField': invalid tag 'args=true': cannot be a flag as well$`,
		},
		"field with 'args' of incorrect type is rejected": {
			config: &struct {
				MyField int `args:"true"`
			}{},
			expectedError: `^invalid field 'struct \{ MyField int "args:\\"true\\"" \}.MyField': invalid tag 'args=true': must be typed as \[\]string$`,
		},
		"struct field cannot use 'args' tag": {
			config: &struct {
				MyField struct{} `args:"true"`
			}{},
			expectedError: `^invalid field 'struct \{ MyField struct \{\} "args:\\"true\\"" \}.MyField': invalid tag 'args=true': cannot be used on struct fields$`,
		},
		"flag name is inferred from field name": {
			config: &struct {
				MyField int `flag:"true"`
			}{},
			expectedFlags: func(tc *testCase) []*flagDef {
				return []*flagDef{
					{
						flagInfo: flagInfo{Name: "my-field", HasValue: true, DefaultValue: "0"},
						Targets:  []reflect.Value{reflect.ValueOf(tc.config).Elem().FieldByName("MyField")},
					},
				}
			},
		},
		"tag 'value-name' is not allowed for bool fields": {
			config: &struct {
				MyField bool `value-name:"VAL"`
			}{},
			expectedError: `^invalid field 'struct \{ MyField bool "value-name:\\"VAL\\"" \}.MyField': invalid tag 'value-name=VAL': not supported for bool fields$`,
		},
		"nested config": {
			config: &struct {
				OuterField1 string   `name:"outer-field1" env:"OUTER_FIELD1" value-name:"outer-V1" desc:"outer-desc1" required:"true" inherited:"true"`
				OuterField2 string   `name:"outer-field2" env:"OUTER_FIELD2" value-name:"outer-V2" desc:"outer-desc2" required:"false" inherited:"false"`
				OuterArgs   []string `args:"true"`
				MyStruct    struct {
					InnerField1 string   `name:"inner-field1" env:"INNER_FIELD1" value-name:"inner-V1" desc:"inner-desc1" required:"true" inherited:"true"`
					InnerField2 string   `name:"inner-field2" env:"INNER_FIELD2" value-name:"inner-V2" desc:"inner-desc2" required:"false" inherited:"false"`
					InnerArgs   []string `args:"true"`
				}
			}{
				OuterField1: "out1",
				OuterField2: "out2",
				MyStruct: struct {
					InnerField1 string   `name:"inner-field1" env:"INNER_FIELD1" value-name:"inner-V1" desc:"inner-desc1" required:"true" inherited:"true"`
					InnerField2 string   `name:"inner-field2" env:"INNER_FIELD2" value-name:"inner-V2" desc:"inner-desc2" required:"false" inherited:"false"`
					InnerArgs   []string `args:"true"`
				}{
					InnerField1: "inner1",
					InnerField2: "inner2",
				},
			},
			expectedFlags: func(tc *testCase) []*flagDef {
				return []*flagDef{
					{
						flagInfo: flagInfo{
							Name:         "outer-field1",
							EnvVarName:   ptrOf("OUTER_FIELD1"),
							HasValue:     true,
							ValueName:    ptrOf("outer-V1"),
							Description:  ptrOf("outer-desc1"),
							Required:     ptrOf(true),
							DefaultValue: "out1",
						},
						Inherited: true,
						Targets:   []reflect.Value{reflect.ValueOf(tc.config).Elem().FieldByName("OuterField1")},
					},
					{
						flagInfo: flagInfo{
							Name:         "outer-field2",
							EnvVarName:   ptrOf("OUTER_FIELD2"),
							HasValue:     true,
							ValueName:    ptrOf("outer-V2"),
							Description:  ptrOf("outer-desc2"),
							Required:     ptrOf(false),
							DefaultValue: "out2",
						},
						Inherited: false,
						Targets:   []reflect.Value{reflect.ValueOf(tc.config).Elem().FieldByName("OuterField2")},
					},
					{
						flagInfo: flagInfo{
							Name:         "inner-field1",
							EnvVarName:   ptrOf("INNER_FIELD1"),
							HasValue:     true,
							ValueName:    ptrOf("inner-V1"),
							Description:  ptrOf("inner-desc1"),
							Required:     ptrOf(true),
							DefaultValue: "inner1",
						},
						Inherited: true,
						Targets:   []reflect.Value{reflect.ValueOf(tc.config).Elem().FieldByName("MyStruct").FieldByName("InnerField1")},
					},
					{
						flagInfo: flagInfo{
							Name:         "inner-field2",
							EnvVarName:   ptrOf("INNER_FIELD2"),
							HasValue:     true,
							ValueName:    ptrOf("inner-V2"),
							Description:  ptrOf("inner-desc2"),
							Required:     ptrOf(false),
							DefaultValue: "inner2",
						},
						Targets: []reflect.Value{reflect.ValueOf(tc.config).Elem().FieldByName("MyStruct").FieldByName("InnerField2")},
					},
				}
			},
			expectedPositionalsTargets: func(tc *testCase) []*[]string {
				valueOfOuterArgs := reflect.ValueOf(tc.config).Elem().FieldByName("OuterArgs").Interface().([]string)
				valueOfInnerArgs := reflect.ValueOf(tc.config).Elem().FieldByName("MyStruct").FieldByName("InnerArgs").Interface().([]string)
				return []*[]string{&valueOfOuterArgs, &valueOfInnerArgs}
			},
		},
		"redeclared field cannot change environment variable": {
			config: &struct {
				MyField1 string `name:"my-field1" env:"MY_FIELD1"`
				MyField2 string `name:"my-field1" env:"MY_FIELD2"`
			}{},
			expectedError: `^invalid field 'struct \{ MyField1 string "name:\\"my-field1\\" env:\\"MY_FIELD1\\""; MyField2 string "name:\\"my-field1\\" env:\\"MY_FIELD2\\"" }.MyField2': invalid tag 'env=MY_FIELD2': cannot redefine environment variable name$`,
		},
		"redeclared field can set environment variable": {
			config: &struct {
				MyField1 string `name:"my-field"`
				MyField2 string `name:"my-field" env:"MF"`
			}{},
			expectedFlags: func(tc *testCase) []*flagDef {
				return []*flagDef{
					{
						flagInfo: flagInfo{
							Name:       "my-field",
							EnvVarName: ptrOf("MF"),
							HasValue:   true,
						},
						Targets: []reflect.Value{
							reflect.ValueOf(tc.config).Elem().FieldByName("MyField1"),
							reflect.ValueOf(tc.config).Elem().FieldByName("MyField2"),
						},
					},
				}
			},
		},
		"redeclared field cannot change has-value": {
			config: &struct {
				MyField1 string `name:"my-field1"`
				MyField2 bool   `name:"my-field1"`
			}{},
			expectedError: `^invalid field 'struct \{ MyField1 string "name:\\"my-field1\\""; MyField2 bool "name:\\"my-field1\\"" }.MyField2': incompatible field types detected \(is one a bool and another isn't\?\)$`,
		},
		"redeclared field cannot change value-name": {
			config: &struct {
				MyField1 string `name:"my-field1" value-name:"V1"`
				MyField2 string `name:"my-field1" value-name:"V2"`
			}{},
			expectedError: `^invalid field 'struct \{ MyField1 string "name:\\"my-field1\\" value-name:\\"V1\\""; MyField2 string "name:\\"my-field1\\" value-name:\\"V2\\"" }.MyField2': invalid tag 'value-name=V2': cannot redefine value name$`,
		},
		"redeclared field can set value-name": {
			config: &struct {
				MyField1 string `name:"my-field"`
				MyField2 string `name:"my-field" value-name:"VVV"`
			}{},
			expectedFlags: func(tc *testCase) []*flagDef {
				return []*flagDef{
					{
						flagInfo: flagInfo{
							Name:      "my-field",
							HasValue:  true,
							ValueName: ptrOf("VVV"),
						},
						Targets: []reflect.Value{
							reflect.ValueOf(tc.config).Elem().FieldByName("MyField1"),
							reflect.ValueOf(tc.config).Elem().FieldByName("MyField2"),
						},
					},
				}
			},
		},
		"redeclared field cannot change description": {
			config: &struct {
				MyField1 string `name:"my-field1" desc:"V1"`
				MyField2 string `name:"my-field1" desc:"V2"`
			}{},
			expectedError: `^invalid field 'struct \{ MyField1 string "name:\\"my-field1\\" desc:\\"V1\\""; MyField2 string "name:\\"my-field1\\" desc:\\"V2\\"" }.MyField2': invalid tag 'desc=V2': cannot redefine description$`,
		},
		"redeclared field can set description": {
			config: &struct {
				MyField1 string `name:"my-field"`
				MyField2 string `name:"my-field" desc:"DESC"`
			}{},
			expectedFlags: func(tc *testCase) []*flagDef {
				return []*flagDef{
					{
						flagInfo: flagInfo{
							Name:        "my-field",
							HasValue:    true,
							Description: ptrOf("DESC"),
						},
						Targets: []reflect.Value{
							reflect.ValueOf(tc.config).Elem().FieldByName("MyField1"),
							reflect.ValueOf(tc.config).Elem().FieldByName("MyField2"),
						},
					},
				}
			},
		},
		"redeclared field cannot change required status": {
			config: &struct {
				MyField1 string `name:"my-field1" required:"true"`
				MyField2 string `name:"my-field1" required:"false"`
			}{},
			expectedError: `^invalid field 'struct \{ MyField1 string "name:\\"my-field1\\" required:\\"true\\""; MyField2 string "name:\\"my-field1\\" required:\\"false\\"" }.MyField2': invalid tag 'required=false': cannot redefine required status$`,
		},
		"redeclared field can set required status": {
			config: &struct {
				MyField1 string `name:"my-field"`
				MyField2 string `name:"my-field" required:"true"`
			}{},
			expectedFlags: func(tc *testCase) []*flagDef {
				return []*flagDef{
					{
						flagInfo: flagInfo{
							Name:     "my-field",
							HasValue: true,
							Required: ptrOf(true),
						},
						Targets: []reflect.Value{
							reflect.ValueOf(tc.config).Elem().FieldByName("MyField1"),
							reflect.ValueOf(tc.config).Elem().FieldByName("MyField2"),
						},
					},
				}
			},
		},
		"redeclared field cannot change default value": {
			config: &struct {
				MyField1 string `name:"my-field1"`
				MyField2 string `name:"my-field1"`
			}{
				MyField1: "v1",
				MyField2: "v2",
			},
			expectedError: `^invalid field 'struct \{ MyField1 string "name:\\"my-field1\\""; MyField2 string "name:\\"my-field1\\"" }.MyField2': incompatible default values detected: 'v1' vs 'v2'$`,
		},
		"redeclared field cannot change inherited status": {
			config: &struct {
				MyField1 string `name:"my-field1" inherited:"true"`
				MyField2 string `name:"my-field1" inherited:"false"`
			}{},
			expectedError: `^invalid field 'struct \{ MyField1 string "name:\\"my-field1\\" inherited:\\"true\\""; MyField2 string "name:\\"my-field1\\" inherited:\\"false\\"" }.MyField2': incompatible inherited status detected: 'true' vs 'false'$`,
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			valueOfConfig := reflect.ValueOf(tc.config)
			if tc.expectedError != "" {
				With(t).Verify(newFlagSet(nil, valueOfConfig)).Will(Fail(tc.expectedError)).OrFail()
			} else {
				fs, err := newFlagSet(nil, valueOfConfig)
				With(t).Verify(err).Will(BeNil()).OrFail()
				if tc.expectedFlags != nil {
					expectedFlags := tc.expectedFlags(&tc)
					With(t).
						Verify(fs.flags).
						Will(EqualTo(
							expectedFlags,
							cmp.AllowUnexported(flagDef{}),
							cmpopts.SortSlices(func(a *flagDef, b *flagDef) bool { return stdcmp.Less(a.Name, b.Name) }),
						)).
						OrFail()
				} else {
					With(t).Verify(fs.flags).Will(BeNil()).OrFail()
				}
				if tc.expectedPositionalsTargets != nil {
					With(t).Verify(fs.positionalsTargets).Will(EqualTo(tc.expectedPositionalsTargets(&tc))).OrFail()
				} else {
					With(t).Verify(fs.positionalsTargets).Will(BeNil()).OrFail()
				}
			}
		})
	}
}

func TestFlagSetGetMergedFlagDefs(t *testing.T) {
	t.Parallel()
	type testCase struct {
		parentConfig  any
		config        any
		expectedError string
		expectedFlags func(tc *testCase) []*mergedFlagDef
	}
	testCases := map[string]testCase{
		"no parent": {
			config: &struct {
				F string `name:"my-field" env:"MY_FIELD" desc:"desc" inherited:"true"`
				S struct {
					F string `name:"my-field" value-name:"VVV" required:"true" inherited:"true"`
				}
			}{
				F: "abc",
				S: struct {
					F string `name:"my-field" value-name:"VVV" required:"true" inherited:"true"`
				}{F: "abc"},
			},
			expectedFlags: func(tc *testCase) []*mergedFlagDef {
				return []*mergedFlagDef{
					{
						flagInfo: flagInfo{
							Name:         "my-field",
							EnvVarName:   ptrOf("MY_FIELD"),
							HasValue:     true,
							ValueName:    ptrOf("VVV"),
							Description:  ptrOf("desc"),
							Required:     ptrOf(true),
							DefaultValue: "abc",
						},
						flagDefs: []*flagDef{
							{
								flagInfo: flagInfo{
									Name:         "my-field",
									EnvVarName:   ptrOf("MY_FIELD"),
									HasValue:     true,
									ValueName:    ptrOf("VVV"),
									Required:     ptrOf(true),
									Description:  ptrOf("desc"),
									DefaultValue: "abc",
								},
								Inherited: true,
								Targets: []reflect.Value{
									reflect.ValueOf(tc.config).Elem().FieldByName("F"),
									reflect.ValueOf(tc.config).Elem().FieldByName("S").FieldByName("F"),
								},
							},
						},
					},
				}
			},
		},
		"flags merged across parents": {
			parentConfig: &struct {
				F1 string `name:"my-field1" env:"MF1" value-name:"VVV" inherited:"true"`
			}{F1: "v1"},
			config: &struct {
				F1  string `name:"my-field1"`
				F11 string `name:"my-field1" desc:"desc1"`
				F2  string `name:"my-field2" env:"MF2" desc:"desc2"`
			}{
				F1:  "v1",
				F11: "v1",
				F2:  "v2",
			},
			expectedFlags: func(tc *testCase) []*mergedFlagDef {
				return []*mergedFlagDef{
					{
						flagInfo: flagInfo{
							Name:         "my-field1",
							EnvVarName:   ptrOf("MF1"),
							HasValue:     true,
							ValueName:    ptrOf("VVV"),
							Description:  ptrOf("desc1"),
							Required:     ptrOf(false),
							DefaultValue: "v1",
						},
						flagDefs: []*flagDef{
							{
								flagInfo: flagInfo{
									Name:         "my-field1",
									HasValue:     true,
									Description:  ptrOf("desc1"),
									DefaultValue: "v1",
								},
								Inherited: false,
								Targets: []reflect.Value{
									reflect.ValueOf(tc.config).Elem().FieldByName("F1"),
									reflect.ValueOf(tc.config).Elem().FieldByName("F11"),
								},
							},
							{
								flagInfo: flagInfo{
									Name:         "my-field1",
									EnvVarName:   ptrOf("MF1"),
									HasValue:     true,
									ValueName:    ptrOf("VVV"),
									DefaultValue: "v1",
								},
								Inherited: true,
								Targets:   []reflect.Value{reflect.ValueOf(tc.parentConfig).Elem().FieldByName("F1")},
							},
						},
					},
					{
						flagInfo: flagInfo{
							Name:         "my-field2",
							EnvVarName:   ptrOf("MF2"),
							HasValue:     true,
							ValueName:    ptrOf("VALUE"),
							Description:  ptrOf("desc2"),
							Required:     ptrOf(false),
							DefaultValue: "v2",
						},
						flagDefs: []*flagDef{
							{
								flagInfo: flagInfo{
									Name:         "my-field2",
									EnvVarName:   ptrOf("MF2"),
									HasValue:     true,
									Description:  ptrOf("desc2"),
									DefaultValue: "v2",
								},
								Targets: []reflect.Value{reflect.ValueOf(tc.config).Elem().FieldByName("F2")},
							},
						},
					},
				}
			},
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var parent *flagSet
			if tc.parentConfig != nil {
				valueOfParentConfig := reflect.ValueOf(tc.parentConfig)
				fs, err := newFlagSet(nil, valueOfParentConfig)
				With(t).Verify(err).Will(BeNil()).OrFail()
				parent = fs
			}
			valueOfConfig := reflect.ValueOf(tc.config)
			if tc.expectedError != "" {
				With(t).Verify(newFlagSet(parent, valueOfConfig)).Will(Fail(tc.expectedError)).OrFail()
			} else {
				fs, err := newFlagSet(parent, valueOfConfig)
				With(t).Verify(err).Will(BeNil()).OrFail()
				if tc.expectedFlags != nil {
					mergedFlagDefs, err := fs.getMergedFlagDefs()
					With(t).Verify(err).Will(BeNil()).OrFail()
					With(t).
						Verify(mergedFlagDefs).
						Will(EqualTo(tc.expectedFlags(&tc), cmp.AllowUnexported(flagDef{}, mergedFlagDef{}))).OrFail()
				} else {
					With(t).Verify(fs.flags).Will(BeNil()).OrFail()
				}
			}
		})
	}
}

func TestFlagSetUsagePrinting(t *testing.T) {
	t.Parallel()
	type testCase struct {
		parentConfig            any
		config                  any
		width                   int
		expectedSingleLineUsage string
		expectedMultiLineUsage  string
	}
	testCases := map[string]testCase{
		"no parent, single flag for multiple fields in nested structure": {
			config: &struct {
				F string `name:"my-field" env:"MY_FIELD" desc:"desc" inherited:"true"`
				S struct {
					F string `name:"my-field" value-name:"VVV" required:"true" inherited:"true"`
				}
			}{
				F: "abc",
				S: struct {
					F string `name:"my-field" value-name:"VVV" required:"true" inherited:"true"`
				}{F: "abc"},
			},
			expectedSingleLineUsage: `--my-field=VVV`,
			expectedMultiLineUsage: `
--my-field=VVV      desc (default value: abc, environment variable: 
                    MY_FIELD)
`,
		},
		"flags merged across parents": {
			parentConfig: &struct {
				F1 string `name:"my-field1" env:"MF1" value-name:"VVV" inherited:"true"`
			}{F1: "v1"},
			config: &struct {
				F1  string `name:"my-field1" required:"true"`
				F11 string `name:"my-field1" desc:"desc1"`
				F2  bool   `name:"my-field2" env:"MF2" desc:"desc2"`
			}{
				F1:  "v1",
				F11: "v1",
			},
			expectedSingleLineUsage: `--my-field1=VVV [--my-field2]`,
			expectedMultiLineUsage: `
--my-field1=VVV     desc1 (default value: v1, environment variable: 
                    MF1)
[--my-field2]       desc2 (default value: false, environment 
                    variable: MF2)
`,
		},
		"positionals without flags": {
			config: &struct {
				Args []string `args:"true"`
			}{},
			expectedSingleLineUsage: `[ARGS...]`,
			expectedMultiLineUsage: `
`,
		},
		"flags and positionals": {
			config: &struct {
				F1   string   `name:"my-field1" value-name:"FF"`
				F2   bool     `name:"my-field2" env:"MF2" desc:"desc2"`
				Args []string `args:"true"`
			}{
				F1: "v1",
			},
			expectedSingleLineUsage: `[--my-field1=FF] [--my-field2] [ARGS...]`,
			expectedMultiLineUsage: `
[--my-field1=FF]    default value: v1, environment variable: MY_FIELD1
[--my-field2]       desc2 (default value: false, environment 
                    variable: MF2)
`,
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var parent *flagSet
			if tc.parentConfig != nil {
				valueOfParentConfig := reflect.ValueOf(tc.parentConfig)
				fs, err := newFlagSet(nil, valueOfParentConfig)
				With(t).Verify(err).Will(BeNil()).OrFail()
				parent = fs
			}
			valueOfConfig := reflect.ValueOf(tc.config)

			fs, err := newFlagSet(parent, valueOfConfig)
			With(t).Verify(err).Will(BeNil()).OrFail()

			width := tc.width
			if width == 0 {
				width = 70
			}

			singleLine := &bytes.Buffer{}
			With(t).Verify(fs.printFlagsSingleLine(singleLine)).Will(Succeed()).OrFail()
			With(t).Verify(singleLine.String()).Will(EqualTo(tc.expectedSingleLineUsage)).OrFail()

			multiLine, err := NewWrappingWriter(width)
			With(t).Verify(err).Will(BeNil()).OrFail()
			With(t).Verify(fs.printFlagsMultiLine(multiLine, "")).Will(Succeed()).OrFail()
			With(t).Verify(multiLine.String()).Will(EqualTo(tc.expectedMultiLineUsage[1:])).OrFail()
		})
	}
}

func TestFlagSetApply(t *testing.T) {
	t.Parallel()
	type testCase struct {
		parentConfig         any
		config               any
		envVars              map[string]string
		args                 []string
		expectedParentConfig any
		expectedConfig       any
		expectedError        string
	}
	testCases := map[string]testCase{
		"CLI overrides environment variables": {
			config: &struct {
				F1 string `name:"my-field1"`
			}{},
			envVars: map[string]string{
				"MY_FIELD1": "should not be used",
			},
			args: []string{"--my-field1=CLI value for F1"},
			expectedConfig: &struct {
				F1 string `name:"my-field1"`
			}{F1: "CLI value for F1"},
		},
		"correct environment variable used for flag": {
			config: &struct {
				F1 string `name:"my-field1" env:"MF1"`
			}{},
			envVars: map[string]string{
				"MY_FIELD1": "should not be used",
				"MF1":       "correct value for F1",
			},
			args: []string{},
			expectedConfig: &struct {
				F1 string `name:"my-field1" env:"MF1"`
			}{F1: "correct value for F1"},
		},
		"default value preserved": {
			config: &struct {
				F1 string `name:"my-field1" env:"MF1"`
				F2 string `name:"my-field2"`
				F3 string `name:"my-field3"`
				F4 string `name:"my-field4"`
			}{F1: "default1", F2: "default2", F3: "default3", F4: "default4"},
			envVars: map[string]string{
				"MY_FIELD1": "should not be used",
				"MF1":       "correct value for F1",
				"MY_FIELD2": "correct value for F2",
			},
			args: []string{"--my-field3=correct value for F3"},
			expectedConfig: &struct {
				F1 string `name:"my-field1" env:"MF1"`
				F2 string `name:"my-field2"`
				F3 string `name:"my-field3"`
				F4 string `name:"my-field4"`
			}{
				F1: "correct value for F1",
				F2: "correct value for F2",
				F3: "correct value for F3",
				F4: "default4",
			},
		},
		"both flags and positionals applied": {
			config: &struct {
				F1   string   `name:"my-field1" env:"MF1"`
				Args []string `args:"true"`
			}{},
			envVars: map[string]string{},
			args: []string{
				"--my-field1=correct value for F1",
				"a",
				"b",
				"c",
			},
			expectedConfig: &struct {
				F1   string   `name:"my-field1" env:"MF1"`
				Args []string `args:"true"`
			}{
				F1:   "correct value for F1",
				Args: []string{"a", "b", "c"},
			},
		},
		"invalid flag error": {
			config: &struct {
				F1 string `name:"my-field1"`
			}{},
			envVars:       map[string]string{},
			args:          []string{"--my-field1=VVV1", "--my-field2=VVV2"},
			expectedError: `^unknown flag: --my-field2$`,
		},
		"required field is missing error": {
			config: &struct {
				F1 string `name:"my-field1"`
				F2 string `name:"my-field2" required:"true"`
			}{F1: "v1"},
			envVars:       map[string]string{},
			args:          []string{"--my-field1=VVV1"},
			expectedError: `^required flag is missing: --my-field2$`,
		},
		"optional string field is not required": {
			config: &struct {
				F1 string `desc:"Some desc."`
				F2 string `required:"true" desc:"Some desc."`
			}{F2: "v2"},
			envVars: map[string]string{},
			args:    []string{},
			expectedConfig: &struct {
				F1 string `desc:"Some desc."`
				F2 string `required:"true" desc:"Some desc."`
			}{F1: "", F2: "v2"},
		},
		"bool flag default value is considered": {
			config: &struct {
				F1 bool `name:"my-field1" required:"true"`
			}{F1: false},
			envVars: map[string]string{},
			args:    []string{},
			expectedConfig: &struct {
				F1 bool `name:"my-field1" required:"true"`
			}{F1: false},
		},
		"bool flag default value overridden by arg": {
			config: &struct {
				F1 bool `name:"my-field1" required:"true"`
			}{F1: false},
			envVars: map[string]string{},
			args:    []string{"--my-field1"},
			expectedConfig: &struct {
				F1 bool `name:"my-field1" required:"true"`
			}{F1: true},
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var parent *flagSet
			if tc.parentConfig != nil {
				valueOfParentConfig := reflect.ValueOf(tc.parentConfig)
				fs, err := newFlagSet(nil, valueOfParentConfig)
				With(t).Verify(err).Will(BeNil()).OrFail()
				parent = fs
			}
			valueOfConfig := reflect.ValueOf(tc.config)

			fs, err := newFlagSet(parent, valueOfConfig)
			With(t).Verify(err).Will(BeNil()).OrFail()

			if tc.expectedError != "" {
				With(t).Verify(fs.apply(tc.envVars, tc.args)).Will(Fail(tc.expectedError)).OrFail()
			} else {
				With(t).Verify(fs.apply(tc.envVars, tc.args)).Will(Succeed()).OrFail()
				With(t).Verify(tc.parentConfig).Will(EqualTo(tc.expectedParentConfig)).OrFail()
				With(t).Verify(tc.config).Will(EqualTo(tc.expectedConfig)).OrFail()
			}
		})
	}
}
