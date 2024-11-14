// Package patcher provides functionality for applying a sequence of patches to JSON documents.
// It defines structures and methods for handling patch operations and applying them to JSON data.
package patcher

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/jsondata"
	"log/slog"
)

// PatchOperation defines a single patch operation to be applied to a JSON document.
// It includes the operation type (Op), the path in the document (Path), and the value to be applied (Value).
type PatchOperation struct {
	Op    string             `json:"op"`    // The operation type (e.g., "add", "remove").
	Path  string             `json:"path"`  // The JSON Pointer path specifying where the operation should be applied.
	Value jsondata.JSONValue `json:"value"` // The value to add, remove, or modify at the specified path.
}

// Patcher is responsible for applying a sequence of patches to a JSON document.
type Patcher struct {
}

// DoPatch takes the raw bytes from both the original document, a json-encoded list of patch objects, and applies them sequentially
// returns whether the patch was successful or not
func (p Patcher) DoPatch(oldRawDoc []byte, rawPatches []byte) (newDoc []byte, err error) {

	var jsonValue jsondata.JSONValue
	slog.Info("PATCHING")

	err = json.Unmarshal(oldRawDoc, &jsonValue)

	if err != nil { //this should NOT fail
		slog.Warn("WARNING: A BAD DOCUMENT PAYLOAD IS IN THE DATABASE")
		return oldRawDoc, fmt.Errorf("malformed raw document payload")
	}

	var patches []PatchOperation
	if err := json.Unmarshal(rawPatches, &patches); err != nil {
		slog.Debug("Invalid PATCH")
		return nil, err
	}
	patchErr := validatePatches(patches)
	if patchErr != nil {
		return oldRawDoc, patchErr
	}
	for _, patchOp := range patches {
		slog.Info("PATCHING 2")
		// Create a new PatchVisitor for the current patch operation
		patchVisitor := NewPatchVisitor(patchOp)

		// Apply the patch to the current jsonValue

		//I dont think you need to type cast here since the visitor is already a visitor.... I think
		patchedValue, err := jsondata.Accept[jsondata.JSONValue](jsonValue, patchVisitor)
		if err != nil || patchVisitor.Failed {
			if patchVisitor.FailureReason != "" {
				return oldRawDoc, err
			}
			return oldRawDoc, err
		}
		if !patchVisitor.pathFound {
			msg := fmt.Sprintf("Error Applying patches:  Path '%s' does not exist in the document", patchOp.Path) // invalid patch
			slog.Error(msg)
			return nil, errors.New(msg)
		}

		// Update jsonValue for the next patch operation
		jsonValue = patchedValue
	}

	updatedData, err := json.Marshal(jsonValue)
	if err != nil {
		return oldRawDoc, err
	}
	slog.Info("PATCHING 4", "updatedData", updatedData)

	return updatedData, nil
}

// validatePatches validates a slice of PatchOperation objects to ensure that each operation is valid.
// It returns an error if any operation is not one of the allowed types: "ArrayAdd", "ArrayRemove", or "ObjectAdd".
func validatePatches(ops []PatchOperation) error {
	for _, p := range ops {
		if p.Op != "ArrayAdd" && p.Op != "ArrayRemove" && p.Op != "ObjectAdd" {
			return fmt.Errorf("bad patch operation: %s", p.Op)
		}
	}
	return nil
}
