package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/RICE-COMP318-FALL24/owldb-p1group24/auth"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/concurrentSkipList"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/db"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/document"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/patcher"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/resourceCreatorService"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/resourceDeleterService"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/resourceGetterService"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/resourcePatcherService"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/server"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/subscriptionManager"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/validation"
)

func setup(schemaFile string) (http.Handler, error) {
	validator, err := validation.New(schemaFile)

	if err != nil {
		fmt.Printf("Error: Bad schema file\n")
		os.Exit(1)
	}
	// Dependency injection and factory initialization
	var docColFactory document.DocumentIndexFactory[document.DocumentIndex[string, *document.Collection]]

	var dbFactory resourceCreatorService.DBFactory[*db.Database[string, *document.Document]]

	var docFactory db.DocFactory[*document.Document]
	var newerColFactory document.CollectionFactory
	newerColFactory = func(colName string) *document.Collection {
		return &document.Collection{
			Name:                colName,
			Docs:                concurrentSkipList.NewSL[string, *document.Document](string(rune(0)), string(rune(127))),
			SubscriptionManager: subscriptionManager.NewColSubManager(concurrentSkipList.NewSL[string, subscriptionManager.Colsubscriber](string(rune(0)), string(rune(127)))),
		}
	}

	var smFactory document.SubscriptionManagerFactory = func() document.SubscriptionManager {
		subs := concurrentSkipList.NewSL[string, *chan []byte](string(rune(0)), string(rune(127)))
		return subscriptionManager.New(subs)
	}

	var idtosubfactory subscriptionManager.IdToSubFactory = func() subscriptionManager.IdToSub[string, *chan []byte] {
		return concurrentSkipList.NewSL[string, *chan []byte](string(rune(0)), string(rune(127)))
	}

	docSubs := concurrentSkipList.NewSL[string, *subscriptionManager.SubscriptionManager](string(rune(0)), string(rune(127)))
	messager := subscriptionManager.NewMessager(idtosubfactory, docSubs)
	docFactory = func(payload []byte, user string, path string) *document.Document {

		newDoc := document.New(payload, user, path, docColFactory, newerColFactory, smFactory, validator, patcher.Patcher{}, messager)
		return newDoc
	}

	docColFactory = func() document.DocumentIndex[string, *document.Collection] {
		newCollections := concurrentSkipList.NewSL[string, *document.Collection](string(rune(0)), string(rune(127)))
		return newCollections
	}

	dbFactory = func(name string) *db.Database[string, *document.Document] {
		newDBIndices := concurrentSkipList.NewSL[string, *document.Document](string(rune(0)), string(rune(127)))
		sm := subscriptionManager.NewColSubManager(concurrentSkipList.NewSL[string, subscriptionManager.Colsubscriber](string(rune(0)), string(rune(127))))
		return db.New[string, *document.Document](name, docFactory, newDBIndices, sm, validator)

	}
	//FOR CRUD OPERATIONS
	// Initialize database and resource services
	dbs := concurrentSkipList.NewSL[string, *db.Database[string, *document.Document]](string(rune(0)), string(rune(127)))
	var rcsDB resourceCreatorService.DatabaseIndex[string, *db.Database[string, *document.Document]] = dbs
	var rgsDB resourceGetterService.DatabaseIndex[string, *db.Database[string, *document.Document]] = dbs
	var rdsDB resourceDeleterService.DatabaseIndex[string, *db.Database[string, *document.Document]] = dbs
	var rpsDB resourcePatcherService.DatabaseIndex[string, *db.Database[string, *document.Document]] = dbs
	rcs := resourceCreatorService.New(rcsDB, dbFactory, validator)
	rgs := resourceGetterService.New(rgsDB)
	rds := resourceDeleterService.New(rdsDB)
	rps := resourcePatcherService.New(rpsDB)

	// Initialize authentication services

	var tokenMap auth.TokenIndex[string, auth.Session] = concurrentSkipList.NewSL[string, auth.Session](string(rune(0)), string(rune(127)))

	authService := auth.New(tokenMap, "tokens.json")

	// Initialize the server handler
	handler := server.New(rds, rgs, rcs, authService, rps)
	return handler, nil
}

type PutResponse struct {
	Uri string `json:"uri"`
}

func TestPutDB(t *testing.T) {
	handler, er := setup("Allschema.json")
	if er != nil {
		t.Fatalf("handler failed to start")
	}
	req := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(`test db put`))
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp := w.Result()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Error reading response body: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("TestPutDB failed")
	}

	var putResp PutResponse
	json.Unmarshal(body, &putResp)

	if putResp.Uri != "/v1/db24" {
		t.Errorf("TestPutDB failed, got %s", putResp.Uri)
	}

}

func TestDeleteDB(t *testing.T) {
	handler, _ := setup("Allschema.json")
	putReq := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(`test db put`))
	putReq.Header.Set("accept", "application/json")
	putReq.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, putReq)

	deleteReq := httptest.NewRequest("DELETE", "/v1/db24", strings.NewReader(`test db delete`))
	deleteReq.Header.Set("accept", "application/json")
	deleteReq.Header.Set("Authorization", "Bearer ADMIN")

	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, deleteReq)

	resp := w2.Result()

	if resp.StatusCode != http.StatusNoContent {

		t.Errorf("TestDeleteDB failed")
	}
}

func TestPutDoc(t *testing.T) {
	handler, _ := setup("Allschema.json")
	putReq := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(`test db put`))
	putReq.Header.Set("accept", "application/json")
	putReq.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, putReq)

	var data = strings.NewReader(`{
  "additionalProp1": "string",
  "additionalProp2": "string",
  "additionalProp3": "string"
}`)
	docPutReq := httptest.NewRequest("PUT", "/v1/db24/doc1", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, docPutReq)
	resp := w2.Result()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		t.Errorf("TestPutDoc failed")
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("TestPutDoc failed")
	}

	var putResp PutResponse
	json.Unmarshal(body, &putResp)

	if putResp.Uri != "/v1/db24/doc1" {
		t.Errorf("TestPutDB failed, got %s", putResp.Uri)
	}

}

func TestPutDocNoDB(t *testing.T) {
	handler, _ := setup("Allschema.json")

	var data = strings.NewReader(`{
  "additionalProp1": "string",
  "additionalProp2": "string",
  "additionalProp3": "string"
}`)
	docPutReq := httptest.NewRequest("PUT", "/v1/db24/doc1", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, docPutReq)
	resp := w2.Result()

	_, err := io.ReadAll(resp.Body)

	if err != nil {
		t.Errorf("TestPutDoc failed")
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("TestPutDoc failed")
	}

}

