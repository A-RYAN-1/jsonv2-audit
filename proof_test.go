package jsonv2audit_test

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"reflect"
	"sort"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"

	jsonv2audit "github.com/A-RYAN-1/jsonv2-audit"
	"github.com/A-RYAN-1/jsonv2-audit/internal/groundtruth"
)

// options is the test's own copy of each rule's v1-keeping option: the
// expression must match Rule.Option, and the value is what the proof applies.
var options = map[string]struct {
	expr string
	opts []jsonv2.Options
}{
	"nil-slice":          {"jsonv2.FormatNilSliceAsNull(true)", []jsonv2.Options{jsonv2.FormatNilSliceAsNull(true)}},
	"nil-map":            {"jsonv2.FormatNilMapAsNull(true)", []jsonv2.Options{jsonv2.FormatNilMapAsNull(true)}},
	"nil-bytes":          {"jsonv2.FormatNilSliceAsNull(true)", []jsonv2.Options{jsonv2.FormatNilSliceAsNull(true)}},
	"byte-array":         {"json.FormatByteArrayAsArray(true)", []jsonv2.Options{json.FormatByteArrayAsArray(true)}},
	"named-byte-slice":   {"json.FormatBytesWithLegacySemantics(true)", []jsonv2.Options{json.FormatBytesWithLegacySemantics(true)}},
	"omitempty":          {"json.OmitEmptyWithLegacySemantics(true)", []jsonv2.Options{json.OmitEmptyWithLegacySemantics(true)}},
	"string-tag":         {"json.StringifyWithLegacySemantics(true)", []jsonv2.Options{json.StringifyWithLegacySemantics(true)}},
	"omitempty-struct":   {"json.OmitEmptyWithLegacySemantics(true)", []jsonv2.Options{json.OmitEmptyWithLegacySemantics(true)}},
	"string-tag-ignored": {"json.ReportErrorsWithLegacySemantics(true)", []jsonv2.Options{json.ReportErrorsWithLegacySemantics(true)}},
	"format-tag":         {"", nil},
	"duration":           {"json.FormatDurationAsNano(true)", []jsonv2.Options{json.FormatDurationAsNano(true)}},
	"ptr-method":         {"json.CallMethodsWithLegacySemantics(true)", []jsonv2.Options{json.CallMethodsWithLegacySemantics(true)}},
	"case-insensitive":   {"jsonv2.MatchCaseInsensitiveNames(true)", []jsonv2.Options{jsonv2.MatchCaseInsensitiveNames(true)}},
	"array-length":       {"json.UnmarshalArrayFromAnyLength(true)", []jsonv2.Options{json.UnmarshalArrayFromAnyLength(true)}},
	"null-resets":        {"json.MergeWithLegacySemantics(true)", []jsonv2.Options{json.MergeWithLegacySemantics(true)}},
	"duplicate-name":     {"json.ReportErrorsWithLegacySemantics(true)", []jsonv2.Options{json.ReportErrorsWithLegacySemantics(true)}},
}

func TestRuleOptionsMatchProof(t *testing.T) {
	n := 0
	for _, r := range jsonv2audit.Rules {
		if !r.Advisory {
			n++
		}
	}
	if len(options) != n {
		t.Fatalf("proof covers %d rules, analyzer has %d", len(options), n)
	}
	for _, r := range jsonv2audit.Rules {
		if r.Advisory {
			continue
		}
		o, ok := options[r.ID]
		if !ok {
			t.Errorf("rule %s has no proof option", r.ID)
			continue
		}
		if o.expr != r.Option {
			t.Errorf("rule %s: Option %q, proof applies %q", r.ID, r.Option, o.expr)
		}
	}
}

