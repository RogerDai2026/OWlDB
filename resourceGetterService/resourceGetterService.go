// Package resourceGetterService serves to route get requests to the correct internal handlers, receiving their responses, and forwarding them back to the server
package resourceGetterService

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

type Getdatabaser interface {
	GetDocumentSerial(docpath string, isSubscribe bool) (payload []byte, subChannel *chan []byte, subId string, statusCode int, docEvent []byte)
	GetColSerial(colpath string, lo string, hi string, isSubscription bool) (payload []byte, stat_code int, subChan *chan []byte, subId string, docEvents [][]byte)
}

// DatabaseIndex represents indices for our databases
type DatabaseIndex[K string, V Getdatabaser] interface {
	Find(key K) (foundValue V, found bool)
}

type ResourceGetterService[K string, V Getdatabaser] struct {
	dbs DatabaseIndex[K, V]
}

func New[K string, V Getdatabaser](dbs DatabaseIndex[K, V]) *ResourceGetterService[K, V] {
	return &ResourceGetterService[K, V]{dbs: dbs}
}

// GetCol retrieves the top-level database in the path, and then delegates the call to a Getdatabase (if found)
func (rgs *ResourceGetterService[K, T]) GetCol(dtb string, colpath string, lower string, upper string, mode bool) (payload []byte, statCode int, subChan *chan []byte, subId string, docEvents [][]byte) {

	if mode {
		slog.Debug(fmt.Sprintf("Received subscription request"))
	}

	db, found := rgs.dbs.Find(K(dtb))
	if !found {
		errmsg, _ := json.Marshal(fmt.Sprintf("Database does not exist"))
		return errmsg, http.StatusNotFound, subChan, subId, docEvents
	}

	return db.GetColSerial(colpath, lower, upper, mode)
}

// GetDoc gets a document by first retrieving the database it belongs to, and forwarding the request at the path pathstr to the database
func (rgs *ResourceGetterService[K, T]) GetDoc(dtb string, pathstr string, subscription bool) (response []byte, statCode int, subCh *chan []byte, id string, docEvent []byte) {

	if subscription {
		slog.Debug(fmt.Sprintf("Received a subscription request"))
	}

	root, ok := rgs.dbs.Find(K(dtb))
	if !ok {
		errmsg, _ := json.Marshal("Error getting document: a database with that name does not exist.")
		return errmsg, http.StatusNotFound, nil, id, docEvent
	}

	doc, subChan, subId, stat, docEv := root.GetDocumentSerial(pathstr, subscription)
	return doc, stat, subChan, subId, docEv
}
