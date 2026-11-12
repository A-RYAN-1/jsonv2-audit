package jsonv2audit_test

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"

	jsonv2audit "github.com/A-RYAN-1/jsonv2-audit"
)

// Octet is the named byte type the generated source also declares.
type Octet byte

// The named types below are declared again, with the same methods, in the
// generated source, so the analyzer and the reflect-built values agree.

// PtrM has pointer-receiver JSON methods.
type PtrM struct{ X int }

func (p *PtrM) MarshalJSON() ([]byte, error) { return []byte(`"pm"`), nil }
func (p *PtrM) UnmarshalJSON(b []byte) error { p.X = len(b); return nil }

// ValM has a value-receiver text marshaler.
type ValM struct{ X int }

func (v ValM) MarshalText() ([]byte, error)  { return []byte("vm"), nil }
func (v *ValM) UnmarshalText(b []byte) error { v.X = len(b); return nil }

// Opt can be omitted entirely: every field is omitempty or omitzero.
type Opt struct {
	A string `json:"a,omitempty"`
	B []int  `json:"b,omitzero"`
}

// Base is embedded into generated structs.
type Base struct {
	A []int `json:"a"`
	B bool  `json:"b,omitempty"`
}

const prelude = `type Octet byte

type PtrM struct{ X int }

func (p *PtrM) MarshalJSON() ([]byte, error) { return []byte(` + "`\"pm\"`" + `), nil }
func (p *PtrM) UnmarshalJSON(b []byte) error { p.X = len(b); return nil }

type ValM struct{ X int }

func (v ValM) MarshalText() ([]byte, error) { return []byte("vm"), nil }
func (v *ValM) UnmarshalText(b []byte) error { v.X = len(b); return nil }

type Opt struct {
	A string ` + "`json:\"a,omitempty\"`" + `
	B []int  ` + "`json:\"b,omitzero\"`" + `
}

type Base struct {
	A []int ` + "`json:\"a\"`" + `
	B bool  ` + "`json:\"b,omitempty\"`" + `
}

`

// genField is one generated type: its Go source, its reflect type and a
// populated value of it.
type genField struct {
	src  string
	typ  reflect.Type
	full reflect.Value
}

var scalarKinds = []func() genField{
	func() genField { return genField{"int", reflect.TypeOf(0), reflect.ValueOf(7)} },
	func() genField { return genField{"bool", reflect.TypeOf(false), reflect.ValueOf(true)} },
	func() genField { return genField{"string", reflect.TypeOf(""), reflect.ValueOf("x")} },
	func() genField { return genField{"float64", reflect.TypeOf(0.0), reflect.ValueOf(1.5)} },
	func() genField { return genField{"[]int", reflect.TypeOf([]int{}), reflect.ValueOf([]int{1, 2})} },
	func() genField { return genField{"[]byte", reflect.TypeOf([]byte{}), reflect.ValueOf([]byte("hi"))} },
	func() genField {
		return genField{"[4]byte", reflect.TypeOf([4]byte{}), reflect.ValueOf([4]byte{1, 2, 3, 4})}
	},
	func() genField { return genField{"[]Octet", reflect.TypeOf([]Octet{}), reflect.ValueOf([]Octet{1, 2})} },
	func() genField {
		return genField{"map[string]int", reflect.TypeOf(map[string]int{}), reflect.ValueOf(map[string]int{"k": 1})}
	},
	func() genField { return genField{"*int", reflect.TypeOf((*int)(nil)), reflect.ValueOf(new(int))} },
	func() genField {
		return genField{"time.Duration", reflect.TypeOf(time.Duration(0)), reflect.ValueOf(time.Second)}
	},
	func() genField {
		return genField{"[]time.Duration", reflect.TypeOf([]time.Duration{}), reflect.ValueOf([]time.Duration{time.Second})}
	},
	func() genField {
		return genField{"time.Time", reflect.TypeOf(time.Time{}), reflect.ValueOf(time.Unix(0, 0).UTC())}
	},
	func() genField { return genField{"PtrM", reflect.TypeOf(PtrM{}), reflect.ValueOf(PtrM{X: 1})} },
	func() genField { return genField{"*PtrM", reflect.TypeOf(&PtrM{}), reflect.ValueOf(&PtrM{X: 1})} },
	func() genField {
		return genField{"map[string]PtrM", reflect.TypeOf(map[string]PtrM{}), reflect.ValueOf(map[string]PtrM{"k": {X: 1}})}
	},
	func() genField { return genField{"[]PtrM", reflect.TypeOf([]PtrM{}), reflect.ValueOf([]PtrM{{X: 1}})} },
	func() genField { return genField{"ValM", reflect.TypeOf(ValM{}), reflect.ValueOf(ValM{X: 1})} },
	func() genField {
		return genField{"Opt", reflect.TypeOf(Opt{}), reflect.ValueOf(Opt{A: "x", B: []int{1}})}
	},
	func() genField { return genField{"*Opt", reflect.TypeOf(&Opt{}), reflect.ValueOf(&Opt{})} },
	func() genField {
		return genField{"json.RawMessage", reflect.TypeOf(json.RawMessage{}), reflect.ValueOf(json.RawMessage("1"))}
	},
}

