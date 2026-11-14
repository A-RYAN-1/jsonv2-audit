// Command mutate is a small mutation tester for analyzer.go. It copies the
// module to a temp directory, applies one source mutation at a time (swap a
// comparison or boolean operator, negate an if condition, flip a boolean
// literal, drop a report call) and runs the library tests. A mutant the tests
// do not fail on has survived and points at a missing test.
//
// Usage, from the module root:
//
//	go run ./scripts/mutate [-file analyzer.go] [-timeout 3m] [-v] [-shard i/n] [-json out.json]
//
// -shard i/n runs every n-th mutant starting at i (0-based), so n machines
// split one run; it defaults to SHARD_INDEX/SHARD_TOTAL when those are set.
// -json writes the counts and the survivors for merging shard results.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type mutant struct {
	desc     string
	off, end int
	repl     string
}

var swap = map[token.Token]token.Token{
	token.EQL: token.NEQ, token.NEQ: token.EQL,
	token.LAND: token.LOR, token.LOR: token.LAND,
	token.LSS: token.GEQ, token.GEQ: token.LSS,
	token.GTR: token.LEQ, token.LEQ: token.GTR,
	token.AND: token.OR, token.OR: token.AND,
}

func main() {
	file := flag.String("file", "analyzer.go", "file to mutate, relative to the module root")
	timeout := flag.Duration("timeout", 3*time.Minute, "per-mutant test timeout")
	verbose := flag.Bool("v", false, "list killed mutants too")
	shardFlag := flag.String("shard", "", "run shard i of n, as i/n (default from SHARD_INDEX/SHARD_TOTAL)")
	jsonOut := flag.String("json", "", "write the result as JSON to this file")
	flag.Parse()
	if *shardFlag == "" && os.Getenv("SHARD_TOTAL") != "" {
		*shardFlag = os.Getenv("SHARD_INDEX") + "/" + os.Getenv("SHARD_TOTAL")
	}
	shard, shards := 0, 1
	if *shardFlag != "" {
		if _, err := fmt.Sscanf(*shardFlag, "%d/%d", &shard, &shards); err != nil || shards < 1 || shard < 0 || shard >= shards {
			check(fmt.Errorf("bad -shard %q: want i/n with 0 <= i < n", *shardFlag))
		}
	}

	src, err := os.ReadFile(*file)
	check(err)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, *file, src, 0)
	check(err)
	off := func(p token.Pos) int { return fset.Position(p).Offset }
	line := func(p token.Pos) int { return fset.Position(p).Line }

	var muts []mutant
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.BinaryExpr:
			if to, ok := swap[n.Op]; ok {
				muts = append(muts, mutant{
					desc: fmt.Sprintf("line %d: %s -> %s", line(n.OpPos), n.Op, to),
					off:  off(n.OpPos), end: off(n.OpPos) + len(n.Op.String()), repl: to.String(),
				})
			}
		case *ast.IfStmt:
			muts = append(muts, mutant{
				desc: fmt.Sprintf("line %d: negate if condition", line(n.If)),
				off:  off(n.Cond.Pos()), end: off(n.Cond.End()),
				repl: "!(" + string(src[off(n.Cond.Pos()):off(n.Cond.End())]) + ")",
			})
		case *ast.Ident:
			if n.Name == "true" || n.Name == "false" {
				to := "true"
				if n.Name == "true" {
					to = "false"
				}
				muts = append(muts, mutant{
					desc: fmt.Sprintf("line %d: %s -> %s", line(n.Pos()), n.Name, to),
					off:  off(n.Pos()), end: off(n.End()), repl: to,
				})
			}
		case *ast.ExprStmt:
			if c, ok := n.X.(*ast.CallExpr); ok {
				if sel, ok := c.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "report" {
					muts = append(muts, mutant{
						desc: fmt.Sprintf("line %d: drop report call", line(n.Pos())),
						off:  off(n.Pos()), end: off(n.End()), repl: "_ = 0",
					})
				}
			}
		}
		return true
	})
	sort.Slice(muts, func(i, j int) bool { return muts[i].off < muts[j].off })
	total := len(muts)
	if shards > 1 {
		var mine []mutant
		for i := shard; i < len(muts); i += shards {
			mine = append(mine, muts[i])
		}
		muts = mine
	}

	dir, err := os.MkdirTemp("", "mutate-*")
	check(err)
	defer func() { _ = os.RemoveAll(dir) }()
	check(copyTree(".", dir))
	target := filepath.Join(dir, *file)

	var survived []string
	killed, invalid := 0, 0
	for i, m := range muts {
		mutated := append(append(append([]byte{}, src[:m.off]...), m.repl...), src[m.end:]...)
		check(os.WriteFile(target, mutated, 0o644))
		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		cmd := exec.CommandContext(ctx, "go", "test", "-short", "-count=1", ".")
		cmd.Dir = dir
		var out bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &out
		err := cmd.Run()
		cancel()
		status := "KILLED  "
		if err != nil && bytes.Contains(out.Bytes(), []byte("build failed")) {
			invalid++ // does not compile: not a test result
			continue
		}
		if err == nil {
			status = "SURVIVED"
			survived = append(survived, m.desc)
		} else {
			killed++
		}
		if *verbose || err == nil {
			fmt.Printf("[%d/%d] %s %s\n", i+1, len(muts), status, m.desc)
		}
	}
	check(os.WriteFile(target, src, 0o644))
	fmt.Printf("\n%s: %d mutants, %d killed, %d survived (score %.1f%%)\n",
		*file, len(muts), killed, len(survived), 100*float64(killed)/float64(len(muts)))
	if shards > 1 {
		fmt.Printf("shard %d/%d of %d mutants\n", shard, shards, total)
	}
	if *jsonOut != "" {
		if survived == nil {
			survived = []string{}
		}
		b, err := json.MarshalIndent(map[string]any{
			"file": *file, "shard": shard, "shards": shards, "total": total,
			"mutants": len(muts), "killed": killed, "invalid": invalid, "survived": survived,
		}, "", "  ")
		check(err)
		check(os.WriteFile(*jsonOut, append(b, '\n'), 0o644))
	}
}

func copyTree(from, to string) error {
	return filepath.WalkDir(from, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "bin") {
			return filepath.SkipDir
		}
		rel, _ := filepath.Rel(from, p)
		dst := filepath.Join(to, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		if strings.HasSuffix(p, ".exe") {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, b, 0o644)
	})
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "mutate:", err)
		os.Exit(1)
	}
}
