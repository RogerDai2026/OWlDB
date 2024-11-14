package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
)

// loginHandler processes user login requests.
// It reads the JSON body containing a username and returns an authentication token upon successful login.
func (dbh *DbHarness) loginhandler(w http.ResponseWriter, r *http.Request) {
	//read body from request
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		errmsg, _ := json.Marshal(err.Error())
		writeResponse(w, http.StatusBadRequest, errmsg)
		return

	}
	//constructing data
	var data struct {
		Username string `json:"username"`
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		errmsg, _ := json.Marshal(err.Error())
		writeResponse(w, http.StatusBadRequest, errmsg)
		return
	}
	if data.Username == "" {
		errmsg, _ := json.Marshal("No username found")
		writeResponse(w, http.StatusBadRequest, errmsg)
		return
	}
	token, loginerr := dbh.auth.Login(data.Username)
	if loginerr != nil {
		errmsg, _ := json.Marshal(loginerr.Error())
		writeResponse(w, http.StatusBadRequest, errmsg)
		return
	}
	response, _ := json.Marshal(struct {
		Token string `json:"token"`
	}{token})

	writeResponse(w, http.StatusOK, response)
}

// logoutHandler processes user logout requests.
// It reads the Authorization header to extract the bearer token and logs the user out if the token is valid.
func (dbh *DbHarness) logoutHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	slog.Info("Delete Handler - Checking Authorization Header")

	// Check if the Authorization header is present
	slog.Info("Checking Authorization Header", "header", r.Header)

	// Extract token from the Authorization header
	token, err := extractToken(r.Header)
	if err != nil {
		slog.Warn("Authorization header missing or invalid", "error", err)
		errmsg, _ := json.Marshal(err.Error())
		writeResponse(w, http.StatusUnauthorized, errmsg)
		return
	}
	slog.Debug("trying to logout?")
	slog.Info("Authorization Header Received", "token", token)

	// Perform logout
	loggedOut, err := dbh.auth.Logout(token)
	if err != nil || !loggedOut {
		slog.Warn("Logout failed", "error", err)
		errmsg, _ := json.Marshal("Missing or invalid bearer token")
		writeResponse(w, http.StatusUnauthorized, errmsg)
		return
	}

	slog.Info("Successfully logged out", "token", token)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusNoContent)
}
