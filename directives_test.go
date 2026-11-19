package parsers_test

import (
	"testing"

	"zeroparse.dev/zp/pkg/runtime"
	"zeroparse.dev/zp/pkg/runtime/input"
)

func TestYAMLDirectiveBackends(t *testing.T) {
	bs := backends(t, "yaml/yaml.zp")
	for _, tc := range []struct {
		directive string
		valid     bool
	}{
		{"%TAG", false}, {"%YAML", false}, {"%TAG !e!", false},
		{"%YAML 1.foo", false}, {"%TAG notahandle tag:x", false},
		{"%YAML 1.2 extra", false}, {"%TAG !e! tag:x extra", false},
		{"%YAMLfoo", true}, {"%TAGx ignored", true}, {"%YAML#name", true},
		{"%YAML 0001.02", true}, {"%TAG !e! notahandle", true},
	} {
		t.Run(tc.directive, func(t *testing.T) {
			for _, b := range bs {
				for _, unit := range []input.PositionUnit{input.CodePoints, input.Bytes} {
					for _, recognize := range []bool{false, true} {
						text, opt := tc.directive+"\n--- x\n", runtime.WithUnit(unit)
						var err error
						if recognize {
							err = b.p.Recognize(text, opt)
						} else {
							_, err = b.p.Parse(text, opt)
						}
						if (err == nil) != tc.valid {
							t.Errorf("backend %s unit %v recognize %v: %v", b.name, unit, recognize, err)
						}
					}
				}
			}
		})
	}
}
