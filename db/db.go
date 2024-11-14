// Package db encapsulates the functionalities of the databases users can create. Note that a database is only functionally responsible for the top-level documents it holds.
package db

import (
	"context"
	"encoding/json"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/index_utils"
)

// DocumentAdder encapsulates the functionalities of the top-level documents with respect to adding new resources to the database
type DocumentAdder interface {
	AddChildDocument(docpath string, payload []byte, docname string, user string, overwrite bool, isPost bool, dbName string) ([]byte, int, string) //adds a child document
	AddChildCollection(colpath string, dbName string) ([]byte, int, string)                                                                         //adds a child collection
}

// DocumentDeleter encapsulates the functionalities of the top-level documents with respect to deleting resources the database
type DocumentGetter interface {
	GetChildDocument(docpath string, isSubscribe bool, dbName string) (payload []byte, status_code int, sub_id string, subChan *chan []byte, docEvent []byte)      //retrieves a document within the document
	GetChildCollection(colpath string, lo string, hi string, isSubscribe bool) (res []byte, stat_code int, subChan *chan []byte, subId string, docEvents [][]byte) // retrieves a collection within the document
	Notify(uri string, payload []byte, evType string)                                                                                                              // notifies all subscribers of a change
}

// DocumentDeleter encapsulates the functionalities of the top-level documents with respect to getting resources from the database
type DocumentDeleter interface {
	DeleteChildDocument(docpath string, dbName string) ([]byte, int) //deletes a child document
	DeleteChildCollection(colpath string) ([]byte, int)              //deletes a child collection
}

type DocumentPatcher interface {
	ApplyPatchDocument(dbName string, docPath string, patch []byte, user string) ([]byte, int) //applies a patch to a document
	DoPatch(patch []byte) ([]byte, error)                                                      //applies a patch to a document
	UpdateDoc(newRaw []byte, user string)                                                      //updates a document
}

// ColSubscriptionManager represents the contract necessary for the database's top-level collection to manage subscriptions
type ColSubscriptionManager interface {
	NotifyAll(colname string)                                             //notifies all subscribers of a change
	AddSubscriber(lo string, hi string) (subChan *chan []byte, id string) //adds a subscriber
	Notify(docname string, evType string, payload []byte)                 //notifies a subscriber of a change
	GenerateEvent(evType string, content []byte) []byte                   //generates an event
}
type Validator interface {
	Validate([]byte) error //validates a document
}

// DBDocumenter encapsulates the behaviors necessary for the top-level documents stored in the database.
// Note that the database is only responsible for actions performed on the top-most level of documents;
// for any children of said documents, the database delegates all functionality to the document itself
// to manage its children
type DBDocumenter interface {
	DocumentAdder
	DocumentGetter
	DocumentDeleter
	DocumentPatcher
	GetSerial() []byte
}

// DocIndex encompasses the behaviors needed for the indices in a document (pointing to collections)
type DocIndex[K string, V any] interface {
	Upsert(key K, check index_utils.UpdateCheck[K, V]) (updated bool, err error)             //Updates or inserts a a value
	Remove(key K) (removedValue V, removed bool)                                             //Removes a value
	Find(key K) (foundValue V, found bool)                                                   // Finds a value
	Query(ctx context.Context, start K, end K) (results []index_utils.Pair[K, V], err error) //Queries the index
}

// DocFactory is a factory function used to create documents
type DocFactory[T DBDocumenter] func(payload []byte, user string, path string) T

// Database represents a database object
type Database[K string, T DBDocumenter] struct {
	name string        //the name of the database
	dcf  DocFactory[T] // dcf creates documents that implement the container interface

	docs DocIndex[K, T] // docs holds documents

	colSubscriptionManager ColSubscriptionManager // colSubscriptionManager manages subscriptions

	validator Validator // validator validates documents
}

// New creates a database object
func New[K string, T DBDocumenter](name string, dcf DocFactory[T], index DocIndex[K, T], manager ColSubscriptionManager, v Validator) *Database[K, T] {
	db := Database[K, T]{}
	db.dcf = dcf
	db.name = name
	db.docs = index
	db.colSubscriptionManager = manager
	db.validator = v
	return &db
}

// NotifyAll notifies all subscribers to a DB that a database has been deleted
func (db *Database[K, T]) NotifyAll(colname string) {
	b, _ := json.Marshal("/")
	db.colSubscriptionManager.NotifyAll(string(b))
}
