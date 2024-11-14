// Description: This file contains the implementation of the Validator struct,
// which is used to validate JSON data against a JSON schema.
package validation

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// Validator holds a compiled JSON schema for validation

type Validator struct {
	schema *jsonschema.Schema
}

// New creates a new instance of Validator by compiling the provided JSON schema file.
// Inputs:
//   - jsonSchemaFilename: The filename of the JSON schema file to compile.
//
// Returns:
//   - A new instance of Validator if the schema is successfully compiled.
func New(jsonSchemaFilename string) (*Validator, error) {

	compiler := jsonschema.NewCompiler()
	//compiler.Draft = jsonschema.Draft2020 // Specify the JSON Schema draft version if needed

	// Compile the schema
	schema, err := compiler.Compile(jsonSchemaFilename)
	if err != nil {
		return nil, fmt.Errorf("error compiling schema '%s': %w", jsonSchemaFilename, err)
	}

	return &Validator{schema: schema}, nil
}

// Validate checks whether the provided JSON data conforms to the compiled schema.
// Inputs:
//   - jsonData: Byte slice representing the JSON data to validate.
//
// Returns:
//   - An error if the JSON data is invalid or does not conform to the schema.
//   - nil if the JSON data is valid.
func (v *Validator) Validate(jsonData []byte) error {
	var jsonObject interface{}

	if err := json.Unmarshal(jsonData, &jsonObject); err != nil {
		slog.Error("Unable to unmarshal JSON data", "error", err)
		return fmt.Errorf("unable to unmarshal JSON data: %w", err)
	}

	if err := v.schema.Validate(jsonObject); err != nil {
		slog.Error("JSON validation failed", "error", err)
		return err
	}

	return nil
}
