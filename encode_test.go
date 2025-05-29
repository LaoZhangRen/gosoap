package gosoap

import (
	"reflect"
	"strconv"
	"testing"
)

func TestRecursiveEncodeWithNumericTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"int", 42, "42"},
		{"int8", int8(8), "8"},
		{"int16", int16(16), "16"},
		{"int32", int32(32), "32"},
		{"int64", int64(64), "64"},
		{"uint", uint(42), "42"},
		{"uint8", uint8(8), "8"},
		{"uint16", uint16(16), "16"},
		{"uint32", uint32(32), "32"},
		{"uint64", uint64(64), "64"},
		{"float32", float32(3.14), "3.14"},
		{"float64", 2.718, "2.718"},
		{"bool_true", true, "true"},
		{"bool_false", false, "false"},
		{"string", "hello", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := &tokenData{}
			tokens.recursiveEncode(tt.input)

			if len(tokens.data) != 1 {
				t.Fatalf("Expected 1 token, got %d", len(tokens.data))
			}

			if charData, ok := tokens.data[0].(xml.CharData); ok {
				result := string(charData)
				if result != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, result)
				}
			} else {
				t.Errorf("Expected xml.CharData, got %T", tokens.data[0])
			}
		})
	}
}

func TestRecursiveEncodeWithMap(t *testing.T) {
	// Test the actual use case: Params with int values
	params := Params{
		"a": 10,
		"b": 5,
	}

	tokens := &tokenData{}
	tokens.recursiveEncode(params)

	// Should generate: StartElement(a), CharData("10"), EndElement(a), StartElement(b), CharData("5"), EndElement(b)
	// Order might vary due to map iteration, so we check both possibilities
	expectedTokenCount := 6 // 2 * (StartElement + CharData + EndElement)
	if len(tokens.data) != expectedTokenCount {
		t.Fatalf("Expected %d tokens, got %d", expectedTokenCount, len(tokens.data))
	}

	// Find the CharData tokens and verify they contain the expected values
	charDataTokens := []string{}
	for _, token := range tokens.data {
		if charData, ok := token.(xml.CharData); ok {
			charDataTokens = append(charDataTokens, string(charData))
		}
	}

	expectedValues := []string{"10", "5"}
	if len(charDataTokens) != len(expectedValues) {
		t.Fatalf("Expected %d CharData tokens, got %d", len(expectedValues), len(charDataTokens))
	}

	// Check that both expected values are present (order doesn't matter)
	for _, expected := range expectedValues {
		found := false
		for _, actual := range charDataTokens {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected value %q not found in CharData tokens: %v", expected, charDataTokens)
		}
	}
}

// Benchmark to ensure performance isn't significantly impacted
func BenchmarkRecursiveEncode(b *testing.B) {
	params := Params{
		"a": 42,
		"b": 3.14,
		"c": "hello",
		"d": true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tokens := &tokenData{}
		tokens.recursiveEncode(params)
	}
}
