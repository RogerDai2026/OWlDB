// Package resourceCreatorService is responsible for the processing of any and all PUT/POST requests to the database, delegating and forwarding any and all requests
// to their appropriate user.
package resourceCreatorService

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strings"

	"github.com/RICE-COMP318-FALL24/owldb-p1group24/index_utils"
)

// Upsertdatabaser defines the interface for uploading collections and documents to a database.
type Upsertdatabaser interface {
	UploadCol(colpath string, dbName string) ([]byte, int, string)                                                                    // Uploads a collection to the database.
	UploadDocument(docpath string, payload []byte, docname, user string, overwrite, isPost bool, dbName string) ([]byte, int, string) // Uploads a document to the database.
}

// DatabaseIndex describes the necessary behaviors for the underlying container of the databases themselves
type DatabaseIndex[K string, V Upsertdatabaser] interface {
	Upsert(key K, check index_utils.UpdateCheck[K, V]) (updated bool, err error)
	Find(key K) (foundValue V, found bool)
}

// ResourceCreatorService is responsible for handling requests to create or upload collections and documents in the database.
type ResourceCreatorService[K string, T Upsertdatabaser] struct {
	dbs       DatabaseIndex[K, T] // The collection of databases.
	dbfactory DBFactory[T]        // A factory function for creating new databases.
	validator Validator           // Validates the schema of documents before uploading.
}

// Validator defines an interface for validating JSON data against a schema.
type Validator interface {
	Validate(jsonData []byte) error // Validates the given JSON data.
}

// New creates a new instance of a ResourceCreatorService. Note that the arguments passed in must themselves be initialized properly to ensure correct behavior
func New[K string, T Upsertdatabaser](dbs DatabaseIndex[K, T], dbfactory DBFactory[T], validator Validator) *ResourceCreatorService[K, T] {
	return &ResourceCreatorService[K, T]{dbs: dbs, dbfactory: dbfactory, validator: validator}
}

// PutCol will put a collection at the database dtb with collection path colpath.
// It returns a JSON-encoded response and a status code indicating success or failure.
func (rcs *ResourceCreatorService[K, T]) PutCol(dtb string, colpath string) ([]byte, int, string) {
	db, found := rcs.dbs.Find(K(dtb))

	if !found {
		errmsg, _ := json.Marshal("Error: no such database exists")
		return errmsg, http.StatusNotFound, ""
	}

	return db.UploadCol(colpath, dtb)
}

// DBFactory is a factory function for creating new databases.
// It returns a JSON-encoded response and a status code indicating success or failure.
type DBFactory[T Upsertdatabaser] func(string) T

// CreateDB creates a database with name dbName
func (rcs *ResourceCreatorService[K, T]) CreateDB(dbName string) ([]byte, int, string) {
	var nullDB T
	check := func(key K, curVal T, exists bool) (newVal T, err error) {
		if exists {
			return nullDB, fmt.Errorf("database with that name exists")
		}
		return rcs.dbfactory(string(key)), nil
	}

	_, err := rcs.dbs.Upsert(K(dbName), check)
	if err != nil {
		errmsg, _ := json.Marshal(err.Error())
		return errmsg, http.StatusBadRequest, ""
	}
	resp := struct {
		Uri string `json:"uri"`
	}{
		Uri: "/v1/" + string(dbName),
	}
	respSerial, _ := json.Marshal(resp)
	return respSerial, http.StatusCreated, resp.Uri
}

// generateResourceName is an internal routine to generate the name of a resource
func generateResourceName() string {
	var name string
	validChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-.~"
	leng := 12
	for i := 0; i < leng; i++ {
		ch := rand.IntN(len(validChars))
		name += string(validChars[ch])
	}
	return name

}

// PostDoc posts a document to the database named dbName.
// Note that if the document is not a top-level document, the post is delegated
// to the parent document on the path to the child collection that will contain this new document
// returns a JSON-encoded response and a status code.
func (rcs *ResourceCreatorService[K, T]) PostDoc(dbName string, colpath string, user string, payload []byte) ([]byte, int, string) {
	colpath = strings.TrimSuffix(colpath, "/")
	db, found := rcs.dbs.Find(K(dbName))
	if !found {
		errmsg, _ := json.Marshal("Error adding document: Database does not exist.")
		return errmsg, http.StatusNotFound, ""

	}
	validErr := rcs.validator.Validate(payload)
	if validErr != nil {
		errmsg, _ := json.Marshal("Malformed document, document does not conform to schema for reason")
		return errmsg, http.StatusBadRequest, ""
	}

	docName := generateResourceName()
	slog.Debug(fmt.Sprintf("Generated name: %s", docName))
	docPath := docName
	if colpath != "" {
		docPath = colpath + "/" + docName
	}
	slog.Debug(fmt.Sprintf("PostDoc: the path for this document is %s", docPath))

	return db.UploadDocument(docPath, payload, docName, user, true, true, string(dbName))
}

// PutDoc puts a document in the database
// It returns a JSON-encoded response and a status code indicating success or failure.
func (rcs *ResourceCreatorService[K, T]) PutDoc(dbName K, docpath string, docname string, payload []byte, overwrite bool, user string) ([]byte, int, string) {
	db, found := rcs.dbs.Find(dbName)
	if !found {
		errmsg, _ := json.Marshal("Collection does not exist")
		return errmsg, http.StatusNotFound, ""

	}

	validErr := rcs.validator.Validate(payload)
	if validErr != nil {
		errmsg, _ := json.Marshal("Malformed document, document does not conform to schema for reason")
		return errmsg, http.StatusBadRequest, ""
	}
	slog.Debug(fmt.Sprintf("calling PutDoc with the following params: overwrite %t", overwrite))
	return db.UploadDocument(docpath, payload, docname, user, overwrite, false, string(dbName))
}
