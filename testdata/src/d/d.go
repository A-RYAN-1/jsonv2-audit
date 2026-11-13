// Package d holds the cases that the mutation tester showed were not pinned
// down: embedded structs, pointer and slice addressability, custom methods
// in one direction only, variables holding DefaultOptionsV1 and calls that
// are not encoding/json roots.
package d

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"fmt"
)

// ptrM has pointer-receiver marshal methods: skipped by v1 for values that
// are not addressable, so it is reported only where the value is not.
type ptrM struct{ X int }

func (p *ptrM) MarshalJSON() ([]byte, error) { return []byte(`1`), nil }

// A root pointer, a pointer field and a slice element are addressable.
type AddrRoot struct{ V ptrM }
type AddrPtrField struct{ P *AddrRoot }
type AddrSlice struct{ S []ptrM } // want `AddrSlice.S: a nil slice encodes as null`

// An embedded pointer makes its fields addressable.
type embPtr struct{ V ptrM }
type EmbPtr struct{ *embPtr }

// A value root is not.
type ValRoot struct{ V ptrM } // want `ValRoot.V: a marshal method on a pointer receiver`

// Calls that are not roots must be ignored.
type Ignored struct{ S []int }

func calls() {
	_, _ = json.Marshal(&AddrRoot{})
	_, _ = json.Marshal(AddrPtrField{})
	_, _ = json.Marshal(AddrSlice{S: []ptrM{{}}})
	_, _ = json.Marshal(EmbPtr{})
	_, _ = json.Marshal(ValRoot{})
	fmt.Println(Ignored{})
	_ = fmt.Sprint(Ignored{})
}

// A named slice type with a pointer-receiver method is custom: no nil rule.
type PL []int

func (p *PL) MarshalJSON() ([]byte, error) { return []byte(`1`), nil }

type PtrSliceField struct {
	L PL // want `PtrSliceField.L: a marshal method on a pointer receiver`
}

// Methods in one direction only.
type OnlyDec []int

func (OnlyDec) UnmarshalJSON([]byte) error { return nil }

type EncodesOnlyDec struct {
	D OnlyDec `json:"d"` // want `EncodesOnlyDec.D: a nil slice encodes as null`
}

type OnlyEnc [3]int

func (OnlyEnc) MarshalJSON() ([]byte, error) { return []byte(`1`), nil }

type DecodesOnlyEnc struct { // want `DecodesOnlyEnc: object names match field names case-insensitively` `DecodesOnlyEnc: a JSON null leaves`
	E OnlyEnc `json:"e"` // want `DecodesOnlyEnc.E: a JSON array of any length`
}

// A type that unmarshals itself through a pointer is not walked on decode.
type selfDecoding struct{ A [3]int }

func (s *selfDecoding) UnmarshalJSON([]byte) error { return nil }

type HoldsSelfDecoding struct { // want `HoldsSelfDecoding: object names match field names case-insensitively` `HoldsSelfDecoding: a JSON null leaves`
	S selfDecoding `json:"s"`
}

// Embedded structs: promoted by default, ordinary fields when tagged, and a
// named non-struct embedded type is an ordinary field.
type Tags []string

type EmbNonStruct struct {
	Tags // want `EmbNonStruct.Tags: a nil slice encodes as null`
}

type Inner struct { // want `Inner: object names match field names case-insensitively`
	Names []string `json:"names"` // want `Inner.Names: a nil slice encodes as null`
}

type EmbTagged struct {
	Inner `json:"inner"`
}

// A promoted struct is not a member of the outer struct, so it neither
// collides with a field of the same name nor makes the outer struct reset on null.
type EmbNoCollision struct { // want `EmbNoCollision: object names match field names case-insensitively` `EmbNoCollision: a JSON null leaves`
	Inner
	Other int `json:"Inner"`
}

type EmbOnly struct { // want `EmbOnly: object names match field names case-insensitively`
	Inner
}

// Unexported and omitted fields do not count.
type OnlyPrivate struct { // want `OnlyPrivate: object names match field names case-insensitively`
	n    int
	Skip int `json:"-"`
	Tags []string
}

type NoFields struct {
	a int
	B int `json:"-"`
}

func decodes() {
	_ = json.Unmarshal(nil, new(DecodesOnlyEnc))
	_ = json.Unmarshal(nil, new(HoldsSelfDecoding))
	_ = json.Unmarshal(nil, new(EmbNoCollision))
	_ = json.Unmarshal(nil, new(EmbOnly))
	_ = json.Unmarshal(nil, new(OnlyPrivate))
	_ = json.Unmarshal(nil, new(NoFields))
	_, _ = json.Marshal(EncodesOnlyDec{})
	_, _ = json.Marshal(PtrSliceField{})
	_, _ = json.Marshal(EmbNonStruct{})
	_, _ = json.Marshal(EmbTagged{})
}

// omitempty: a custom marshaler decides its own output; unexported fields
// are ignored when deciding whether a struct can be omitted; a struct with
// an omitempty number cannot.
type Flag bool

func (Flag) MarshalJSON() ([]byte, error) { return []byte(`1`), nil }

type FlagField struct {
	F Flag `json:"f,omitempty"`
}

type custom struct{}

func (custom) MarshalJSON() ([]byte, error) { return []byte(`1`), nil }

type CustomPtr struct {
	P *custom `json:"p,omitempty"`
}

type quiet struct {
	A string `json:"a,omitempty"`
	x int
}

type OmitWithPrivate struct {
	Q quiet `json:"q,omitempty"` // want `OmitWithPrivate.Q: omitempty never omits a struct`
}

type withNumber struct {
	N int `json:"n,omitempty"` // want `withNumber.N: omitempty omits false and 0`
}

type OmitNumberStruct struct {
	W withNumber `json:"w,omitempty"`
}

// Two fields with one name make a struct that v1 drops entirely, so
// omitempty on it can still omit it in v2.
type dupAll struct { // want `dupAll: two fields of one struct`
	A int `json:"x"`
	B int `json:"x"`
}

type OmitDup struct {
	D dupAll `json:"d,omitempty"` // want `OmitDup.D: omitempty never omits a struct`
}

func encodes() {
	_, _ = json.Marshal(FlagField{})
	_, _ = json.Marshal(CustomPtr{})
	_, _ = json.Marshal(OmitWithPrivate{})
	_, _ = json.Marshal(OmitNumberStruct{})
	_, _ = json.Marshal(OmitDup{})
}

// DefaultOptionsV1 variables: followed only when written once, by
// initialisation; a parameter that is assigned, or a variable whose address
// is taken, is not.
func viaParameter(o jsonv2.Options) {
	o = json.DefaultOptionsV1()
	_, _ = jsonv2.Marshal(Ignored{}, o)
}

func viaAddress() {
	o := json.DefaultOptionsV1()
	p := &o
	_ = p
	_, _ = jsonv2.Marshal(Ignored{}, o)
}

func viaInit() {
	o := json.DefaultOptionsV1()
	_, _ = jsonv2.Marshal(Ignored{}, o) // want `Ignored: Marshal with json.DefaultOptionsV1\(\)`
}

// A rule with no option says so and does not suggest one.
type FormatOnly struct {
	T int `json:"t,format:x"` // want `FormatOnly.T: the format tag option was removed in Go 1.27 and fails at run time in v1 and v2; remove it or use a custom marshaler$`
}

func format() { _, _ = json.Marshal(FormatOnly{}) }
