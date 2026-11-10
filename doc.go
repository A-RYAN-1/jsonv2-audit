// Package jsonv2audit is a go/analysis analyzer that finds the places where
// a module's encoding/json behaviour changes when it moves to
// encoding/json/v2 (Go 1.27), and names the option that keeps the v1
// behaviour.
//
// The analyzer starts at every call to json.Marshal, json.MarshalIndent,
// json.Unmarshal, (*json.Encoder).Encode and (*json.Decoder).Decode, follows
// the static type of the value through pointers, slices, arrays, maps,
// struct fields and embedded structs, and stops at types with their own
// marshal or unmarshal methods and at interfaces, whose dynamic type is not
// known. Because it follows types from the call sites, a type defined in
// one package and marshaled in another is still checked; the finding is
// reported at the field when the field is in the analyzed package, and at
// the call otherwise.
//
// Each finding has a category from [Rules]. Every rule is proved by a test
// that marshals a real value with both versions, checks that the outputs
// differ exactly when the analyzer reports, and checks that the rule's
// option makes v2 produce the v1 output.
//
// What it cannot see: values behind interfaces, invalid UTF-8 and duplicate
// object names in input (v2 rejects both), and JSON whose key case differs
// from the field names. For those, run the go-json-experiment/jsonsplit
// wrapper on real traffic.
package jsonv2audit
