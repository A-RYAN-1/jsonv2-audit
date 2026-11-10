package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestFindingsAndSummary(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"-C", "../..", "./internal/groundtruth"}, &out, &errb); code != 1 {
		t.Fatalf("exit %d, stderr %s", code, errb.String())
	}
	s := out.String()
	for _, want := range []string{
		"[nil-slice] NilSlice.S:",
		"[duration] Duration.D:",
		"to keep v1 behaviour under v2: ",
		"json.FormatDurationAsNano(true)",
		"needs a code change: format-tag",
		"4 decoded struct type(s) rely on case-insensitive",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("output lacks %q:\n%s", want, s)
		}
	}
}

func TestAllListsCaseFindings(t *testing.T) {
	var out, errb bytes.Buffer
	run([]string{"-all", "-C", "../..", "./internal/groundtruth"}, &out, &errb)
	if !strings.Contains(out.String(), "[case-insensitive] CaseFold:") {
		t.Errorf("-all output lacks the case-insensitive line:\n%s", out.String())
	}
}

func TestJSON(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"-json", "-C", "../..", "./internal/groundtruth"}, &out, &errb); code != 1 {
		t.Fatalf("exit %d, stderr %s", code, errb.String())
	}
	var sums []Summary
	if err := json.Unmarshal(out.Bytes(), &sums); err != nil {
		t.Fatal(err)
	}
	if len(sums) != 1 || len(sums[0].Findings) == 0 || len(sums[0].Options) == 0 {
		t.Fatalf("unexpected summary: %+v", sums)
	}
}

func TestClean(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"-C", "../..", "."}, &out, &errb); code != 0 {
		t.Fatalf("exit %d: %s %s", code, out.String(), errb.String())
	}
	if !strings.Contains(out.String(), "no differences found") {
		t.Errorf("got %q", out.String())
	}
}

func TestRulesAndBadArgs(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"-rules"}, &out, &errb); code != 0 || !strings.Contains(out.String(), "case-insensitive") {
		t.Fatalf("exit %d: %s", code, out.String())
	}
	if code := run([]string{"-nope"}, &out, &errb); code != 2 {
		t.Fatalf("bad flag exit %d", code)
	}
	if code := run([]string{"-C", "../..", "./does/not/exist"}, &out, &errb); code != 2 {
		t.Fatalf("missing package exit %d", code)
	}
}
