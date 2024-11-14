package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
)

// writeResponse writes a response with appropriate headers
func writeResponse(w http.ResponseWriter, code int, response []byte) {
	if code != http.StatusNoContent {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(code)
		w.Write(response)
	} else {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(code)
	}

}

// parses out bearer token
func extractToken(header http.Header) (string, error) {
	authHeader := header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("authorization header missing")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return "", errors.New("invalid authorization token format")
	}

	return token, nil
}

// validates Bounds for query params
func validateBounds(param string) bool {
	pattern := "^(\\[|\\()[^[\\]()]*,[^[\\]()]*(\\]|\\))$"
	match, err := regexp.MatchString(pattern, param)
	if err != nil {
		return false
	}
	if match {
		return true
	}
	return false
}

// validateSubscribe validates the subscribe query field
func validateSubscribe(param string) bool {
	pattern := "^(no)?subscribe$"
	match, err := regexp.MatchString(pattern, param)
	if err != nil {
		return false
	}
	if match {
		return true
	}
	return false
}

// validateOverwrite checks that overwrite is a correct
func validateOverwrite(param string) bool {
	pattern := "^(no)?overwrite$"
	match, err := regexp.MatchString(pattern, param)

	if err != nil {
		return false
	}
	if !match {
		return false
	}
	return true
}

// validateColPath validates the collection path
func validateColPath(colpath string) error {
	if colpath == "" {
		return nil
	}
	c2 := strings.TrimSuffix(colpath, "/")

	splitPath := strings.Split(c2, "/")

	if len(splitPath)%2 != 0 {
		return fmt.Errorf("bad resource path")
	}
	return nil
}

// requestPreprocessor preprocesses requests, ensuring that no double slashes occur
func requestPreprocessor(mx http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			if strings.HasPrefix(r.URL.Path, "/v1/") {
				optionsHandler(w, r)
			} else if r.URL.Path == "/auth" {
				authOptionsHandler(w, r)
			} else {
				defaultOptionsHandler(w, r)
			}
			return
		}

		if strings.Contains(r.URL.Path, "//") {
			errmsg, err := json.Marshal("Bad Uri: contains //")
			if err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			writeResponse(w, http.StatusBadRequest, errmsg)
			return
		}

		mx.ServeHTTP(w, r)
	})
}

// validatePutColPath validates paths for put requests for collections
func validatePutColPath(path string) error {
	if path == "" {
		return fmt.Errorf("Bad Uri")
	}
	c2 := strings.TrimSuffix(path, "/")
	sp := strings.Split(c2, "/")
	if len(sp)%2 != 0 {
		return fmt.Errorf("Bad Uri")

	}
	return nil
}

// parseResourcePath parses the resource path
func parseResourcePath(path string) (database string, resourcePath string) {
	splitPath := strings.Split(path, "/")
	if len(splitPath) == 1 {
		return splitPath[0], ""
	}
	resourcePath = strings.TrimPrefix(path, splitPath[0]+"/")
	return splitPath[0], resourcePath
}

// validateDocPath validates that the path is to a valid document
func validateDocPath(docpath string) error {
	if docpath == "" {
		return fmt.Errorf("bad resource path")
	}
	splitPath := strings.Split(docpath, "/")
	if len(splitPath)%2 != 1 {
		return fmt.Errorf("bad resource path")
	}
	return nil
}

// parseBounds is an internal helper routine to parse the interval parameter. Returns the upper and lower bounds
func parseBounds(rawBounds string) (string, string) {
	if rawBounds == "[,]" || rawBounds == "" {
		return string(rune(0)), string(rune(127))
	}
	rawBounds = strings.Trim(rawBounds, "[]")

	splitStr := strings.Split(rawBounds, ",")
	var lower string
	var upper string
	if splitStr[0] == "" {
		slog.Debug("no lower bound was specified,so we set to lower min")
		lower = string(rune(0))
	} else {
		lower = splitStr[0]
	}
	if splitStr[1] == "" {
		slog.Debug("no upper bound was specified, so we set to max")
		upper = string(rune(127))
	} else {
		upper = splitStr[1]
	}

	return lower, upper
}

// internal helper routines to validate inputs
// validateUrl is an internal routine that determines whether the uri is a valid uri.
func validateUrl(url string) error {
	if strings.Contains(url, "//") {
		return fmt.Errorf("malformed uri: //")
	}
	return nil
}
