package command

import (
	"testing"

	. "github.com/arikkfir/justest"
)

func TestWrappingWriter(t *testing.T) {
	t.Parallel()
	type testCase struct {
		inputs         [][]byte
		width          int
		prefix         string
		expectedString string
	}
	testCases := map[string]testCase{
		"single input, simple single line under width": {
			inputs: [][]byte{
				[]byte("hello world"),
			},
			width: 80,
			expectedString: `
hello world
`,
		},
		"single input, multi-line, all lines under width": {
			inputs: [][]byte{
				[]byte("hello world\ntest test test\none two three"),
			},
			width: 80,
			expectedString: `
hello world
test test test
one two three
`,
		},
		"single input, multi-line, 1st line over width": {
			inputs: [][]byte{
				[]byte("hello world\ntest test\none two"),
			},
			width: 10,
			expectedString: `
hello 
world
test test
one two
`,
		},
		"multi-input, multi-line, 1st line over width": {
			inputs: [][]byte{
				[]byte("hel"),
				[]byte("lo wor"),
				[]byte("ld\ntest "),
				[]byte("test\none two"),
			},
			width: 10,
			expectedString: `
hello 
world
test test
one two
`,
		},
		"multi-input, multi-line, 2nd line over width": {
			inputs: [][]byte{
				[]byte("hel"),
				[]byte("lo\ntesting "),
				[]byte("test\none two"),
			},
			width: 10,
			expectedString: `
hello
testing 
test
one two
`,
		},
		"multi-input, multi-line, 2nd line over width, special symbols": {
			inputs: [][]byte{
				[]byte("hel"),
				[]byte("lo\nabc -"),
				[]byte("-key=v\none two"),
			},
			width: 10,
			expectedString: `
hello
abc 
--key=v
one two
`,
		},
		"multi-input, multi-line, 2nd line over width, split with hard break": {
			inputs: [][]byte{
				[]byte("hel"),
				[]byte("lo\nabc -"),
				[]byte("-very-long-key=v\none two"),
			},
			width: 10,
			expectedString: `
hello
abc 
--very-long-key=v
one two
`,
		},
		"multi-input, multi-line, 2nd line over width & cannot be broken": {
			inputs: [][]byte{
				[]byte("hel"),
				[]byte("lo\n--very-long-key=v\none two"),
			},
			width: 10,
			expectedString: `
hello
--very-long-key=v
one two
`,
		},
		"multi-input, multi-line, 2nd line splits exactly on width": {
			inputs: [][]byte{
				[]byte("hel"),
				[]byte("lo\n--very=v12\none two"),
			},
			width: 10,
			expectedString: `
hello
--very=v12
one two
`,
		},
		"prefixed single input, simple single line under width": {
			inputs: [][]byte{
				[]byte("hello world"),
			},
			width:  80,
			prefix: "    ",
			expectedString: `
    hello world
`,
		},
		"prefixed single input, multi-line, all lines under width": {
			inputs: [][]byte{
				[]byte("hello world\ntest test test\none two three"),
			},
			width:  80,
			prefix: "    ",
			expectedString: `
    hello world
    test test test
    one two three
`,
		},
		"prefixed single input, multi-line, 1st line over width": {
			inputs: [][]byte{
				[]byte("hello world\ntest test\none two"),
			},
			width:  10,
			prefix: "    ",
			expectedString: `
    hello 
    world
    test 
    test
    one 
    two
`,
		},
		"prefixed multi-input, multi-line, 1st line over width": {
			inputs: [][]byte{
				[]byte("hel"),
				[]byte("lo wor"),
				[]byte("ld\ntest "),
				[]byte("test\none two"),
			},
			width:  10,
			prefix: "    ",
			expectedString: `
    hello 
    world
    test 
    test
    one 
    two
`,
		},
		"prefixed multi-input, multi-line, 2nd line over width": {
			inputs: [][]byte{
				[]byte("hel"),
				[]byte("lo\ntesting "),
				[]byte("test\none two"),
			},
			width:  10,
			prefix: "    ",
			expectedString: `
    hello
    testing 
    test
    one 
    two
`,
		},
		"prefixed multi-input, multi-line, 2nd line over width, special symbols": {
			inputs: [][]byte{
				[]byte("hel"),
				[]byte("lo\nabc -"),
				[]byte("-key=v\none two"),
			},
			width:  10,
			prefix: "    ",
			expectedString: `
    hello
    abc 
    --key=v
    one 
    two
`,
		},
		"prefixed multi-input, multi-line, 2nd line over width, split with hard break": {
			inputs: [][]byte{
				[]byte("hel"),
				[]byte("lo\nabc -"),
				[]byte("-very-long-key=v\none two"),
			},
			width:  10,
			prefix: "    ",
			expectedString: `
    hello
    abc 
    --very-long-key=v
    one 
    two
`,
		},
		"prefixed multi-input, multi-line, 2nd line over width & cannot be broken": {
			inputs: [][]byte{
				[]byte("hel"),
				[]byte("lo\n--very-long-key=v\none two"),
			},
			width:  10,
			prefix: "    ",
			expectedString: `
    hello
    --very-long-key=v
    one 
    two
`,
		},
		"prefixed multi-input, multi-line, 2nd line splits exactly on width": {
			inputs: [][]byte{
				[]byte("hel"),
				[]byte("lo\n--very=v12\none two"),
			},
			width:  10,
			prefix: "    ",
			expectedString: `
    hello
    --very=v12
    one 
    two
`,
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			w, err := NewWrappingWriter(tc.width)
			With(t).Verify(err).Will(BeNil()).OrFail()
			if tc.prefix != "" {
				With(t).Verify(w.SetLinePrefix(tc.prefix)).Will(Succeed()).OrFail()
			}

			for _, input := range tc.inputs {
				With(t).Verify(w.Write(input)).Will(Succeed()).OrFail()
			}

			With(t).Verify(w.String()).Will(EqualTo(tc.expectedString[1 : len(tc.expectedString)-1])).OrFail()
		})
	}
}
