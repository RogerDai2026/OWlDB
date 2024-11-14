package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// putHandler dispatches PUT requests based on the end of the path;
// a terminating slash will redirect to collections, and documents otherwise
func (dbh *DbHarness) putHandler(w http.ResponseWriter, r *http.Request) {

	resource := r.PathValue("resource")
	if len(resource) == 0 {
		emsg, _ := json.Marshal("Bad Resource Path")
		writeResponse(w, http.StatusBadRequest, emsg)
		return
	}
	splitPath := strings.Split(resource, "/")
	if len(splitPath) == 1 {
		dbh.createDBHandler(w, r)
		return
	}
	if strings.HasSuffix(resource, "/") {
		dbh.putColHandler(w, r)
	} else {
		dbh.putDocHandler(w, r)
	}
}

// createDBHandler handles requests to create databases.
// Validates the URL path and the bearer token from the Authorization header. If valid, it creates the database.
func (dbh *DbHarness) createDBHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	patherr := validateUrl(r.URL.Path) //validates path

	if patherr != nil {
		errmsg, _ := json.Marshal(patherr.Error())
		writeResponse(w, http.StatusBadRequest, errmsg)
		return
	}
	token, err := extractToken(r.Header)
	if err != nil {
		emg, e := json.Marshal("Invalid or Expired Bearer token")
		if e != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		writeResponse(w, http.StatusUnauthorized, emg)
	}

	_, autherr := dbh.auth.ValidateSession(token)
	if autherr != nil {
		errmsg, _ := json.Marshal(autherr.Error())
		writeResponse(w, http.StatusUnauthorized, errmsg)
		return
	}

	dbName := r.PathValue("resource")
	if dbName == "" {
		errmsg, _ := json.Marshal("bad resource path")
		writeResponse(w, http.StatusBadRequest, errmsg)
		return
	}
	response, status, uri := dbh.rc.CreateDB(dbName)
	if status == http.StatusCreated {
		w.Header().Set("Location", uri)
		writeResponse(w, status, response)
		return
	}
	writeResponse(w, status, response)
}

// putDocHandler handles requests for creating or updating documents in a database.
// Validates the document path, JSON body, and the bearer token before proceeding with the operation.
func (dbh *DbHarness) putDocHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	patherr := validateUrl(r.URL.Path)

	if patherr != nil {
		errmsg, e := json.Marshal(patherr.Error())
		if e != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		writeResponse(w, http.StatusBadRequest, errmsg)
		return
	}
	//parse database name
	resourcePath := r.PathValue("resource")
	dbName, docPath := parseResourcePath(resourcePath)

	err := validateDocPath(docPath)
	if err != nil {
		errmsg, _ := json.Marshal(err.Error())
		writeResponse(w, http.StatusBadRequest, errmsg)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error: unable to read body", http.StatusBadRequest)
		return
	}
	ok := json.Valid(body)
	if !ok {
		errmsg, _ := json.Marshal("Malformed json object")
		writeResponse(w, http.StatusBadRequest, errmsg)
		return
	}
	splitPath := strings.Split(docPath, "/")
	params := r.URL.Query()
	isOverwrite := params.Get("mode")
	if isOverwrite != "" {
		if !validateOverwrite(isOverwrite) {
			errmsg, _ := json.Marshal("Malformed overwrite parameter")
			writeResponse(w, http.StatusBadRequest, errmsg)
		}
	}

	var overwrite bool
	if isOverwrite == "nooverwrite" {
		overwrite = false
	} else {
		overwrite = true
	}

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
	slog.Debug(fmt.Sprintf("overwrite is %t", overwrite))
	user, autherr := dbh.auth.ValidateSession(token)
	if autherr != nil {
		errmsg, _ := json.Marshal(autherr.Error())
		writeResponse(w, http.StatusUnauthorized, errmsg)
		return
	}
	err = validateDocPath(docPath)
	if err != nil {
		errmsg, _ := json.Marshal(err.Error())
		writeResponse(w, http.StatusBadRequest, errmsg)
		return
	}
	response, status, uri := dbh.rc.PutDoc(dbName, docPath, splitPath[len(splitPath)-1], body, overwrite, user)
	if status == http.StatusCreated || status == http.StatusOK {
		w.Header().Set("Location", uri)
		writeResponse(w, status, response)
		return
	}
	writeResponse(w, status, response)
}

