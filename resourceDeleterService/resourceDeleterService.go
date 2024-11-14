// Package resourceDeleterService is responsible for deleting resources held on OwlDB
package resourceDeleterService

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Notifier is an interface that defines a method for notifying subscribers of changes to a resource.
type Notifier interface {
	NotifyAll(colname string)
}

// Deletedatabaser encompasses the behaviors needed for ResourceDeleterService to operate on
type Deletedatabaser interface {
	Notifier
	DeleteDoc(docpath string) ([]byte, int) //DeleteDoc should delete the document at the provided path
	DeleteCol(colpath string) ([]byte, int) //DeleteCol should delete the collection at the provided path
}

// DatabaseIndex is a generic interface that defines operations for managing databases.
type DatabaseIndex[K string, V Deletedatabaser] interface {
	Remove(key K) (removedVal V, removed bool)
	Find(key K) (foundVal V, found bool)
}

// ResourceDeleterService is responsible for the deletion of resources
type ResourceDeleterService[K string, V Deletedatabaser] struct {
	dbs DatabaseIndex[K, V]
}

// New creates a new ResourceDeleterService, ready to use as long as the dbs has been initialized.
func New[K string, T Deletedatabaser](dbs DatabaseIndex[K, T]) *ResourceDeleterService[K, T] {
	return &ResourceDeleterService[K, T]{dbs: dbs}
}

// DeleteDoc deletes the document located at the path docpath, under the database dbName
func (rds *ResourceDeleterService[K, T]) DeleteDoc(dbName string, docpath string) ([]byte, int) {
	dtb, found := rds.dbs.Find(K(dbName))

	if !found {
		errmsg, _ := json.Marshal("Error: database does not exist")
		return errmsg, http.StatusNotFound
	}
	return dtb.DeleteDoc(docpath)
}

// DeleteCol deletes the document loacted at the path colpath, under the database dtb
func (rds *ResourceDeleterService[K, T]) DeleteCol(dtb string, colpath string) ([]byte, int) {
	db, found := rds.dbs.Find(K(dtb))

	if !found {
		errmsg, _ := json.Marshal("Error: no such database exists")
		return errmsg, http.StatusNotFound
	}

	return db.DeleteCol(colpath)
}

// DeleteDB deletes the database named dtb. It returns a json-encoded response, and a status code
func (rds *ResourceDeleterService[K, T]) DeleteDB(dtb string) ([]byte, int) {
	db, success := rds.dbs.Remove(K(dtb))
	if success {
		msg, _ := json.Marshal("Deleted.")
		slog.Debug("About to notify subscribers that this database is deleted")
		db.NotifyAll("/")
		return msg, http.StatusNoContent
	} else {
		errmsg, _ := json.Marshal("Error: database does not exist")
		return errmsg, http.StatusNotFound
	}
}
