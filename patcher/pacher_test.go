package patcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/jsondata"
	"reflect"
	"strings"
	"testing"
)

// MustNewJSONValue is a helper function that panics if NewJSONValue fails.
func MustNewJSONValue(value interface{}) jsondata.JSONValue {
	v, err := jsondata.NewJSONValue(value)
	if err != nil {
		panic(fmt.Sprintf("Failed to create JSONValue: %v", err))
	}
	return v
}

func TestPatchVisitorArrayAddDuplicate(t *testing.T) {
	initialData := map[string]interface{}{
		"riceFriends": []interface{}{"Alice", "Bob", "Charlie"},
	}
	jsonValueCurrent, _ := jsondata.NewJSONValue(initialData)

	// Attempting to add "Bob", which is already in the array
	operation := PatchOperation{
		Op:    "ArrayAdd",
		Path:  "/riceFriends",
		Value: jsondata.JSONValue(MustNewJSONValue("Bob")),
	}

	visitor := PatchVisitor{
		Patch:         operation,
		CurrentPath:   "",
		Failed:        false,
		FailureReason: "",
	}

	modifiedData, err := jsondata.Accept[jsondata.JSONValue](jsonValueCurrent, &visitor)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	fmt.Println("modifiedData", modifiedData)

	modifiedJSON, err := json.Marshal(modifiedData)
	if err != nil {
		t.Errorf("Failed to marshal modified data: %v", err)
	}
	fmt.Println("modifiedJSON", modifiedJSON)

	var result map[string]interface{}
	err = json.Unmarshal(modifiedJSON, &result)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}
	fmt.Println("result", result)
	expected := []interface{}{"Alice", "Bob", "Charlie"}
	if !reflect.DeepEqual(result["riceFriends"], expected) {
		t.Errorf("Expected 'riceFriends' to be %v, but got %v", expected, result["riceFriends"])
	}
}