func TestPutDuplicateDB(t *testing.T) {
	handler, _ := setup("Allschema.json")
	req := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(`test db put`))
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	req2 := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(`test db put`))
	req2.Header.Set("accept", "application/json")
	req2.Header.Set("Authorization", "Bearer ADMIN")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	resp := w2.Result()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("TestDuplicateDB failed, expected status code 400,got %d", resp.StatusCode)
	}
}

func TestGetDoc(t *testing.T) {
	handler, _ := setup("Allschema.json")
	req := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(`test db put`))
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var data = strings.NewReader(`{
  "additionalProp1": "string",
  "additionalProp2": "string",
  "additionalProp3": "string"
}`)
	docPutReq := httptest.NewRequest("PUT", "/v1/db24/doc1", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, docPutReq)

	docGetReq := httptest.NewRequest("GET", "/v1/db24/doc1", data)
	docGetReq.Header.Set("accept", "application/json")
	docGetReq.Header.Set("Authorization", "Bearer ADMIN")
	docGetReq.Header.Set("Content-Type", "application/json")

	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, docGetReq)
	r3 := httptest.NewRecorder()

	h := r3.Result().StatusCode

	if h != http.StatusOK {
		t.Errorf("TestGetDoc failed: got ")
	}
}

type Doc struct {
	Key string `json:"key"`
}

type docstruct struct {
	Path string `json:"path"`
	Doc  Doc    `json:"doc"`
	Meta string `json:"meta"`
}

func TestGetMultipleDocs(t *testing.T) {
	handler, _ := setup("Allschema.json")
	req := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(`test db put`))
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var data = strings.NewReader(`{
  "key":"hello1"
}`)
	docPutReq := httptest.NewRequest("PUT", "/v1/db24/doc1", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, docPutReq)
	fmt.Printf("%d", w2.Result().StatusCode)
	data = strings.NewReader(`{
  "key":"hello2"
}`)
	docPutReq = httptest.NewRequest("PUT", "/v1/db24/doc2", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	w2 = httptest.NewRecorder()
	handler.ServeHTTP(w2, docPutReq)
	fmt.Printf("%d", w2.Result().StatusCode)
	data = strings.NewReader(`{
  "key":"hello3"
}`)
	docPutReq = httptest.NewRequest("PUT", "/v1/db24/doc3", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	w2 = httptest.NewRecorder()
	handler.ServeHTTP(w2, docPutReq)

	data = strings.NewReader(`{
  "key":"hello4"
}`)
	docPutReq = httptest.NewRequest("PUT", "/v1/db24/doc4", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, docPutReq)

	colGetReq := httptest.NewRequest("GET", "/v1/db24/?interval=%5B%2C%5D", strings.NewReader(`test col get`))
	colGetReq.Header.Set("accept", "application/json")
	colGetReq.Header.Set("Authorization", "Bearer ADMIN")
	finalres := httptest.NewRecorder()
	handler.ServeHTTP(finalres, colGetReq)

	code := finalres.Result().StatusCode

	if code != http.StatusOK {
		t.Errorf("Failed")
	}

	body, err := io.ReadAll(finalres.Result().Body)
	if err != nil {
		t.Errorf("Failed")
	}
	var docs []docstruct
	err = json.Unmarshal(body, &docs)

	if len(docs) != 4 {
		t.Errorf("Get multiple docs failed, expected 4 docs, got %d", len(docs))
	}

}

func TestDeleteTopDoc(t *testing.T) {
	handler, err := setup("Allschema.json")

	if err != nil {
		t.Errorf("Bad schema, server failed to start")

	}

	putReq := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(""))

	putReq.Header.Set("accept", "application/json")
	putReq.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, putReq)

	var data = strings.NewReader(`{"key":"hello1"}`)

	docPutReq := httptest.NewRequest("PUT", "/v1/db24/b", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, docPutReq)

	docDeleteReq := httptest.NewRequest("DELETE", "/v1/db24/b", strings.NewReader(""))
	docDeleteReq.Header.Set("accept", "*/*")
	docDeleteReq.Header.Set("Authorization", "Bearer ADMIN")

	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, docDeleteReq)

	if w3.Result().StatusCode != http.StatusNoContent {
		t.Errorf("Delete top failed")
	}
}

func TestGetMultipleDocsInRange(t *testing.T) {
	handler, _ := setup("Allschema.json")
	req := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(`test db put`))
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var data = strings.NewReader(`{
  "key":"hello1"
}`)
	docPutReq := httptest.NewRequest("PUT", "/v1/db24/b", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, docPutReq)

	data = strings.NewReader(`{
  "key":"hello2"
}`)
	docPutReq = httptest.NewRequest("PUT", "/v1/db24/c", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	w2 = httptest.NewRecorder()
	handler.ServeHTTP(w2, docPutReq)

	data = strings.NewReader(`{
  "key":"hello3"
}`)
	docPutReq = httptest.NewRequest("PUT", "/v1/db24/d", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	w2 = httptest.NewRecorder()
	handler.ServeHTTP(w2, docPutReq)

	data = strings.NewReader(`{
  "key":"hello4"
}`)
	docPutReq = httptest.NewRequest("PUT", "/v1/db24/e", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, docPutReq)

	colGetReq := httptest.NewRequest("GET", "/v1/db24/?interval=%5Bd%2Ce%5D", strings.NewReader(`test col get`))
	colGetReq.Header.Set("accept", "application/json")
	colGetReq.Header.Set("Authorization", "Bearer ADMIN")
	finalres := httptest.NewRecorder()
	handler.ServeHTTP(finalres, colGetReq)

	code := finalres.Result().StatusCode

	if code != http.StatusOK {
		t.Errorf("Failed")
	}

	body, err := io.ReadAll(finalres.Result().Body)
	if err != nil {
		t.Errorf("Failed")
	}
	var docs []docstruct
	err = json.Unmarshal(body, &docs)
	if len(docs) != 2 {
		t.Errorf("error getting documents in a range: expected 2, got %d", 2)
	}

}

func TestPutDocOverwrite(t *testing.T) {
	handler, _ := setup("Allschema.json")
	req := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(`test db put`))
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var data = strings.NewReader(`{
  "key":"hello1"
}`)
	docPutReq := httptest.NewRequest("PUT", "/v1/db24/b", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, docPutReq)
	var data2 = strings.NewReader(`{
  "key":"hello2"
}`)
	docPutReq2 := httptest.NewRequest("PUT", "/v1/db24/b", data2)
	docPutReq2.Header.Set("accept", "application/json")
	docPutReq2.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq2.Header.Set("Content-Type", "application/json")
	docPutReq2.URL.Query().Set("mode", "overwrite")
	w2 = httptest.NewRecorder()
	handler.ServeHTTP(w2, docPutReq2)
	res := w2.Result().Body

	_, err := io.ReadAll(res)
	if err != nil {
		t.Errorf("Failed")
	}
	if w2.Result().StatusCode != http.StatusOK {

		t.Errorf("Failed")
	}
	docGetReq := httptest.NewRequest("GET", "/v1/db24/b", data)
	docGetReq.Header.Set("accept", "application/json")
	docGetReq.Header.Set("Authorization", "Bearer ADMIN")
	docGetReq.Header.Set("Content-Type", "application/json")

	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, docGetReq)

	body, err := io.ReadAll(w3.Result().Body)
	if err != nil {
		t.Errorf("Failed")
	}
	if w3.Result().StatusCode != http.StatusOK {

		t.Errorf("Failed")
	}

	var doc docstruct
	err = json.Unmarshal(body, &doc)

	fmt.Printf("%+v", doc)
	fmt.Printf("\n")

}

