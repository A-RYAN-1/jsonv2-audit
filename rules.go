package jsonv2audit

// Rule is one behaviour difference between encoding/json (v1) and
// encoding/json/v2 that can be decided from the types alone.
type Rule struct {
	// ID is the diagnostic category, stable across releases.
	ID string
	// Encode and Decode say on which paths the rule applies.
	Encode, Decode bool
	// Summary is the difference, written as "v1 does X; v2 does Y".
	Summary string
	// Option is the Go expression that keeps the v1 behaviour when the code
	// moves to v2, with json meaning encoding/json and jsonv2 meaning
	// encoding/json/v2. Empty when no option exists and the code must change.
	Option string
	// Folded rules fire on most decoded structs, so the text output counts
	// them per package (with FoldedNote) instead of listing them; -all lists
	// them.
	Folded     bool
	FoldedNote string
}

// Rules lists every rule the analyzer reports, in a stable order.
var Rules = []Rule{
	{
		ID: "nil-slice", Encode: true,
		Summary: "a nil slice encodes as null in v1 and [] in v2",
		Option:  "jsonv2.FormatNilSliceAsNull(true)",
	},
	{
		ID: "nil-map", Encode: true,
		Summary: "a nil map encodes as null in v1 and {} in v2",
		Option:  "jsonv2.FormatNilMapAsNull(true)",
	},
	{
		ID: "nil-bytes", Encode: true,
		Summary: `a nil []byte encodes as null in v1 and "" in v2`,
		Option:  "jsonv2.FormatNilSliceAsNull(true)",
	},
	{
		ID: "byte-array", Encode: true, Decode: true,
		Summary: "a [N]byte is a JSON array of numbers in v1 and a base64 string in v2",
		Option:  "json.FormatByteArrayAsArray(true)",
	},
	{
		ID: "named-byte-slice", Encode: true, Decode: true,
		Summary: "a slice of a named byte type is base64 in v1 and a JSON array in v2",
		Option:  "json.FormatBytesWithLegacySemantics(true)",
	},
	{
		ID: "omitempty", Encode: true,
		Summary: "omitempty omits false and 0 in v1 but keeps them in v2 (omitzero omits them in both)",
		Option:  "json.OmitEmptyWithLegacySemantics(true)",
	},
	{
		ID: "omitempty-struct", Encode: true,
		Summary: "omitempty never omits a struct (or a non-nil pointer to one) in v1, and omits it in v2 when it encodes as {}",
		Option:  "json.OmitEmptyWithLegacySemantics(true)",
	},
	{
		ID: "string-tag", Encode: true, Decode: true,
		Summary: "the string tag option quotes bools and strings in v1 and is an error on them in v2",
		Option:  "json.StringifyWithLegacySemantics(true)",
	},
	{
		ID: "string-tag-ignored", Encode: true, Decode: true,
		Summary: "the string tag option is ignored on slices, maps, structs and interfaces in v1 and is an error in v2 (better: remove the option)",
		Option:  "json.ReportErrorsWithLegacySemantics(true)",
	},
	{
		ID: "format-tag", Encode: true, Decode: true,
		Summary: "the format tag option was removed in Go 1.27 and fails at run time in v1 and v2; remove it or use a custom marshaler",
	},
	{
		ID: "duration", Encode: true, Decode: true,
		Summary: "time.Duration is integer nanoseconds in v1 and has no default representation in v2 (run-time error)",
		Option:  "json.FormatDurationAsNano(true)",
	},
	{
		ID: "ptr-method", Encode: true,
		Summary: "a marshal method on a pointer receiver is skipped for non-addressable values (map values, values passed directly) in v1 and always called in v2",
		Option:  "json.CallMethodsWithLegacySemantics(true)",
	},
	{
		ID: "case-insensitive", Decode: true, Folded: true,
		FoldedNote: "decoded struct type(s) rely on case-insensitive name matching",
		Summary:    "object names match field names case-insensitively in v1 and exactly in v2 (or tag a field with case:ignore)",
		Option:     "jsonv2.MatchCaseInsensitiveNames(true)",
	},
	{
		ID: "array-length", Decode: true,
		Summary: "a JSON array of any length fills a Go array in v1 (extra elements dropped, missing ones left alone) and must match its length exactly in v2",
		Option:  "json.UnmarshalArrayFromAnyLength(true)",
	},
	{
		ID: "null-resets", Decode: true, Folded: true,
		FoldedNote: "decoded struct type(s) have a field that v2 resets on JSON null",
		Summary:    "a JSON null leaves a struct, number, string or bool field unchanged in v1 and resets it to its zero value in v2 (matters when decoding into a value that already holds data)",
		Option:     "json.MergeWithLegacySemantics(true)",
	},
	{
		ID: "duplicate-name", Encode: true, Decode: true,
		Summary: "two fields of one struct with the same JSON name are both silently ignored in v1 and an error in v2",
		Option:  "json.ReportErrorsWithLegacySemantics(true)",
	},
}

var ruleByID = func() map[string]Rule {
	m := make(map[string]Rule, len(Rules))
	for _, r := range Rules {
		m[r.ID] = r
	}
	return m
}()

// RuleByID returns the rule with the given diagnostic category.
func RuleByID(id string) (Rule, bool) {
	r, ok := ruleByID[id]
	return r, ok
}
