package parsers_test

import (
	"bufio"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"syntaxir.dev/sxir/pkg/compiler"
	"syntaxir.dev/sxir/pkg/grammar"
	"syntaxir.dev/sxir/pkg/runtime"
	"syntaxir.dev/sxir/pkg/runtime/vm"
)

var update = flag.Bool("update", false, "update golden files")

// generation is a sxir gen command of generate.go.
type generation struct {
	grammar, module, out string
	hasTypes             bool
	config               compiler.GoConfig
}

// generations reads the go:generate lines of generate.go.
func generations(t *testing.T) []generation {
	t.Helper()
	return generationsIn(t, ".")
}

// generationsIn reads the go:generate lines of the generate.go in dir. The grammar and output paths
// are relative to dir.
func generationsIn(t *testing.T, dir string) []generation {
	t.Helper()
	f, err := os.Open(filepath.Join(dir, "generate.go"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var gens []generation
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		args, ok := strings.CutPrefix(sc.Text(), "//go:generate go run syntaxir.dev/sxir/cmd/sxir gen ")
		if !ok {
			continue
		}
		var g generation
		fs := strings.Fields(args)
		for i := 0; i < len(fs); i++ {
			switch fs[i] {
			case "-g":
				i++
				g.grammar = fs[i]
			case "-module":
				i++
				g.module = fs[i]
			case "-o":
				i++
				g.out = fs[i]
			case "-lib":
				i++
				if fs[i] != "import" {
					t.Fatalf("generate.go: unknown library mode %s", fs[i])
				}
				g.config.Lib = compiler.LibImport
			case "-types":
				g.hasTypes = true
				g.config.Types = true
			case "-expectations":
				g.config.Expectations = true
			case "-recognize":
				g.config.Recognize = true
			case "-limits":
				g.config.LimitControls = true
			case "-typed-only":
				g.config.TypedOnly = true
			case "-document":
				g.config.Document = true
			case "-stream":
				g.config.Stream = true
			case "-nodoc":
				g.config.OmitPackageDoc = true
			default:
				t.Fatalf("generate.go: unknown argument %s", fs[i])
			}
		}
		g.grammar = filepath.Join(dir, g.grammar)
		g.out = filepath.Join(dir, g.out)
		gens = append(gens, g)
	}
	if len(gens) == 0 {
		t.Fatal("generate.go generates nothing")
	}
	return gens
}

// backend is one way of parsing with a grammar: the reference parser, or the bytecode VM with
// its recursive or iterative machine.
type backend struct {
	name string
	p    runtime.Parser
}

// backends compiles the grammar at path from the rule main for every backend.
func backends(t *testing.T, path string) []backend {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := grammar.FromBytes(path, src)
	ref, err := compiler.NewReferenceParser(s, compiler.WithStart("main"))
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	p, err := compiler.Compile(s, compiler.WithStart("main"))
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	vp := p.(*vm.Parser)
	return []backend{{"reference", ref}, {"bytecode", vp}, {"iterative", vp.Iterative()}}
}

// TestGeneratedParsersAreUpToDate checks that every file of a generated package is what sxir gen
// generates from its grammar (run go generate . after changing a grammar or the generator).
func TestGeneratedParsersAreUpToDate(t *testing.T) {
	for _, g := range generations(t) {
		src, err := os.ReadFile(g.grammar)
		if err != nil {
			t.Fatal(err)
		}
		c := g.config
		c.Module = g.module
		want, err := compiler.GenerateGo(grammar.FromBytes(g.grammar, src), c, compiler.WithStart("main"))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range want {
			path := filepath.Join(g.out, filepath.FromSlash(f.Path))
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !sameGenerated(t, path, string(got), string(f.Content)) {
				t.Errorf("%s is out of date; run go generate .", path)
			}
		}
	}
}

// TestBackends parses the inputs of each parser's testdata (testdata/*.txt) with the reference parser
// and both machines of the bytecode VM, and compares the trees and errors with the golden files (testdata/*.golden). The modules test
// their generated parsers against the same files. Run go test . -update to rewrite them.
func TestBackends(t *testing.T) {
	for _, g := range generations(t) {
		t.Run(filepath.Dir(g.grammar), func(t *testing.T) {
			bs := backends(t, g.grammar)
			inputs, _ := filepath.Glob(filepath.Join(filepath.Dir(g.grammar), "testdata", "*.txt"))
			if len(inputs) == 0 {
				t.Fatal("no inputs")
			}
			for _, in := range inputs {
				data, err := os.ReadFile(in)
				if err != nil {
					t.Fatal(err)
				}
				golden := strings.TrimSuffix(in, ".txt") + ".golden"
				for i, b := range bs {
					// Print the tree and the errors, including those recovered with #recover.
					var got string
					var opts []runtime.ParseOption
					if g.config.Expectations {
						opts = append(opts, runtime.WithExpectations())
					}
					n, err := b.p.Parse(string(data), opts...)
					if n != nil {
						got = n.String()
					}
					if err != nil {
						if got != "" {
							got += "\n"
						}
						got += "error: " + err.Error()
					}
					got += "\n"
					if *update && i == 0 {
						if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
							t.Fatal(err)
						}
					}
					want, err := os.ReadFile(golden)
					if err != nil {
						t.Fatalf("%v (run with -update to create it)", err)
					}
					if got != string(want) {
						t.Errorf("%s (%s)\n got  %s\n want %s", in, b.name, got, want)
					}
				}
			}
		})
	}
}