func TestDocNooverwrite(t *testing.T) {
	handler, _ := setup("Allschema.json")
	req := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(`test db put`))
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var data = strings.NewReader(`{
  "key":"hello1"
}`)
	docPutReq := httptest.NewRequest("PUT", "/v1/db24/b", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, docPutReq)
	var data2 = strings.NewReader(`{
  "key":"hello2"
}`)
	docPutReq2 := httptest.NewRequest("PUT", "/v1/db24/b?mode=nooverwrite", data2)
	docPutReq2.Header.Set("accept", "application/json")
	docPutReq2.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq2.Header.Set("Content-Type", "application/json")

	w2 = httptest.NewRecorder()
	handler.ServeHTTP(w2, docPutReq2)
	res := w2.Result().Body

	_, err := io.ReadAll(res)
	if err != nil {
		t.Errorf("Failed")
	}
	if w2.Result().StatusCode != http.StatusPreconditionFailed {

		t.Errorf("Failed")
	}

}

func TestDeleteDoc(t *testing.T) {
	handler, err := setup("Allschema.json")
	if err != nil {
		t.Errorf("TestDelete failed to start, %s", err.Error())
	}
	req := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(""))

	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusCreated {
		t.Errorf("Delete Doc failed: unable to add DB,got %d", w.Result().StatusCode)
	}
	var data = strings.NewReader(`{"key":"hello1"}`)
	docPutReq := httptest.NewRequest("PUT", "/v1/db24/b", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, docPutReq)

	colPutReq := httptest.NewRequest("PUT", "/v1/db24/b/c/", data)
	colPutReq.Header.Set("accept", "application/json")
	colPutReq.Header.Set("Authorization", "Bearer ADMIN")
	colPutReq.Header.Set("Content-Type", "application/json")

	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, colPutReq)

	data2 := strings.NewReader(`{"key":"hello1"}`)
	docPutReq = httptest.NewRequest("PUT", "/v1/db24/b/c/d", data2)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	w2 = httptest.NewRecorder()
	handler.ServeHTTP(w2, docPutReq)
	if w2.Result().StatusCode != http.StatusCreated {
		t.Errorf("DeleteDoc failed: unable to create top doc, got %d", w2.Result().StatusCode)
	}
	deleteDocReq := httptest.NewRequest("DELETE", "/v1/db24/b/c/d", strings.NewReader(""))
	deleteDocReq.Header.Set("accept", "application/json")
	deleteDocReq.Header.Set("Authorization", "Bearer ADMIN")
	deleteDocReq.Header.Set("Content-Type", "application/json")

	w4 := httptest.NewRecorder()
	handler.ServeHTTP(w4, deleteDocReq)

	if w4.Result().StatusCode != http.StatusNoContent {

		t.Errorf("Failed,got %d", w4.Result().StatusCode)
	}

}

func TestDeleteDocNonExistent(t *testing.T) {
	handler, err := setup("Allschema.json")
	if err != nil {
		t.Errorf("TestDelete failed to start, %s", err.Error())
	}
	req := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(""))

	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusCreated {
		t.Errorf("Delete Doc failed: unable to add DB,got %d", w.Result().StatusCode)
	}
	var data = strings.NewReader(`{"key":"hello1"}`)
	docPutReq := httptest.NewRequest("PUT", "/v1/db24/b", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, docPutReq)

	colPutReq := httptest.NewRequest("PUT", "/v1/db24/b/c/", data)
	colPutReq.Header.Set("accept", "application/json")
	colPutReq.Header.Set("Authorization", "Bearer ADMIN")
	colPutReq.Header.Set("Content-Type", "application/json")

	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, colPutReq)

	data2 := strings.NewReader(`{"key":"hello1"}`)
	docPutReq = httptest.NewRequest("PUT", "/v1/db24/b/c/d", data2)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	w2 = httptest.NewRecorder()
	handler.ServeHTTP(w2, docPutReq)
	if w2.Result().StatusCode != http.StatusCreated {
		t.Errorf("DeleteDoc failed: unable to create top doc, got %d", w2.Result().StatusCode)
	}
	deleteDocReq := httptest.NewRequest("DELETE", "/v1/db24/b/c/d", strings.NewReader(""))
	deleteDocReq.Header.Set("accept", "application/json")
	deleteDocReq.Header.Set("Authorization", "Bearer ADMIN")
	deleteDocReq.Header.Set("Content-Type", "application/json")

	w4 := httptest.NewRecorder()
	handler.ServeHTTP(w4, deleteDocReq)

	if w4.Result().StatusCode != http.StatusNoContent {

		t.Errorf("Failed,got %d", w4.Result().StatusCode)
	}
	deleteDocReq = httptest.NewRequest("DELETE", "/v1/db24/b/c/d", strings.NewReader(""))
	deleteDocReq.Header.Set("accept", "application/json")
	deleteDocReq.Header.Set("Authorization", "Bearer ADMIN")
	deleteDocReq.Header.Set("Content-Type", "application/json")

	w4 = httptest.NewRecorder()
	handler.ServeHTTP(w4, deleteDocReq)
	if w4.Result().StatusCode != http.StatusNotFound {

		t.Errorf("Delete Non-existent failed, got status code %d instead of 404", w4.Result().StatusCode)
	}
}