// genStruct builds a random struct type and its source, nesting up to depth.
func genStruct(r *rand.Rand, depth int) genField {
	n := 1 + r.Intn(4)
	var fields []reflect.StructField
	var src strings.Builder
	src.WriteString("struct {\n")
	var full []reflect.Value
	embedded := false
	for i := 0; i < n; i++ {
		if !embedded && r.Intn(8) == 0 {
			embedded = true
			base := Base{A: []int{1}, B: true}
			if r.Intn(2) == 0 {
				fields = append(fields, reflect.StructField{Name: "Base", Type: reflect.TypeOf(base), Anonymous: true})
				src.WriteString("\tBase\n")
				full = append(full, reflect.ValueOf(base))
			} else {
				fields = append(fields, reflect.StructField{Name: "Base", Type: reflect.TypeOf(&base), Anonymous: true})
				src.WriteString("\t*Base\n")
				full = append(full, reflect.ValueOf(&base))
			}
			continue
		}
		var f genField
		if depth > 0 && r.Intn(5) == 0 {
			f = genStruct(r, depth-1)
			if r.Intn(2) == 0 {
				p := reflect.New(f.typ)
				p.Elem().Set(f.full)
				f = genField{"*" + f.src, reflect.PointerTo(f.typ), p}
			}
		} else {
			f = scalarKinds[r.Intn(len(scalarKinds))]()
		}
		var opts []string
		for _, o := range []string{"omitempty", "omitzero", "string"} {
			if r.Intn(4) == 0 {
				opts = append(opts, o)
			}
		}
		tag := fmt.Sprintf("f%d", i)
		if r.Intn(12) == 0 {
			tag, opts = "-", nil
		}
		if len(opts) > 0 {
			tag += "," + strings.Join(opts, ",")
		}
		name := fmt.Sprintf("F%d", i)
		fields = append(fields, reflect.StructField{Name: name, Type: f.typ, Tag: reflect.StructTag(`json:"` + tag + `"`)})
		fmt.Fprintf(&src, "\t%s %s `json:\"%s\"`\n", name, f.src, tag)
		full = append(full, f.full)
	}
	src.WriteString("}")
	typ := reflect.StructOf(fields)
	v := reflect.New(typ).Elem()
	for i, fv := range full {
		v.Field(i).Set(fv)
	}
	return genField{src.String(), typ, v}
}

type generated struct {
	dir   string
	types []genField
}

// generate writes n random types as a module in a temp dir.
func generate(t testing.TB, seed int64, n int) generated {
	t.Helper()
	r := rand.New(rand.NewSource(seed))
	g := generated{dir: t.TempDir()}
	var src strings.Builder
	src.WriteString("package gen\n\nimport (\n\t\"encoding/json\"\n\t\"time\"\n)\n\nvar _ time.Duration\n\n")
	src.WriteString(prelude)
	for i := 0; i < n; i++ {
		f := genStruct(r, 1)
		g.types = append(g.types, f)
		fmt.Fprintf(&src, "type T%d %s\n\n", i, f.src)
	}
	src.WriteString("func roots() {\n")
	for i := range g.types {
		fmt.Fprintf(&src, "\t_, _ = json.Marshal(T%d{})\n\t_ = json.Unmarshal(nil, new(T%d))\n", i, i)
	}
	src.WriteString("}\n\nvar _ = roots\n")
	must(t, os.WriteFile(filepath.Join(g.dir, "go.mod"), []byte("module gen\n\ngo 1.27\n"), 0o644))
	must(t, os.WriteFile(filepath.Join(g.dir, "gen.go"), []byte(src.String()), 0o644))
	return g
}