// Test for ArrayRemove
func TestPatchVisitor_ArrayRemove(t *testing.T) {
	initialData := map[string]interface{}{
		"riceFriends": []interface{}{"Alice", "Bob", "Charlie"},
	}
	jsonValueCurrent, _ := jsondata.NewJSONValue(initialData)

	operation := PatchOperation{
		Op:    "ArrayRemove",
		Path:  "/riceFriends",
		Value: jsondata.JSONValue(MustNewJSONValue("Bob")),
	}

	visitor := NewPatchVisitor(operation)

	modifiedData, err := jsondata.Accept[jsondata.JSONValue](jsonValueCurrent, visitor)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	modifiedJSON, err := json.Marshal(modifiedData)
	if err != nil {
		t.Errorf("Failed to marshal modified data: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(modifiedJSON, &result)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}

	expected := []interface{}{"Alice", "Charlie"}
	if !reflect.DeepEqual(result["riceFriends"], expected) {
		t.Errorf("Expected 'riceFriends' to be %v, but got %v", expected, result["riceFriends"])
	}
}

// Test for ArrayAdd
func TestPatchVisitor_ArrayAdd_EmptyArray(t *testing.T) {
	initialData := map[string]interface{}{
		"riceFriends": []interface{}{},
	}
	jsonValueCurrent, _ := jsondata.NewJSONValue(initialData)

	operation := PatchOperation{
		Op:    "ArrayAdd",
		Path:  "/riceFriends",
		Value: jsondata.JSONValue(MustNewJSONValue("David")),
	}

	visitor := NewPatchVisitor(operation)

	modifiedData, err := jsondata.Accept[jsondata.JSONValue](jsonValueCurrent, visitor)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	modifiedJSON, err := json.Marshal(modifiedData)
	if err != nil {
		t.Errorf("Failed to marshal modified data: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(modifiedJSON, &result)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}

	expected := []interface{}{"David"}
	if !reflect.DeepEqual(result["riceFriends"], expected) {
		t.Errorf("Expected 'riceFriends' to be %v, but got %v", expected, result["riceFriends"])
	}
}

func TestPatchVisitor_ArrayRemove_NonExistentValue(t *testing.T) {
	initialData := map[string]interface{}{
		"riceFriends": []interface{}{"Alice", "Bob", "Charlie"},
	}
	// Removed unused jsonValueCurrent initialization
	jsonValueCurrent, _ := jsondata.NewJSONValue(initialData)

	operation := PatchOperation{
		Op:    "ArrayRemove",
		Path:  "/riceFriends",
		Value: jsondata.JSONValue(MustNewJSONValue("David")),
	}

	visitor := NewPatchVisitor(operation)

	_, err := jsondata.Accept[jsondata.JSONValue](jsonValueCurrent, visitor)
	if err == nil {
		t.Fatalf("Expected error when removing non-existent value, but got nil")
	}

	expectedError := "Value not found in array"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', but got: %v", expectedError, err)
	}
}

func TestPatcher_InvalidPatchOperation(t *testing.T) {
	initialDoc := []byte(`{
		"riceFriends": ["Alice", "Bob", "Charlie"]
	}`)

	// Invalid patch operation: "InvalidOperation" is not supported
	rawPatches := []byte(`[
		{"op": "InvalidOperation", "path": "/riceFriends", "value": "David"}
	]`)

	patcher := Patcher{}

	_, err := patcher.DoPatch(initialDoc, rawPatches)
	if err == nil {
		t.Fatalf("Expected error for invalid patch operation, but got nil")
	}

	expectedError := "bad patch operation"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', but got: %v", expectedError, err)
	}
}

// TestPatchVisitor_ObjectAdd_NestedObject_AddNewKey
func TestPatchVisitor_ObjectAdd_NestedObject_AddNewKey(t *testing.T) {
	initialData := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": map[string]interface{}{
					"existingKey": "existingValue",
				},
			},
		},
	}
	jsonValueCurrent, _ := jsondata.NewJSONValue(initialData)

	// Patch to add "newKey": "newValue" to "/level1/level2/level3"
	operation := PatchOperation{
		Op:    "ObjectAdd",
		Path:  "/level1/level2/level3/newKey",
		Value: jsondata.JSONValue(MustNewJSONValue("newValue")),
	}

	visitor := NewPatchVisitor(operation)

	modifiedData, err := jsondata.Accept[jsondata.JSONValue](jsonValueCurrent, visitor)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Marshal and unmarshal to compare
	modifiedJSON, err := json.Marshal(modifiedData)
	if err != nil {
		t.Errorf("Failed to marshal modified data: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(modifiedJSON, &result)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}

	expected := map[string]interface{}{
		"existingKey": "existingValue",
		"newKey":      "newValue",
	}

	level3, ok := result["level1"].(map[string]interface{})["level2"].(map[string]interface{})["level3"].(map[string]interface{})
	if !ok {
		t.Fatalf("Failed to access nested level3 map")
	}

	if !reflect.DeepEqual(level3["newKey"], expected["newKey"]) {
		t.Errorf("Expected 'newKey' to be %v, but got %v", expected["newKey"], level3["newKey"])
	}
}

//// TestPatchVisitor_ObjectAdd_SpecialCharacters
//func TestPatchVisitor_ObjectAdd_SpecialCharacters(t *testing.T) {
//	initialData := map[string]interface{}{
//		"settings": map[string]interface{}{
//			"theme": "dark",
//		},
//	}
//	jsonValueCurrent, _ := jsondata.NewJSONValue(initialData)
//
//	// Patch to add a key with special characters
//	operation := PatchOperation{
//		Op:    "ObjectAdd",
//		Path:  "/settings/special~key/with/slash",
//		Value: jsondata.JSONValue(MustNewJSONValue("specialValue")),
//	}
//
//	visitor := NewPatchVisitor(operation)
//
//	modifiedData, err := jsondata.Accept[jsondata.JSONValue](jsonValueCurrent, visitor)
//	if err != nil {
//		t.Errorf("Unexpected error: %v", err)
//	}
//
//}

// TestPatchVisitor_ObjectAdd_DeeplyNestedObject
func TestPatchVisitor_ObjectAdd_DeeplyNestedObject(t *testing.T) {
	initialData := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": map[string]interface{}{
					"level4": map[string]interface{}{
						"existingKey": "existingValue",
					},
				},
			},
		},
	}
	jsonValueCurrent, _ := jsondata.NewJSONValue(initialData)

	// Patch to add "newKey": "newValue" to "/level1/level2/level3/level4"
	operation := PatchOperation{
		Op:    "ObjectAdd",
		Path:  "/level1/level2/level3/level4/newKey",
		Value: jsondata.JSONValue(MustNewJSONValue("newValue")),
	}

	visitor := NewPatchVisitor(operation)

	modifiedData, err := jsondata.Accept[jsondata.JSONValue](jsonValueCurrent, visitor)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Marshal and unmarshal to compare
	modifiedJSON, err := json.Marshal(modifiedData)
	if err != nil {
		t.Errorf("Failed to marshal modified data: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(modifiedJSON, &result)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}

	level4, ok := result["level1"].(map[string]interface{})["level2"].(map[string]interface{})["level3"].(map[string]interface{})["level4"].(map[string]interface{})
	if !ok {
		t.Fatalf("Failed to access nested level4 map")
	}

	if !reflect.DeepEqual(level4["newKey"], "newValue") {
		t.Errorf("Expected 'newKey' to be 'newValue', but got %v", level4["newKey"])
	}
}

// TestPatchVisitor_ArrayAdd_MultipleUniqueValues
func TestPatchVisitor_ArrayAdd_MultipleUniqueValues(t *testing.T) {
	initialData := map[string]interface{}{
		"friends": []interface{}{"Alice", "Bob"},
	}
	jsonValueCurrent, _ := jsondata.NewJSONValue(initialData)

	// Define multiple ArrayAdd operations
	operations := []PatchOperation{
		{
			Op:    "ArrayAdd",
			Path:  "/friends",
			Value: jsondata.JSONValue(MustNewJSONValue("Charlie")),
		},
		{
			Op:    "ArrayAdd",
			Path:  "/friends",
			Value: jsondata.JSONValue(MustNewJSONValue("David")),
		},
	}

	for _, op := range operations {
		visitor := NewPatchVisitor(op)
		var err error
		jsonValueCurrent, err = jsondata.Accept[jsondata.JSONValue](jsonValueCurrent, visitor)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	}

	// Marshal and unmarshal to compare
	modifiedJSON, err := json.Marshal(jsonValueCurrent)
	if err != nil {
		t.Errorf("Failed to marshal modified data: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(modifiedJSON, &result)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}

	expected := []interface{}{"Alice", "Bob", "Charlie", "David"}
	if !reflect.DeepEqual(result["friends"], expected) {
		t.Errorf("Expected 'friends' to be %v, but got %v", expected, result["friends"])
	}
}

// TestPatchVisitor_ArrayAdd_SpecialCharacters
func TestPatchVisitor_ArrayAdd_SpecialCharacters(t *testing.T) {
	initialData := map[string]interface{}{
		"tags": []interface{}{"go", "json"},
	}
	jsonValueCurrent, _ := jsondata.NewJSONValue(initialData)

	// Patch to add a tag with special characters
	operation := PatchOperation{
		Op:    "ArrayAdd",
		Path:  "/tags",
		Value: jsondata.JSONValue(MustNewJSONValue("c++")),
	}

	visitor := NewPatchVisitor(operation)

	modifiedData, err := jsondata.Accept[jsondata.JSONValue](jsonValueCurrent, visitor)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Marshal and unmarshal to compare
	modifiedJSON, err := json.Marshal(modifiedData)
	if err != nil {
		t.Errorf("Failed to marshal modified data: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(modifiedJSON, &result)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}

	expected := []interface{}{"go", "json", "c++"}
	if !reflect.DeepEqual(result["tags"], expected) {
		t.Errorf("Expected 'tags' to be %v, but got %v", expected, result["tags"])
	}
}

// TestPatchVisitor_ArrayAdd_NestedArray
func TestPatchVisitor_ArrayAdd_NestedArray(t *testing.T) {
	initialData := map[string]interface{}{
		"teams": []interface{}{
			map[string]interface{}{
				"name": "TeamA",
				"members": []interface{}{
					"Alice",
					"Bob",
				},
			},
		},
	}
	jsonValueCurrent, _ := jsondata.NewJSONValue(initialData)

	// Patch to add "Charlie" to "/teams/0/members"
	operation := PatchOperation{
		Op:    "ArrayAdd",
		Path:  "/teams/0/members",
		Value: jsondata.JSONValue(MustNewJSONValue("Charlie")),
	}

	visitor := NewPatchVisitor(operation)

	modifiedData, err := jsondata.Accept[jsondata.JSONValue](jsonValueCurrent, visitor)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Marshal and unmarshal to compare
	modifiedJSON, err := json.Marshal(modifiedData)
	if err != nil {
		t.Errorf("Failed to marshal modified data: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(modifiedJSON, &result)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}

	expected := []interface{}{
		map[string]interface{}{
			"name": "TeamA",
			"members": []interface{}{
				"Alice",
				"Bob",
				"Charlie",
			},
		},
	}

	if !reflect.DeepEqual(result["teams"], expected) {
		t.Errorf("Expected 'teams' to be %v, but got %v", expected, result["teams"])
	}
}

// TestPatchVisitor_ArrayRemove_MultipleExistingValues
func TestPatchVisitor_ArrayRemove_MultipleExistingValues(t *testing.T) {
	initialData := map[string]interface{}{
		"friends": []interface{}{"Alice", "Bob", "Charlie", "David"},
	}
	jsonValueCurrent, _ := jsondata.NewJSONValue(initialData)

	// Define multiple ArrayRemove operations
	operations := []PatchOperation{
		{
			Op:    "ArrayRemove",
			Path:  "/friends",
			Value: jsondata.JSONValue(MustNewJSONValue("Bob")),
		},
		{
			Op:    "ArrayRemove",
			Path:  "/friends",
			Value: jsondata.JSONValue(MustNewJSONValue("Charlie")),
		},
	}

	for _, op := range operations {
		visitor := NewPatchVisitor(op)
		var err error
		jsonValueCurrent, err = jsondata.Accept[jsondata.JSONValue](jsonValueCurrent, visitor)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	}

	// Marshal and unmarshal to compare
	modifiedJSON, err := json.Marshal(jsonValueCurrent)
	if err != nil {
		t.Errorf("Failed to marshal modified data: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(modifiedJSON, &result)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}

	expected := []interface{}{"Alice", "David"}
	if !reflect.DeepEqual(result["friends"], expected) {
		t.Errorf("Expected 'friends' to be %v, but got %v", expected, result["friends"])
	}
}

// TestPatchVisitor_ArrayRemove_FromEmptyArray
func TestPatchVisitor_ArrayRemove_FromEmptyArray(t *testing.T) {
	initialData := map[string]interface{}{
		"friends": []interface{}{},
	}
	jsonValueCurrent, _ := jsondata.NewJSONValue(initialData)

	// Patch to remove "Alice" from an empty array
	operation := PatchOperation{
		Op:    "ArrayRemove",
		Path:  "/friends",
		Value: jsondata.JSONValue(MustNewJSONValue("Alice")),
	}

	visitor := NewPatchVisitor(operation)

	_, err := jsondata.Accept[jsondata.JSONValue](jsonValueCurrent, visitor)
	if err == nil {
		t.Fatalf("Expected error when removing from empty array, but got none")
	}

	expectedError := "Value not found in array"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', but got: %v", expectedError, err)
	}
}

// TestPatchVisitor_ArrayRemove_NestedArray
func TestPatchVisitor_ArrayRemove_NestedArray(t *testing.T) {
	initialData := map[string]interface{}{
		"teams": []interface{}{
			map[string]interface{}{
				"name":    "TeamA",
				"members": []interface{}{"Alice", "Bob", "Charlie"},
			},
		},
	}
	jsonValueCurrent, _ := jsondata.NewJSONValue(initialData)

	// Patch to remove "Bob" from "/teams/0/members"
	operation := PatchOperation{
		Op:    "ArrayRemove",
		Path:  "/teams/0/members",
		Value: jsondata.JSONValue(MustNewJSONValue("Bob")),
	}

	visitor := NewPatchVisitor(operation)

	modifiedData, err := jsondata.Accept[jsondata.JSONValue](jsonValueCurrent, visitor)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Marshal and unmarshal to compare
	modifiedJSON, err := json.Marshal(modifiedData)
	if err != nil {
		t.Errorf("Failed to marshal modified data: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(modifiedJSON, &result)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}

	expected := []interface{}{
		map[string]interface{}{
			"name": "TeamA",
			"members": []interface{}{
				"Alice",
				"Charlie",
			},
		},
	}

	if !reflect.DeepEqual(result["teams"], expected) {
		t.Errorf("Expected 'teams' to be %v, but got %v", expected, result["teams"])
	}
}

// TestPatchVisitor_SequentialPatchOperations
func TestPatchVisitor_SequentialPatchOperations(t *testing.T) {
	initialData := map[string]interface{}{
		"friends": []interface{}{"Alice", "Bob"},
		"settings": map[string]interface{}{
			"theme": "dark",
		},
	}
	jsonValueCurrent, _ := jsondata.NewJSONValue(initialData)

	// Define a sequence of patch operations
	operations := []PatchOperation{
		{
			Op:    "ArrayAdd",
			Path:  "/friends",
			Value: jsondata.JSONValue(MustNewJSONValue("Charlie")),
		},
		{
			Op:    "ObjectAdd",
			Path:  "/settings/notifications",
			Value: jsondata.JSONValue(MustNewJSONValue(true)),
		},
		{
			Op:    "ArrayRemove",
			Path:  "/friends",
			Value: jsondata.JSONValue(MustNewJSONValue("Bob")),
		},
		{
			Op:    "ArrayAdd",
			Path:  "/friends",
			Value: jsondata.JSONValue(MustNewJSONValue("David")),
		},
	}

	for _, op := range operations {
		visitor := NewPatchVisitor(op)
		var err error
		jsonValueCurrent, err = jsondata.Accept[jsondata.JSONValue](jsonValueCurrent, visitor)
		if err != nil {
			t.Errorf("Unexpected error during operation %v: %v", op, err)
		}
	}

	// Marshal and unmarshal to compare
	modifiedJSON, err := json.Marshal(jsonValueCurrent)
	if err != nil {
		t.Errorf("Failed to marshal modified data: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(modifiedJSON, &result)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}

	expectedFriends := []interface{}{"Alice", "Charlie", "David"}
	expectedSettings := map[string]interface{}{
		"theme":         "dark",
		"notifications": true,
	}

	// Check 'friends' array
	if !reflect.DeepEqual(result["friends"], expectedFriends) {
		t.Errorf("Expected 'friends' to be %v, but got %v", expectedFriends, result["friends"])
	}

	// Check 'settings' object
	if !reflect.DeepEqual(result["settings"], expectedSettings) {
		t.Errorf("Expected 'settings' to be %v, but got %v", expectedSettings, result["settings"])
	}
}

// TestPatchVisitor_SequentialPatchOperations_MultipleTypes
func TestPatchVisitor_SequentialPatchOperations_MultipleTypes(t *testing.T) {
	initialData := map[string]interface{}{
		"settings": map[string]interface{}{
			"theme": "light",
		},
		"friends": []interface{}{"Alice", "Bob"},
	}
	jsonValueCurrent, _ := jsondata.NewJSONValue(initialData)

	// Define a sequence of diverse patch operations
	operations := []PatchOperation{
		{
			Op:    "ObjectAdd",
			Path:  "/settings/notifications",
			Value: jsondata.JSONValue(MustNewJSONValue(true)),
		},
		{
			Op:    "ArrayAdd",
			Path:  "/friends",
			Value: jsondata.JSONValue(MustNewJSONValue("Charlie")),
		},
		{
			Op:    "ArrayRemove",
			Path:  "/friends",
			Value: jsondata.JSONValue(MustNewJSONValue("Alice")),
		},
		{
			Op:    "ObjectAdd",
			Path:  "/settings/language",
			Value: jsondata.JSONValue(MustNewJSONValue("en-US")),
		},
		{
			Op:    "ArrayAdd",
			Path:  "/friends",
			Value: jsondata.JSONValue(MustNewJSONValue("David")),
		},
		{
			Op:    "ArrayRemove",
			Path:  "/friends",
			Value: jsondata.JSONValue(MustNewJSONValue("Bob")),
		},
	}

	for _, op := range operations {
		visitor := NewPatchVisitor(op)
		var err error
		jsonValueCurrent, err = jsondata.Accept[jsondata.JSONValue](jsonValueCurrent, visitor)
		if err != nil {
			t.Errorf("Unexpected error during operation %v: %v", op, err)
		}
	}

	// Marshal and unmarshal to compare
	modifiedJSON, err := json.Marshal(jsonValueCurrent)
	if err != nil {
		t.Errorf("Failed to marshal modified data: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(modifiedJSON, &result)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}

	expectedFriends := []interface{}{"Charlie", "David"}
	expectedSettings := map[string]interface{}{
		"theme":         "light",
		"notifications": true,
		"language":      "en-US",
	}

	// Check 'friends' array
	if !reflect.DeepEqual(result["friends"], expectedFriends) {
		t.Errorf("Expected 'friends' to be %v, but got %v", expectedFriends, result["friends"])
	}

	// Check 'settings' object
	if !reflect.DeepEqual(result["settings"], expectedSettings) {
		t.Errorf("Expected 'settings' to be %v, but got %v", expectedSettings, result["settings"])
	}
}

func TestPatcher_DoPatch_MalformedOriginalDocument(t *testing.T) {
	// Malformed JSON (missing closing brace)
	oldRawDoc := []byte(`{ "data": "value" `)

	// Valid patches (won't be used due to malformed document)
	rawPatches := []byte(`[{"op": "ObjectAdd", "path": "/newKey", "value": "newValue"}]`)

	patcher := Patcher{}

	newDoc, err := patcher.DoPatch(oldRawDoc, rawPatches)

	if err == nil {
		t.Fatalf("Expected error due to malformed original document, but got nil")
	}

	expectedError := "malformed raw document payload"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', but got '%v'", expectedError, err)
	}

	if !bytes.Equal(newDoc, oldRawDoc) {
		t.Errorf("Expected original document to be returned, but got '%s'", newDoc)
	}
}

func TestPatcher_DoPatch_MalformedPatches(t *testing.T) {
	// Valid original document
	oldRawDoc := []byte(`{ "data": "value" }`)

	// Malformed patches (missing closing bracket)
	rawPatches := []byte(`[{"op": "ObjectAdd", "path": "/newKey", "value": "newValue"]`)

	patcher := Patcher{}

	newDoc, err := patcher.DoPatch(oldRawDoc, rawPatches)

	if err == nil {
		t.Fatalf("Expected error due to malformed patches, but got nil")
	}

	expectedError := "invalid character"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', but got '%v'", expectedError, err)
	}

	if newDoc != nil {
		t.Errorf("Expected newDoc to be nil, but got '%s'", newDoc)
	}
}

func TestPatcher_DoPatch_InvalidPatchOperation(t *testing.T) {
	oldRawDoc := []byte(`{ "data": "value" }`)

	patches := []PatchOperation{
		{
			Op:    "InvalidOp",
			Path:  "/newKey",
			Value: MustNewJSONValue("newValue"),
		},
	}
	rawPatches, _ := json.Marshal(patches)

	patcher := Patcher{}

	newDoc, err := patcher.DoPatch(oldRawDoc, rawPatches)

	if err == nil {
		t.Fatalf("Expected error due to invalid patch operation, but got nil")
	}

	expectedError := "bad patch operation: InvalidOp"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', but got '%v'", expectedError, err)
	}

	if !bytes.Equal(newDoc, oldRawDoc) {
		t.Errorf("Expected original document to be returned, but got '%s'", newDoc)
	}
}

func TestPatcher_DoPatch_PathNotFound(t *testing.T) {
	oldRawDoc := []byte(`{ "data": { "key": "value" } }`)

	patches := []PatchOperation{
		{
			Op:    "ObjectAdd",
			Path:  "/nonexistent/path",
			Value: MustNewJSONValue("newValue"),
		},
	}
	rawPatches, _ := json.Marshal(patches)

	patcher := Patcher{}

	newDoc, err := patcher.DoPatch(oldRawDoc, rawPatches)

	if err == nil {
		t.Fatalf("Expected error due to non-existent path, but got nil")
	}

	expectedError := "Error Applying patches:  Path '/nonexistent/path' does not exist in the document"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error message to contain '%s', but got '%v'", expectedError, err)
	}

	if newDoc != nil {
		t.Errorf("Expected newDoc to be nil, but got '%s'", newDoc)
	}
}

func TestPatcher_DoPatch_ArrayAddToObject(t *testing.T) {
	oldRawDoc := []byte(`{ "data": { "key": "value" } }`)

	patches := []PatchOperation{
		{
			Op:    "ArrayAdd",
			Path:  "/data",
			Value: MustNewJSONValue("newValue"),
		},
	}
	rawPatches, _ := json.Marshal(patches)

	patcher := Patcher{}

	newDoc, err := patcher.DoPatch(oldRawDoc, rawPatches)

	if err == nil {
		t.Fatalf("Expected error due to type mismatch, but got nil")
	}

	expectedError := "error applying patches: path '/data' ends in an object, expected array"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error message to contain '%s', but got '%v'", expectedError, err)
	}

	if !bytes.Equal(newDoc, oldRawDoc) {
		t.Errorf("Expected original document to be returned, but got '%s'", newDoc)
	}
}

func TestPatchVisitor_PatchToScalar(t *testing.T) {
	jsonValue := MustNewJSONValue("scalar value")

	patch := PatchOperation{
		Op:    "ObjectAdd",
		Path:  "",
		Value: MustNewJSONValue("newValue"),
	}

	visitor := NewPatchVisitor(patch)

	_, err := jsondata.Accept[jsondata.JSONValue](jsonValue, visitor)

	if err == nil {
		t.Fatalf("Expected error due to applying patch to scalar, but got nil")
	}

	expectedError := "error applying patches:  find string along path"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error message to contain '%s', but got '%v'", expectedError, err)
	}
}

func TestPatchV_ArrayRemove_Nonexistence(t *testing.T) {
	initialData := map[string]interface{}{
		"array": []interface{}{"item1", "item2"},
	}
	jsonValue, _ := jsondata.NewJSONValue(initialData)

	patch := PatchOperation{
		Op:    "ArrayRemove",
		Path:  "/array",
		Value: MustNewJSONValue("item3"),
	}

	visitor := NewPatchVisitor(patch)

	_, err := jsondata.Accept[jsondata.JSONValue](jsonValue, visitor)

	if err == nil {
		t.Fatalf("Expected error due to removing non-existent value, but got nil")
	}

	expectedError := "Value not found in array at path '/array'"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error message to contain '%s', but got '%v'", expectedError, err)
	}
}

func TestSplitJSONPointer_Invalid(t *testing.T) {
	pointer := "invalid/pointer"

	_, err := splitJSONPointer(pointer)

	if err == nil {
		t.Fatalf("Expected error due to invalid JSON Pointer, but got nil")
	}

	expectedError := "invalid JSON Pointer"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error message to contain '%s', but got '%v'", expectedError, err)
	}
}

func TestNormalizeJSONPointer(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "/"},
		{"/", "/"},
		{"/path", "/path"},
	}

	for _, tt := range tests {
		result := normalizeJSONPointer(tt.input)
		if result != tt.expected {
			t.Errorf("Expected '%s', got '%s'", tt.expected, result)
		}
	}
}

func TestPatchVisitor_ArrayAdd_OnObject(t *testing.T) {
	// Initial JSON data: an object
	initialData := map[string]interface{}{
		"user": map[string]interface{}{
			"name": "Alice",
		},
	}

	// Create a JSONValue from the initial data
	jsonValue, err := jsondata.NewJSONValue(initialData)
	if err != nil {
		t.Fatalf("Failed to create JSONValue: %v", err)
	}

	// Create a PatchOperation that attempts to perform ArrayAdd on an object path
	patch := PatchOperation{
		Op:    "ArrayAdd",
		Path:  "/user",
		Value: MustNewJSONValue("Bob"),
	}

	// Create a PatchVisitor with the patch
	visitor := NewPatchVisitor(patch)

	// Apply the patch
	_, err = jsondata.Accept[jsondata.JSONValue](jsonValue, visitor)

	// Check for the expected error
	if err == nil {
		t.Fatalf("Expected error when applying ArrayAdd on an object, but got none")
	}

	expectedError := fmt.Sprintf("error applying patches: path '%s' ends in an object, expected array", patch.Path)
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', but got '%s'", expectedError, err.Error())
	}
}

func TestPatchVisitor_ArrayRemove_OnObject(t *testing.T) {
	// Initial JSON data: an object
	initialData := map[string]interface{}{
		"config": map[string]interface{}{
			"enabled": true,
		},
	}

	// Create a JSONValue from the initial data
	jsonValue, err := jsondata.NewJSONValue(initialData)
	if err != nil {
		t.Fatalf("Failed to create JSONValue: %v", err)
	}

	// Create a PatchOperation that attempts to perform ArrayRemove on an object path
	patch := PatchOperation{
		Op:    "ArrayRemove",
		Path:  "/config",
		Value: MustNewJSONValue(true),
	}

	// Create a PatchVisitor with the patch
	visitor := NewPatchVisitor(patch)

	// Apply the patch
	_, err = jsondata.Accept[jsondata.JSONValue](jsonValue, visitor)

	// Check for the expected error
	if err == nil {
		t.Fatalf("Expected error when applying ArrayRemove on an object, but got none")
	}

	expectedError := fmt.Sprintf("error applying patches: path '%s' ends in an object, expected array", patch.Path)
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', but got '%s'", expectedError, err.Error())
	}
}

func TestPatchVisitor_ObjectAdd_InvalidPath(t *testing.T) {
	// Initial JSON data
	initialData := map[string]interface{}{
		"data": "value",
	}

	// Create a JSONValue from the initial data
	jsonValue, err := jsondata.NewJSONValue(initialData)
	if err != nil {
		t.Fatalf("Failed to create JSONValue: %v", err)
	}

	// Attempt to add with an invalid path (empty string)
	patch := PatchOperation{
		Op:    "ObjectAdd",
		Path:  "",
		Value: MustNewJSONValue("newValue"),
	}

	// Create a PatchVisitor with the patch
	visitor := NewPatchVisitor(patch)

	// Apply the patch
	_, err = jsondata.Accept[jsondata.JSONValue](jsonValue, visitor)

	// Check for the expected error
	if err == nil {
		t.Fatalf("Expected error due to invalid path, but got none")
	}

	expectedError := fmt.Sprintf("Invalid path '%s': no key specified", patch.Path)
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', but got '%s'", expectedError, err.Error())
	}
}

func TestPatcher_DoPatch_InvalidOldRawDoc(t *testing.T) {
	// Malformed JSON in oldRawDoc (missing closing brace)
	oldRawDoc := []byte(`{ "data": "value" `)

	// Valid patches
	patches := []PatchOperation{
		{
			Op:    "ObjectAdd",
			Path:  "/newKey",
			Value: MustNewJSONValue("newValue"),
		},
	}
	rawPatches, err := json.Marshal(patches)
	if err != nil {
		t.Fatalf("Failed to marshal patches: %v", err)
	}

	patcher := Patcher{}

	// Apply patches
	newDoc, err := patcher.DoPatch(oldRawDoc, rawPatches)

	// Check for expected error
	if err == nil {
		t.Fatalf("Expected error due to invalid oldRawDoc, but got none")
	}

	expectedError := "malformed raw document payload"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', but got '%s'", expectedError, err.Error())
	}

	// Verify that the original document is returned
	if !bytes.Equal(newDoc, oldRawDoc) {
		t.Errorf("Expected original document to be returned, but got '%s'", newDoc)
	}
}

