// Package parsers generates and checks the parsers in its subdirectories, packages of this module
// (syntaxir.dev/alpha-parsers/<language>) that depend only on the standard library, the SyntaxIR library (syntaxir.dev/sxir/codegen/go/lib) and core (syntaxir.dev/sxir/core).
//
// Each subdirectory holds the grammar (<language>.sxir), the parser generated from it (parser.go, api.go, and types/types.go; they import the library), code
// written by hand around the generated API, and tests against the language's reference implementation
// or conformance suite. The tests of this package check that every generated file is up to date and that
// every backend of the engine gives the results recorded in the subdirectories' golden files, which the
// generated parsers are checked against in their own packages. From the root of the repository,
// regenerate the parsers with go generate ., test one parser with go test ./<language>/..., and
// all of them with ./test.sh.
package parsers

//go:generate go run syntaxir.dev/sxir/cmd/sxir gen -g cel/cel.sxir -module syntaxir.dev/alpha-parsers/cel -types -typed-only -no-limits -nodoc -lib import -o cel
//go:generate go run syntaxir.dev/sxir/cmd/sxir gen -g csv/csv.sxir -module syntaxir.dev/alpha-parsers/csv -types -recognize -nodoc -lib import -o csv
//go:generate go run syntaxir.dev/sxir/cmd/sxir gen -g cue/cue.sxir -module syntaxir.dev/alpha-parsers/cue -types -typed-only -no-limits -nodoc -lib import -o cue
//go:generate go run syntaxir.dev/sxir/cmd/sxir gen -g duckdb/duckdb.sxir -module syntaxir.dev/alpha-parsers/duckdb -types -typed-only -no-limits -nodoc -lib import -o duckdb
//go:generate go run syntaxir.dev/sxir/cmd/sxir gen -g golang/golang.sxir -expectations -module syntaxir.dev/alpha-parsers/golang -types -typed-only -no-limits -nodoc -lib import -o golang
//go:generate go run syntaxir.dev/sxir/cmd/sxir gen -g json/json.sxir -module syntaxir.dev/alpha-parsers/json -types -recognize -nodoc -lib import -o json
//go:generate go run syntaxir.dev/sxir/cmd/sxir gen -g python/python.sxir -expectations -module syntaxir.dev/alpha-parsers/python -types -typed-only -no-limits -nodoc -lib import -o python
//go:generate go run syntaxir.dev/sxir/cmd/sxir gen -g typescript/typescript.sxir -expectations -module syntaxir.dev/alpha-parsers/typescript -types -typed-only -no-limits -nodoc -lib import -o typescript
//go:generate go run syntaxir.dev/sxir/cmd/sxir gen -g xml/xml.sxir -expectations -module syntaxir.dev/alpha-parsers/xml -types -recognize -nodoc -lib import -o xml
//go:generate go run syntaxir.dev/sxir/cmd/sxir gen -g yaml/yaml.sxir -module syntaxir.dev/alpha-parsers/yaml -types -recognize -nodoc -lib import -o yaml
