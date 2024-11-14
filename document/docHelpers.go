// Package document encompasses the functionalities and behaviors of document objects in OwlDB
package document

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/jsondata"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// serialize returns a JSON-encoded byte slice containing the content of document d
// It is used to serialize the document for storage or transmission
func (d *Document) serialize() ([]byte, error) {

	var docJSON jsondata.JSONValue

	_ = json.Unmarshal(d.Info.Doc, &docJSON)

	serialStruct := struct {
		Path string             `json:"path"`
		Doc  jsondata.JSONValue `json:"doc"`
		Meta metadata           `json:"meta"`
	}{
		Path: d.Info.Path,
		Doc:  docJSON,
		Meta: d.Info.Meta,
	}

	return json.Marshal(serialStruct)

}

// GetSerial returns a serialized representation of the document contents
// as a JSON-encoded byte slice
func (d *Document) GetSerial() []byte {
	res, _ := d.serialize()
	return res
}

// SubscriptionManagerFactory is a function type that creates a new SubscriptionManager
type SubscriptionManagerFactory func() SubscriptionManager

// AddChildDocument adds a child document at path docpath.
// payload: a raw
func (d *Document) AddChildDocument(docpath string, payload []byte, docname string, user string, overwrite bool, isPost bool, dbName string) ([]byte, int, string) {

	splitPath := strings.Split(docpath, "/")
	parentColName := splitPath[len(splitPath)-2]
	splitPathToParent := splitPath[:len(splitPath)-2]

	parentDoc, found := d.traverseDocuments(splitPathToParent)
	if !found {
		errmsg, _ := json.Marshal("Collection does not exist")
		return errmsg, http.StatusNotFound, ""
	}
	parentCol, foundCol := parentDoc.collections.Find(parentColName)
	if !foundCol {
		errmsg, _ := json.Marshal("Collection does not exist")
		return errmsg, http.StatusNotFound, ""
	}

	newDoc := New(payload, user, docpath, d.docCollectionFactory, d.collectionFactory, d.smFactory, d.validator, d.patcher, d.messager)
	var didOverwrite bool
	check := func(key string, curVal *Document, exists bool) (newVal *Document, err error) {

		if exists && !overwrite {
			return nil, fmt.Errorf("document already exists")
		}
		if exists && overwrite {
			didOverwrite = true
			curVal.UpdateDoc(payload, user)
			slog.Debug("ABOUT TO NOTIFY ABOUT A PUT OVERWRITE")
			curVal.messager.NotifyDocs(dbName+"/"+docpath, "update", curVal.GetSerial())
			parentCol.SubscriptionManager.Notify(docname, "update", curVal.GetSerial())
			return curVal, nil
		}
		if isPost {
			parentCol.SubscriptionManager.Notify(docname, "update", newDoc.GetSerial())
			return newDoc, nil
		}

		newDoc.messager.NotifyDocs(dbName+"/"+docpath, "update", newDoc.GetSerial())
		parentCol.SubscriptionManager.Notify(docname, "update", newDoc.GetSerial())
		return newDoc, nil
	}

	//attempting to upsert
	_, err := parentCol.Docs.Upsert(docname, check)

	if err != nil {
		errmsg, _ := json.Marshal(err.Error())
		return errmsg, http.StatusPreconditionFailed, ""
	}

	resp, uri := generatePutResponse(docpath, dbName)
	if didOverwrite {
		return resp, http.StatusOK, uri
	}
	return resp, http.StatusCreated, uri

}

// AddChildCollection adds an empty collection to the document at resourcepath colpath.
// Returns a JSON-encoded response object and a status code
func (d *Document) AddChildCollection(colpath string, dbName string) ([]byte, int, string) {
	slog.Debug(fmt.Sprintf("Adding a child collection with path %s", colpath))
	colpath = strings.TrimSuffix(colpath, "/")
	newSplitPath := strings.Split(colpath, "/")
	childColName := newSplitPath[len(newSplitPath)-1]
	parentDoc, found := d.traverseDocuments(newSplitPath[:len(newSplitPath)-1])
	if !found {
		errmsg, _ := json.Marshal("Collection does not exist")
		return errmsg, http.StatusNotFound, ""
	}
	newCol := d.collectionFactory(childColName)
	check := func(key string, curVal *Collection, exists bool) (newVal *Collection, err error) {
		if exists {
			return nil, fmt.Errorf("Collection Already Exists")
		}
		return newCol, nil

	}
	_, err := parentDoc.collections.Upsert(childColName, check)
	if err != nil {
		errmsg, _ := json.Marshal(err.Error())
		return errmsg, http.StatusBadRequest, ""
	}
	resp, uri := generatePutResponse(colpath+"/", dbName)

	return resp, http.StatusCreated, uri
}