func TestGetDeeplyNestedDoc(t *testing.T) {
	handler, _ := setup("Allschema.json")
	req := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(`test db put`))
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	var data = strings.NewReader(`{"key":"hello1"}`)
	docPutReq := httptest.NewRequest("PUT", "/v1/db24/b", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, docPutReq)

	colPutReq := httptest.NewRequest("PUT", "/v1/db24/b/c/", data)
	colPutReq.Header.Set("accept", "application/json")
	colPutReq.Header.Set("Authorization", "Bearer ADMIN")
	colPutReq.Header.Set("Content-Type", "application/json")

	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, colPutReq)

	data2 := strings.NewReader(`{"key":"hello1"}`)
	docPutReq = httptest.NewRequest("PUT", "/v1/db24/b/c/d", data2)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	w2 = httptest.NewRecorder()
	handler.ServeHTTP(w2, docPutReq)

	b, _ := io.ReadAll(w2.Result().Body)
	fmt.Printf(string(b))
	colPutReq = httptest.NewRequest("PUT", "/v1/db24/b/c/d/e/", data)
	colPutReq.Header.Set("accept", "application/json")
	colPutReq.Header.Set("Authorization", "Bearer ADMIN")
	colPutReq.Header.Set("Content-Type", "application/json")
	w2 = httptest.NewRecorder()
	handler.ServeHTTP(w2, colPutReq)

	var finaldata = strings.NewReader(`{
	"key":"FINAL DOCUMENT"
	}`)
	docPutReq = httptest.NewRequest("PUT", "/v1/db24/b/c/d/e/f", finaldata)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	w2 = httptest.NewRecorder()
	handler.ServeHTTP(w2, docPutReq)

	dat := strings.NewReader("")
	nestedDocGetReq := httptest.NewRequest("GET", "/v1/db24/b/c/d/e/f", dat)
	nestedDocGetReq.Header.Set("accept", "application/json")
	nestedDocGetReq.Header.Set("Authorization", "Bearer ADMIN")
	nestedDocGetReq.Header.Set("Content-Type", "application/json")

	final := httptest.NewRecorder()
	handler.ServeHTTP(final, nestedDocGetReq)

}

