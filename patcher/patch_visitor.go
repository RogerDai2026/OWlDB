// Package patcher provides functionality for applying patches to JSON data using a visitor pattern.
// It supports operations such as adding and removing elements from JSON objects and arrays.
package patcher

import (
	"errors"
	"fmt"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/jsondata"
	"log/slog"
	"strings"
)

// PatchVisitor applies a single patch operation to a JSONValue during traversal.
type PatchVisitor struct {
	Patch         PatchOperation // The patch operation to apply
	CurrentPath   string // The current path in the JSON document
	Failed        bool // Flag indicating if the patch operation failed
	FailureReason string // Reason for the failure
	pathFound     bool // Flag indicating if the path was found in the document
}

// NewPatchVisitor creates a new PatchVisitor for a single patch operation.
func NewPatchVisitor(patch PatchOperation) *PatchVisitor {
	return &PatchVisitor{
		Patch:         patch, // The patch operation to apply
		CurrentPath:   "", // The current path in the JSON document
		Failed:        false, // Flag indicating if the patch operation failed
		FailureReason: "", // Reason for the failure
		pathFound:     false, // Flag indicating if the path was found in the document
	}
}

// Map applies the ObjectAdd operation.
// Map processes a JSON object (map) by iterating through its key-value pairs.
// It applies ObjectAdd operations if specified in the patches and then recursively traverses the map.
// Returns the updated JSONValue and an error if the operation fails.

func (pv *PatchVisitor) Map(m map[string]jsondata.JSONValue) (jsondata.JSONValue, error) {
	newMap := make(map[string]jsondata.JSONValue)

	// Copy existing key-value pairs to the new map
	for existingKey, existingValue := range m {
		newMap[existingKey] = existingValue
	}

	//if it's last path we have for patch, check whether it's arrayadd or arrayremove, if so, return error
	if (pv.Patch.Op == "ArrayAdd" || pv.Patch.Op == "ArrayRemove") &&
		pv.CurrentPath == normalizeJSONPointer(pv.Patch.Path) {

		// Set failure flags and reason
		pv.Failed = true
		pv.FailureReason = fmt.Sprintf("error applying patches: path '%s' ends in an object, expected array", pv.Patch.Path)

		// Log the error for debugging
		slog.Error("Invalid patch operation on object", "operation", pv.Patch.Op, "path", pv.Patch.Path)

		// Return an empty JSONValue and the error
		return jsondata.JSONValue{}, errors.New(pv.FailureReason)
	}
	if pv.Patch.Op == "ObjectAdd" && pv.CurrentPath == parentJSONPointer(pv.Patch.Path) {
		patchSegments, err := splitJSONPointer(pv.Patch.Path)
		if err != nil {
			pv.Failed = true
			pv.FailureReason = fmt.Sprintf("Invalid JSON Pointer '%s': %v", pv.Patch.Path, err)
			return jsondata.JSONValue{}, errors.New(pv.FailureReason)
		}

		if len(patchSegments) == 0 {
			pv.Failed = true
			pv.FailureReason = fmt.Sprintf("Invalid path '%s': no key specified", pv.Patch.Path)
			return jsondata.JSONValue{}, errors.New(pv.FailureReason)
		}

		key := patchSegments[len(patchSegments)-1]

		if _, exists := newMap[key]; exists {
			// Key exists, do nothing
			slog.Debug("ObjectAdd operation skipped: key already exists", "key", key, "path", pv.CurrentPath)
			return jsondata.NewJSONValue(m)
		}

		// Add the key-value pair
		pv.pathFound = true
		newMap[key] = pv.Patch.Value
		slog.Debug("ObjectAdd operation applied", "key", key, "path", pv.CurrentPath)
	}

	// Traverse the map
	for k, v := range newMap {
		previousPath := pv.CurrentPath
		pv.CurrentPath = pv.CurrentPath + "/" + escapeJSONPointer(k)

		//check if pathfound == true, if so, it means already did object add. Can return
		// Before recursive call
		if pv.pathFound {
			// The operation has been applied, we can return early
			return jsondata.NewJSONValue(newMap)
		}

		// Recursively accept
		newValue, err := jsondata.Accept[jsondata.JSONValue](v, pv)
		if err != nil {
			return jsondata.JSONValue{}, err
		}
		newMap[k] = newValue

		// Restore the previous path
		pv.CurrentPath = previousPath
	}

	return jsondata.NewJSONValue(newMap)
}

