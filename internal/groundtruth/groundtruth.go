// Package groundtruth holds one type per rule plus clean types. The proof
// test runs the analyzer over this package and checks each prediction
// against what encoding/json and encoding/json/v2 really do with the values
// in Encode and Decode.
package groundtruth

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"time"
)

// Types that differ, one rule each (the type name is the case name).

type NilSlice struct {
	S []int `json:"s"`
}

type NilMap struct {
	M map[string]int `json:"m"`
}

type NilBytes struct {
	B []byte `json:"b"`
}

type ByteArray struct {
	A [4]byte `json:"a"`
}

type Octet byte

type NamedByteSlice struct {
	O []Octet `json:"o"`
}

type OmitBool struct {
	B bool `json:"b,omitempty"`
}

type OmitInt struct {
	N int `json:"n,omitempty"`
}

type allOptional struct {
	A string `json:"a,omitempty"`
}

type OmitStruct struct {
	S allOptional `json:"s,omitempty"`
}

type OmitPtr struct {
	P *allOptional `json:"p,omitempty"`
}

type StringBool struct {
	B bool `json:"b,string"`
}

type StringSlice struct {
	S []int `json:"s,string,omitempty"` //nolint:staticcheck // SA5008: the invalid tag is the case under test
}

type FormatTag struct {
	T time.Time `json:"t,format:RFC3339"`
}

type Duration struct {
	D time.Duration `json:"d"`
}

type DurationSlice struct {
	D []time.Duration `json:"d,omitempty"`
}

type ptrMarshaler struct{ X int }

func (p *ptrMarshaler) MarshalJSON() ([]byte, error) { return []byte(`"custom"`), nil }

type PtrMethod struct {
	M map[string]ptrMarshaler `json:"m,omitempty"`
}

type CaseFold struct {
	Name string `json:"name"`
}

type ArrayLength struct {
	A [3]int `json:"a"`
}

type nullInner struct {
	A int `json:"a"`
	B int `json:"b"`
}

type NullResets struct {
	I nullInner `json:"i"`
	N int       `json:"n"`
	S string    `json:"s"`
}

type DuplicateName struct {
	X int
	B int `json:"X"`
}

// Clean types: the analyzer must report nothing, and v1 and v2 must agree.

type CleanScalars struct {
	S  string    `json:"s,omitempty"`
	N  int       `json:"n"`
	F  float64   `json:"f,string"`
	B  bool      `json:"b"`
	T  time.Time `json:"t"`
	P  *int      `json:"p,omitempty"`
	Z  int       `json:"z,omitzero"`
	Ig []int     `json:"-"`
}

type CleanCollections struct {
	S []int          `json:"s,omitempty"`
	M map[string]int `json:"m,omitempty"`
	B []byte         `json:"b,omitzero"`
	R json.RawMessage
}

type CleanPtrMethod struct {
	P *ptrMarshaler `json:"p"`
}

// Encode maps a case name to the value that is marshaled.
var Encode = map[string]any{
	"NilSlice":         NilSlice{},
	"NilMap":           NilMap{},
	"NilBytes":         NilBytes{},
	"ByteArray":        ByteArray{A: [4]byte{1, 2, 3, 4}},
	"NamedByteSlice":   NamedByteSlice{O: []Octet{1, 2}},
	"OmitBool":         OmitBool{},
	"OmitInt":          OmitInt{},
	"OmitStruct":       OmitStruct{},
	"OmitPtr":          OmitPtr{P: &allOptional{}},
	"StringBool":       StringBool{B: true},
	"StringSlice":      StringSlice{S: []int{1}},
	"Duration":         Duration{D: time.Second},
	"DurationSlice":    DurationSlice{D: []time.Duration{time.Second}},
	"PtrMethod":        PtrMethod{M: map[string]ptrMarshaler{"k": {}}},
	"DuplicateName":    DuplicateName{X: 1, B: 2},
	"CleanScalars":     CleanScalars{S: "x", N: 1, F: 1.5, T: time.Unix(0, 0).UTC()},
	"CleanCollections": CleanCollections{S: []int{1}, M: map[string]int{"a": 1}, B: []byte("hi"), R: json.RawMessage(`{"a":1}`)},
	"CleanPtrMethod":   CleanPtrMethod{P: &ptrMarshaler{}},
}

// Decoder is a decode case: the input and a constructor for the target.
type Decoder struct {
	Input string
	New   func() any
}