// Here we insert a child collection to a top-level document,and give it documents with the names e,f,g,h.
// We then test various queries
func TestCollectionQueries(t *testing.T) {

	handler, _ := setup("Allschema.json")
	req := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(`test db put`))
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	var data = strings.NewReader(`{"key":"hello1"}`)
	docPutReq := httptest.NewRequest("PUT", "/v1/db24/b", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, docPutReq)

	colPutReq := httptest.NewRequest("PUT", "/v1/db24/b/c/", data)
	colPutReq.Header.Set("accept", "application/json")
	colPutReq.Header.Set("Authorization", "Bearer ADMIN")
	colPutReq.Header.Set("Content-Type", "application/json")

	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, colPutReq)

	data = strings.NewReader(`{"key":"hello1"}`)
	docPutReq = httptest.NewRequest("PUT", "/v1/db24/b/c/e", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	w2 = httptest.NewRecorder()
	handler.ServeHTTP(w2, docPutReq)

	data = strings.NewReader(`{"key":"hello2"}`)
	docPutReq = httptest.NewRequest("PUT", "/v1/db24/b/c/f", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	w2 = httptest.NewRecorder()
	handler.ServeHTTP(w2, docPutReq)

	data = strings.NewReader(`{
  "key":"hello3"
}`)
	docPutReq = httptest.NewRequest("PUT", "/v1/db24/b/c/g", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	w2 = httptest.NewRecorder()
	handler.ServeHTTP(w2, docPutReq)

	data = strings.NewReader(`{"key":"hello4"}`)
	docPutReq = httptest.NewRequest("PUT", "/v1/db24/b/c/h", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	httptest.NewRecorder()
	handler.ServeHTTP(w3, docPutReq)

	colGetReq := httptest.NewRequest("GET", "/v1/db24/b/c/?interval=[f,h]", strings.NewReader(`test col get`))
	colGetReq.Header.Set("accept", "application/json")
	colGetReq.Header.Set("Authorization", "Bearer ADMIN")
	finalres := httptest.NewRecorder()
	handler.ServeHTTP(finalres, colGetReq)

	code := finalres.Result().StatusCode

	if code != http.StatusOK {
		t.Errorf("Failed")
	}

	body, err := io.ReadAll(finalres.Result().Body)
	fmt.Printf(string(body))
	if err != nil {
		t.Errorf("Failed")
	}
	var docs []docstruct
	//f,g,h documents
	err = json.Unmarshal(body, &docs)
	if len(docs) != 3 {
		t.Errorf("Failed")
	}

	colGetReq = httptest.NewRequest("GET", "/v1/db24/b/c/?interval=[,h]", strings.NewReader(`test col get`))
	colGetReq.Header.Set("accept", "application/json")
	colGetReq.Header.Set("Authorization", "Bearer ADMIN")
	finalres = httptest.NewRecorder()
	handler.ServeHTTP(finalres, colGetReq)

	var docs2 []docstruct
	//e,f,g,h documents
	body2, err2 := io.ReadAll(finalres.Result().Body)
	if err2 != nil {
		t.Errorf("Failed")
	}
	err = json.Unmarshal(body2, &docs2)
	if len(docs2) != 4 {
		t.Errorf("Failed")
	}
}

func TestDeleteCol(t *testing.T) {
	handler, _ := setup("Allschema.json")
	req := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(`test db put`))
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	var data = strings.NewReader(`{"key":"hello1"}`)
	docPutReq := httptest.NewRequest("PUT", "/v1/db24/b", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	putDoc := httptest.NewRecorder()
	handler.ServeHTTP(putDoc, docPutReq)
	colPutReq := httptest.NewRequest("PUT", "/v1/db24/b/c/", data)
	colPutReq.Header.Set("accept", "application/json")
	colPutReq.Header.Set("Authorization", "Bearer ADMIN")
	colPutReq.Header.Set("Content-Type", "application/json")
	putCol := httptest.NewRecorder()
	handler.ServeHTTP(putCol, colPutReq)

	//deleting collection:
	data = strings.NewReader(`{"key":"hello1"}`)
	colDelete := httptest.NewRequest("DELETE", "/v1/db24/b/c/", data)
	colDelete.Header.Set("accept", "application/json")
	colDelete.Header.Set("Authorization", "Bearer ADMIN")
	colDelete.Header.Set("Content-Type", "application/json")
	delCol := httptest.NewRecorder()
	handler.ServeHTTP(delCol, colDelete)

	if delCol.Result().StatusCode != http.StatusNoContent {
		t.Errorf("Failed")
	}

	data = strings.NewReader(`{"key":"hello1"}`)
	colGet := httptest.NewRequest("GET", "/v1/db24/b/c/", data)
	colGet.Header.Set("accept", "application/json")
	colGet.Header.Set("Authorization", "Bearer ADMIN")
	colGet.Header.Set("Content-Type", "application/json")
	getCol := httptest.NewRecorder()
	handler.ServeHTTP(getCol, colDelete)

	if getCol.Result().StatusCode != http.StatusNotFound {
		t.Errorf("Failed")
	}
}

// Test deleting a nonexistent collection
func TestDeleteColNonExistent(t *testing.T) {
	handler, _ := setup("Allschema.json")
	req := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(`test db put`))
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	var data = strings.NewReader(`{"key":"hello1"}`)

	//putting a document
	docPutReq := httptest.NewRequest("PUT", "/v1/db24/b", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	putDoc := httptest.NewRecorder()
	handler.ServeHTTP(putDoc, docPutReq)

	data = strings.NewReader(`{"key":"hello1"}`)

	//deleting a nonexistent collection
	colDelete := httptest.NewRequest("DELETE", "/v1/db24/b/c/", data)
	colDelete.Header.Set("accept", "application/json")
	colDelete.Header.Set("Authorization", "Bearer ADMIN")
	colDelete.Header.Set("Content-Type", "application/json")
	delCol := httptest.NewRecorder()
	handler.ServeHTTP(delCol, colDelete)

	if delCol.Result().StatusCode != http.StatusNotFound {
		t.Errorf("Failed")
	}
}

// TestColAlreadyExists tests inserting a duplicate collection
func TestColAlreadyExists(t *testing.T) {
	handler, _ := setup("Allschema.json")
	req := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(`test db put`))
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	var data = strings.NewReader(`{"key":"hello1"}`)
	docPutReq := httptest.NewRequest("PUT", "/v1/db24/b", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	putDoc := httptest.NewRecorder()
	handler.ServeHTTP(putDoc, docPutReq)

	colPutReq := httptest.NewRequest("PUT", "/v1/db24/b/c/", data)
	colPutReq.Header.Set("accept", "application/json")
	colPutReq.Header.Set("Authorization", "Bearer ADMIN")
	colPutReq.Header.Set("Content-Type", "application/json")
	putCol := httptest.NewRecorder()
	handler.ServeHTTP(putCol, colPutReq)
	//duplicate
	colPutReq = httptest.NewRequest("PUT", "/v1/db24/b/c/", data)
	colPutReq.Header.Set("accept", "application/json")
	colPutReq.Header.Set("Authorization", "Bearer ADMIN")
	colPutReq.Header.Set("Content-Type", "application/json")
	putCol = httptest.NewRecorder()
	handler.ServeHTTP(putCol, colPutReq)

	if putCol.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("Failed")
	}

}

func TestMultipleCol(t *testing.T) {

}

// Tests an unauthorized put on the collection
func TestPutColNoAuth(t *testing.T) {
	handler, _ := setup("Allschema.json")
	req := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(`test db put`))
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	var data = strings.NewReader(`{"key":"hello1"}`)
	docPutReq := httptest.NewRequest("PUT", "/v1/db24/b", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	putDoc := httptest.NewRecorder()
	handler.ServeHTTP(putDoc, docPutReq)

	colPutReq := httptest.NewRequest("PUT", "/v1/db24/b/c/", data)
	colPutReq.Header.Set("accept", "application/json")

	colPutReq.Header.Set("Content-Type", "application/json")
	putCol := httptest.NewRecorder()
	handler.ServeHTTP(putCol, colPutReq)
	if putCol.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("Failed")
	}
	fmt.Printf("\n")
}

// TestDeleteColNoAuth tries deleting a collection without authenticating first
func TestDeleteColNoAuth(t *testing.T) {
	handler, _ := setup("Allschema.json")
	req := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(`test db put`))
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	var data = strings.NewReader(`{"key":"hello1"}`)
	docPutReq := httptest.NewRequest("PUT", "/v1/db24/b", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	putDoc := httptest.NewRecorder()
	handler.ServeHTTP(putDoc, docPutReq)
	colPutReq := httptest.NewRequest("PUT", "/v1/db24/b/c/", data)
	colPutReq.Header.Set("accept", "application/json")
	colPutReq.Header.Set("Authorization", "Bearer ADMIN")
	colPutReq.Header.Set("Content-Type", "application/json")
	putCol := httptest.NewRecorder()
	handler.ServeHTTP(putCol, colPutReq)

	//deleting collection:
	data = strings.NewReader(`{"key":"hello1"}`)
	colDelete := httptest.NewRequest("DELETE", "/v1/db24/b/c/", data)
	colDelete.Header.Set("accept", "application/json")

	colDelete.Header.Set("Content-Type", "application/json")
	delCol := httptest.NewRecorder()
	handler.ServeHTTP(delCol, colDelete)

	if delCol.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("Failed1")
	}

	data = strings.NewReader(`{"key":"hello1"}`)
	colGet := httptest.NewRequest("DELETE", "/v1/db24/b/c/", data)
	colGet.Header.Set("accept", "application/json")
	colGet.Header.Set("Authorization", "Bearer ADMIN")
	colGet.Header.Set("Content-Type", "application/json")
	getCol := httptest.NewRecorder()
	handler.ServeHTTP(getCol, colGet)

	if getCol.Result().StatusCode != http.StatusNoContent {
		t.Errorf("Failed")
	}
}

