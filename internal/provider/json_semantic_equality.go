// Copyright (c) Arthur Diniz <arthurbdiniz@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// jsonSemanticEqualityModifier is a plan modifier that suppresses diffs
// when two JSON strings are semantically equivalent (same content, different formatting).
type jsonSemanticEqualityModifier struct{}

// JSONSemanticEquality returns a plan modifier that compares JSON strings semantically.
// If the state and config values parse to equivalent JSON structures (ignoring whitespace
// and key ordering), the modifier uses the state value to prevent unnecessary updates.
func JSONSemanticEquality() planmodifier.String {
	return jsonSemanticEqualityModifier{}
}

func (m jsonSemanticEqualityModifier) Description(_ context.Context) string {
	return "Compares JSON strings semantically, ignoring whitespace and key ordering differences."
}

func (m jsonSemanticEqualityModifier) MarkdownDescription(_ context.Context) string {
	return "Compares JSON strings semantically, ignoring whitespace and key ordering differences."
}

func (m jsonSemanticEqualityModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If there's no state value, nothing to compare against
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}

	// If there's no config value (computed only), nothing to compare
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	// If there's no plan value, nothing to modify
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}

	stateJSON := req.StateValue.ValueString()
	configJSON := req.ConfigValue.ValueString()

	// If string values are already identical, no need to compare semantically
	if stateJSON == configJSON {
		return
	}

	// Parse and compare JSON semantically
	if jsonSemanticEqual(stateJSON, configJSON) {
		// JSON is semantically equal - use state value to suppress diff
		resp.PlanValue = types.StringValue(stateJSON)
	}
}

// jsonSemanticEqual compares two JSON strings for semantic equality.
// Returns true if both strings parse to equivalent JSON structures.
// It normalizes both structures by removing null values and optional
// fields that n8n might not return (like executeOnce, alwaysOutputData).
func jsonSemanticEqual(a, b string) bool {
	var objA, objB interface{}

	if err := json.Unmarshal([]byte(a), &objA); err != nil {
		return false
	}

	if err := json.Unmarshal([]byte(b), &objB); err != nil {
		return false
	}

	// Normalize both objects to handle n8n API inconsistencies
	normalizedA := normalizeForComparison(objA)
	normalizedB := normalizeForComparison(objB)

	return reflect.DeepEqual(normalizedA, normalizedB)
}

// normalizeForComparison recursively processes JSON data to normalize it for comparison.
// It removes null values and optional node fields that n8n might not consistently return.
func normalizeForComparison(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			// Skip null values
			if value == nil {
				continue
			}
			// Skip optional node fields that n8n doesn't consistently return
			// These fields have default values and may be omitted from API responses
			if isOptionalNodeField(key) {
				continue
			}
			result[key] = normalizeForComparison(value)
		}
		// Return nil for empty maps to handle {} vs missing field equivalence
		if len(result) == 0 {
			return nil
		}
		return result
	case []interface{}:
		result := make([]interface{}, 0, len(v))
		for _, item := range v {
			normalized := normalizeForComparison(item)
			// Don't add nil items to arrays
			if normalized != nil {
				result = append(result, normalized)
			}
		}
		// Return nil for empty arrays
		if len(result) == 0 {
			return nil
		}
		return result
	default:
		return v
	}
}

// isOptionalNodeField returns true for node fields that n8n doesn't consistently
// return in API responses. These have default values and should be ignored
// when comparing config vs state.
func isOptionalNodeField(key string) bool {
	optionalFields := map[string]bool{
		"executeOnce":      true, // Default: false
		"alwaysOutputData": true, // Default: false
		"retryOnFail":      true, // Default: false
		"onError":          true, // Has default value
		"continueOnFail":   true, // Default: false
		"disabled":         true, // Default: false
	}
	return optionalFields[key]
}

// NormalizeJSON takes a JSON string and returns a normalized version
// with consistent formatting. This can be used to ensure consistent
// storage in state.
func NormalizeJSON(input string) (string, error) {
	var obj interface{}
	if err := json.Unmarshal([]byte(input), &obj); err != nil {
		return "", err
	}

	// Marshal with consistent formatting (no indentation, sorted keys for maps)
	normalized, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}

	return string(normalized), nil
}
