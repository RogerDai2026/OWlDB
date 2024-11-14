package validation_test

import (
	"encoding/json"
	"fmt"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/mocks"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/validation"
	"testing"
)

type validDoc struct {
	Path string          `json:"path"`
	Doc  json.RawMessage `json:"doc"`
	Meta json.RawMessage `json:"meta"`
}

// tests when a document conforms to a schema
func TestValidator_Validate(t *testing.T) {

	v, err := validation.New("document2.json")

	if err != nil {
		fmt.Printf(err.Error())
		t.Errorf("TestValidator_Validate failed")
	}

	validDc := validDoc{Doc: mocks.MockPayload(), Path: "hello"}

	b, _ := json.Marshal(validDc)

	err = v.Validate(b)
	if err != nil {
		t.Errorf("Error")
	}
}

type invalidDoc struct {
	Doc  string
	Path string
	Meta string
}

func TestValidator_ValidateInvalidInputs(t *testing.T) {

	v, err := validation.New("document2.json")

	if err != nil {
		fmt.Printf(err.Error())
		t.Errorf("TestValidator_Validate failed")
	}
	invalidDc := invalidDoc{Doc: "hello", Path: "hello", Meta: "hello"}
	b, _ := json.Marshal(invalidDc)
	validErr := v.Validate(b)
	if validErr == nil {
		t.Errorf("TestValidator_ValidateInvalidInputs failed")
	}
}