// tests posting to the top-level collection in a database
func TestPostTopDoc(t *testing.T) {
	handler, _ := setup("Allschema.json")
	req := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(`test db put`))
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var data = strings.NewReader(`{"key":"hello1"}`)
	docPutReq := httptest.NewRequest("POST", "/v1/db24/", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	putDoc := httptest.NewRecorder()
	handler.ServeHTTP(putDoc, docPutReq)

	if putDoc.Result().StatusCode != http.StatusCreated {
		t.Errorf("Failed")
	}

	uri := putDoc.Result().Header.Get("Location")

	getUri := uri

	getPostedDoc := httptest.NewRequest("GET", getUri, strings.NewReader(""))
	getPostedDoc.Header.Set("accept", "application/json")
	getPostedDoc.Header.Set("Authorization", "Bearer ADMIN")
	getPostedDoc.Header.Set("Content-Type", "application/json")

	getPosted := httptest.NewRecorder()
	handler.ServeHTTP(getPosted, getPostedDoc)
	if getPosted.Result().StatusCode != http.StatusOK {
		t.Errorf("Fail")
	}

}

// tests posting multiple documents
func TestPostMultipleDocs(t *testing.T) {
	handler, _ := setup("Allschema.json")
	req := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(`test db put`))
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	docPutReq := httptest.NewRequest("POST", "/v1/db24/", strings.NewReader(`{"key":"hello1"}`))
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	putDoc := httptest.NewRecorder()
	handler.ServeHTTP(putDoc, docPutReq)

	if putDoc.Result().StatusCode != http.StatusCreated {
		t.Errorf("Failed")
	}
	docPutReq = httptest.NewRequest("POST", "/v1/db24/", strings.NewReader(`{"key":"hello1"}`))
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	putDoc = httptest.NewRecorder()

	handler.ServeHTTP(putDoc, docPutReq)
	if putDoc.Result().StatusCode != http.StatusCreated {
		t.Errorf("Failed")
	}
	docPutReq = httptest.NewRequest("POST", "/v1/db24/", strings.NewReader(`{"key":"hello1"}`))
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	putDoc = httptest.NewRecorder()
	handler.ServeHTTP(putDoc, docPutReq)
	if putDoc.Result().StatusCode != http.StatusCreated {
		t.Errorf("Failed")
	}
	docPutReq = httptest.NewRequest("POST", "/v1/db24/", strings.NewReader(`{"key":"hello1"}`))
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	putDoc = httptest.NewRecorder()
	handler.ServeHTTP(putDoc, docPutReq)
	if putDoc.Result().StatusCode != http.StatusCreated {
		t.Errorf("Failed")
	}

}

// tests posting to the top-level database
func TestPostTop(t *testing.T) {
	handler, _ := setup("Allschema.json")
	req := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(`test db put`))
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var data = strings.NewReader(`{"key":"hello1"}`)
	docPostReq := httptest.NewRequest("POST", "/v1/db24/", data)
	docPostReq.Header.Set("accept", "application/json")
	docPostReq.Header.Set("Authorization", "Bearer ADMIN")
	docPostReq.Header.Set("Content-Type", "application/json")
	postDoc := httptest.NewRecorder()
	handler.ServeHTTP(postDoc, docPostReq)
	if postDoc.Result().StatusCode != http.StatusCreated {
		t.Errorf("Failed")
	}
}

// tests that an unauthorized post failed
func TestPostNoAuth(t *testing.T) {
	handler, _ := setup("Allschema.json")
	req := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(`test db put`))
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var data = strings.NewReader(`{"key":"hello1"}`)
	docPostReq := httptest.NewRequest("POST", "/v1/db24/", data)
	docPostReq.Header.Set("accept", "application/json")

	docPostReq.Header.Set("Content-Type", "application/json")
	postDoc := httptest.NewRecorder()
	handler.ServeHTTP(postDoc, docPostReq)

	if postDoc.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("Failed")
	}

	data = strings.NewReader(`{"key":"hello1"}`)

	docPutReq := httptest.NewRequest("PUT", "/v1/db24/b", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	putDoc := httptest.NewRecorder()
	handler.ServeHTTP(putDoc, docPutReq)

	if putDoc.Result().StatusCode != http.StatusCreated {
		t.Errorf("Failed")
	}

	colPutReq := httptest.NewRequest("PUT", "/v1/db24/b/c/", data)
	colPutReq.Header.Set("accept", "application/json")
	colPutReq.Header.Set("Authorization", "Bearer ADMIN")
	colPutReq.Header.Set("Content-Type", "application/json")
	putCol := httptest.NewRecorder()
	handler.ServeHTTP(putCol, colPutReq)
	data = strings.NewReader(`{"key":"hello1"}`)
	docPostReq = httptest.NewRequest("POST", "/v1/db24/b/c/", data)
	docPostReq.Header.Set("accept", "application/json")

	docPostReq.Header.Set("Content-Type", "application/json")
	postDoc = httptest.NewRecorder()
	handler.ServeHTTP(postDoc, docPostReq)

	if postDoc.Result().StatusCode != http.StatusUnauthorized {
		fmt.Printf("%d", postDoc.Result().StatusCode)
	}

}

// tests that a nested collection can be posted to
func TestPostNestedCollection(t *testing.T) {
	handler, _ := setup("Allschema.json")
	req := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(`test db put`))
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var data = strings.NewReader(`{"key":"hello1"}`)
	docPutReq := httptest.NewRequest("PUT", "/v1/db24/b", data)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	putDoc := httptest.NewRecorder()
	handler.ServeHTTP(putDoc, docPutReq)

	colPutReq := httptest.NewRequest("PUT", "/v1/db24/b/c/", data)
	colPutReq.Header.Set("accept", "application/json")
	colPutReq.Header.Set("Authorization", "Bearer ADMIN")
	colPutReq.Header.Set("Content-Type", "application/json")
	putCol := httptest.NewRecorder()
	handler.ServeHTTP(putCol, colPutReq)

	data = strings.NewReader(`{"key":"hello1"}`)
	docPostReq := httptest.NewRequest("POST", "/v1/db24/b/c/", data)
	docPostReq.Header.Set("accept", "application/json")
	docPostReq.Header.Set("Authorization", "Bearer ADMIN")
	docPostReq.Header.Set("Content-Type", "application/json")
	postDoc := httptest.NewRecorder()
	handler.ServeHTTP(postDoc, docPostReq)

	if postDoc.Result().StatusCode != http.StatusCreated {
		t.Errorf("Failed")
	}

	uri := postDoc.Result().Header.Get("Location")

	getPostedDoc := httptest.NewRequest("GET", uri, strings.NewReader(""))
	getPostedDoc.Header.Set("accept", "application/json")
	getPostedDoc.Header.Set("Authorization", "Bearer ADMIN")
	getPostedDoc.Header.Set("Content-Type", "application/json")

	getPosted := httptest.NewRecorder()
	handler.ServeHTTP(getPosted, getPostedDoc)

	if getPosted.Result().StatusCode != http.StatusOK {
		t.Errorf("Failed")
	}
}