// findings runs the analyzer over the groundtruth package and returns the
// rule IDs reported per type name.
func findings(t *testing.T) map[string]map[string]bool {
	t.Helper()
	cfg := &packages.Config{Mode: packages.LoadAllSyntax}
	pkgs, err := packages.Load(cfg, "./internal/groundtruth")
	if err != nil {
		t.Fatal(err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		t.Fatal("groundtruth does not load")
	}
	g, err := checker.Analyze([]*analysis.Analyzer{jsonv2audit.Analyzer}, pkgs, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]map[string]bool{}
	for _, act := range g.Roots {
		if act.Err != nil {
			t.Fatal(act.Err)
		}
		for _, d := range act.Diagnostics {
			typ := d.Message[:strings.IndexAny(d.Message, ".:[")]
			if got[typ] == nil {
				got[typ] = map[string]bool{}
			}
			got[typ][d.Category] = true
		}
	}
	return got
}

func rulesFor(found map[string]bool, decode bool) []string {
	var ids []string
	for id := range found {
		r, _ := jsonv2audit.RuleByID(id)
		if r.Advisory {
			continue
		}
		if (decode && r.Decode) || (!decode && r.Encode) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func compat(ids []string) []jsonv2.Options {
	var o []jsonv2.Options
	for _, id := range ids {
		o = append(o, options[id].opts...)
	}
	return o
}

func TestPredictionsMatchEncoding(t *testing.T) {
	got := findings(t)
	for name, v := range groundtruth.Encode {
		ids := rulesFor(got[name], false)
		b1, e1 := json.Marshal(v)
		b2, e2 := jsonv2.Marshal(v)
		differs := string(b1) != string(b2) || (e1 == nil) != (e2 == nil)
		if differs != (len(ids) > 0) {
			t.Errorf("%s: analyzer reported %v, but v1 gave %s (%v) and v2 gave %s (%v)", name, ids, b1, e1, b2, e2)
			continue
		}
		if len(ids) == 0 {
			continue
		}
		b3, e3 := jsonv2.Marshal(v, compat(ids)...)
		if string(b3) != string(b1) || (e1 == nil) != (e3 == nil) {
			t.Errorf("%s: with %v v2 gave %s (%v), v1 gave %s (%v)", name, ids, b3, e3, b1, e1)
		}
	}
}

func TestPredictionsMatchDecoding(t *testing.T) {
	got := findings(t)
	for name, c := range groundtruth.Decode {
		ids := rulesFor(got[name], true)
		v1, v2, v3 := c.New(), c.New(), c.New()
		e1 := json.Unmarshal([]byte(c.Input), v1)
		e2 := jsonv2.Unmarshal([]byte(c.Input), v2)
		differs := !reflect.DeepEqual(v1, v2) || (e1 == nil) != (e2 == nil)
		if differs != (len(ids) > 0) {
			t.Errorf("%s: analyzer reported %v, but v1 gave %+v (%v) and v2 gave %+v (%v)", name, ids, v1, e1, v2, e2)
			continue
		}
		e3 := jsonv2.Unmarshal([]byte(c.Input), v3, compat(ids)...)
		if !reflect.DeepEqual(v1, v3) || (e1 == nil) != (e3 == nil) {
			t.Errorf("%s: with %v v2 gave %+v (%v), v1 gave %+v (%v)", name, ids, v3, e3, v1, e1)
		}
	}
}

func TestFormatTagFailsInBoth(t *testing.T) {
	got := findings(t)
	if !got["FormatTag"]["format-tag"] {
		t.Fatalf("format-tag not reported: %v", got["FormatTag"])
	}
	if _, err := json.Marshal(groundtruth.FormatTagValue); err == nil {
		t.Error("v1 accepted a format tag")
	}
	if _, err := jsonv2.Marshal(groundtruth.FormatTagValue); err == nil {
		t.Error("v2 accepted a format tag")
	}
}

func TestEveryRuleHasACase(t *testing.T) {
	got := findings(t)
	seen := map[string]bool{}
	for _, ids := range got {
		for id := range ids {
			seen[id] = true
		}
	}
	for _, r := range jsonv2audit.Rules {
		if !seen[r.ID] {
			t.Errorf("rule %s has no groundtruth case", r.ID)
		}
	}
}

// TestV1OptionsAreSufficient checks the v1-options advice against real v2
// output: for every groundtruth call that passes DefaultOptionsV1, v2 with
// only the options the analyzer says the value needs gives the same result
// as v2 with all of DefaultOptionsV1, so the options it lists as not needed
// can be dropped.
func TestV1OptionsAreSufficient(t *testing.T) {
	pkgs, err := packages.Load(&packages.Config{Mode: packages.LoadAllSyntax}, "./internal/groundtruth")
	if err != nil {
		t.Fatal(err)
	}
	g, err := checker.Analyze([]*analysis.Analyzer{jsonv2audit.Analyzer}, pkgs, nil)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string][]jsonv2.Options{}
	for _, o := range options {
		if o.expr != "" {
			byName[strings.SplitN(o.expr, "(", 2)[0]] = o.opts
		}
	}
	checked := 0
	for _, act := range g.Roots {
		for _, d := range act.Diagnostics {
			if d.Category != "v1-options" {
				continue
			}
			typ, rest, _ := strings.Cut(d.Message, ": ")
			fn, _, _ := strings.Cut(rest, " ")
			var opts []jsonv2.Options
			if _, after, ok := strings.Cut(d.Message, "this value needs "); ok {
				list, _, _ := strings.Cut(after, "; pass these instead")
				for _, name := range strings.Split(list, ", ") {
					o, known := byName[name]
					if !known {
						t.Fatalf("%s: unknown option %q in %q", typ, name, d.Message)
					}
					opts = append(opts, o...)
				}
			}
			checked++
			if fn == "Marshal" {
				v := groundtruth.Encode[typ]
				if v == nil {
					continue
				}
				ref, e1 := jsonv2.Marshal(v, json.DefaultOptionsV1())
				got, e2 := jsonv2.Marshal(v, opts...)
				if string(ref) != string(got) || (e1 == nil) != (e2 == nil) {
					t.Errorf("%s encode: DefaultOptionsV1 gave %s (%v), the needed options gave %s (%v)", typ, ref, e1, got, e2)
				}
				continue
			}
			c, ok := groundtruth.Decode[typ]
			if !ok {
				continue
			}
			r1, r2 := c.New(), c.New()
			e1 := jsonv2.Unmarshal([]byte(c.Input), r1, json.DefaultOptionsV1())
			e2 := jsonv2.Unmarshal([]byte(c.Input), r2, opts...)
			if !reflect.DeepEqual(r1, r2) || (e1 == nil) != (e2 == nil) {
				t.Errorf("%s decode: DefaultOptionsV1 gave %+v (%v), the needed options gave %+v (%v)", typ, r1, e1, r2, e2)
			}
		}
	}
	if checked < 20 {
		t.Errorf("only %d v1-options findings checked", checked)
	}
}
