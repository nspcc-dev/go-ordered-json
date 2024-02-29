// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"testing"
	"unicode"
)

type Optionals struct {
	Sr string `json:"sr"`
	So string `json:"so,omitempty"`
	Sw string `json:"-"`

	Ir int `json:"omitempty"` // actually named omitempty, not an option
	Io int `json:"io,omitempty"`

	Slr []string `json:"slr,random"`
	Slo []string `json:"slo,omitempty"`

	Mr map[string]any `json:"mr"`
	Mo map[string]any `json:",omitempty"`

	Fr float64 `json:"fr"`
	Fo float64 `json:"fo,omitempty"`

	Br bool `json:"br"`
	Bo bool `json:"bo,omitempty"`

	Ur uint `json:"ur"`
	Uo uint `json:"uo,omitempty"`

	Str struct{} `json:"str"`
	Sto struct{} `json:"sto,omitempty"`
}

var optionalsExpected = `{
 "sr": "",
 "omitempty": 0,
 "slr": null,
 "mr": {},
 "fr": 0,
 "br": false,
 "ur": 0,
 "str": {},
 "sto": {}
}`

func TestOmitEmpty(t *testing.T) {
	var o Optionals
	o.Sw = "something"
	o.Mr = map[string]any{}
	o.Mo = map[string]any{}

	got, err := MarshalIndent(&o, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if got := string(got); got != optionalsExpected {
		t.Errorf(" got: %s\nwant: %s\n", got, optionalsExpected)
	}
}

type StringTag struct {
	BoolStr bool   `json:",string"`
	IntStr  int64  `json:",string"`
	StrStr  string `json:",string"`
}

var stringTagExpected = `{
 "BoolStr": "true",
 "IntStr": "42",
 "StrStr": "\u0022xzbit\u0022"
}`

func TestStringTag(t *testing.T) {
	var s StringTag
	s.BoolStr = true
	s.IntStr = 42
	s.StrStr = "xzbit"
	got, err := MarshalIndent(&s, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if got := string(got); got != stringTagExpected {
		t.Fatalf(" got: %s\nwant: %s\n", got, stringTagExpected)
	}

	// Verify that it round-trips.
	var s2 StringTag
	err = NewDecoder(bytes.NewReader(got)).Decode(&s2)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !reflect.DeepEqual(s, s2) {
		t.Fatalf("decode didn't match.\nsource: %#v\nEncoded as:\n%s\ndecode: %#v", s, string(got), s2)
	}
}

// byte slices are special even if they're renamed types.
type renamedByte byte
type renamedByteSlice []byte
type renamedRenamedByteSlice []renamedByte

func TestEncodeRenamedByteSlice(t *testing.T) {
	s := renamedByteSlice("abc")
	result, err := Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	expect := `"YWJj"`
	if string(result) != expect {
		t.Errorf(" got %s want %s", result, expect)
	}
	r := renamedRenamedByteSlice("abc")
	result, err = Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(result) != expect {
		t.Errorf(" got %s want %s", result, expect)
	}
}

var unsupportedValues = []any{
	math.NaN(),
	math.Inf(-1),
	math.Inf(1),
}

func TestUnsupportedValues(t *testing.T) {
	for _, v := range unsupportedValues {
		if _, err := Marshal(v); err != nil {
			if _, ok := err.(*UnsupportedValueError); !ok {
				t.Errorf("for %v, got %T want UnsupportedValueError", v, err)
			}
		} else {
			t.Errorf("for %v, expected error", v)
		}
	}
}

// Ref has Marshaler and Unmarshaler methods with pointer receiver.
type Ref int

func (*Ref) MarshalJSON() ([]byte, error) {
	return []byte(`"ref"`), nil
}

func (r *Ref) UnmarshalJSON([]byte) error {
	*r = 12
	return nil
}

// Val has Marshaler methods with value receiver.
type Val int

func (Val) MarshalJSON() ([]byte, error) {
	return []byte(`"val"`), nil
}

// RefText has Marshaler and Unmarshaler methods with pointer receiver.
type RefText int

func (*RefText) MarshalText() ([]byte, error) {
	return []byte(`"ref"`), nil
}

func (r *RefText) UnmarshalText([]byte) error {
	*r = 13
	return nil
}

// ValText has Marshaler methods with value receiver.
type ValText int

func (ValText) MarshalText() ([]byte, error) {
	return []byte(`"val"`), nil
}

func TestRefValMarshal(t *testing.T) {
	var s = struct {
		R0 Ref
		R1 *Ref
		R2 RefText
		R3 *RefText
		V0 Val
		V1 *Val
		V2 ValText
		V3 *ValText
	}{
		R0: 12,
		R1: new(Ref),
		R2: 14,
		R3: new(RefText),
		V0: 13,
		V1: new(Val),
		V2: 15,
		V3: new(ValText),
	}
	const want = `{"R0":"ref","R1":"ref","R2":"\u0022ref\u0022","R3":"\u0022ref\u0022","V0":"val","V1":"val","V2":"\u0022val\u0022","V3":"\u0022val\u0022"}`
	b, err := Marshal(&s)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got := string(b); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// C implements Marshaler and returns unescaped JSON.
type C int

func (C) MarshalJSON() ([]byte, error) {
	return []byte(`"<&>"`), nil
}

// CText implements Marshaler and returns unescaped text.
type CText int

func (CText) MarshalText() ([]byte, error) {
	return []byte(`"<&>"`), nil
}

func TestMarshaler_NeoGo_PR2174(t *testing.T) {
	source := "IOU（欠条币）：一种支持负数的NEP-17（非严格意义上的）资产，合约无存储区，账户由区块链浏览器统计"
	b, err := Marshal(source)
	if err != nil {
		t.Fatalf("Marshal(c): %v", err)
	}
	want := `"` + `IOU\uFF08\u6B20\u6761\u5E01\uFF09\uFF1A\u4E00\u79CD\u652F\u6301\u8D1F\u6570\u7684NEP-17\uFF08\u975E\u4E25\u683C\u610F\u4E49\u4E0A\u7684\uFF09\u8D44\u4EA7\uFF0C\u5408\u7EA6\u65E0\u5B58\u50A8\u533A\uFF0C\u8D26\u6237\u7531\u533A\u5757\u94FE\u6D4F\u89C8\u5668\u7EDF\u8BA1` + `"`
	if got := string(b); got != want {
		t.Errorf("Marshal(c) = %#q, want %#q", got, want)
	}
}

func TestMarshalerEscaping(t *testing.T) {
	var c C
	want := `"\u003C\u0026\u003E"`
	b, err := Marshal(c)
	if err != nil {
		t.Fatalf("Marshal(c): %v", err)
	}
	if got := string(b); got != want {
		t.Errorf("Marshal(c) = %#q, want %#q", got, want)
	}

	var ct CText
	want = `"\u0022\u003C\u0026\u003E\u0022"`
	b, err = Marshal(ct)
	if err != nil {
		t.Fatalf("Marshal(ct): %v", err)
	}
	if got := string(b); got != want {
		t.Errorf("Marshal(ct) = %#q, want %#q", got, want)
	}
}

type IntType int

type MyStruct struct {
	IntType
}

func TestAnonymousNonstruct(t *testing.T) {
	var i IntType = 11
	a := MyStruct{i}
	const want = `{"IntType":11}`

	b, err := Marshal(a)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got := string(b); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

type unexportedIntType int

type MyStructWithUnexportedIntType struct {
	unexportedIntType
}

func TestAnonymousNonstructWithUnexportedType(t *testing.T) {
	a := MyStructWithUnexportedIntType{11}
	const want = `{}`

	b, err := Marshal(a)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got := string(b); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

type MyStructContainingUnexportedStruct struct {
	unexportedStructType1
	unexportedIntType
}

type unexportedStructType1 struct {
	ExportedIntType1
	unexportedIntType
	unexportedStructType2
}

type unexportedStructType2 struct {
	ExportedIntType2
	unexportedIntType
}

type ExportedIntType1 int
type ExportedIntType2 int

func TestUnexportedAnonymousStructWithExportedType(t *testing.T) {
	s2 := unexportedStructType2{3, 4}
	s1 := unexportedStructType1{1, 2, s2}
	a := MyStructContainingUnexportedStruct{s1, 6}
	const want = `{"ExportedIntType1":1,"ExportedIntType2":3}`

	b, err := Marshal(a)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got := string(b); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

type BugA struct {
	S string
}

type BugB struct {
	BugA
	S string
}

type BugC struct {
	S string
}

// Legal Go: We never use the repeated embedded field (S).
type BugX struct {
	A int
	BugA
	BugB
}

// Issue 16042. Even if a nil interface value is passed in
// as long as it implements MarshalJSON, it should be marshaled.
type nilMarshaler string

func (nm *nilMarshaler) MarshalJSON() ([]byte, error) {
	if nm == nil {
		return Marshal("0zenil0")
	}
	return Marshal("zenil:" + string(*nm))
}

// Issue 16042.
func TestNilMarshal(t *testing.T) {
	testCases := []struct {
		v    any
		want string
	}{
		{v: nil, want: `null`},
		{v: new(float64), want: `0`},
		{v: []any(nil), want: `null`},
		{v: []string(nil), want: `null`},
		{v: map[string]string(nil), want: `null`},
		{v: []byte(nil), want: `null`},
		{v: struct{ M string }{"gopher"}, want: `{"M":"gopher"}`},
		{v: struct{ M Marshaler }{}, want: `{"M":null}`},
		{v: struct{ M Marshaler }{(*nilMarshaler)(nil)}, want: `{"M":"0zenil0"}`},
		{v: struct{ M any }{(*nilMarshaler)(nil)}, want: `{"M":null}`},
	}

	for _, tt := range testCases {
		out, err := Marshal(tt.v)
		if err != nil || string(out) != tt.want {
			t.Errorf("Marshal(%#v) = %#q, %#v, want %#q, nil", tt.v, out, err, tt.want)
			continue
		}
	}
}

// Issue 5245.
func TestEmbeddedBug(t *testing.T) {
	v := BugB{
		BugA{"A"},
		"B",
	}
	b, err := Marshal(v)
	if err != nil {
		t.Fatal("Marshal:", err)
	}
	want := `{"S":"B"}`
	got := string(b)
	if got != want {
		t.Fatalf("Marshal: got %s want %s", got, want)
	}
	// Now check that the duplicate field, S, does not appear.
	x := BugX{
		A: 23,
	}
	b, err = Marshal(x)
	if err != nil {
		t.Fatal("Marshal:", err)
	}
	want = `{"A":23}`
	got = string(b)
	if got != want {
		t.Fatalf("Marshal: got %s want %s", got, want)
	}
}

type BugD struct { // Same as BugA after tagging.
	XXX string `json:"S"`
}

// BugD's tagged S field should dominate BugA's.
type BugY struct {
	BugA
	BugD
}

// Test that a field with a tag dominates untagged fields.
func TestTaggedFieldDominates(t *testing.T) {
	v := BugY{
		BugA{"BugA"},
		BugD{"BugD"},
	}
	b, err := Marshal(v)
	if err != nil {
		t.Fatal("Marshal:", err)
	}
	want := `{"S":"BugD"}`
	got := string(b)
	if got != want {
		t.Fatalf("Marshal: got %s want %s", got, want)
	}
}

// There are no tags here, so S should not appear.
type BugZ struct {
	BugA
	BugC
	BugY // Contains a tagged S field through BugD; should not dominate.
}

func TestDuplicatedFieldDisappears(t *testing.T) {
	v := BugZ{
		BugA{"BugA"},
		BugC{"BugC"},
		BugY{
			BugA{"nested BugA"},
			BugD{"nested BugD"},
		},
	}
	b, err := Marshal(v)
	if err != nil {
		t.Fatal("Marshal:", err)
	}
	want := `{}`
	got := string(b)
	if got != want {
		t.Fatalf("Marshal: got %s want %s", got, want)
	}
}

func TestStringBytes(t *testing.T) {
	t.Parallel()
	// Test that encodeState.stringBytes and encodeState.string use the same encoding.
	var r []rune
	for i := '\u0000'; i <= unicode.MaxRune; i++ {
		r = append(r, i)
	}
	s := string(r) + "\xff\xff\xffhello" // some invalid UTF-8 too

	for _, escapeHTML := range []bool{true, false} {
		es := &encodeState{}
		es.string(s, escapeHTML)

		esBytes := &encodeState{}
		esBytes.stringBytes([]byte(s), escapeHTML)

		enc := es.Buffer.String()
		encBytes := esBytes.Buffer.String()
		if enc != encBytes {
			i := 0
			for i < len(enc) && i < len(encBytes) && enc[i] == encBytes[i] {
				i++
			}
			enc = enc[i:]
			encBytes = encBytes[i:]
			i = 0
			for i < len(enc) && i < len(encBytes) && enc[len(enc)-i-1] == encBytes[len(encBytes)-i-1] {
				i++
			}
			enc = enc[:len(enc)-i]
			encBytes = encBytes[:len(encBytes)-i]

			if len(enc) > 20 {
				enc = enc[:20] + "..."
			}
			if len(encBytes) > 20 {
				encBytes = encBytes[:20] + "..."
			}

			t.Errorf("with escapeHTML=%t, encodings differ at %#q vs %#q",
				escapeHTML, enc, encBytes)
		}
	}
}

func TestIssue10281(t *testing.T) {
	type Foo struct {
		N Number
	}
	x := Foo{Number(`invalid`)}

	b, err := Marshal(&x)
	if err == nil {
		t.Errorf("Marshal(&x) = %#q; want error", b)
	}
}

func TestHTMLEscape(t *testing.T) {
	var b, want bytes.Buffer
	m := `{"M":"<html>foo &` + "\xe2\x80\xa8 \xe2\x80\xa9" + `</html>"}`
	want.Write([]byte(`{"M":"\u003Chtml\u003Efoo \u0026\u2028 \u2029\u003C/html\u003E"}`))
	HTMLEscape(&b, []byte(m))
	if !bytes.Equal(b.Bytes(), want.Bytes()) {
		t.Errorf("HTMLEscape(&b, []byte(m)) = %s; want %s", b.Bytes(), want.Bytes())
	}
}

// golang.org/issue/8582
func TestEncodePointerString(t *testing.T) {
	type stringPointer struct {
		N *int64 `json:"n,string"`
	}
	var n int64 = 42
	b, err := Marshal(stringPointer{N: &n})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got, want := string(b), `{"n":"42"}`; got != want {
		t.Errorf("Marshal = %s, want %s", got, want)
	}
	var back stringPointer
	err = Unmarshal(b, &back)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if back.N == nil {
		t.Fatalf("Unmarshaled nil N field")
	}
	if *back.N != 42 {
		t.Fatalf("*N = %d; want 42", *back.N)
	}
}

var encodeStringTests = []struct {
	in  string
	out string
}{
	{"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f", `"\u0000\u0001\u0002\u0003\u0004\u0005\u0006\u0007\b\t\n\u000B\f\r\u000E\u000F\u0010\u0011\u0012\u0013\u0014\u0015\u0016\u0017\u0018\u0019\u001A\u001B\u001C\u001D\u001E\u001F"`},
	{"\x20\x21\x22\x23\x24\x25\x26\x27\x28\x29\x2a\x2b\x2c\x2d\x2e\x2f\x30\x31\x32\x33\x34\x35\x36\x37\x38\x39\x3a\x3b\x3c\x3d\x3e\x3f", `" !\u0022#$%\u0026\u0027()*\u002B,-./0123456789:;\u003C=\u003E?"`},
	{"\x40\x41\x44\x45\x44\x45\x46\x47\x48\x49\x4a\x4b\x4c\x4d\x4e\x4f\x50\x51\x54\x55\x54\x55\x56\x57\x58\x59\x5a\x5b\x5c\x5d\x5e\x5f", `"@ADEDEFGHIJKLMNOPQTUTUVWXYZ[\\]^_"`},
	{"\x60\x61\x66\x67\x64\x65\x66\x67\x68\x69\x6a\x6b\x6c\x6d\x6e\x6f\x70\x71\x76\x77\x74\x75\x76\x77\x78\x79\x7a\x7b\x7c\x7d\x7e\x7f", `"\u0060afgdefghijklmnopqvwtuvwxyz{|}~\u007F"`},
	{"\x80\x81\x88\x89\x84\x85\x86\x87\x88\x89\x8a\x8b\x8c\x8d\x8e\x8f\x90\x91\x98\x99\x94\x95\x96\x97\x98\x99\x9a\x9b\x9c\x9d\x9e\x9f", `"\u0080\u0081\u0088\u0089\u0084\u0085\u0086\u0087\u0088\u0089\u008A\u008B\u008C\u008D\u008E\u008F\u0090\u0091\u0098\u0099\u0094\u0095\u0096\u0097\u0098\u0099\u009A\u009B\u009C\u009D\u009E\u009F"`},
	{"\xa0\xa1\xaa\xab\xa4\xa5\xa6\xa7\xa8\xa9\xaa\xab\xac\xad\xae\xaf\xb0\xb1\xba\xbb\xb4\xb5\xb6\xb7\xb8\xb9\xba\xbb\xbc\xbd\xbe\xbf", `"\u00A0\u00A1\u00AA\u00AB\u00A4\u00A5\u00A6\u00A7\u00A8\u00A9\u00AA\u00AB\u00AC\u00AD\u00AE\u00AF\u00B0\u00B1\u00BA\u00BB\u00B4\u00B5\u00B6\u00B7\u00B8\u00B9\u00BA\u00BB\u00BC\u00BD\u00BE\u00BF"`},
	{"\xc0\xc1\xcc\xcd\xc4\xc5\xc6\xc7\xc8\xc9\xca\xcb\xcc\xcd\xce\xcf\xd0\xd1\xdc\xdd\xd4\xd5\xd6\xd7\xd8\xd9\xda\xdb\xdc\xdd\xde\xdf", `"\u00C0\u00C1\u00CC\u00CD\u00C4\u00C5\u00C6\u00C7\u00C8\u00C9\u00CA\u00CB\u00CC\u00CD\u00CE\u00CF\u00D0\u00D1\u00DC\u00DD\u00D4\u00D5\u00D6\u00D7\u00D8\u00D9\u00DA\u00DB\u00DC\u00DD\u00DE\u00DF"`},
	{"\xe0\xe1\xee\xef\xe4\xe5\xe6\xe7\xe8\xe9\xea\xeb\xec\xed\xee\xef\xf0\xf1\xfe\xff\xf4\xf5\xf6\xf7\xf8\xf9\xfa\xfb\xfc\xfd\xfe\xff", `"\u00E0\u00E1\u00EE\u00EF\u00E4\u00E5\u00E6\u00E7\u00E8\u00E9\u00EA\u00EB\u00EC\u00ED\u00EE\u00EF\u00F0\u00F1\u00FE\u00FF\u00F4\u00F5\u00F6\u00F7\u00F8\u00F9\u00FA\u00FB\u00FC\u00FD\u00FE\u00FF"`},
	{"\xff\xd0", `"\u00FF\u00D0"`},
	{"测试", `"\u6D4B\u8BD5"`},
}

func TestEncodeString(t *testing.T) {
	for _, tt := range encodeStringTests {
		b, err := Marshal(tt.in)
		if err != nil {
			t.Errorf("Marshal(%q): %v", tt.in, err)
			continue
		}
		out := string(b)
		if out != tt.out {
			t.Errorf("Marshal(%q) = %#q, want %#q", tt.in, out, tt.out)
		}
	}
}

type jsonbyte byte

func (b jsonbyte) MarshalJSON() ([]byte, error) { return tenc(`{"JB":%d}`, b) }

type textbyte byte

func (b textbyte) MarshalText() ([]byte, error) { return tenc(`TB:%d`, b) }

type jsonint int

func (i jsonint) MarshalJSON() ([]byte, error) { return tenc(`{"JI":%d}`, i) }

type textint int

func (i textint) MarshalText() ([]byte, error) { return tenc(`TI:%d`, i) }

func tenc(format string, a ...any) ([]byte, error) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, format, a...)
	return buf.Bytes(), nil
}

// Issue 13783
func TestEncodeBytekind(t *testing.T) {
	testdata := []struct {
		data any
		want string
	}{
		{byte(7), "7"},
		{jsonbyte(7), `{"JB":7}`},
		{textbyte(4), `"TB:4"`},
		{jsonint(5), `{"JI":5}`},
		{textint(1), `"TI:1"`},
		{[]byte{0, 1}, `"AAE="`},
		{[]jsonbyte{0, 1}, `[{"JB":0},{"JB":1}]`},
		{[][]jsonbyte{{0, 1}, {3}}, `[[{"JB":0},{"JB":1}],[{"JB":3}]]`},
		{[]textbyte{2, 3}, `["TB:2","TB:3"]`},
		{[]jsonint{5, 4}, `[{"JI":5},{"JI":4}]`},
		{[]textint{9, 3}, `["TI:9","TI:3"]`},
		{[]int{9, 3}, `[9,3]`},
	}
	for _, d := range testdata {
		js, err := Marshal(d.data)
		if err != nil {
			t.Error(err)
			continue
		}
		got, want := string(js), d.want
		if got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	}
}

func TestTextMarshalerMapKeysAreSorted(t *testing.T) {
	b, err := Marshal(map[unmarshalerText]int{
		{"x", "y"}: 1,
		{"y", "x"}: 2,
		{"a", "z"}: 3,
		{"z", "a"}: 4,
	})
	if err != nil {
		t.Fatalf("Failed to Marshal text.Marshaler: %v", err)
	}
	const want = `{"a:z":3,"x:y":1,"y:x":2,"z:a":4}`
	if string(b) != want {
		t.Errorf("Marshal map with text.Marshaler keys: got %#q, want %#q", b, want)
	}
}

var re = regexp.MustCompile

// syntactic checks on form of marshaled floating point numbers.
var badFloatREs = []*regexp.Regexp{
	re(`p`),                     // no binary exponential notation
	re(`^\+`),                   // no leading + sign
	re(`^-?0[^.]`),              // no unnecessary leading zeros
	re(`^-?\.`),                 // leading zero required before decimal point
	re(`\.(e|$)`),               // no trailing decimal
	re(`\.[0-9]+0(e|$)`),        // no trailing zero in fraction
	re(`^-?(0|[0-9]{2,})\..*e`), // exponential notation must have normalized mantissa
	re(`e[0-9]`),                // positive exponent must be signed
	re(`e[+-]0`),                // exponent must not have leading zeros
	re(`e-[1-6]$`),              // not tiny enough for exponential notation
	re(`e+(.|1.|20)$`),          // not big enough for exponential notation
	re(`^-?0\.0000000`),         // too tiny, should use exponential notation
	re(`^-?[0-9]{22}`),          // too big, should use exponential notation
	re(`[1-9][0-9]{16}[1-9]`),   // too many significant digits in integer
	re(`[1-9][0-9.]{17}[1-9]`),  // too many significant digits in decimal
	// below here for float32 only
	re(`[1-9][0-9]{8}[1-9]`),  // too many significant digits in integer
	re(`[1-9][0-9.]{9}[1-9]`), // too many significant digits in decimal
}

func TestMarshalFloat(t *testing.T) {
	t.Parallel()
	nfail := 0
	test := func(f float64, bits int) {
		vf := any(f)
		if bits == 32 {
			f = float64(float32(f)) // round
			vf = float32(f)
		}
		bout, err := Marshal(vf)
		if err != nil {
			t.Errorf("Marshal(%T(%g)): %v", vf, vf, err)
			nfail++
			return
		}
		out := string(bout)

		// result must convert back to the same float
		g, err := strconv.ParseFloat(out, bits)
		if err != nil {
			t.Errorf("Marshal(%T(%g)) = %q, cannot parse back: %v", vf, vf, out, err)
			nfail++
			return
		}
		if f != g || fmt.Sprint(f) != fmt.Sprint(g) { // fmt.Sprint handles ±0
			t.Errorf("Marshal(%T(%g)) = %q (is %g, not %g)", vf, vf, out, float32(g), vf)
			nfail++
			return
		}

		bad := badFloatREs
		if bits == 64 {
			bad = bad[:len(bad)-2]
		}
		for _, re := range bad {
			if re.MatchString(out) {
				t.Errorf("Marshal(%T(%g)) = %q, must not match /%s/", vf, vf, out, re)
				nfail++
				return
			}
		}
	}

	var (
		bigger  = math.Inf(+1)
		smaller = math.Inf(-1)
	)

	var digits = "1.2345678901234567890123"
	for i := len(digits); i >= 2; i-- {
		for exp := -30; exp <= 30; exp++ {
			for _, sign := range "+-" {
				for bits := 32; bits <= 64; bits += 32 {
					s := fmt.Sprintf("%c%se%d", sign, digits[:i], exp)
					f, err := strconv.ParseFloat(s, bits)
					if err != nil {
						log.Fatal(err)
					}
					next := math.Nextafter
					if bits == 32 {
						next = func(g, h float64) float64 {
							return float64(math.Nextafter32(float32(g), float32(h)))
						}
					}
					test(f, bits)
					test(next(f, bigger), bits)
					test(next(f, smaller), bits)
					if nfail > 50 {
						t.Fatalf("stopping test early")
					}
				}
			}
		}
	}
	test(0, 64)
	test(math.Copysign(0, -1), 64)
	test(0, 32)
	test(math.Copysign(0, -1), 32)
}

func TestMarshalRawMessageValue(t *testing.T) {
	type (
		T1 struct {
			M RawMessage `json:",omitempty"`
		}
		T2 struct {
			M *RawMessage `json:",omitempty"`
		}
	)

	var (
		rawNil   = RawMessage(nil)
		rawEmpty = RawMessage([]byte{})
		rawText  = RawMessage([]byte(`"foo"`))
	)

	tests := []struct {
		in   any
		want string
		ok   bool
	}{
		// Test with nil RawMessage.
		{rawNil, "null", true},
		{&rawNil, "null", true},
		{[]any{rawNil}, "[null]", true},
		{&[]any{rawNil}, "[null]", true},
		{[]any{&rawNil}, "[null]", true},
		{&[]any{&rawNil}, "[null]", true},
		{struct{ M RawMessage }{rawNil}, `{"M":null}`, true},
		{&struct{ M RawMessage }{rawNil}, `{"M":null}`, true},
		{struct{ M *RawMessage }{&rawNil}, `{"M":null}`, true},
		{&struct{ M *RawMessage }{&rawNil}, `{"M":null}`, true},
		{map[string]any{"M": rawNil}, `{"M":null}`, true},
		{&map[string]any{"M": rawNil}, `{"M":null}`, true},
		{map[string]any{"M": &rawNil}, `{"M":null}`, true},
		{&map[string]any{"M": &rawNil}, `{"M":null}`, true},
		{T1{rawNil}, "{}", true},
		{T2{&rawNil}, `{"M":null}`, true},
		{&T1{rawNil}, "{}", true},
		{&T2{&rawNil}, `{"M":null}`, true},

		// Test with empty, but non-nil, RawMessage.
		{rawEmpty, "", false},
		{&rawEmpty, "", false},
		{[]any{rawEmpty}, "", false},
		{&[]any{rawEmpty}, "", false},
		{[]any{&rawEmpty}, "", false},
		{&[]any{&rawEmpty}, "", false},
		{struct{ X RawMessage }{rawEmpty}, "", false},
		{&struct{ X RawMessage }{rawEmpty}, "", false},
		{struct{ X *RawMessage }{&rawEmpty}, "", false},
		{&struct{ X *RawMessage }{&rawEmpty}, "", false},
		{map[string]any{"nil": rawEmpty}, "", false},
		{&map[string]any{"nil": rawEmpty}, "", false},
		{map[string]any{"nil": &rawEmpty}, "", false},
		{&map[string]any{"nil": &rawEmpty}, "", false},
		{T1{rawEmpty}, "{}", true},
		{T2{&rawEmpty}, "", false},
		{&T1{rawEmpty}, "{}", true},
		{&T2{&rawEmpty}, "", false},

		// Test with RawMessage with some text.
		//
		// The tests below marked with Issue6458 used to generate "ImZvbyI=" instead "foo".
		// This behavior was intentionally changed in Go 1.8.
		// See https://github.com/golang/go/issues/14493#issuecomment-255857318
		{rawText, `"foo"`, true}, // Issue6458
		{&rawText, `"foo"`, true},
		{[]any{rawText}, `["foo"]`, true},  // Issue6458
		{&[]any{rawText}, `["foo"]`, true}, // Issue6458
		{[]any{&rawText}, `["foo"]`, true},
		{&[]any{&rawText}, `["foo"]`, true},
		{struct{ M RawMessage }{rawText}, `{"M":"foo"}`, true}, // Issue6458
		{&struct{ M RawMessage }{rawText}, `{"M":"foo"}`, true},
		{struct{ M *RawMessage }{&rawText}, `{"M":"foo"}`, true},
		{&struct{ M *RawMessage }{&rawText}, `{"M":"foo"}`, true},
		{map[string]any{"M": rawText}, `{"M":"foo"}`, true},  // Issue6458
		{&map[string]any{"M": rawText}, `{"M":"foo"}`, true}, // Issue6458
		{map[string]any{"M": &rawText}, `{"M":"foo"}`, true},
		{&map[string]any{"M": &rawText}, `{"M":"foo"}`, true},
		{T1{rawText}, `{"M":"foo"}`, true}, // Issue6458
		{T2{&rawText}, `{"M":"foo"}`, true},
		{&T1{rawText}, `{"M":"foo"}`, true},
		{&T2{&rawText}, `{"M":"foo"}`, true},
	}

	for i, tt := range tests {
		b, err := Marshal(tt.in)
		if ok := (err == nil); ok != tt.ok {
			if err != nil {
				t.Errorf("test %d, unexpected failure: %v", i, err)
			} else {
				t.Errorf("test %d, unexpected success", i)
			}
		}
		if got := string(b); got != tt.want {
			t.Errorf("test %d, Marshal(%#v) = %q, want %q", i, tt.in, got, tt.want)
		}
	}
}

type ObjEnc struct {
	A int
	B OrderedObject
	C any
}

var orderedObjectTests = []struct {
	in  any
	out string
}{
	{OrderedObject{}, `{}`},
	{OrderedObject{Member{"A", []int{1, 2, 3}}, Member{"B", 23}, Member{"C", "C"}}, `{"A":[1,2,3],"B":23,"C":"C"}`},
	{ObjEnc{A: 234, B: OrderedObject{Member{"K", "V"}, Member{"V", 4}}}, `{"A":234,"B":{"K":"V","V":4},"C":null}`},
	{ObjEnc{A: 234, B: OrderedObject{}, C: OrderedObject{{"A", 0}}}, `{"A":234,"B":{},"C":{"A":0}}`},
	{[]OrderedObject{{{"A", "Ay"}, {"B", "Bee"}}, {{"A", "Nay"}}}, `[{"A":"Ay","B":"Bee"},{"A":"Nay"}]`},
	{map[string]OrderedObject{"A": {{"A", "Ay"}, {"B", "Bee"}}}, `{"A":{"A":"Ay","B":"Bee"}}`},
	{map[string]any{"A": OrderedObject{{"A", "Ay"}, {"B", "Bee"}}}, `{"A":{"A":"Ay","B":"Bee"}}`},
}

func TestEncodeOrderedObject(t *testing.T) {
	for i, o := range orderedObjectTests {
		d, err := Marshal(o.in)
		if err != nil {
			t.Errorf("Unexpected error %v", err)
			continue
		}
		ds := string(d)
		if o.out != ds {
			t.Errorf("#%d expected '%v', was '%v'", i, o.out, ds)
		}
	}
}