// tests correct login
func TestAuthLogin(t *testing.T) {
	handler, _ := setup("Allschema.json")
	data := strings.NewReader(`{ "username":"TestUser" }`)
	loginReq := httptest.NewRequest("POST", "/auth", data)
	loginReq.Header.Set("accept", "application/json")
	loginReq.Header.Set("Content-Type", "application/json")

	r2 := httptest.NewRecorder()
	handler.ServeHTTP(r2, loginReq)
	if r2.Result().StatusCode != http.StatusOK {
		t.Errorf("TestAuthLogin failed")
	}
}

type tokenResp struct {
	Token string `json:"token"`
}

// tests correct logout
func TestAuthLogout(t *testing.T) {
	handler, _ := setup("Allschema.json")
	data := strings.NewReader(`{ "username":"TestUser" }`)
	loginReq := httptest.NewRequest("POST", "/auth", data)
	loginReq.Header.Set("accept", "application/json")
	loginReq.Header.Set("Content-Type", "application/json")

	r2 := httptest.NewRecorder()
	handler.ServeHTTP(r2, loginReq)
	if r2.Result().StatusCode != http.StatusOK {
		t.Errorf("TestAuthLogout failed")
	}
	loginBody, err := io.ReadAll(r2.Result().Body)
	if err != nil {
		t.Errorf("TestAuthLogout failed")
	}
	var tokenR tokenResp
	err = json.Unmarshal(loginBody, &tokenR)
	if err != nil {
		t.Errorf("TestAuthLogout failed")
	}
	token := tokenR.Token

	bearerHeader := "Bearer " + token

	logoutReq := httptest.NewRequest("DELETE", "/auth", strings.NewReader(""))

	logoutReq.Header.Set("accept", "*/*")
	logoutReq.Header.Set("Authorization", bearerHeader)

	logoutListener := httptest.NewRecorder()

	handler.ServeHTTP(logoutListener, logoutReq)

	if logoutListener.Result().StatusCode != http.StatusNoContent {
		t.Errorf("Logout failed")
	}

}

// tests an attempt to logout with the same token twice
func TestAuthDoubleLogout(t *testing.T) {
	handler, _ := setup("Allschema.json")
	data := strings.NewReader(`{ "username":"TestUser" }`)
	loginReq := httptest.NewRequest("POST", "/auth", data)
	loginReq.Header.Set("accept", "application/json")
	loginReq.Header.Set("Content-Type", "application/json")

	r2 := httptest.NewRecorder()
	handler.ServeHTTP(r2, loginReq)
	if r2.Result().StatusCode != http.StatusOK {
		t.Errorf("TestAuthLogout failed")
	}
	loginBody, err := io.ReadAll(r2.Result().Body)
	if err != nil {
		t.Errorf("TestAuthLogout failed")
	}
	var tokenR tokenResp
	err = json.Unmarshal(loginBody, &tokenR)
	if err != nil {
		t.Errorf("TestAuthLogout failed")
	}
	token := tokenR.Token

	bearerHeader := "Bearer " + token

	logoutReq := httptest.NewRequest("DELETE", "/auth", strings.NewReader(""))

	logoutReq.Header.Set("accept", "*/*")
	logoutReq.Header.Set("Authorization", bearerHeader)

	logoutListener := httptest.NewRecorder()

	handler.ServeHTTP(logoutListener, logoutReq)

	logoutReq = httptest.NewRequest("DELETE", "/auth", strings.NewReader(""))

	logoutReq.Header.Set("accept", "*/*")
	logoutReq.Header.Set("Authorization", bearerHeader)

	logoutListener = httptest.NewRecorder()

	handler.ServeHTTP(logoutListener, logoutReq)

	if logoutListener.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("Double logout failed, got stat code %d", logoutListener.Result().StatusCode)
	}

}

// tests malformed collection uri
func TestBadColUri(t *testing.T) {

	handler, _ := setup("Allschema.json")

	req := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(`test db put`))
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	badPutCol := httptest.NewRequest("PUT", "/v1/db24/doc1/col2/col3/", strings.NewReader(""))

	badPutCol.Header.Set("accept", "application/json")
	badPutCol.Header.Set("Authorization", "Bearer ADMIN")
	badPutRecorder := httptest.NewRecorder()
	handler.ServeHTTP(badPutRecorder, badPutCol)

	if badPutRecorder.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("BadColUri failed, got %d stat code", badPutRecorder.Result().StatusCode)
	}

}

// tests malformed document uri
func TestBadDocUri(t *testing.T) {
	handler, _ := setup("Allschema.json")

	req := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(`test db put`))
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	badPutCol := httptest.NewRequest("PUT", "/v1/db24/doc1/col2", strings.NewReader(""))

	badPutCol.Header.Set("accept", "application/json")
	badPutCol.Header.Set("Authorization", "Bearer ADMIN")
	badPutRecorder := httptest.NewRecorder()
	handler.ServeHTTP(badPutRecorder, badPutCol)

	if badPutRecorder.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("BadColUri failed, got %d stat code", badPutRecorder.Result().StatusCode)
	}
	fmt.Printf("\n")
}

// tests putting a correct, complex document, and an incorrect document into the database
func TestObjectSchema(t *testing.T) {
	handler, err := setup("complexSchema1.json")
	if err != nil {
		t.Errorf("schema failed to compile")
	}
	req := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(`test db put`))
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	docContents := strings.NewReader(`{"groceries": ["fruit",43,["Tony Soprano"],{"key":"value"}]}`)

	goodDocPut := httptest.NewRequest("PUT", "/v1/db24/doc1", docContents)
	goodDocPut.Header.Set("accept", "application/json")
	goodDocPut.Header.Set("Authorization", "Bearer ADMIN")
	goodDocPut.Header.Set("Content-Type", "application/json")

	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, goodDocPut)

	if w2.Result().StatusCode != http.StatusCreated {

		t.Errorf("Fail")
	}

	badDocContents := strings.NewReader(`{"christopher moltisanti":"I must be loyle to my capo"}`)

	badDocPut := httptest.NewRequest("PUT", "/v1/db24/doc2", badDocContents)

	badDocPut.Header.Set("accept", "application/json")
	badDocPut.Header.Set("Authorization", "Bearer ADMIN")
	badDocPut.Header.Set("Content-Type", "application/json")

	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, badDocPut)

	if w3.Result().StatusCode != http.StatusBadRequest {

		t.Errorf("Fail")
	}
}

