// Copyright (c) Arthur Diniz <arthurbdiniz@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package n8n

import (
	"encoding/json"
	"testing"
)

// TestNodeCredentialsRoundTrip verifies that credentials survive JSON marshaling/unmarshaling.
func TestNodeCredentialsRoundTrip(t *testing.T) {
	// Create a node with credentials
	originalNode := Node{
		ID:          "get-doc",
		Name:        "Get Google Doc",
		Type:        "n8n-nodes-base.googleDocs",
		TypeVersion: 2,
		Position:    []int{220, 300},
		Parameters: map[string]interface{}{
			"operation":   "get",
			"documentURL": "https://docs.google.com/document/d/test",
		},
		Credentials: map[string]interface{}{
			"googleDocsOAuth2Api": map[string]interface{}{
				"id":   "UGckzzkZk8YRR0Lt",
				"name": "Google Docs account",
			},
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(originalNode)
	if err != nil {
		t.Fatalf("Failed to marshal node to JSON: %v", err)
	}

	// Unmarshal back to Node
	var roundTripNode Node
	if err := json.Unmarshal(jsonData, &roundTripNode); err != nil {
		t.Fatalf("Failed to unmarshal node from JSON: %v", err)
	}

	// Verify credentials are preserved
	if roundTripNode.Credentials == nil {
		t.Fatal("Credentials were lost during JSON round-trip")
	}

	creds, ok := roundTripNode.Credentials["googleDocsOAuth2Api"]
	if !ok {
		t.Fatal("googleDocsOAuth2Api credential was lost during JSON round-trip")
	}

	credMap, ok := creds.(map[string]interface{})
	if !ok {
		t.Fatal("Credential value is not a map")
	}

	if credMap["id"] != "UGckzzkZk8YRR0Lt" {
		t.Errorf("Credential ID mismatch: expected %q, got %q", "UGckzzkZk8YRR0Lt", credMap["id"])
	}

	if credMap["name"] != "Google Docs account" {
		t.Errorf("Credential name mismatch: expected %q, got %q", "Google Docs account", credMap["name"])
	}
}

// TestNodeWithoutCredentials verifies that nodes without credentials work correctly.
func TestNodeWithoutCredentials(t *testing.T) {
	// Create a node without credentials (like a Manual Trigger)
	originalNode := Node{
		ID:          "trigger",
		Name:        "Manual Trigger",
		Type:        "n8n-nodes-base.manualTrigger",
		TypeVersion: 1,
		Position:    []int{0, 0},
		Parameters:  map[string]interface{}{},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(originalNode)
	if err != nil {
		t.Fatalf("Failed to marshal node to JSON: %v", err)
	}

	// Verify "credentials" is not in the JSON (due to omitempty)
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &jsonMap); err != nil {
		t.Fatalf("Failed to unmarshal JSON to map: %v", err)
	}

	if _, exists := jsonMap["credentials"]; exists {
		t.Error("credentials field should be omitted from JSON when empty")
	}

	// Unmarshal back to Node
	var roundTripNode Node
	if err := json.Unmarshal(jsonData, &roundTripNode); err != nil {
		t.Fatalf("Failed to unmarshal node from JSON: %v", err)
	}

	// Credentials should be nil
	if roundTripNode.Credentials != nil {
		t.Error("Credentials should be nil for nodes without credentials")
	}
}

// TestWorkflowWithMixedCredentials tests a workflow with some nodes having credentials and some not.
func TestWorkflowWithMixedCredentials(t *testing.T) {
	workflow := Workflow{
		ID:     "test-workflow",
		Name:   "Test Workflow",
		Active: true,
		Nodes: []Node{
			{
				ID:          "trigger",
				Name:        "Manual Trigger",
				Type:        "n8n-nodes-base.manualTrigger",
				TypeVersion: 1,
				Position:    []int{0, 0},
				Parameters:  map[string]interface{}{},
			},
			{
				ID:          "google-docs",
				Name:        "Get Doc",
				Type:        "n8n-nodes-base.googleDocs",
				TypeVersion: 2,
				Position:    []int{220, 0},
				Parameters:  map[string]interface{}{"operation": "get"},
				Credentials: map[string]interface{}{
					"googleDocsOAuth2Api": map[string]interface{}{
						"id":   "cred-1",
						"name": "Google Account",
					},
				},
			},
			{
				ID:          "slack",
				Name:        "Send Message",
				Type:        "n8n-nodes-base.slack",
				TypeVersion: 2,
				Position:    []int{440, 0},
				Parameters:  map[string]interface{}{"channel": "#general"},
				Credentials: map[string]interface{}{
					"slackOAuth2Api": map[string]interface{}{
						"id":   "cred-2",
						"name": "Slack Bot",
					},
				},
			},
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(workflow)
	if err != nil {
		t.Fatalf("Failed to marshal workflow to JSON: %v", err)
	}

	// Unmarshal back to Workflow
	var roundTripWorkflow Workflow
	if err := json.Unmarshal(jsonData, &roundTripWorkflow); err != nil {
		t.Fatalf("Failed to unmarshal workflow from JSON: %v", err)
	}

	// Verify we have 3 nodes
	if len(roundTripWorkflow.Nodes) != 3 {
		t.Fatalf("Expected 3 nodes, got %d", len(roundTripWorkflow.Nodes))
	}

	// Verify trigger has no credentials
	if roundTripWorkflow.Nodes[0].Credentials != nil {
		t.Error("Manual Trigger should not have credentials")
	}

	// Verify Google Docs has credentials
	if roundTripWorkflow.Nodes[1].Credentials == nil {
		t.Fatal("Google Docs node should have credentials")
	}
	if _, ok := roundTripWorkflow.Nodes[1].Credentials["googleDocsOAuth2Api"]; !ok {
		t.Error("Google Docs node missing googleDocsOAuth2Api credential")
	}

	// Verify Slack has credentials
	if roundTripWorkflow.Nodes[2].Credentials == nil {
		t.Fatal("Slack node should have credentials")
	}
	if _, ok := roundTripWorkflow.Nodes[2].Credentials["slackOAuth2Api"]; !ok {
		t.Error("Slack node missing slackOAuth2Api credential")
	}
}

// TestAPIResponseParsing simulates parsing an n8n API response with credentials.
func TestAPIResponseParsing(t *testing.T) {
	// This is what the n8n API returns
	apiResponse := `{
		"nodes": [
			{
				"id": "get-doc",
				"name": "Get Google Doc",
				"type": "n8n-nodes-base.googleDocs",
				"typeVersion": 2,
				"position": [220, 300],
				"parameters": {
					"operation": "get",
					"documentURL": "https://docs.google.com/document/d/test"
				},
				"credentials": {
					"googleDocsOAuth2Api": {
						"id": "UGckzzkZk8YRR0Lt",
						"name": "Google Docs account"
					}
				}
			}
		]
	}`

	var data struct {
		Nodes []Node `json:"nodes"`
	}

	if err := json.Unmarshal([]byte(apiResponse), &data); err != nil {
		t.Fatalf("Failed to parse API response: %v", err)
	}

	if len(data.Nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(data.Nodes))
	}

	node := data.Nodes[0]
	if node.Credentials == nil {
		t.Fatal("Credentials were not parsed from API response")
	}

	creds, ok := node.Credentials["googleDocsOAuth2Api"]
	if !ok {
		t.Fatal("googleDocsOAuth2Api credential was not parsed")
	}

	credMap, ok := creds.(map[string]interface{})
	if !ok {
		t.Fatalf("Credential is not a map, got %T", creds)
	}

	if credMap["id"] != "UGckzzkZk8YRR0Lt" {
		t.Errorf("Credential ID mismatch: got %v", credMap["id"])
	}
}
