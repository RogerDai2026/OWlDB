// Package db provides functions for managing documents within a document-based database,
// including uploading, deleting, patching, and retrieving documents.
package db

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"net/http"
)

// UploadDocument uploads a document to the database at the specified path.
// This function handles both PUT and POST requests. It supports options for overwriting
// an existing document and allows for top-level document management as well as child documents.
// Returns a serialized response, status code, and potential errors if the operation fails.
func (db *Database[K, T]) UploadDocument(docpath string, payload []byte, docname, user string, overwrite, isPost bool, dbName string) ([]byte, int, string) {

	slog.Debug(fmt.Sprintf("uploading document at path %+v to the database %s dbName, isPost is %t", docpath, dbName, isPost))

	stringPath := string(docpath)

	splitPath := strings.Split(stringPath, "/")

	if len(splitPath) == 1 { //we are posting to the top DB
		return db.uploadTop(K(docpath), payload, docname, user, overwrite, isPost, dbName)
	}

	topDocName := K(splitPath[0])

	parentDoc, found := db.docs.Find(topDocName)
	if !found {
		errmsg, _ := json.Marshal("Document does not exist")
		return errmsg, http.StatusNotFound, ""
	}
	return parentDoc.AddChildDocument(stringPath, payload, docname, user, overwrite, isPost, dbName)

}

// uploadTop uploads a document to the top level of the database.
// It creates a new document or overwrites an existing one depending on the overwrite flag.
// Returns a response indicating the result of the operation and the associated status code.
func (db *Database[K, T]) uploadTop(docpath K, payload []byte, docname string, user string, overwrite bool, isPost bool, dbName string) ([]byte, int, string) {

	slog.Debug(fmt.Sprintf("uploading to the top of database %+v, isPost %t", string(docpath), isPost))

	newDoc := db.dcf(payload, user, string(docpath))

	var nullDoc T
	var stat_code int
	check := func(docname K, curVal T, exists bool) (newVal T, err error) {
		if !exists {
			stat_code = http.StatusCreated
			db.colSubscriptionManager.Notify(string(docname), "update", newDoc.GetSerial())
			newDoc.Notify(dbName+"/"+string(docname), newDoc.GetSerial(), "update")
			return newDoc, nil
		} else { //the document exists
			if !overwrite {
				stat_code = http.StatusPreconditionFailed
				return nullDoc, fmt.Errorf("document already exists")
			} else {
				stat_code = http.StatusOK
				curVal.UpdateDoc(payload, user)
				db.colSubscriptionManager.Notify(string(docname), "update", curVal.GetSerial())
				curVal.Notify(db.name+"/"+string(docname), curVal.GetSerial(), "update")
			}
		}

		return curVal, nil
	}

	_, err := db.docs.Upsert(K(docname), check)

	if err != nil { //doc failed -
		var statCode int = 400
		if !isPost {
			statCode = 412
		}
		b, _ := json.Marshal(err.Error())
		return b, statCode, ""
	}
	slog.Debug("AAAAAAAAAAAA")
	respJSON := struct {
		Uri string `json:"uri"`
	}{
		Uri: "/v1/" + dbName + "/" + string(docpath),
	}

	b, _ := json.Marshal(respJSON)

	//slog.Debug(fmt.Sprintf("Notifying subscribers of the topmost database about document %s", docname))

	return b, stat_code, respJSON.Uri
}

// GetDocumentSerial retrieves and returns a serialized version of the document located at docpath.
// If isSubscribe is true, the function returns a channel for subscription to document changes.
// Returns the serialized payload, a subscription channel, subscription ID, status code, and any document events.
func (db *Database[K, T]) GetDocumentSerial(docpath string, isSubscribe bool) (payload []byte, subChannel *chan []byte, subId string, statusCode int, docEvent []byte) {

	slog.Debug(fmt.Sprintf("GetDocumentSerial,docpath is %s", docpath))

	splitPath := strings.Split(docpath, "/")

	topDocName := K(splitPath[0])

	topDoc, found := db.docs.Find(topDocName)

	if !found {

		errmsg, _ := json.Marshal("Document does not exist")

		return errmsg, nil, "", http.StatusNotFound, docEvent
	}

	payload, statusCode, subId, subChannel, docEvent = topDoc.GetChildDocument(docpath, isSubscribe, db.name)
	return payload, subChannel, subId, statusCode, docEvent
}

// DeleteDoc deletes the document at the specified path.
// This method can handle both top-level documents and child documents.
// Returns a serialized response and a status code indicating success or failure.
func (db *Database[K, T]) DeleteDoc(docpath string) ([]byte, int) {

	splitPath := strings.Split(docpath, "/")

	if len(splitPath) == 1 {
		return db.deleteTop(docpath)
	}

	topDocName := K(splitPath[0])

	topDoc, found := db.docs.Find(topDocName)

	if !found {

		errmsg, _ := json.Marshal("Document does not exist")

		return errmsg, http.StatusNotFound
	}
	//delegated to the documents
	return topDoc.DeleteChildDocument(docpath, db.name)
}

