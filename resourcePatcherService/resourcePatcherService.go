// Package resourcePatcherService is responsible for processing and dispatching PATCH requests to their appropriate handlers
package resourcePatcherService

import (
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/index_utils"
	"net/http"
)

// Patchdatabaser is an interface that represents an object capable of applying patches to documents.
type Patchdatabaser interface {
	Patch(docPath string, patches []byte, user string) ([]byte, int)
}

// DatabaseIndex is a generic interface that defines operations for managing databases.
type DatabaseIndex[K string, V Patchdatabaser] interface {
	// Upsert inserts or updates a value in the database index based on the given key and update function.
	Upsert(key K, check index_utils.UpdateCheck[K, V]) (updated bool, err error)

	// Find looks up a value based on the given key. It returns the found value and whether it was found.
	Find(key K) (foundValue V, found bool)
}

// Validator is an interface for validating data against a schema or set of rules.
type Validator interface {
	Validate([]byte) error
}

// ResourcePatcherService manages and processes PATCH operations for documents within databases.
type ResourcePatcherService[K string, T Patchdatabaser] struct {
	dbs DatabaseIndex[K, T]
}

// New creates a new instance of ResourcePatcherService with the given database index.
func New[K string, T Patchdatabaser](dbs DatabaseIndex[K, T]) *ResourcePatcherService[K, T] {
	return &ResourcePatcherService[K, T]{dbs: dbs}
}

// PatchDoc processes a PATCH request for a document within a specific database.
// Returns the response as a byte slice and the corresponding HTTP status code.
func (rps *ResourcePatcherService[K, T]) PatchDoc(dtb string, docpath string, patches []byte, user string) ([]byte, int, string) {

	db, found := rps.dbs.Find(K(dtb))
	if found == false {
		return []byte(`{"error": "Database not exist"}`), http.StatusNotFound, ""
	} // find the database

	responses, status := db.Patch(docpath, patches, user)

	return responses, status, "/v1/" + dtb + "/" + docpath
}
