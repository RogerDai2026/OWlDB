package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// a writeFlusher is the composition of an http.ResponseWriter and the http.Flusher. Needed for SSEs
type writeFlusher interface {
	http.ResponseWriter
	http.Flusher
}

// getHandler dispatches get requests depending on whether they end with a /
func (dbh *DbHarness) getHandler(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	if len(resource) == 0 {
		emsg, _ := json.Marshal("Bad Resource Path")
		writeResponse(w, http.StatusBadRequest, emsg)
		return
	}
	splitPath := strings.Split(resource, "/")
	if len(splitPath) == 1 {
		dbh.getDocHandler(w, r)
		return
	}
	if strings.HasSuffix(resource, "/") {
		dbh.getColHandler(w, r)
	} else {
		dbh.getDocHandler(w, r)
	}
}

// getDocHandler handles get requests made to documents
func (dbh *DbHarness) getDocHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	patherr := validateUrl(r.URL.Path)

	if patherr != nil {
		errmsg, _ := json.Marshal(patherr.Error())
		writeResponse(w, http.StatusBadRequest, errmsg)
		return
	}
	token, err := extractToken(r.Header)
	if err != nil {
		ermsg, _ := json.Marshal("Missing or invalid bearer token")
		writeResponse(w, http.StatusUnauthorized, ermsg)
		return
	}

	_, autherr := dbh.auth.ValidateSession(token)
	if autherr != nil {
		errmsg, _ := json.Marshal(autherr.Error())
		writeResponse(w, http.StatusUnauthorized, errmsg)
		return
	}
	qs := r.URL.Query()
	mode := qs.Get("mode")
	subscribe := (mode == "subscribe")
	if mode != "" {
		if !validateSubscribe(mode) {
			emsg, _ := json.Marshal("malformed subscribe parameter")
			writeResponse(w, http.StatusBadRequest, emsg)
			return
		}
	}

	path := r.PathValue("resource")
	dtb, docpath := parseResourcePath(path)
	err = validateDocPath(docpath)
	if err != nil {
		errmsg, e := json.Marshal(err.Error())
		if e != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		writeResponse(w, http.StatusBadRequest, errmsg)
		return
	}
	response, status, subChan, _, docEv := dbh.rg.GetDoc(dtb, docpath, subscribe)
	if status != http.StatusOK {
		writeResponse(w, status, response)
		return
	}
	if subscribe && status == http.StatusOK {
		wf, ok := w.(writeFlusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		wf.Header().Set("Content-Type", "text/event-stream")
		wf.Header().Set("Cache-Control", "no-cache")
		wf.Header().Set("Connection", "keep-alive")
		wf.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Last-Event-ID")
		wf.Header().Set("Access-Control-Allow-Origin", "*")
		wf.WriteHeader(http.StatusOK)
		wf.Flush()
		var evt bytes.Buffer
		evt.WriteString(string(docEv))

		slog.Info("Sending", "msg", evt.String())

		// Send event
		wf.Write(evt.Bytes())
		wf.Flush()
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				var evt bytes.Buffer
				evt.WriteString(":keep-alive\n\n")
				wf.Write(evt.Bytes())
				wf.Flush()
			case <-r.Context().Done():

				continue
			case event := <-*subChan:
				var evt bytes.Buffer
				evt.WriteString(string(event))

				slog.Info("Sending", "msg", evt.String())

				// Send event
				wf.Write(evt.Bytes())
				wf.Flush()
			}

		}
	}
	writeResponse(w, status, response)
}

// getColHandler handles requests to collections
func (dbh *DbHarness) getColHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	slog.Info(fmt.Sprintf("Request headers: %+v", r.Context()))
	patherr := validateUrl(r.URL.Path)

	if patherr != nil {
		http.Error(w, patherr.Error(), http.StatusBadRequest)
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
		return
	}
	_, autherr := dbh.auth.ValidateSession(token)
	if autherr != nil {
		errmsg, _ := json.Marshal(autherr.Error())
		writeResponse(w, http.StatusUnauthorized, errmsg)
		return
	}
	path := r.PathValue("resource")
	dtb, colpath := parseResourcePath(path)

	err = validateColPath(colpath)
	if err != nil {
		errmsg, e := json.Marshal(err.Error())
		if e != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		writeResponse(w, http.StatusBadRequest, errmsg)
		return
	}
	qs := r.URL.Query()
	mode := qs.Get("mode")
	subscribe := (mode == "subscribe")
	if mode != "" {
		if !validateSubscribe(mode) {
			errmsg, e := json.Marshal("Malformed subscribe param")
			if e != nil {
				http.Error(w, e.Error(), http.StatusInternalServerError)
				return
			}
			writeResponse(w, http.StatusBadRequest, errmsg)

			return
		}
	}

	bounds := qs.Get("interval")
	if bounds != "" {
		if !validateBounds(bounds) {
			errmsg, _ := json.Marshal("Malformed interval param")
			writeResponse(w, http.StatusBadRequest, errmsg)

			return
		}
	}

	lower, upper := parseBounds(bounds)
	slog.Debug(fmt.Sprintf("These are the bounds received: %s lower, %s upper", lower, upper))
	b, status, subChan, _, docEvents := dbh.rg.GetCol(dtb, colpath, lower, upper, subscribe)
	slog.Debug(fmt.Sprintf("%d", status))
	if subscribe && status == http.StatusOK { //subscription request
		wf, ok := w.(writeFlusher)
		slog.Debug(fmt.Sprintf("Beginning sse connection, payload is %s", string(b)))
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		wf.Header().Set("Content-Type", "text/event-stream")
		wf.Header().Set("Cache-Control", "no-cache")
		wf.Header().Set("Connection", "keep-alive")
		wf.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Last-Event-ID")
		wf.Header().Set("Access-Control-Allow-Origin", "*")
		wf.WriteHeader(http.StatusOK)
		wf.Flush()
		ticker := time.NewTicker(15 * time.Second) //keep-alive comments
		defer ticker.Stop()
		for _, ev := range docEvents { //writing each document as an SSE
			var evt bytes.Buffer
			evt.WriteString(string(ev))

			slog.Info("Sending", "msg", evt.String())

			// Send event
			wf.Write(evt.Bytes())
			wf.Flush()
		}
		for {
			select {

			case <-ticker.C: //send a comment
				var evt bytes.Buffer
				evt.WriteString(":keep-alive\n\n")
				wf.Write(evt.Bytes())
				wf.Flush()

			case <-r.Context().Done():

				continue
			case event := <-*subChan:
				var evt bytes.Buffer
				evt.WriteString(string(event))

				slog.Info("Sending", "msg", evt.String())

				// Send event
				wf.Write(evt.Bytes())
				wf.Flush()
			}

		}

	} else { //not a subscription request
		writeResponse(w, status, b)
	}

}