// deleteTop handles the case where the topmost document must be deleted
// Returns a response and status code indicating the outcome of the operation.
func (db *Database[K, T]) deleteTop(docpath string) ([]byte, int) {

	slog.Debug(fmt.Sprintf("deleting the top document,resource path is %s", docpath))

	removedDoc, removed := db.docs.Remove(K(docpath))

	//This is the only way removes can fail
	if !removed {

		errmsg, _ := json.Marshal("Document does not exist")

		return errmsg, http.StatusNotFound
	}

	payload := "/" + docpath
	b, _ := json.Marshal(payload)
	removedDoc.Notify(db.name+"/"+docpath, b, "delete")
	db.colSubscriptionManager.Notify(docpath, "delete", b)
	//success
	return nil, http.StatusNoContent
}

// deleteTop handles the deletion of a top-level document.
// Returns a response and status code indicating the outcome of the operation.
func (db *Database[K, T]) Patch(docPath string, patches []byte, user string) ([]byte, int) {

	splitPath := strings.Split(docPath, "/")
	if len(splitPath) == 0 {
		return []byte(`{"error": "Invalid document path"}`), http.StatusBadRequest
	}
	topDocName := K(splitPath[0])
	topDoc, found := db.docs.Find(topDocName)
	if !found {
		errmsg, _ := json.Marshal("Document does not exist")
		return errmsg, http.StatusNotFound
	}
	if len(splitPath) == 1 {
		return db.patchTop(docPath, patches, user)
	}

	slog.Debug(fmt.Sprintf("From Patch in db, user is %s", user))
	responseBytes, statusCode := topDoc.ApplyPatchDocument(db.name, docPath, patches, user)
	if statusCode != http.StatusOK {
		return responseBytes, statusCode
	}
	return responseBytes, statusCode
}

// Patch applies a JSON patch to the document at the specified path
// and returns a response indicating the outcome of the operation.	
func (db *Database[K, T]) patchTop(docName string, patch []byte, user string) ([]byte, int) {

	var nullDoc T
	var newDocPayload []byte
	slog.Debug(fmt.Sprintf("user is %s", user))
	chk := func(name K, curDoc T, exists bool) (newDoc T, err error) { //guaranteed to be atomic due to locks in SL
		if !exists {
			return nullDoc, fmt.Errorf("Document does not exist")
		}

		newRaw, patchErr := curDoc.DoPatch(patch)
		if patchErr != nil { //DO NOT UPDATE
			slog.Debug("DO NOT UPDATE,WE GOT AN ERROR")
			return nullDoc, patchErr
		}
		err = db.validator.Validate(newRaw) //is this a valid patch?
		if err != nil {
			return nullDoc, err
		} //this will not update
		slog.Debug("About to update DOCUMENT SUBSCRIBERS")
		curDoc.UpdateDoc(newRaw, user)

		newDocPayload = curDoc.GetSerial() //serializing new documents
		curDoc.Notify(db.name+"/"+docName, newDocPayload, "update")
		db.colSubscriptionManager.Notify(docName, "update", newDocPayload)
		return curDoc, nil
	}
	updated, er := db.docs.Upsert(K(docName), chk)

	msg := "patch applied"
	if !updated {

		if er.Error() == "Document does not exist" {
			errmsg, _ := json.Marshal(er.Error())
			return errmsg, http.StatusNotFound
		} else if strings.HasPrefix(er.Error(), "bad patch operation") {
			errmsg, _ := json.Marshal(er.Error())
			return errmsg, http.StatusBadRequest
		} else {
			msg = er.Error()
		}
	} else {
	}

	return generatePatchResponse(db.name, docName, !updated, msg), http.StatusOK
}

// generatePatchResponse generates a response for a patch operation.
// The response includes the URI of the patched document, a flag indicating whether the patch failed,
// and a message describing the outcome of the operation.
func generatePatchResponse(dbName string, docPath string, patchFailed bool, msg string) []byte {

	uri := "/v1/" + dbName + "/" + docPath

	response := struct {
		Uri         string `json:"uri"`
		PatchFailed bool   `json:"patchFailed"`
		Message     string `json:"string"`
	}{
		Uri:         uri,
		PatchFailed: patchFailed,
		Message:     msg,
	}

	b, err := json.Marshal(response)
	if err != nil {
		slog.Warn("WARNING from docHelpers: JSON marshaling failed!")
	}
	return b
}
