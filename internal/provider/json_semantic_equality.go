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
func jsonSemanticEqual(a, b string) bool {
	var objA, objB interface{}

	if err := json.Unmarshal([]byte(a), &objA); err != nil {
		return false
	}

	if err := json.Unmarshal([]byte(b), &objB); err != nil {
		return false
	}

	return reflect.DeepEqual(objA, objB)
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