// tests that attempting to access a forbidden method correctly returns a 405
func TestForbiddenMethod(t *testing.T) {
	handler, err := setup("complexSchema1.json")
	if err != nil {
		t.Errorf("schema failed to compile")
	}
	req := httptest.NewRequest("CONNECT", "/v1/db24", strings.NewReader(`test db put`))
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("TestForbiddenFailed,got %d", w.Result().StatusCode)
	}
}

func TestPutNestedOverWrite(t *testing.T) {

}

// SUBSCRIPTION TESTS

//func TestSubscribeDB(t *testing.T) {
//	handler, err := setup("Allschema.json")
//
//	if err != nil {
//		t.Errorf("TestSubscribeDB failed, schema did not compile")
//	}
//	ts := httptest.NewServer(handler)
//	defer ts.Close()
//
//	body := strings.NewReader("")
//	rq, err := http.NewRequest("PUT", ts.URL+"/v1/db24", body)
//	rq.Header.Set("accept", "application/json")
//	rq.Header.Set("Authorization", "Bearer ADMIN")
//
//	if err != nil {
//		t.Errorf("SSE test failed, bad http request")
//	}
//	//Docbody := strings.NewReader(`{"key":"value"}`)
//	client := &http.Client{}
//	_, err = client.Do(rq)
//	if err != nil {
//		t.Errorf("SSE test failed, bad request")
//	}
//
//	sseRecorder := make(chan []byte)
//	doneChannel := make(chan struct{})
//	var wg sync.WaitGroup
//	wg.Add(1)
//	go func() {
//		defer wg.Done()
//		subReq, err2 := http.NewRequest("GET", ts.URL+"/v1/db24/?mode=subscribe", strings.NewReader(""))
//
//		if err2 != nil {
//			t.Errorf("SSE DB failed, bad GET")
//		}
//		subReq.Header.Set("accept", "application/json")
//		subReq.Header.Set("Authorization", "Bearer ADMIN")
//		getClient := &http.Client{}
//		resp, err3 := getClient.Do(subReq)
//
//		if err3 != nil {
//			t.Errorf("SubscribeDB failed, bad get")
//		}
//
//		scanner := bufio.NewScanner(resp.Body)
//		ct := 0
//
//		for scanner.Scan() {
//			line := scanner.Bytes()
//			ct += 1
//			slog.Info(fmt.Sprintf(string(line)))
//			sseRecorder <- line
//			if ct == 4 {
//				break
//			}
//
//		}
//
//		err := resp.Body.Close()
//		slog.Info("DONEEEEEEEEEEEEE\n")
//		if err != nil {
//
//			return
//		}
//
//	}()
//
//	var wg2 sync.WaitGroup
//	wg2.Add(1)
//
//	go func() {
//		defer wg2.Done()
//		body2 := strings.NewReader(`{"key":"value"}`)
//		rq1, err := http.NewRequest("PUT", ts.URL+"/v1/db24/b", body2)
//		rq1.Header.Set("accept", "application/json")
//		rq1.Header.Set("Authorization", "Bearer ADMIN")
//		if err != nil {
//			t.Errorf("SSE DB test failed, bad put")
//		}
//		c2 := &http.Client{}
//		resp, err := c2.Do(rq1)
//		if err != nil {
//			fmt.Printf(err.Error())
//			t.Errorf("SSE DB test failed, bad put")
//		}
//		resp.Body.Close()
//
//		fmt.Printf("exiting goroutine 2\n")
//		close(doneChannel)
//		return
//	}()
//
//	var res [][]byte
//	go func() {
//
//		for {
//			select {
//			case event := <-sseRecorder:
//
//				res = append(res, event)
//
//			case <-doneChannel:
//
//				break
//			}
//		}
//
//	}()
//	wg.Wait()
//	wg2.Wait()
//	return
//
//}
//

// Patch Test
func TestPatchVisitorArrayAdd(t *testing.T) {
	handler, err := setup("Allschema.json")
	if err != nil {
		t.Errorf("Failed: got error %s", err.Error())
	}

	// create the database
	putReq := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(`test db put`))
	putReq.Header.Set("accept", "application/json")
	putReq.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, putReq)

	// create a document with an empty array "friends"
	var initialData = strings.NewReader(`{"friends": []}`)
	docPutReq := httptest.NewRequest("PUT", "/v1/db24/doc1", initialData)
	docPutReq.Header.Set("accept", "application/json")
	docPutReq.Header.Set("Authorization", "Bearer ADMIN")
	docPutReq.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, docPutReq)

	// Now apply a patch
	patchData := strings.NewReader(`[
       {
           "op": "ArrayAdd",
           "path": "/friends",
           "value": "neyida"
       }
   ]`)
	patchReq := httptest.NewRequest("PATCH", "/v1/db24/doc1", patchData)
	patchReq.Header.Set("accept", "application/json")
	patchReq.Header.Set("Authorization", "Bearer ADMIN")
	patchReq.Header.Set("Content-Type", "application/json-patch+json")
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, patchReq)

	// Check the result of the patch
	if w3.Result().StatusCode != http.StatusOK {
		t.Errorf("Expected PATCH to succeed, got status code %d", w3.Result().StatusCode)
	}

	// Get the updated document to verify the patch was applied
	getReq := httptest.NewRequest("GET", "/v1/db24/doc1", nil)
	getReq.Header.Set("accept", "application/json")
	getReq.Header.Set("Authorization", "Bearer ADMIN")
	w4 := httptest.NewRecorder()
	handler.ServeHTTP(w4, getReq)

	var result map[string]interface{}
	err = json.Unmarshal(w4.Body.Bytes(), &result)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	slog.Info("patch resutl", slog.Any("result", result))

	// Debug print
	fmt.Printf("Resulting document: %v\n", result)

	if doc, ok := result["doc"].(map[string]interface{}); ok {
		expected := []interface{}{"neyida"}
		if !reflect.DeepEqual(doc["friends"], expected) {
			t.Errorf("Expected 'friends' to be %v, but got %v", expected, doc["friends"])
		}
	} else {
		t.Errorf("Doc object not found in result")
	}
}
