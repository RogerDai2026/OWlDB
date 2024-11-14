package resourcePatcherService

import (
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/mocks"
	"net/http"
	"testing"
)

// dummyPatcher is a mock patcher for testing purposes.
type dummyPatcher struct {
	didPatch bool
}

// evilDummyPatcher is another mock patcher that simulates invalid patching scenarios.
type evilDummyPatcher struct {
	didPatch bool
}

// Validate simulates validation in dummyPatcher.
func (d *dummyPatcher) Validate(bytes []byte) error {
	d.didPatch = true
	return nil
}

// dummyValidator simulates a validator for testing patch operations.
type dummyValidator struct {
	didValidate bool
}

// DoPatch simulates patching in dummyValidator.
func (d *dummyValidator) DoPatch(oldRawDoc []byte, rawPatches []byte) (newDoc []byte, err error) {
	d.didValidate = true
	return []byte("GOOD DOC"), nil
}

// setup initializes a ResourcePatcherService instance without any databases.
func setup() *ResourcePatcherService[string, Patchdatabaser] {

	mockSL := mocks.NewMockSL[string, Patchdatabaser]()

	rps := New[string, Patchdatabaser](mockSL)

	return rps
}

// mockDB simulates a database for testing purposes.
type mockDB struct {
}

// Patch simulates a patch operation on a mockDB.
func (m *mockDB) Patch(docPath string, patches []byte, user string) ([]byte, int) {

	return []byte("Pretend this is a success message"), 200
}

// setupContainsDB initializes a ResourcePatcherService instance with a mock database.
func setupContainsDB() *ResourcePatcherService[string, Patchdatabaser] {
	mockSL := mocks.NewMockSL[string, Patchdatabaser]()

	chk := func(string, Patchdatabaser, bool) (Patchdatabaser, error) {
		return &mockDB{}, nil
	}

	mockSL.Upsert("db", chk)

	return New[string, Patchdatabaser](mockSL)
}

// TestResourcePatcherService_PatchDocNoDBFound tests patching when no database is found.
func TestResourcePatcherService_PatchDocNoDBFound(t *testing.T) {

	rps := setup()

	_, stat, _ := rps.PatchDoc("db", "doc1/col1/doc1", []byte("PATCHES"), "USER")

	if stat != http.StatusNotFound {
		t.Errorf("TestResourcePatcherService_PatchDocNoDBFound")
	}
}

// TestResourcePatcherService_PatchDocNoDBFound tests patching when no database is found.
func TestResourcePatcherService_PatchDocDBFound(t *testing.T) {
	rps := setupContainsDB()

	_, stat, _ := rps.PatchDoc("db", "doc1", []byte("PATCHES"), "USER")

	if stat != http.StatusOK {
		t.Errorf("TestResourcePatcherService_PatchDocDBFound failed")
	}

}

// TestResourcePatcherService_PatchDocDBFound tests patching when a database is found.
func TestResourcePatcherService_PatchDoc(t *testing.T) {
	rps := setupContainsDB()

	_, stat, _ := rps.PatchDoc("db", "doc1", []byte("PATCHES"), "USER")

	if stat != http.StatusOK {
		t.Errorf("TestResourcePatcherService_PatchDocDBFound failed")
	}
}

// TestResourcePatcherService_PatchDoc tests a basic patch operation when the database exists.
func TestResourcePatcherService_PatchDocObjectAdd(t *testing.T) {
	rps := setupContainsDB()

	// Define a patch in JSON format
	patch := `
    [
        { "op": "ObjectAdd", "path": "/field", "value": "new value" },
        { "op": "ObjectAdd", "path": "/existingField", "value": "updated value" }
    ]`

	// Convert the JSON patch to a byte slice
	patchBytes := []byte(patch)

	_, stat, _ := rps.PatchDoc("db", "doc1", patchBytes, "USER")

	if stat != http.StatusOK {
		t.Errorf("TestResourcePatcherService_PatchDocObjectAdd failed: expected status 200, got %d", stat)
	}
}

// TestResourcePatcherService_PatchDocObjectAdd tests the ObjectAdd operation in a patch.
func TestResourcePatcherService_PatchDocArrayAdd(t *testing.T) {
	rps := setupContainsDB()

	// Define a patch in JSON format
	patch := `
	[
		{ "op": "ObjectAdd", "path": "/field", "value": {"user": "new user"} },
		{ "op": "ArrayAdd", "path": "/field", "value": "updated value" }
	]`

	// Convert the JSON patch to a byte slice
	patchBytes := []byte(patch)

	_, stat, _ := rps.PatchDoc("db", "doc1", patchBytes, "USER")

	if stat != http.StatusOK {
		t.Errorf("TestResourcePatcherService_PatchDocWithPatchBytes failed")
	}
}

// TestResourcePatcherService_PatchDocArrayAdd tests the ArrayAdd operation in a patch.
func TestResourcePatcherService_PatchDocArrayRemove(t *testing.T) {
	rps := setupContainsDB()

	// Define a patch in JSON format
	patch := `
	[
		{ "op": "ObjectAdd", "path": "/field", "value": {"user": "new user"} },
		{ "op": "ArrayAdd", "path": "/field", "value": "updated value" },
		{ "op": "ArrayRemove", "path": "/field", "value": "updated value" }

	]`

	// Convert the JSON patch to a byte slice
	patchBytes := []byte(patch)

	_, stat, _ := rps.PatchDoc("db", "doc1", patchBytes, "USER")

	if stat != http.StatusOK {
		t.Errorf("TestResourcePatcherService_PatchDocWithPatchBytes failed")
	}
}