// Decode maps a case name to a decode case.
var Decode = map[string]Decoder{
	"CaseFold":   {`{"NAME":"x"}`, func() any { return new(CaseFold) }},
	"ByteArray":  {`{"a":[1,2,3,4]}`, func() any { return new(ByteArray) }},
	"Duration":   {`{"d":1000}`, func() any { return new(Duration) }},
	"StringBool": {`{"b":"true"}`, func() any { return new(StringBool) }},

	"ArrayLength": {`{"a":[1,2]}`, func() any { return new(ArrayLength) }},
	"NullResets": {`{"i":null,"n":null,"s":null}`, func() any {
		return &NullResets{I: nullInner{A: 3, B: 4}, N: 5, S: "s"}
	}},
	"DuplicateName": {`{"x":1}`, func() any { return new(DuplicateName) }},
}

// FormatTag fails in both versions, so it has no value to compare; the proof
// test checks that both report an error.
var FormatTagValue = FormatTag{}

// roots gives the analyzer the same calls the proof test makes.
func roots() {
	_, _ = json.Marshal(NilSlice{})
	_, _ = json.Marshal(NilMap{})
	_, _ = json.Marshal(NilBytes{})
	_, _ = json.Marshal(ByteArray{})
	_, _ = json.Marshal(NamedByteSlice{})
	_, _ = json.Marshal(OmitBool{})
	_, _ = json.Marshal(OmitInt{})
	_, _ = json.Marshal(OmitStruct{})
	_, _ = json.Marshal(OmitPtr{})
	_, _ = json.Marshal(StringBool{})
	_, _ = json.Marshal(StringSlice{})
	_, _ = json.Marshal(FormatTag{})
	_, _ = json.Marshal(Duration{})
	_, _ = json.Marshal(DurationSlice{})
	_, _ = json.Marshal(PtrMethod{})
	_, _ = json.Marshal(CleanScalars{})
	_, _ = json.Marshal(CleanCollections{})
	_, _ = json.Marshal(CleanPtrMethod{})
	_ = json.Unmarshal(nil, new(CaseFold))
	_ = json.Unmarshal(nil, new(ByteArray))
	_ = json.Unmarshal(nil, new(Duration))
	_ = json.Unmarshal(nil, new(StringBool))
	_ = json.Unmarshal(nil, new(ArrayLength))
	_ = json.Unmarshal(nil, new(NullResets))
	_, _ = json.Marshal(DuplicateName{})
	_ = json.Unmarshal(nil, new(DuplicateName))
}

// v2roots gives the analyzer v2 calls that pass DefaultOptionsV1, one per
// case, for the v1-options proof.
func v2roots() {
	o := json.DefaultOptionsV1()
	_, _ = jsonv2.Marshal(NilSlice{}, o)
	_, _ = jsonv2.Marshal(NilMap{}, o)
	_, _ = jsonv2.Marshal(NilBytes{}, o)
	_, _ = jsonv2.Marshal(ByteArray{}, o)
	_, _ = jsonv2.Marshal(NamedByteSlice{}, o)
	_, _ = jsonv2.Marshal(OmitBool{}, o)
	_, _ = jsonv2.Marshal(OmitInt{}, o)
	_, _ = jsonv2.Marshal(OmitStruct{}, o)
	_, _ = jsonv2.Marshal(OmitPtr{}, o)
	_, _ = jsonv2.Marshal(StringBool{}, o)
	_, _ = jsonv2.Marshal(StringSlice{}, o)
	_, _ = jsonv2.Marshal(FormatTag{}, o)
	_, _ = jsonv2.Marshal(Duration{}, o)
	_, _ = jsonv2.Marshal(DurationSlice{}, o)
	_, _ = jsonv2.Marshal(PtrMethod{}, o)
	_, _ = jsonv2.Marshal(CleanScalars{}, o)
	_, _ = jsonv2.Marshal(CleanCollections{}, o)
	_, _ = jsonv2.Marshal(CleanPtrMethod{}, o)
	_ = jsonv2.Unmarshal(nil, new(CaseFold), o)
	_ = jsonv2.Unmarshal(nil, new(ByteArray), o)
	_ = jsonv2.Unmarshal(nil, new(Duration), o)
	_ = jsonv2.Unmarshal(nil, new(StringBool), o)
	_ = jsonv2.Unmarshal(nil, new(ArrayLength), o)
	_ = jsonv2.Unmarshal(nil, new(NullResets), o)
	_, _ = jsonv2.Marshal(DuplicateName{}, o)
	_ = jsonv2.Unmarshal(nil, new(DuplicateName), o)
}

var _ = roots
var _ = v2roots
