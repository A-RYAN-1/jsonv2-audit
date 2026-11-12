package jsonv2audit

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

const docURL = "https://github.com/A-RYAN-1/jsonv2-audit"

// Analyzer reports the types reached from encoding/json calls whose
// marshaling or unmarshaling changes under encoding/json/v2. Its -skip flag
// takes a comma-separated list of rule IDs not to report; under go vet it
// is spelled -jsonv2audit.skip.
var Analyzer = newAnalyzer(ruleSet{})

// Config selects what an analyzer made by [NewAnalyzer] reports.
type Config struct {
	// Skip lists rule IDs (see [Rules]) that are not reported.
	Skip []string
}

// NewAnalyzer returns an analyzer like [Analyzer] that applies cfg. It is
// the entry point for tools that configure analyzers in code, such as the
// golangci-lint module plugin. It returns an error for an unknown rule ID.
func NewAnalyzer(cfg Config) (*analysis.Analyzer, error) {
	skip := ruleSet{}
	for _, id := range cfg.Skip {
		if err := skip.add(id); err != nil {
			return nil, err
		}
	}
	return newAnalyzer(skip), nil
}

func newAnalyzer(skip ruleSet) *analysis.Analyzer {
	a := &analysis.Analyzer{
		Name:     "jsonv2audit",
		Doc:      "report encoding/json types whose behaviour changes under encoding/json/v2, with the option that keeps v1 behaviour",
		URL:      docURL,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run:      func(pass *analysis.Pass) (any, error) { return run(pass, skip) },
	}
	a.Flags.Var(skip, "skip", "comma-separated rule IDs not to report")
	return a
}

// ruleSet is a set of rule IDs; as a flag.Value it accepts a
// comma-separated list and rejects unknown IDs.
type ruleSet map[string]bool

func (s ruleSet) add(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	if _, ok := ruleByID[id]; !ok {
		return fmt.Errorf("unknown rule %q (jsonv2-audit -rules lists them)", id)
	}
	s[id] = true
	return nil
}

func (s ruleSet) String() string {
	ids := make([]string, 0, len(s))
	for _, r := range Rules {
		if s[r.ID] {
			ids = append(ids, r.ID)
		}
	}
	return strings.Join(ids, ",")
}

func (s ruleSet) Set(v string) error {
	clear(s)
	for _, id := range strings.Split(v, ",") {
		if err := s.add(id); err != nil {
			return err
		}
	}
	return nil
}

type mode int

const (
	encode mode = iota
	decode
)

// roots maps the encoding/json entry points to the index of the argument
// that carries the Go value, and the direction.
var roots = map[string]struct {
	arg  int
	mode mode
}{
	"encoding/json.Marshal":           {0, encode},
	"encoding/json.MarshalIndent":     {0, encode},
	"(*encoding/json.Encoder).Encode": {0, encode},
	"encoding/json.Unmarshal":         {1, decode},
	"(*encoding/json.Decoder).Decode": {0, decode},
}

// site is where a finding is reported and the name it is reported under.
type site struct {
	pos  token.Pos
	name string
}

type state struct {
	pass     *analysis.Pass
	skip     ruleSet
	reported map[string]bool
}

type walker struct {
	*state
	call token.Pos
	mode mode
	seen map[string]bool
}

func run(pass *analysis.Pass, skip ruleSet) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	st := &state{pass: pass, skip: skip, reported: map[string]bool{}}
	insp.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		fn, _ := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
		if fn == nil {
			return
		}
		r, ok := roots[fn.FullName()]
		if !ok || r.arg >= len(call.Args) {
			return
		}
		t := pass.TypesInfo.TypeOf(call.Args[r.arg])
		if t == nil {
			return
		}
		w := &walker{state: st, call: call.Pos(), mode: r.mode, seen: map[string]bool{}}
		w.root(t, "value passed to "+fn.Name())
	})
	return nil, nil
}

func (w *walker) root(t types.Type, name string) {
	t = types.Unalias(t)
	if _, ok := t.Underlying().(*types.Interface); ok {
		return // the dynamic type is not known statically
	}
	s := site{pos: w.call, name: name}
	if p, ok := t.Underlying().(*types.Pointer); ok {
		w.walk(p.Elem(), true, s)
		return
	}
	if w.mode == encode && !w.custom(t, false) {
		w.nilRules(t, s)
	}
	w.walk(t, false, s)
}