func TestPatcher_DoPatch_InvalidRawPatches(t *testing.T) {
	// Valid oldRawDoc
	oldRawDoc := []byte(`{ "data": "value" }`)

	// Malformed rawPatches (missing closing bracket)
	rawPatches := []byte(`[{"op": "ObjectAdd", "path": "/newKey", "value": "newValue"}`)

	patcher := Patcher{}

	// Apply patches
	newDoc, err := patcher.DoPatch(oldRawDoc, rawPatches)

	// Check for expected error
	if err == nil {
		t.Fatalf("Expected error due to invalid rawPatches, but got none")
	}

	expectedErrorSubstring := "unexpected end of JSON input"
	if !strings.Contains(err.Error(), expectedErrorSubstring) {
		t.Errorf("Expected error message to contain '%s', but got '%s'", expectedErrorSubstring, err.Error())
	}

	// Verify that newDoc is nil as per your code
	if newDoc != nil {
		t.Errorf("Expected newDoc to be nil, but got '%s'", newDoc)
	}
}

func TestPatcher_DoPatch_InvalidPatch(t *testing.T) {
	// Valid oldRawDoc
	oldRawDoc := []byte(`{ "data": "value" }`)

	// Patches with invalid operation
	patches := []PatchOperation{
		{
			Op:    "InvalidOp",
			Path:  "/newKey",
			Value: MustNewJSONValue("newValue"),
		},
	}
	rawPatches, err := json.Marshal(patches)
	if err != nil {
		t.Fatalf("Failed to marshal patches: %v", err)
	}

	patcher := Patcher{}

	// Apply patches
	newDoc, err := patcher.DoPatch(oldRawDoc, rawPatches)

	// Check for expected error
	if err == nil {
		t.Fatalf("Expected error due to invalid patch operation, but got none")
	}

	expectedError := "bad patch operation: InvalidOp"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', but got '%s'", expectedError, err.Error())
	}

	// Verify that the original document is returned
	if !bytes.Equal(newDoc, oldRawDoc) {
		t.Errorf("Expected original document to be returned, but got '%s'", newDoc)
	}
}

