package cue

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	"syntaxir.dev/sxir/pkg/compiler"
	"syntaxir.dev/sxir/pkg/grammar"
	"syntaxir.dev/sxir/pkg/runtime"
	"syntaxir.dev/sxir/pkg/runtime/input"
)

// The generated parser has only the typed entry point, so the check that it agrees with the grammar does not
// rest on other entry points of the same generated code: the oracle is the reference parser, which runs
// cue.sxir itself.
var interpreter = sync.OnceValue(func() runtime.Parser {
	src, err := os.ReadFile("cue.sxir")
	if err != nil {
		panic(err)
	}
	p, err := compiler.NewReferenceParser(grammar.FromBytes("cue.sxir", src), compiler.WithStart("main"))
	if err != nil {
		panic(err)
	}
	return p
})

// spanOf returns the range of a typed value that carries a Span, or false.
func spanOf(v any) (start, end int, ok bool) {
	r := reflect.ValueOf(v)
	for r.Kind() == reflect.Interface || r.Kind() == reflect.Pointer {
		if r.IsNil() {
			return 0, 0, false
		}
		r = r.Elem()
	}
	if r.Kind() != reflect.Struct {
		return 0, 0, false
	}
	f := r.FieldByName("Span")
	if !f.IsValid() {
		return 0, 0, false
	}
	s := f.Interface().(interface{ Range() (int, int) })
	start, end = s.Range()
	return start, end, true
}

// interpreterDiff compares ParseValue with the interpreter p, in both position units: they must accept the same input
// with the same error (message, line, column and offset), and the value must cover the range of p's node. It
// returns what differs, or "".
func interpreterDiff(src string, p runtime.Parser) string {
	for _, unit := range []struct {
		name string
		g    PositionUnit
		i    input.PositionUnit
	}{{"code points", CodePoints, input.CodePoints}, {"bytes", Bytes, input.Bytes}} {
		v, gerr := ParseValue(src, WithUnit(unit.g))
		n, ierr := p.Parse(src, runtime.WithUnit(unit.i))
		if (gerr == nil) != (ierr == nil) || (gerr != nil && gerr.Error() != ierr.Error()) {
			return fmt.Sprintf("%q (%s): ParseValue: %v; interpreter: %v", src, unit.name, gerr, ierr)
		}
		if gerr != nil {
			continue
		}
		if start, end, ok := spanOf(v); ok && n != nil && (start != n.Start() || end != n.End()) {
			return fmt.Sprintf("%q (%s): ParseValue covers %d..%d; interpreter %d..%d", src, unit.name, start, end, n.Start(), n.End())
		}
	}
	return ""
}

func agreeWithInterpreter(t testing.TB, src string) {
	t.Helper()
	if d := interpreterDiff(src, interpreter()); d != "" {
		t.Fatal(d)
	}
}

// TestInterpreterOracle checks that the comparison can fail: against an interpreter of a grammar that differs
// it reports a difference, for an accepted input and a rejected one, and the range comparison has a range.
func TestInterpreterOracle(t *testing.T) {
	src, err := os.ReadFile("cue.sxir")
	if err != nil {
		t.Fatal(err)
	}
	mutated := strings.Replace(string(src), `"&&"`, `"&|"`, 1)
	if mutated == string(src) {
		t.Fatal("the mutation did not apply")
	}
	other, err := compiler.NewReferenceParser(grammar.FromString("cue.sxir", mutated), compiler.WithStart("main"))
	if err != nil {
		t.Fatal(err)
	}
	for _, in := range []string{"x: a && b", "x: a &| b", "x: a &&", "x: \"é\" + 1"} {
		if d := interpreterDiff(in, interpreter()); d != "" {
			t.Errorf("the grammar's own interpreter differs: %s", d)
		}
	}
	if interpreterDiff("x: a && b", other) == "" {
		t.Error("a mutated interpreter agrees on an input it rejects")
	}
	if interpreterDiff("x: a &| b", other) == "" {
		t.Error("a mutated interpreter agrees on an input it accepts")
	}
	v, err := ParseValue("x: \"é\" + 1", WithUnit(Bytes))
	if err != nil {
		t.Fatal(err)
	}
	if start, end, ok := spanOf(v); !ok || end <= start {
		t.Errorf("the span of %T is %d..%d (%v): the range comparison is void", v, start, end, ok)
	}
}