// GetChildDocument retrieves a document contained in another document located at the forward-slash-delimited path.
// returns a JSON-encoded version of the document, and a status code
func (d *Document) GetChildDocument(docpath string, isSubscribe bool, dbName string) ([]byte, int, string, *chan []byte, []byte) {

	splitPath := strings.Split(docpath, "/")

	if len(splitPath) == 0 { //returning itself

		if isSubscribe {
			subChan, subId := d.messager.AddDocSubscriber(dbName + "/" + docpath)
			return d.GetSerial(), http.StatusOK, subId, subChan, d.sm.GenerateEvent("update", d.GetSerial())
		}

		return d.GetSerial(), http.StatusOK, "", nil, nil
	}

	resDoc, found := d.traverseDocuments(splitPath)
	if !found {
		errmsg, _ := json.Marshal("Document does not exist")
		return errmsg, http.StatusNotFound, "", nil, nil
	}

	payload := resDoc.GetSerial()

	if isSubscribe {

		ch, subId := resDoc.messager.AddDocSubscriber(dbName + "/" + docpath)
		return payload, http.StatusOK, subId, ch, resDoc.sm.GenerateEvent("update", payload)
	}

	return payload, http.StatusOK, "", nil, nil

}

// GetChildCollection gets a child collection belonging to the document d (or one of it's descendant documents). If
// this request is part of a subscription request, a channel and unique identifier will also be returned, and this
// will be used to provide future updates to the client
func (d *Document) GetChildCollection(colpath string, lo string, hi string, isSubscribe bool) ([]byte, int, *chan []byte, string, [][]byte) {
	colpath = strings.TrimSuffix(colpath, "/")
	newSplitPath := strings.Split(colpath, "/")
	childColName := newSplitPath[len(newSplitPath)-1]
	parentDoc, found := d.traverseDocuments(newSplitPath[:len(newSplitPath)-1])
	if !found {
		errmsg, _ := json.Marshal("Collection does not exist")
		return errmsg, http.StatusNotFound, nil, "", nil
	}
	childCol, foundCol := parentDoc.collections.Find(childColName)
	if !foundCol {
		errmsg, _ := json.Marshal("Collection does not exist")
		return errmsg, http.StatusNotFound, nil, "", nil
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()
	payload, stat := childCol.CSerialize(ctx, lo, hi)
	if isSubscribe {
		slog.Debug("Getting a new channel for this subscription request")
		subChan, subId := childCol.SubscriptionManager.AddSubscriber(lo, hi)
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(15*time.Second))
		defer cancel()
		docs, _ := childCol.Docs.Query(ctx, lo, hi)
		docBytes := make([][]byte, 0)
		for _, v := range docs {
			event := childCol.SubscriptionManager.GenerateEvent("update", v.Value.GetSerial())
			docBytes = append(docBytes, event)
		}
		return payload, stat, subChan, subId, docBytes
	}
	return payload, stat, nil, "", nil
}

// DeleteChildDocument deletes a child document of the parent document d
// Returns a response (if an error occurred) and a status code
func (d *Document) DeleteChildDocument(docpath string, dbName string) ([]byte, int) {

	splitPath := strings.Split(docpath, "/")

	parentPath := splitPath[:len(splitPath)-2]

	parentColName := splitPath[len(splitPath)-2] //we need to Remove() on the collection

	victimName := splitPath[len(splitPath)-1]

	parentDoc, foundDoc := d.traverseDocuments(parentPath)

	if !foundDoc {
		errmsg, _ := json.Marshal("Document does not exist")
		return errmsg, http.StatusNotFound
	}

	parentCol, foundCol := parentDoc.collections.Find(parentColName)

	if !foundCol {
		errmsg, _ := json.Marshal("Document does not exist")
		return errmsg, http.StatusNotFound
	}
	removedDoc, removed := parentCol.Docs.Remove(victimName)

	if !removed {
		errmsg, _ := json.Marshal("Document does not exist")
		return errmsg, http.StatusNotFound
	}
	//notifying all documents
	removedDoc.Notify(dbName+"/"+docpath, []byte("/"+docpath), "delete")
	colmsg, _ := json.Marshal("/" + docpath)

	parentCol.SubscriptionManager.Notify(victimName, "delete", colmsg)
	return nil, http.StatusNoContent

}

// DeleteChildCollection deletes a collection within the document
// Returns a response (if an error occurred) and a status code
func (d *Document) DeleteChildCollection(colpath string) ([]byte, int) {
	newPath := colpath
	newPath = strings.TrimSuffix(newPath, "/")
	newSplitPath := strings.Split(newPath, "/")
	childColName := newSplitPath[len(newSplitPath)-1]
	parentDoc, found := d.traverseDocuments(newSplitPath[:len(newSplitPath)-1])
	if !found { //the only way find fails is if the document is being deleted/can't be found, etc... so...
		errmsg, _ := json.Marshal("Owning document does not exist")
		return errmsg, http.StatusNotFound
	}
	removedCol, removed := parentDoc.collections.Remove(childColName)

	if !removed { //the only way remove fails is if the collection doesn't exist, so...
		errmsg, _ := json.Marshal("Collection does not exist")
		return errmsg, http.StatusNotFound
	}
	slog.Debug("Notifiying subscribers that this collection is deleted")
	b, _ := json.Marshal("/" + colpath)
	removedCol.SubscriptionManager.NotifyAll(string(b))
	return nil, http.StatusNoContent
}

