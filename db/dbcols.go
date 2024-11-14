// Package db provides methods for managing collections within a document-based database.
// It allows for retrieving, deleting, and uploading collections in a hierarchical structure.
package db

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// GetColSerial retrieves and serializes the collection at the specified path.
// It returns a byte slice of the serialized collection, a status code, and an optional subscription channel if applicable.
// The range is defined by the 'lo' and 'hi' keys, and if isSubscription is true, the function will return events for document changes.
func (db *Database[K, T]) GetColSerial(colpath string, lo string, hi string, isSubscription bool) ([]byte, int, *chan []byte, string, [][]byte) {

	splitPath := strings.Split(colpath, "/")

	topDocName := splitPath[0] //get the name of the parent document

	if len(colpath) == 0 { //TOP-LEVEL DATABASE GET
		res, stat := db.serialTop(lo, hi)
		if isSubscription {
			docBytes := make([][]byte, 0)
			subChan, subId := db.colSubscriptionManager.AddSubscriber(lo, hi)
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(15*time.Second))
			defer cancel()
			docs, _ := db.docs.Query(ctx, K(lo), K(hi))
			for _, v := range docs {
				event := db.colSubscriptionManager.GenerateEvent("update", v.Value.GetSerial())
				docBytes = append(docBytes, event)
			}
			return res, stat, subChan, subId, docBytes
		}

		return res, stat, nil, "", nil
	}

	topDoc, found := db.docs.Find(K(topDocName)) //look for a doc with this name

	if !found {

		errmsg, _ := json.Marshal("Collection does not exist")

		return errmsg, http.StatusNotFound, nil, "", nil
	}
	//delegated to the documents
	return topDoc.GetChildCollection(colpath, lo, hi, isSubscription)

}

// DeleteCol deletes the collection located at the path colpath in the database.
// Returns a response (if an error occurred) and a status code.
func (db *Database[K, T]) DeleteCol(colpath string) ([]byte, int) {

	slog.Debug(fmt.Sprintf("Deleting the collection at path %+v", colpath))

	splitPath := strings.Split(colpath, "/")

	topDocName := splitPath[0] //get the parent document

	topDoc, found := db.docs.Find(K(topDocName)) //looking for the db

	if !found {

		errmsg, _ := json.Marshal("Collection does not exist")

		return errmsg, http.StatusNotFound
	}
	return topDoc.DeleteChildCollection(colpath)
}

// UploadCol uploads a collection at the path colpath, at the database dbName
// Returns a response (if an error occurred) and a status code.
func (db *Database[K, T]) UploadCol(colpath string, dbName string) ([]byte, int, string) {

	slog.Debug(fmt.Sprintf("UploadCol: uploading a collection at path %s,database name %s", colpath, dbName))

	splitPath := strings.Split(colpath, "/")

	topDocName := splitPath[0]

	topDoc, found := db.docs.Find(K(topDocName)) //find the top document

	if !found { //we couldn't find it :(

		errmsg, _ := json.Marshal("Collection does not exist")

		return errmsg, http.StatusNotFound, ""
	}
	return topDoc.AddChildCollection(colpath, dbName)
}

// serialTop is an internal routine that returns a serialized representation of the top-level collection
// represented as a JSON-encoded byte slice. Also returns a status code
func (db *Database[K, T]) serialTop(lo string, hi string) ([]byte, int) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()
	res, err := db.docs.Query(ctx, K(lo), K(hi)) //query on top-level collection
	if err != nil {
		errmsg, _ := json.Marshal(err.Error())
		return errmsg, http.StatusBadRequest
	}
	resArr := []json.RawMessage{}
	for _, pair := range res { //serializing all documents
		bytes := pair.Value.GetSerial()
		var rj json.RawMessage = bytes
		resArr = append(resArr, rj)
	}
	resSerial, _ := json.Marshal(resArr)
	return resSerial, http.StatusOK
}
