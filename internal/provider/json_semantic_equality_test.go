// Copyright (c) Arthur Diniz <arthurbdiniz@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"
)

func TestJsonSemanticEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{
			name:     "identical strings",
			a:        `{"key": "value"}`,
			b:        `{"key": "value"}`,
			expected: true,
		},
		{
			name:     "different whitespace",
			a:        `{"key": "value"}`,
			b:        `{ "key" : "value" }`,
			expected: true,
		},
		{
			name:     "different key order",
			a:        `{"a": 1, "b": 2}`,
			b:        `{"b": 2, "a": 1}`,
			expected: true,
		},
		{
			name:     "nested objects with different formatting",
			a:        `{"outer":{"inner":"value"}}`,
			b:        `{"outer": {"inner": "value"}}`,
			expected: true,
		},
		{
			name:     "arrays with same elements",
			a:        `[1, 2, 3]`,
			b:        `[1,2,3]`,
			expected: true,
		},
		{
			name:     "different values",
			a:        `{"key": "value1"}`,
			b:        `{"key": "value2"}`,
			expected: false,
		},
		{
			name:     "missing key",
			a:        `{"a": 1, "b": 2}`,
			b:        `{"a": 1}`,
			expected: false,
		},
		{
			name:     "arrays with different order",
			a:        `[1, 2, 3]`,
			b:        `[3, 2, 1]`,
			expected: false,
		},
		{
			name:     "complex nested structure with whitespace",
			a:        `{"nodes":[{"id":"1","name":"test"},{"id":"2","name":"test2"}]}`,
			b:        `{ "nodes": [ { "id": "1", "name": "test" }, { "id": "2", "name": "test2" } ] }`,
			expected: true,
		},
		{
			name: "multiline vs single line",
			a: `{
				"id": "test",
				"name": "Test Workflow"
			}`,
			b:        `{"id": "test", "name": "Test Workflow"}`,
			expected: true,
		},
		{
			name:     "null values",
			a:        `{"key": null}`,
			b:        `{"key":null}`,
			expected: true,
		},
		{
			name:     "boolean values",
			a:        `{"active": true, "disabled": false}`,
			b:        `{"active":true,"disabled":false}`,
			expected: true,
		},
		{
			name:     "numeric values",
			a:        `{"count": 42, "rate": 3.14}`,
			b:        `{"count":42,"rate":3.14}`,
			expected: true,
		},
		{
			name:     "invalid json a",
			a:        `{invalid}`,
			b:        `{"key": "value"}`,
			expected: false,
		},
		{
			name:     "invalid json b",
			a:        `{"key": "value"}`,
			b:        `{invalid}`,
			expected: false,
		},
		{
			name:     "both invalid json",
			a:        `{invalid}`,
			b:        `{also-invalid}`,
			expected: false,
		},
		{
			name:     "empty objects",
			a:        `{}`,
			b:        `{ }`,
			expected: true,
		},
		{
			name:     "empty arrays",
			a:        `[]`,
			b:        `[ ]`,
			expected: true,
		},
		{
			name:     "realistic node comparison",
			a:        `[{"id":"manual-trigger","name":"When clicking 'Test workflow'","parameters":{},"position":[240,300],"type":"n8n-nodes-base.manualTrigger","typeVersion":1}]`,
			b:        `[{"id": "manual-trigger", "name": "When clicking 'Test workflow'", "parameters": {}, "position": [240, 300], "type": "n8n-nodes-base.manualTrigger", "typeVersion": 1}]`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := jsonSemanticEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("jsonSemanticEqual(%q, %q) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestNormalizeJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
	}{
		{
			name:        "simple object",
			input:       `{ "key" : "value" }`,
			shouldError: false,
		},
		{
			name:        "array",
			input:       `[ 1, 2, 3 ]`,
			shouldError: false,
		},
		{
			name:        "invalid json",
			input:       `{invalid}`,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeJSON(tt.input)
			if tt.shouldError {
				if err == nil {
					t.Errorf("NormalizeJSON(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("NormalizeJSON(%q) unexpected error: %v", tt.input, err)
				}
				// Verify the result is valid JSON
				if _, err := NormalizeJSON(result); err != nil {
					t.Errorf("NormalizeJSON result is not valid JSON: %v", err)
				}
			}
		})
	}
}

func TestNormalizeJSONIdempotent(t *testing.T) {
	input := `{ "key" : "value", "nested": { "a": 1 } }`

	first, err := NormalizeJSON(input)
	if err != nil {
		t.Fatalf("First normalization failed: %v", err)
	}

	second, err := NormalizeJSON(first)
	if err != nil {
		t.Fatalf("Second normalization failed: %v", err)
	}

	if first != second {
		t.Errorf("NormalizeJSON is not idempotent: %q != %q", first, second)
	}
}