func TestPatchVisitor_Bool_Targeted(t *testing.T) {
	// Initial JSON data with a boolean at path "/isActive"
	initialData := map[string]interface{}{
		"isActive": true,
		"user": map[string]interface{}{
			"name":       "Alice",
			"isVerified": false,
		},
	}
	jsonValue, err := jsondata.NewJSONValue(initialData)
	if err != nil {
		t.Fatalf("Failed to create JSONValue: %v", err)
	}

	// Patch operation targeting the boolean value at "/isActive"
	patch := PatchOperation{
		Op:    "ObjectAdd",
		Path:  "/isActive",
		Value: MustNewJSONValue("newValue"),
	}

	// Create the PatchVisitor
	visitor := NewPatchVisitor(patch)

	// Apply the patch
	_, err = jsondata.Accept[jsondata.JSONValue](jsonValue, visitor)

	// Check for the expected error
	if err != nil {
		t.Fatalf("Error during visit: %v", err)
	}
}

func TestPatchVisitor_Float_Targeted(t *testing.T) {
	// Initial JSON data with a boolean at path "/isActive"
	initialData := map[string]interface{}{
		"isActive": 91.4,
		"user": map[string]interface{}{
			"name":       "Alice",
			"isVerified": false,
		},
	}
	jsonValue, err := jsondata.NewJSONValue(initialData)
	if err != nil {
		t.Fatalf("Failed to create JSONValue: %v", err)
	}

	// Patch operation targeting the boolean value at "/isActive"
	patch := PatchOperation{
		Op:    "ObjectAdd",
		Path:  "/isActive",
		Value: MustNewJSONValue("newValue"),
	}

	// Create the PatchVisitor
	visitor := NewPatchVisitor(patch)

	// Apply the patch
	_, err = jsondata.Accept[jsondata.JSONValue](jsonValue, visitor)

	// Check for the expected error
	if err != nil {
		t.Fatalf("Error during visit: %v", err)
	}
}

