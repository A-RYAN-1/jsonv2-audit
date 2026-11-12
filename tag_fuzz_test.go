package jsonv2audit

import (
	"strings"
	"testing"
)

func FuzzParseTag(f *testing.F) {
	for _, s := range []string{
		"", "name", "name,omitempty", ",omitempty", "-", "-,", "a,b,c",
		"n, string , omitzero", "n,,,x", "n,format:RFC3339", " ,", ",,", "a\x00,b",
		"ünï,omitempty", "n,omitempty,omitempty",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, tag string) {
		name, opts := parseTag(tag)

		// The name is everything before the first comma, untouched.
		want, rest, _ := strings.Cut(tag, ",")
		if name != want {
			t.Fatalf("name %q, want %q", name, want)
		}
		// Every option is non-empty, trimmed, comma-free and came from the tag.
		for o := range opts {
			if o == "" || o != strings.TrimSpace(o) || strings.Contains(o, ",") {
				t.Fatalf("bad option %q from %q", o, tag)
			}
			if !strings.Contains(rest, o) {
				t.Fatalf("option %q not in %q", o, rest)
			}
		}
		// Every non-blank comma-separated piece is reported.
		for _, p := range strings.Split(rest, ",") {
			if p = strings.TrimSpace(p); p != "" && !opts[p] {
				t.Fatalf("option %q of %q dropped", p, tag)
			}
		}
		// Without a comma there are no options.
		if !strings.Contains(tag, ",") && len(opts) != 0 {
			t.Fatalf("options %v from comma-free %q", opts, tag)
		}
	})
}