// nilRules reports a slice, map or []byte that is not omitted when nil.
func (w *walker) nilRules(t types.Type, s site) {
	if w.custom(t, true) {
		return
	}
	switch u := t.Underlying().(type) {
	case *types.Slice:
		if isPlainByte(u.Elem()) {
			w.report(s, "nil-bytes")
		} else {
			w.report(s, "nil-slice")
		}
	case *types.Map:
		w.report(s, "nil-map")
	}
}

func (w *walker) walk(t types.Type, addr bool, s site) {
	t = types.Unalias(t)
	if isDuration(t) {
		w.report(s, "duration")
		return
	}
	if w.mode == encode {
		onValue, onPtr := w.methods(t, "MarshalJSON", "MarshalText")
		if onValue {
			return
		}
		if onPtr {
			if !addr {
				w.report(s, "ptr-method")
			}
			return
		}
	} else if v, p := w.methods(t, "UnmarshalJSON", "UnmarshalText"); v || p {
		return
	}
	switch u := t.Underlying().(type) {
	case *types.Pointer:
		w.walk(u.Elem(), true, s)
	case *types.Slice:
		if isPlainByte(u.Elem()) {
			return
		}
		if isNamedByte(u.Elem()) {
			w.report(s, "named-byte-slice")
			return
		}
		w.walk(u.Elem(), true, site{s.pos, s.name + "[]"})
	case *types.Array:
		if isPlainByte(u.Elem()) || isNamedByte(u.Elem()) {
			w.report(s, "byte-array")
			return
		}
		w.walk(u.Elem(), addr, site{s.pos, s.name + "[]"})
	case *types.Map:
		w.walk(u.Elem(), false, site{s.pos, s.name + "[]"})
	case *types.Struct:
		w.walkStruct(t, u, addr, s.name)
	}
}

func (w *walker) walkStruct(t types.Type, st *types.Struct, addr bool, path string) {
	key := types.TypeString(t, nil)
	if addr {
		key += "|addr"
	}
	if w.seen[key] {
		return
	}
	w.seen[key] = true
	name := w.typeName(t, path)

	if w.mode == decode && hasSerializedField(st) {
		w.report(site{w.typePos(t), name}, "case-insensitive")
	}
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		tag, hasTag := reflect.StructTag(st.Tag(i)).Lookup("json")
		if tag == "-" {
			continue
		}
		jsonName, opts := parseTag(tag)
		ft := types.Unalias(f.Type())
		if f.Embedded() && jsonName == "" {
			if _, ok := deref(ft).Underlying().(*types.Struct); ok {
				_, isPtr := ft.Underlying().(*types.Pointer)
				w.walk(ft, addr || isPtr, site{w.fieldPos(f), name + "." + f.Name()})
				continue
			}
		}
		if !f.Exported() {
			continue
		}
		s := site{w.fieldPos(f), name + "." + f.Name()}
		if hasTag {
			w.tagRules(ft, opts, s)
		}
		if w.mode == encode {
			omit, zero := opts["omitempty"], opts["omitzero"]
			if omit {
				if id := w.omitDiffers(ft); id != "" {
					w.report(s, id)
				}
			}
			if !omit && !zero {
				w.nilRules(ft, s)
			}
		}
		w.walk(ft, addr, s)
	}
}

func (w *walker) tagRules(ft types.Type, opts map[string]bool, s site) {
	for o := range opts {
		if strings.HasPrefix(o, "format:") {
			w.report(s, "format-tag")
		}
	}
	if opts["string"] {
		b, ok := deref(ft).Underlying().(*types.Basic)
		switch {
		case !ok:
			w.report(s, "string-tag-ignored")
		case b.Info()&(types.IsBoolean|types.IsString) != 0:
			w.report(s, "string-tag")
		}
	}
}

// omitDiffers returns the rule that applies when omitempty on a field of
// type t behaves differently in v1 and v2, or "".
func (w *walker) omitDiffers(t types.Type) string {
	if w.custom(t, true) {
		return "" // the output of a custom marshaler is not known
	}
	switch u := t.Underlying().(type) {
	case *types.Basic:
		if u.Info()&(types.IsBoolean|types.IsNumeric) != 0 {
			return "omitempty"
		}
	case *types.Struct:
		if allOmittable(u) {
			return "omitempty-struct"
		}
	case *types.Pointer:
		if w.custom(u.Elem(), true) {
			return ""
		}
		if st, ok := u.Elem().Underlying().(*types.Struct); ok && allOmittable(st) {
			return "omitempty-struct"
		}
	}
	return ""
}