func must(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func analyze(t testing.TB, dir string) ([]*packages.Package, map[string]map[string]bool) {
	t.Helper()
	pkgs, err := packages.Load(&packages.Config{Mode: packages.LoadAllSyntax, Dir: dir}, "./...")
	must(t, err)
	if packages.PrintErrors(pkgs) > 0 {
		src, _ := os.ReadFile(filepath.Join(dir, "gen.go"))
		t.Fatalf("generated module does not load:\n%s", src)
	}
	return pkgs, report(t, pkgs)
}

func report(t testing.TB, pkgs []*packages.Package) map[string]map[string]bool {
	t.Helper()
	g, err := checker.Analyze([]*analysis.Analyzer{jsonv2audit.Analyzer}, pkgs, nil)
	must(t, err)
	got := map[string]map[string]bool{}
	for _, act := range g.Roots {
		must(t, act.Err)
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

var upperKey = regexp.MustCompile(`"(f\d+)":`)

// TestPropertyPredictionsMatchEncoding generates random struct types and
// checks, against the real v1 and v2 output, that (1) when the analyzer
// reports nothing for a path, v1 and v2 agree, and (2) when it reports, the
// options it suggests make v2 agree with v1.
func TestPropertyPredictionsMatchEncoding(t *testing.T) {
	seeds := 3
	if testing.Short() {
		seeds = 1
	}
	if v, err := strconv.Atoi(os.Getenv("JSONV2AUDIT_SEEDS")); err == nil && v > 0 {
		seeds = v
	}
	for seed := int64(1); seed <= int64(seeds); seed++ {
		g := generate(t, seed, 150)
		_, got := analyze(t, g.dir)
		for i, f := range g.types {
			name := fmt.Sprintf("T%d", i)
			// Findings inside a named helper type are reported once, under the
			// helper's name; they apply to every generated type that uses it.
			found := map[string]bool{}
			for id := range got[name] {
				found[id] = true
			}
			for _, h := range []string{"Base", "Opt", "PtrM", "ValM"} {
				if strings.Contains(f.src, h) {
					for id := range got[h] {
						found[id] = true
					}
				}
			}
			encIDs := rulesFor(found, false)
			decIDs := rulesFor(found, true)
			zero := reflect.New(f.typ).Elem()
			for label, v := range map[string]reflect.Value{"zero": zero, "full": f.full} {
				b1, e1 := json.Marshal(v.Interface())
				b2, e2 := jsonv2.Marshal(v.Interface())
				differs := string(b1) != string(b2) || (e1 == nil) != (e2 == nil)
				if differs && len(encIDs) == 0 {
					t.Errorf("seed %d %s (%s) encode: analyzer reported nothing but v1 gave %s (%v) and v2 gave %s (%v)\n%s", seed, name, label, b1, e1, b2, e2, f.src)
					continue
				}
				b3, e3 := jsonv2.Marshal(v.Interface(), compat(encIDs)...)
				if string(b3) != string(b1) || (e1 == nil) != (e3 == nil) {
					t.Errorf("seed %d %s (%s) encode: with %v v2 gave %s (%v), v1 gave %s (%v)\n%s", seed, name, label, encIDs, b3, e3, b1, e1, f.src)
				}
				if e1 != nil {
					continue
				}
				for variant, in := range map[string]string{"plain": string(b1), "upper": upperKey.ReplaceAllStringFunc(string(b1), strings.ToUpper)} {
					v1, v2, v3 := reflect.New(f.typ), reflect.New(f.typ), reflect.New(f.typ)
					d1 := json.Unmarshal([]byte(in), v1.Interface())
					d2 := jsonv2.Unmarshal([]byte(in), v2.Interface())
					dDiffers := !reflect.DeepEqual(v1.Interface(), v2.Interface()) || (d1 == nil) != (d2 == nil)
					if dDiffers && len(decIDs) == 0 {
						t.Errorf("seed %d %s (%s, %s) decode of %s: analyzer reported nothing but v1 gave %+v (%v) and v2 gave %+v (%v)\n%s", seed, name, label, variant, in, v1.Interface(), d1, v2.Interface(), d2, f.src)
						continue
					}
					d3 := jsonv2.Unmarshal([]byte(in), v3.Interface(), compat(decIDs)...)
					if !reflect.DeepEqual(v1.Interface(), v3.Interface()) || (d1 == nil) != (d3 == nil) {
						t.Errorf("seed %d %s (%s, %s) decode of %s: with %v v2 gave %+v (%v), v1 gave %+v (%v)\n%s", seed, name, label, variant, in, decIDs, v3.Interface(), d3, v1.Interface(), d1, f.src)
					}
				}
			}
		}
		if t.Failed() {
			return
		}
		seen := map[string]bool{}
		for _, ids := range got {
			for id := range ids {
				seen[id] = true
			}
		}
		var missing []string
		for _, r := range jsonv2audit.Rules {
			if !seen[r.ID] && r.ID != "format-tag" && r.ID != "ptr-method" {
				missing = append(missing, r.ID)
			}
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			t.Logf("seed %d: generator never triggered %v", seed, missing)
		}
	}
}

// BenchmarkAnalyzeGenerated measures the analyzer over a generated module of
// 400 struct types (800 call sites), type-checked once outside the timer.
func BenchmarkAnalyzeGenerated(b *testing.B) {
	g := generate(b, 99, 400)
	pkgs, _ := analyze(b, g.dir)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		report(b, pkgs)
	}
}
