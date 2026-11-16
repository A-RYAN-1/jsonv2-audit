package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// vettool builds this command once per test binary run.
func vettool(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("builds the vet tool and runs go vet")
	}
	bin := filepath.Join(t.TempDir(), "jsonv2-audit-vet")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

// vet runs go vet with the tool in testdata/mod, a module whose type is
// declared in one package and marshaled in another.
func vet(t *testing.T, bin string, flags ...string) (string, error) {
	t.Helper()
	args := append([]string{"vet", "-vettool=" + bin}, flags...)
	cmd := exec.Command("go", append(args, "./...")...)
	cmd.Dir = filepath.Join("testdata", "mod")
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestGoVet(t *testing.T) {
	bin := vettool(t)

	out, err := vet(t, bin)
	if err == nil {
		t.Fatalf("go vet passed, want findings:\n%s", out)
	}
	for _, want := range []string{
		"Pet.Nicknames: a nil slice encodes as null in v1 and [] in v2",
		"Pet.Age: omitempty omits false and 0 in v1",
		"api.go:13:",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("go vet output lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "model.go") {
		t.Errorf("a finding on an imported type should be reported at the call, not in model.go:\n%s", out)
	}

	if strings.Contains(out, "case-insensitive") {
		t.Errorf("case-insensitive is off by default:\n%s", out)
	}
	out, err = vet(t, bin, "-jsonv2audit.all")
	if err == nil || !strings.Contains(out, "case-insensitive") {
		t.Errorf("-jsonv2audit.all should report case-insensitive, got err %v:\n%s", err, out)
	}

	out, err = vet(t, bin, "-jsonv2audit.skip=nil-slice,omitempty")
	if err != nil {
		t.Fatalf("go vet with both rules skipped failed: %v\n%s", err, out)
	}

	out, err = vet(t, bin, "-jsonv2audit.skip=nil-slcie")
	if err == nil || !strings.Contains(out, `unknown rule "nil-slcie"`) {
		t.Errorf("an unknown rule should fail with its name, got err %v:\n%s", err, out)
	}
}