// allOmittable says whether every serialized field of st can be left out,
// so that v2 may encode the struct as {} and omit it.
func allOmittable(st *types.Struct) bool {
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		tag := reflect.StructTag(st.Tag(i)).Get("json")
		if tag == "-" || (!f.Exported() && !f.Embedded()) {
			continue
		}
		_, opts := parseTag(tag)
		if opts["omitzero"] {
			continue
		}
		if opts["omitempty"] {
			if b, ok := f.Type().Underlying().(*types.Basic); ok && b.Info()&(types.IsBoolean|types.IsNumeric) != 0 {
				return false
			}
			continue
		}
		return false
	}
	return true
}

// custom says whether t has a marshal (encode) or unmarshal (decode) method
// on its value, or on its pointer when ptrToo is set.
func (w *walker) custom(t types.Type, ptrToo bool) bool {
	var v, p bool
	if w.mode == encode {
		v, p = w.methods(t, "MarshalJSON", "MarshalText")
	} else {
		v, p = w.methods(t, "UnmarshalJSON", "UnmarshalText")
	}
	return v || (ptrToo && p)
}

// methods reports whether any of names is in the method set of t, and
// whether it is only in the method set of *t.
func (w *walker) methods(t types.Type, names ...string) (onValue, onPtr bool) {
	if _, ok := t.Underlying().(*types.Interface); ok {
		return false, false
	}
	for _, n := range names {
		if isMethod(t, n) {
			return true, false
		}
	}
	if _, isPtr := t.(*types.Pointer); isPtr {
		return false, false
	}
	pt := types.NewPointer(t)
	for _, n := range names {
		if isMethod(pt, n) {
			return false, true
		}
	}
	return false, false
}

func isMethod(t types.Type, name string) bool {
	obj, _, _ := types.LookupFieldOrMethod(t, false, nil, name)
	_, ok := obj.(*types.Func)
	return ok
}

func (w *walker) report(s site, id string) {
	key := id + "|" + s.name
	if w.skip[id] || w.reported[key] {
		return
	}
	w.reported[key] = true
	r := ruleByID[id]
	msg := s.name + ": " + r.Summary
	if r.Option != "" {
		msg += "; to keep v1 behaviour pass " + r.Option
	}
	w.pass.Report(analysis.Diagnostic{Pos: s.pos, Category: id, Message: msg, URL: docURL + "#" + id})
}

func (w *walker) fieldPos(f *types.Var) token.Pos {
	if f.Pkg() == w.pass.Pkg && f.Pos().IsValid() {
		return f.Pos()
	}
	return w.call
}

func (w *walker) typePos(t types.Type) token.Pos {
	if n, ok := t.(*types.Named); ok && n.Obj().Pkg() == w.pass.Pkg && n.Obj().Pos().IsValid() {
		return n.Obj().Pos()
	}
	return w.call
}

// typeName names a struct in messages: a named type by its name, an
// anonymous struct by the path that reached it, so two anonymous structs
// never share a name.
func (w *walker) typeName(t types.Type, path string) string {
	if _, ok := t.(*types.Named); ok {
		return types.TypeString(t, types.RelativeTo(w.pass.Pkg))
	}
	return path
}

func hasSerializedField(st *types.Struct) bool {
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		if reflect.StructTag(st.Tag(i)).Get("json") != "-" && (f.Exported() || f.Embedded()) {
			return true
		}
	}
	return false
}

// parseTag splits a json struct tag into its name and options.
func parseTag(tag string) (string, map[string]bool) {
	name, rest, _ := strings.Cut(tag, ",")
	opts := map[string]bool{}
	for rest != "" {
		var o string
		o, rest, _ = strings.Cut(rest, ",")
		if o = strings.TrimSpace(o); o != "" {
			opts[o] = true
		}
	}
	return name, opts
}

func deref(t types.Type) types.Type {
	for {
		p, ok := t.Underlying().(*types.Pointer)
		if !ok {
			return t
		}
		t = p.Elem()
	}
}

func isDuration(t types.Type) bool {
	n, ok := t.(*types.Named)
	return ok && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == "time" && n.Obj().Name() == "Duration"
}

func isPlainByte(t types.Type) bool {
	t = types.Unalias(t)
	b, ok := t.(*types.Basic)
	return ok && b.Kind() == types.Byte
}

func isNamedByte(t types.Type) bool {
	t = types.Unalias(t)
	if _, ok := t.(*types.Named); !ok {
		return false
	}
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Kind() == types.Byte
}
