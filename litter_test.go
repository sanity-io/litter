package litter_test

import (
	"testing"

	"github.com/sanity-io/litter"
)

type CustomInt int

func TestPointerFormatting_StrictGo(t *testing.T) {
	cfg := &litter.Options{
		StrictGo: true,
	}

	intVal := 42
	strVal := "hello"
	boolVal := true
	floatVal := 12.3
	int8Val := int8(10)
	uintVal := uint(10)
	aliasVal := CustomInt(20)
	nestedVal := &intVal

	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"int pointer", &intVal, "new(42)"},
		{"string pointer", &strVal, "new(\"hello\")"},
		{"bool pointer", &boolVal, "new(true)"},
		{"float pointer", &floatVal, "new(12.3)"},
		{"int8 pointer", &int8Val, "new(int8(10))"},
		{"uint pointer", &uintVal, "new(uint(10))"},
		{"alias pointer", &aliasVal, "new(litter_test.CustomInt(20))"},
		{"nested pointer", &nestedVal, "new(new(42))"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := cfg.Sdump(tc.input)
			if got != tc.expected {
				t.Errorf("got %q, expected %q", got, tc.expected)
			}
		})
	}
}

func TestPointerFormatting_Default(t *testing.T) {
	cfg := &litter.Options{
		StrictGo: false,
	}

	intVal := 42
	strVal := "hello"
	got := cfg.Sdump(&intVal, &strVal)
	expected := "&42 &\"hello\""
	if got != expected {
		t.Errorf("got %q, expected %q", got, expected)
	}
}