const (
	headerPrefix = "// Code generated by sxir "
	headerSuffix = ". DO NOT EDIT."
)

// stampVersion replaces the version in the first line of a generated file, "// Code generated by sxir
// <version>. DO NOT EDIT.", with a fixed token. The generator writes "(devel)" when it runs under go.work
// (as hack/sync-alpha-parsers.sh does) and the module version when it runs from the go.mod pin, so the
// version says how the generator was resolved, not what it generated. ok is false when the first line does
// not have that shape or has no version.
func stampVersion(s string) (stamped string, ok bool) {
	line, rest, _ := strings.Cut(s, "\n")
	v, found := strings.CutPrefix(line, headerPrefix)
	if !found {
		return s, false
	}
	v, found = strings.CutSuffix(v, headerSuffix)
	if !found || v == "" || strings.ContainsAny(v, " \t") {
		return s, false
	}
	return headerPrefix + "VERSION" + headerSuffix + "\n" + rest, true
}

// sameGenerated reports whether got is what the generator wrote as want, up to the version in the first
// line. Both must have the header: a file without it, or with another shape, is never the same.
func sameGenerated(t *testing.T, path, got, want string) bool {
	t.Helper()
	w, ok := stampVersion(want)
	if !ok {
		t.Fatalf("%s: the generator output does not start with %q<version>%q", path, headerPrefix, headerSuffix)
	}
	g, ok := stampVersion(got)
	return ok && g == w
}

func TestSameGenerated(t *testing.T) {
	const body = "\npackage p\n\nvar x = 1\n"
	hdr := func(v string) string { return headerPrefix + v + headerSuffix + body }
	for _, tc := range []struct {
		name, got, want string
		same            bool
	}{
		{"identical", hdr("(devel)"), hdr("(devel)"), true},
		{"version only", hdr("(devel)"), hdr("v0.0.0-20261119095319-5400579d480b"), true},
		{"content differs", hdr("(devel)"), hdr("(devel)") + "var y = 2\n", false},
		{"content differs under another version", hdr("(devel)") + "var y = 2\n", hdr("v1.2.3"), false},
		{"edited code after the header", strings.Replace(hdr("v1.2.3"), "x = 1", "x = 2", 1), hdr("v1.2.3"), false},
		{"header missing", "package p\n\nvar x = 1\n", hdr("(devel)"), false},
		{"header without version", headerPrefix + headerSuffix[1:] + body, hdr("(devel)"), false},
		{"header with other wording", "// Code generated by hand. DO NOT EDIT." + body, hdr("(devel)"), false},
		{"header not on the first line", "\n" + hdr("(devel)"), hdr("(devel)"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := sameGenerated(t, "p.go", tc.got, tc.want); got != tc.same {
				t.Errorf("sameGenerated = %v, want %v", got, tc.same)
			}
		})
	}
}