// postDocHandler handles operations related to posting new documents into a collection.
// It ensures the URL path is valid, checks the bearer token, and validates the JSON body before proceeding.
func (dbh *DbHarness) postDocHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	patherr := validateUrl(r.URL.Path)

	if patherr != nil {
		errmsg, _ := json.Marshal(patherr.Error())
		writeResponse(w, http.StatusBadRequest, errmsg)
		return
	}
	resourcePath := r.PathValue("resource")
	dbName, colpath := parseResourcePath(resourcePath)

	if !strings.HasSuffix(r.URL.Path, "/") {
		errmsg, _ := json.Marshal("Bad Resource Path")
		writeResponse(w, http.StatusBadRequest, errmsg)
		return
	}
	if e := validateColPath(colpath); e != nil {
		errmsg, _ := json.Marshal("Bad Resource Path")
		writeResponse(w, http.StatusBadRequest, errmsg)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		errmsg, _ := json.Marshal(err.Error())
		writeResponse(w, http.StatusBadRequest, errmsg)
		return
	}
	ok := json.Valid(body)
	if !ok {
		errmsg, _ := json.Marshal("Bad JSON")
		writeResponse(w, http.StatusBadRequest, errmsg)
		return
	}
	token, err := extractToken(r.Header)
	if err != nil {
		emg, e := json.Marshal("Missing or invalid bearer token")
		if e != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		writeResponse(w, http.StatusUnauthorized, emg)
	}

	user, autherr := dbh.auth.ValidateSession(token)
	if autherr != nil {

		resp, _ := json.Marshal("Missing or invalid bearer token")
		writeResponse(w, http.StatusUnauthorized, resp)
		return
	}
	resp, stat, uri := dbh.rc.PostDoc(dbName, colpath, user, body)
	if stat == http.StatusCreated {
		w.Header().Set("Location", uri)
	}
	writeResponse(w, stat, resp)

}

// putColHandler processes PUT requests for collections.
// It validates the collection path and the bearer token, then attempts to create or update the collection.
func (dbh *DbHarness) putColHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	patherr := validateUrl(r.URL.Path)

	if patherr != nil {
		errmsg, _ := json.Marshal(patherr.Error())
		writeResponse(w, http.StatusBadRequest, errmsg)
		return
	}
	resourcePath := r.PathValue("resource")
	dtb, colpath := parseResourcePath(resourcePath)

	err := validatePutColPath(colpath)
	if err != nil {
		errmsg, _ := json.Marshal(err.Error())
		writeResponse(w, http.StatusBadRequest, errmsg)
		return
	}
	token, err := extractToken(r.Header)
	if err != nil {
		emg, _ := json.Marshal("Invalid or Expired Bearer token")

		writeResponse(w, http.StatusUnauthorized, emg)
	}
	_, autherr := dbh.auth.ValidateSession(token)
	if autherr != nil {
		errmsg, _ := json.Marshal(autherr.Error())
		writeResponse(w, http.StatusUnauthorized, errmsg)
		return
	}
	// If valid, it proceeds with the collection operation.
	// Returns a 201 Created status upon success or an appropriate error status otherwise.
	resp, stat, uri := dbh.rc.PutCol(dtb, colpath)
	if stat == http.StatusCreated {
		w.Header().Set("Location", uri)
		writeResponse(w, stat, resp)
		return
	}
	writeResponse(w, stat, resp)
}

// patchDocHandler processes PATCH requests for documents.
// It validates the document path, JSON body, and the bearer token before forwarding the request to the patching service.
func (dbh *DbHarness) patchDocHanlder(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()
	// Validates the URL path, JSON body, and Authorization header.
	writer.Header().Set("Access-Control-Allow-Origin", "*")
	writer.Header().Set("Content-Type", "application/json")
	path := request.PathValue("resource")
	dbName, docPath := parseResourcePath(path)
	if dbName == "" {
		errmsg, _ := json.Marshal("Bad resource path")
		writeResponse(writer, http.StatusBadRequest, errmsg)
		return
	}
	body, err := io.ReadAll(request.Body)
	if err != nil {
		errmsg, _ := json.Marshal(err.Error())
		writeResponse(writer, http.StatusBadRequest, errmsg)
		return
	}
	ok := json.Valid(body)
	if !ok {
		errmsg, _ := json.Marshal("Not a valid json object")
		writeResponse(writer, http.StatusBadRequest, errmsg)
		return
	}
	token, err := extractToken(request.Header)
	if err != nil {
		emg, e := json.Marshal("Missing or invalid Bearer token")
		if e != nil {
			http.Error(writer, "Internal error", http.StatusInternalServerError)
			return
		}
		writeResponse(writer, http.StatusUnauthorized, emg)
		return
	}
	user, autherr := dbh.auth.ValidateSession(token)
	if autherr != nil {
		errmsg, _ := json.Marshal(autherr.Error())
		writeResponse(writer, http.StatusUnauthorized, errmsg)
		return
	}
	err = validateDocPath(docPath) //check that this is a valid document path
	if err != nil {
		errmsg, _ := json.Marshal(err.Error())
		writeResponse(writer, http.StatusBadRequest, errmsg)
		return
	}
	resp, status, uri := dbh.rp.PatchDoc(dbName, docPath, body, user)
	if status == http.StatusOK {
		writer.Header().Set("Location", uri)
	}
	writeResponse(writer, status, resp)
}
