// Package server provides the handlers for DELETE requests to OwlDB. 
// It supports deletion of databases, collections, and documents, and ensures request validation, 
// token authentication, and response formatting.
package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

// deleteHandler parses the resource path and dispatches requests accordingly 
func (dbh *DbHarness) deleteHandler(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	if len(resource) == 0 { //empty strings are not allowed
		emsg, _ := json.Marshal("Bad Resource Path")
		writeResponse(w, http.StatusBadRequest, emsg)
		return
	}
	splitPath := strings.Split(resource, "/")
	if len(splitPath) == 1 { //must be a database
		dbh.deleteDBHandler(w, r)
		return
	}
	if strings.HasSuffix(resource, "/") { //must be a collection
		dbh.deleteColHandler(w, r)
	} else {
		dbh.deleteDocHandler(w, r) //must be a document
	}
}

// deleteColHandler handles DELETE requests to remove a collection from a specified database.
// It validates the resource path, extracts the Bearer token, and checks the token's validity before performing the deletion.
func (dbh *DbHarness) deleteColHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	patherr := validateUrl(r.URL.Path)
    //validating the path
	if patherr != nil {
		errmsg, _ := json.Marshal(patherr.Error())
		writeResponse(w, http.StatusBadRequest, errmsg)
		return
	
	}
	// Extract and validate the Bearer token
	token, err := extractToken(r.Header)
	if err != nil {
		errmsg, _ := json.Marshal(err.Error())
		writeResponse(w, http.StatusUnauthorized, errmsg)
		return
	}
	_, autherr := dbh.auth.ValidateSession(token)
	if autherr != nil {
		errmsg, _ := json.Marshal(autherr.Error())
		writeResponse(w, http.StatusUnauthorized, errmsg)
		return
	}
	// Parse the resource path and validate
	resourcePath := r.PathValue("resource")
	dbName, colpath := parseResourcePath(resourcePath)
	err = validateColPath(colpath)
	if err != nil {
		errmsg, e := json.Marshal(err.Error())
		if e != nil {
			http.Error(w, e.Error(), http.StatusInternalServerError)
			return
		}
		writeResponse(w, http.StatusBadRequest, errmsg)
		return
	}
	response, status := dbh.rd.DeleteCol(dbName, colpath)
	writeResponse(w, status, response)
}

// deleteDocHandler handles DELETE requests to remove a document from a specified database and collection.
func (dbh *DbHarness) deleteDocHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	patherr := validateUrl(r.URL.Path)
 //validating the path
	if patherr != nil {
		errmsg, _ := json.Marshal(patherr.Error())
		writeResponse(w, http.StatusBadRequest, errmsg)
		return
	}
	// Extract and validate the Bearer token
	token, err := extractToken(r.Header)
	if err != nil {
		emg, e := json.Marshal("Invalid or Expired Bearer token")
		if e != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		writeResponse(w, http.StatusUnauthorized, emg)
		return
	}
	// Authenticate the token

	_, autherr := dbh.auth.ValidateSession(token)
	if autherr != nil {
		errmsg, _ := json.Marshal(autherr.Error())
		writeResponse(w, http.StatusUnauthorized, errmsg)
		return
	}
	path := r.PathValue("resource")
	dbName, docpath := parseResourcePath(path)
	err = validateDocPath(docpath)
	// Validate the document path
	if err != nil {
		errmsg, _ := json.Marshal(err.Error())
		writeResponse(w, http.StatusBadRequest, errmsg)
		return
	}
	response, status := dbh.rd.DeleteDoc(dbName, docpath)
	writeResponse(w, status, response)
}

// deleteDBHandler handles requests to delete databases
func (dbh *DbHarness) deleteDBHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	patherr := validateUrl(r.URL.Path) //validating that the path is well-formed

	if patherr != nil {
		errmsg, _ := json.Marshal(patherr.Error())
		writeResponse(w, http.StatusBadRequest, errmsg)
		return
	}

	dbName := r.PathValue("resource")
	token, err := extractToken(r.Header)
	if err != nil {
		emg, e := json.Marshal("Invalid or Expired Bearer token")
		if e != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		writeResponse(w, http.StatusUnauthorized, emg)
		return
	}
	_, autherr := dbh.auth.ValidateSession(token) //authenticating
	if autherr != nil {
		errmsg, _ := json.Marshal(autherr.Error())
		writeResponse(w, http.StatusUnauthorized, errmsg)
		return
	}

	response, status := dbh.rd.DeleteDB(dbName)

	writeResponse(w, status, response)

}