func TestPatchVisitor_Null_Targeted(t *testing.T) {
	// Initial JSON data with a null at path "/user/middleName"
	initialData := map[string]interface{}{
		"user": map[string]interface{}{
			"firstName":  "Dave",
			"middleName": nil,
			"lastName":   "Smith",
		},
	}
	jsonValue, err := jsondata.NewJSONValue(initialData)
	if err != nil {
		t.Fatalf("Failed to create JSONValue: %v", err)
	}

	// Patch operation targeting the null value at "/user/middleName"
	patch := PatchOperation{
		Op:    "ObjectAdd",
		Path:  "/user/middleName",
		Value: MustNewJSONValue("newValue"),
	}

	// Create the PatchVisitor
	visitor := NewPatchVisitor(patch)

	// Apply the patch
	_, err = jsondata.Accept[jsondata.JSONValue](jsonValue, visitor)

	// Check for the expected error
	if err != nil {
		t.Fatalf("Expected error when applying patch to null, but got none")
	}
}

func TestParentJSONPointer(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Root path",
			input:    "/",
			expected: "",
		},
		{
			name:     "Single segment",
			input:    "/a",
			expected: "",
		},
		{
			name:     "Two segments",
			input:    "/a/b",
			expected: "/a",
		},
		{
			name:     "Multiple segments",
			input:    "/a/b/c",
			expected: "/a/b",
		},
		{
			name:     "Segments with escaped characters",
			input:    "/a~1b/c~0d",
			expected: "/a~1b",
		},
		{
			name:     "Path ending with slash",
			input:    "/a/b/",
			expected: "/a/b",
		},
		{
			name:     "Path with multiple trailing slashes",
			input:    "/a/b///",
			expected: "/a/b//",
		},
		{
			name:     "Path with tilde and slash sequences",
			input:    "/~0/~1/~01/~10",
			expected: "/~0/~1/~01",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parentJSONPointer(tt.input)
			if result != tt.expected {
				t.Errorf("parentJSONPointer('%s') expected '%s', got '%s'", tt.input, tt.expected, result)
			}
		})
	}
}