// Slice applies ArrayAdd and ArrayRemove operations.
// Slice processes a JSON array by iterating through its elements.
// Returns the updated JSONValue and an error if the operation fails.
func (pv *PatchVisitor) Slice(s []jsondata.JSONValue) (jsondata.JSONValue, error) {
	slog.Info("PATCHING 3")
	newSlice := make([]jsondata.JSONValue, len(s))
	copy(newSlice, s)

	if (pv.Patch.Op == "ArrayAdd" || pv.Patch.Op == "ArrayRemove") &&
		normalizeJSONPointer(pv.CurrentPath) == normalizeJSONPointer(pv.Patch.Path) {
		if pv.Patch.Op == "ArrayAdd" {
			// Add the value if it's not already in the array
			exists := false
			pv.pathFound = true
			for _, item := range newSlice {
				if item.Equal(pv.Patch.Value) {
					exists = true
					break
				}
			}
			if !exists {
				newSlice = append(newSlice, pv.Patch.Value)
			}
		} else if pv.Patch.Op == "ArrayRemove" {
			// Remove the value if it's in the array
			found := false
			pv.pathFound = true
			tempSlice := []jsondata.JSONValue{}
			for _, item := range newSlice {
				if item.Equal(pv.Patch.Value) {
					found = true
				} else {
					tempSlice = append(tempSlice, item)
				}
			}
			if !found {
				pv.Failed = true
				pv.FailureReason = fmt.Sprintf("Value not found in array at path '%s'", pv.CurrentPath)
				return jsondata.JSONValue{}, errors.New(pv.FailureReason)
			}
			newSlice = tempSlice
		}
	}

	// Traverse the array
	for i, v := range newSlice {
		previousPath := pv.CurrentPath
		pv.CurrentPath = fmt.Sprintf("%s/%d", pv.CurrentPath, i)

		// Recursively accept
		newValue, err := jsondata.Accept[jsondata.JSONValue](v, pv)
		if err != nil {
			return jsondata.JSONValue{}, err
		}
		newSlice[i] = newValue

		// Restore the previous path
		pv.CurrentPath = previousPath
	}
	slog.Info(fmt.Sprintf("newslice is %+v", newSlice))

	return jsondata.NewJSONValue(newSlice)
}

// Bool returns an error if a patch operation targets a boolean value at the current path
func (pv *PatchVisitor) Bool(b bool) (jsondata.JSONValue, error) {
	if normalizeJSONPointer(pv.CurrentPath) == normalizeJSONPointer(pv.Patch.Path) {
		// Patch operation targets a boolean, which is invalid
		pv.Failed = true
		pv.FailureReason = fmt.Sprintf("Cannot apply operation '%s' on a boolean at path '%s'", pv.Patch.Op, pv.Patch.Path)
		return jsondata.JSONValue{}, errors.New(pv.FailureReason)
	}
	// Continue traversal without error
	return jsondata.NewJSONValue(b)
}

// Float64 returns an error if a patch operation targets a float64 value at the current path
func (pv *PatchVisitor) Float64(f float64) (jsondata.JSONValue, error) {
	if normalizeJSONPointer(pv.CurrentPath) == normalizeJSONPointer(pv.Patch.Path) {
		// Patch operation targets a float64, which is invalid
		pv.Failed = true
		pv.FailureReason = fmt.Sprintf("Cannot apply operation '%s' on a number at path '%s'", pv.Patch.Op, pv.Patch.Path)
		return jsondata.JSONValue{}, errors.New(pv.FailureReason)
	}
	// Continue traversal without error
	return jsondata.NewJSONValue(f)
}

// String returns an error if a patch operation targets a string value at the current path
func (pv *PatchVisitor) String(s string) (jsondata.JSONValue, error) {
	if normalizeJSONPointer(pv.CurrentPath) == normalizeJSONPointer(pv.Patch.Path) {
		// Patch operation targets a string, which is invalid
		pv.Failed = true
		pv.FailureReason = fmt.Sprintf("error applying patches:  find string along path")
		return jsondata.JSONValue{}, errors.New(pv.FailureReason)
	}
	// Continue traversal without error
	return jsondata.NewJSONValue(s)
}

// Null returns an error if a patch operation targets a null value at the current path
func (pv *PatchVisitor) Null() (jsondata.JSONValue, error) {
	if normalizeJSONPointer(pv.CurrentPath) == normalizeJSONPointer(pv.Patch.Path) {
		// Patch operation targets null, which is invalid
		pv.Failed = true
		pv.FailureReason = fmt.Sprintf("error applying patches:  find null along path")
		return jsondata.JSONValue{}, errors.New(pv.FailureReason)
	}
	// Continue traversal without error
	return jsondata.NewJSONValue(nil)
}

// Helper functions

// parentJSONPointer returns the parent path of a given JSON pointer.
func parentJSONPointer(ptr string) string {
	if ptr == "" || ptr == "/" {
		return ""
	}
	idx := strings.LastIndex(ptr, "/")
	if idx <= 0 {
		return ""
	}
	return ptr[:idx]
}

// normalizeJSONPointer normalizes a JSON pointer to ensure it follows the correct format.
func normalizeJSONPointer(ptr string) string {
	if ptr == "" {
		return "/"
	}
	return ptr
}

// splitJSONPointer splits a JSON pointer into its individual segments.
func splitJSONPointer(ptr string) ([]string, error) {
	if ptr == "" {
		return []string{}, nil
	}
	if ptr[0] != '/' {
		return nil, fmt.Errorf("invalid JSON Pointer: %s", ptr)
	}
	parts := strings.Split(ptr[1:], "/")
	for i, part := range parts {
		parts[i] = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
	}
	return parts, nil
}

// escapeJSONPointer escapes a string to be safely used as a JSON pointer.
func escapeJSONPointer(s string) string {
	s = strings.ReplaceAll(s, "~", "~0")
	s = strings.ReplaceAll(s, "/", "~1")
	return s
}