// generatePutResponse creates a JSON response object for put requests
// Returns a JSON-encoded byte slice
func generatePutResponse(uri string, dbName string) ([]byte, string) {
	respJson := struct {
		Uri string `json:"uri"`
	}{
		Uri: "/v1/" + dbName + "/" + uri,
	}
	b, _ := json.Marshal(respJson)
	return b, respJson.Uri
}

// traverseDocuments is helper method used to navigate from a parent document to it's descendant documents
// It returns the document at the end of the path, and a boolean indicating whether the document was found
func (d *Document) traverseDocuments(splitPath []string) (*Document, bool) {
	curDoc := d

	if len(splitPath) == 1 {
		return d, true
	}
	for i := 0; i < len(splitPath)-1; i += 2 {
		tmp, found := curDoc.traverseDocument(splitPath[i+1], splitPath[i+2])
		if !found {
			return nil, false
		}
		curDoc = tmp
	}
	return curDoc, true
}

// traverseDocument finds the document with name docName in collection colName, a collection belonging to the parent doc
// Returns the document and a boolean indicating whether the document was found
func (d *Document) traverseDocument(colName string, docName string) (*Document, bool) {
	var resDoc *Document
	var foundDoc bool
	nextCol, found := d.collections.Find(colName)
	if !found {
		return resDoc, found
	}
	resDoc, foundDoc = nextCol.Docs.Find(docName)
	return resDoc, foundDoc

}

// ApplyPatchDocument applies a JSON patch to the document at the specified path
// Returns a JSON-encoded response object and a status code
func (d *Document) ApplyPatchDocument(dbName string, docPath string, patch []byte, user string) ([]byte, int) {
	// Step 1: Split docPath into path segments and clean it
	splitPath := strings.Split(docPath, "/")

	// Step 2: Check if documentPath is for the root document
	splitPathToParent := splitPath[:len(splitPath)-2]
	parentColName := splitPath[len(splitPath)-2]
	parentDoc, found := d.traverseDocuments(splitPathToParent)

	if !found {
		errmsg, _ := json.Marshal("document at that path does not exist")
		return errmsg, http.StatusNotFound
	}

	parentCol, found := parentDoc.collections.Find(parentColName)

	if !found {
		errmsg, _ := json.Marshal("owning collection does not exist")
		return errmsg, http.StatusNotFound
	}
	docName := splitPath[len(splitPath)-1]
	// Use the same update function for documents within collections

	var nullDoc *Document
	var newDocPayload []byte
	chk := func(docName string, curDoc *Document, exists bool) (newDoc *Document, err error) {
		if !exists { //can't update something that doesn't exist
			return nullDoc, fmt.Errorf("document does not exist")
		}
		newRaw, patchErr := curDoc.DoPatch(patch)
		if patchErr != nil { //DO NOT UPDATE
			return curDoc, patchErr
		}
		err = d.validator.Validate(newRaw)
		if err != nil {
			return curDoc, err
		}
		curDoc.UpdateDoc(newRaw, user)

		newDocPayload = curDoc.GetSerial()
		//Notifying subscribers
		parentCol.SubscriptionManager.Notify(docName, "update", newDocPayload)
		curDoc.messager.NotifyDocs(dbName+"/"+docPath, "update", newDocPayload)
		return curDoc, nil
	}
	updated, er := parentCol.Docs.Upsert(docName, chk)
	msg := "patch applied"
	if !updated {

		if er.Error() == "document does not exist" {
			errmsg, _ := json.Marshal(er.Error())
			return errmsg, http.StatusNotFound
		} else if strings.HasPrefix(er.Error(), "bad patch operation") {
			errmsg, _ := json.Marshal(er.Error())
			return errmsg, http.StatusBadRequest
		} else {
			msg = er.Error()
		}
	}
	return generatePatchResponse(dbName, docName, !updated, msg), http.StatusOK
}

// getRawBody gets ONLY THE BODY OF THE JSON DOCUMENT. DO NOT EVER GIVE THIS TO THE USER... FOR PATCHES ONLY!!!
// Returns a deep copy of the document's body
func (d *Document) getRawBody() []byte {
	deepCopy := make([]byte, len(d.Info.Doc))
	copy(deepCopy, d.Info.Doc)
	return deepCopy
}

// DoPatch applies a JSON patch to the document
// Returns the patched document as a JSON-encoded byte slice
func (d *Document) DoPatch(patch []byte) (newRaw []byte, err error) {
	oldRaw := d.getRawBody()
	return d.patcher.DoPatch(oldRaw, patch)
}

// UpdateDoc mutates the documents contents (DO NOT CALL THIS WITHOUT SYNCHRONIZATION)
func (d *Document) UpdateDoc(newRawDoc []byte, user string) {
	d.Info.Doc = newRawDoc
	d.Info.Meta.LastModifiedBy = user
	d.Info.Meta.LastModifiedAt = time.Now().UnixMilli()

}

// generatePatchResponse generates a response for a patch operation
// Returns a JSON-encoded byte slice
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

// Notify forwards the event notification to an internal subscription handler
func (d *Document) Notify(uri string, payload []byte, evType string) {
	d.messager.NotifyDocs(uri, evType, payload)
}