func TestJsonSemanticEqualWithNullValues(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{
			name:     "null value vs missing key",
			a:        `{"key": "value", "main": null}`,
			b:        `{"key": "value"}`,
			expected: true,
		},
		{
			name:     "nested null value",
			a:        `{"outer": {"inner": null, "keep": "this"}}`,
			b:        `{"outer": {"keep": "this"}}`,
			expected: true,
		},
		{
			name:     "connection with main null vs ai_languageModel",
			a:        `{"Google Gemini Chat Model1": {"main": null}}`,
			b:        `{"Google Gemini Chat Model1": {"ai_languageModel": [[[{"index": 0, "node": "AI Agent", "type": "ai_languageModel"}]]]}}`,
			expected: false, // These are actually different - one has content
		},
		{
			name:     "both have main null",
			a:        `{"Google Gemini Chat Model1": {"main": null}}`,
			b:        `{"Google Gemini Chat Model1": {"main": null}}`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := jsonSemanticEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("jsonSemanticEqual(%q, %q) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestJsonSemanticEqualWithOptionalFields(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{
			name:     "executeOnce present vs missing",
			a:        `{"id": "node1", "executeOnce": false}`,
			b:        `{"id": "node1"}`,
			expected: true,
		},
		{
			name:     "alwaysOutputData present vs missing",
			a:        `{"id": "node1", "alwaysOutputData": false}`,
			b:        `{"id": "node1"}`,
			expected: true,
		},
		{
			name:     "multiple optional fields",
			a:        `{"id": "node1", "executeOnce": false, "alwaysOutputData": true, "retryOnFail": false}`,
			b:        `{"id": "node1"}`,
			expected: true,
		},
		{
			name:     "nested in array",
			a:        `[{"id": "node1", "executeOnce": false}, {"id": "node2"}]`,
			b:        `[{"id": "node1"}, {"id": "node2"}]`,
			expected: true,
		},
		{
			name:     "realistic node comparison with optional fields",
			a:        `[{"id":"get-articles","name":"Get articles","executeOnce":false,"alwaysOutputData":false}]`,
			b:        `[{"id":"get-articles","name":"Get articles"}]`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := jsonSemanticEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("jsonSemanticEqual(%q, %q) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestJsonSemanticEqualWithParameterOrder(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{
			name:     "parameters in different order should be equal",
			a:        `[{"id":"node1","parameters":[{"key":"options","value":"a"},{"key":"sheetName","value":"b"},{"key":"documentId","value":"c"}]}]`,
			b:        `[{"id":"node1","parameters":[{"key":"documentId","value":"c"},{"key":"options","value":"a"},{"key":"sheetName","value":"b"}]}]`,
			expected: true,
		},
		{
			name:     "parameters with different values should not be equal",
			a:        `[{"id":"node1","parameters":[{"key":"options","value":"a"}]}]`,
			b:        `[{"id":"node1","parameters":[{"key":"options","value":"different"}]}]`,
			expected: false,
		},
		{
			name:     "parameters with missing key should not be equal",
			a:        `[{"id":"node1","parameters":[{"key":"options","value":"a"},{"key":"extra","value":"b"}]}]`,
			b:        `[{"id":"node1","parameters":[{"key":"options","value":"a"}]}]`,
			expected: false,
		},
		{
			name:     "nested nodes with parameters in different order",
			a:        `[{"id":"n1","parameters":[{"key":"a"},{"key":"b"}]},{"id":"n2","parameters":[{"key":"x"},{"key":"y"}]}]`,
			b:        `[{"id":"n1","parameters":[{"key":"b"},{"key":"a"}]},{"id":"n2","parameters":[{"key":"y"},{"key":"x"}]}]`,
			expected: true,
		},
		{
			name:     "realistic google sheets node parameter order",
			a:        `[{"id":"read-tickers","name":"Read_tickers","parameters":[{"key":"options","type":"unknown","value":"map[]"},{"key":"sheetName","type":"unknown","value":"stocks"},{"key":"documentId","type":"unknown","value":"123"}]}]`,
			b:        `[{"id":"read-tickers","name":"Read_tickers","parameters":[{"key":"documentId","type":"unknown","value":"123"},{"key":"options","type":"unknown","value":"map[]"},{"key":"sheetName","type":"unknown","value":"stocks"}]}]`,
			expected: true,
		},
		{
			name:     "regular arrays without key field should preserve order",
			a:        `{"position":[100, 200]}`,
			b:        `{"position":[200, 100]}`,
			expected: false,
		},
		{
			name:     "empty parameters arrays should be equal",
			a:        `[{"id":"node1","parameters":[]}]`,
			b:        `[{"id":"node1","parameters":[]}]`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := jsonSemanticEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("jsonSemanticEqual(%q, %q) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}
