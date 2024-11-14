// Package server Description: This file contains the implementation of the HTTP server that serves the API.
package server

import (
	"log/slog"
	"net/http"
)

type resourceCreator interface {
	PostDoc(dbName string, colpath string, user string, payload []byte) ([]byte, int, string)
	PutDoc(dbName string, docpath string, docname string, payload []byte, overwrite bool, user string) ([]byte, int, string)
	PutCol(dtb string, colpath string) ([]byte, int, string)
	CreateDB(dbName string) ([]byte, int, string)
}

type resourceGetter interface {
	GetDoc(dtb string, pathstr string, subscription bool) (response []byte, statCode int, subCh *chan []byte, id string, docEvent []byte)

	GetCol(dtb string, colpath string, lower string, upper string, mode bool) (payload []byte, statCode int, subChan *chan []byte, subId string, docEvents [][]byte)
}

type resourceDeleter interface {
	DeleteCol(dtb string, colpath string) ([]byte, int)
	DeleteDoc(dbName string, docpath string) ([]byte, int)
	DeleteDB(dbName string) ([]byte, int)
}

type resourcePatcher interface {
	PatchDoc(dtb string, docpath string, patches []byte, user string) ([]byte, int, string)
}

// DbHarness serves as a structure that exposes the resource services to HTTP endpoints
type DbHarness struct {
	rg   resourceGetter  //rg manages all requests to GET resources
	rd   resourceDeleter //rd manages all requests to DELETE resources
	rc   resourceCreator //rc manages all requests to PUT & POST resources
	rp   resourcePatcher //rp manages all requests to PATCH resources
	auth Authorizer      //auth manages all authorization mechanisms
}

// Authorizer encapsulates the necessary functionalities for authentication
type Authorizer interface {
	CreateSession(username string) (string, error) //creates a session
	ValidateSession(token string) (string, error)  //validates a session
	Login(username string) (string, error)         //logs in
	Logout(token string) (bool, error)             //logs out
}

// New creates a new HTTP server, taking a resourceDeleter, resourceGetter, and resourceCreator
func New(rd resourceDeleter, rg resourceGetter, rc resourceCreator, auth Authorizer, rp resourcePatcher) http.Handler {

	dbharness := DbHarness{
		rg:   rg,
		rd:   rd,
		rc:   rc,
		rp:   rp,
		auth: auth,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("PUT /v1/{resource...}", dbharness.putHandler)
	mux.HandleFunc("GET /v1/{resource...}", dbharness.getHandler)
	mux.HandleFunc("POST /v1/{resource...}", dbharness.postDocHandler)
	mux.HandleFunc("PATCH /v1/{resource...}", dbharness.patchDocHanlder)
	mux.HandleFunc("DELETE /v1/{resource...}", dbharness.deleteHandler)

	mux.HandleFunc("OPTIONS /auth", authOptionsHandler)
	mux.HandleFunc("OPTIONS /v1/", optionsHandler)
	mux.HandleFunc("POST /auth", dbharness.loginhandler)
	mux.HandleFunc("DELETE /auth", dbharness.logoutHandler)

	return requestPreprocessor(mux)
}

// optionsHandler is needed to handle preflighted requests; the swagger testing thing
// sends an OPTIONS request before anything else
func optionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Allow", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	w.WriteHeader(http.StatusOK)
}

// defaultOptionsHandler handles all OPTIONS requests to invalid URLs. Since no methods
// are allowed on invalid requests, no allowed methods are returned
func defaultOptionsHandler(w http.ResponseWriter, r *http.Request) {
	slog.Debug("hello from default options")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Allow", "")
	w.Header().Set("Access-Control-Allow-Methods", "")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.WriteHeader(http.StatusOK)
}

// authOptionsHandler handles OPTIONS requests to the auth endpoint.
func authOptionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Allow", "POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Methods", "POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	w.WriteHeader(http.StatusOK)
}
