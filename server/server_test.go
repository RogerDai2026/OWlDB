package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type mockResourceDeleter struct {
	didDeleteCol bool
	didDeleteDoc bool
	didDeleteDB  bool
}

func (m *mockResourceDeleter) DeleteCol(dtb string, colpath string) ([]byte, int) {
	m.didDeleteCol = true
	return nil, 204
}

func (m *mockResourceDeleter) DeleteDoc(dbName string, docpath string) ([]byte, int) {
	m.didDeleteDoc = true
	return nil, 204
}

func (m *mockResourceDeleter) DeleteDB(dbName string) ([]byte, int) {
	m.didDeleteDB = true
	return nil, 204
}

type mockResourceGetter struct {
	didGetDoc bool
	didGetCol bool
}

func (m *mockResourceGetter) GetDoc(dtb string, pathstr string, subscription bool) (response []byte, statCode int, subCh *chan []byte, id string, docEvent []byte) {
	m.didGetDoc = true
	if subscription {
		ch := make(chan []byte)
		return []byte("payload"), 200, &ch, "", nil
	}
	return nil, 200, nil, "", nil
}

func (m *mockResourceGetter) GetCol(dtb string, colpath string, lower string, upper string, mode bool) (payload []byte, statCode int, subChan *chan []byte, subId string, docEvents [][]byte) {
	m.didGetCol = true
	if mode {
		ch := make(chan []byte)
		return []byte("payload"), 200, &ch, "", nil
	}
	return []byte("payload"), 200, nil, "", nil

}

type mockResourcePatcher struct {
	didPatchDoc bool
}

func (m *mockResourcePatcher) PatchDoc(dtb string, docpath string, patches []byte, user string) ([]byte, int, string) {
	m.didPatchDoc = true
	return []byte("hello"), http.StatusOK, ""
}

type mockCreator struct {
	didPostDoc  bool
	didPutDoc   bool
	didPutCol   bool
	didCreateDB bool
}

func (m *mockCreator) PostDoc(dbName string, colpath string, user string, payload []byte) ([]byte, int, string) {
	m.didPostDoc = true
	return []byte("hello"), http.StatusCreated, "Posted Doc"
}

func (m *mockCreator) PutDoc(dbName string, docpath string, docname string, payload []byte, overwrite bool, user string) ([]byte, int, string) {
	m.didPutDoc = true
	if overwrite {
		//simulates a replacement
		return []byte("hello"), http.StatusOK, "Overwrite Success"
	} else if !overwrite {
		return []byte("hello"), http.StatusPreconditionFailed, "Nooverwrite Success"
	}
	return []byte("hello"), http.StatusCreated, ""
}

func (m *mockCreator) PutCol(dtb string, colpath string) ([]byte, int, string) {
	m.didPutCol = true
	return []byte("hello"), http.StatusCreated, ""
}

func (m *mockCreator) CreateDB(dbName string) ([]byte, int, string) {
	m.didCreateDB = true

	return []byte("hello"), http.StatusCreated, ""
}

type mockAuthorizer struct {
	didLogout          bool
	didCreateSession   bool
	didLogin           bool
	didValidateSession bool
}

func (m *mockAuthorizer) CreateSession(username string) (string, error) {
	m.didCreateSession = true
	return "token", nil
}

func (m *mockAuthorizer) ValidateSession(token string) (string, error) {
	m.didValidateSession = true
	return "user", nil
}

func (m *mockAuthorizer) Login(username string) (string, error) {
	m.didLogin = true
	return "token", nil
}

func (m *mockAuthorizer) Logout(token string) (bool, error) {
	m.didLogout = true
	return true, nil
}

func setup() http.Handler {
	return New(&mockResourceDeleter{}, &mockResourceGetter{}, &mockCreator{}, &mockAuthorizer{}, &mockResourcePatcher{})
}

func TestGetDoc(t *testing.T) {
	srv := setup()

	getReq := httptest.NewRequest("GET", "/v1/db24/doc1", strings.NewReader(""))
	getReq.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, getReq)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("TestGetDoc failed")
	}

}

func TestBadDocUri(t *testing.T) {
	srv := setup()
	getReq := httptest.NewRequest("GET", "/v1/db24/doc1/doc2", strings.NewReader(""))
	getReq.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, getReq)

	if w.Result().StatusCode != http.StatusBadRequest {
		fmt.Printf("statusCode")
		t.Errorf("TestGetDoc failed,got stat code %d", w.Result().StatusCode)
	}
}

func TestBadColUri(t *testing.T) {
	srv := setup()
	getReq := httptest.NewRequest("GET", "/v1/db24/doc1/doc2/col3/", strings.NewReader(""))
	getReq.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, getReq)

	if w.Result().StatusCode != http.StatusBadRequest {
		fmt.Printf("statusCode")
		t.Errorf("TestGetBadColUri failed,got stat code %d", w.Result().StatusCode)
	}
}

func TestGetCol(t *testing.T) {
	srv := setup()
	getReq := httptest.NewRequest("GET", "/v1/db24/doc1/col1/", strings.NewReader(""))
	getReq.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, getReq)

	if w.Result().StatusCode != http.StatusOK {
		fmt.Printf("statusCode")
		t.Errorf("TestGetBadColUri failed,got stat code %d", w.Result().StatusCode)
	}
}

func TestPutDoc(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("PUT", "/v1/db24/doc1/col1/doc2", strings.NewReader(`{"k":"v"}`))
	r.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("PutDoc failed")
	}

}

func TestPutCol(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("PUT", "/v1/db24/doc1/col1/", strings.NewReader(""))
	r.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusCreated {
		t.Errorf("PutCol failed")
	}
}

func TestPutDocOverwrite(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("PUT", "/v1/db", strings.NewReader(""))
	r.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusCreated {
		t.Errorf("Failed to create simulated DB")
	}

	r2 := httptest.NewRequest("PUT", "/v1/db24/doc1?mode=overwrite", strings.NewReader(`{"first":"pute"}`))
	r2.Header.Set("Authorization", "Bearer ADMIN")
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, r2)
	if w2.Result().StatusCode != http.StatusOK {
		t.Errorf("TestPutDocOverwrite failed, expected status code 201, got %d", w2.Result().StatusCode)
	}
	if w2.Header().Get("Location") != "Overwrite Success" {
		t.Errorf("TestPutDocOverwrite failed, expected Location: Overwrite Success, got %s", w2.Header().Get("Location"))
	}
}

func TestPutDocNooverwrite(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("PUT", "/v1/db24/doc1?mode=nooverwrite", strings.NewReader(`{"k":"v"}`))
	r.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusPreconditionFailed {
		t.Errorf("TestPutDocNooverwrite failed,got stat code %d, expected 412", w.Result().StatusCode)
	}

}

func TestPutDB(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("PUT", "/v1/db24", strings.NewReader(""))
	r.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusCreated {
		t.Errorf("testPutDB failed")
	}
}

func TestDeleteDB(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("DELETE", "/v1/db24", strings.NewReader(""))
	w := httptest.NewRecorder()
	r.Header.Set("Authorization", "Bearer ADMIN")
	srv.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusNoContent {
		t.Errorf("TestDeleteDB failed")
	}
}

func TestDeleteDBNoAuth(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("DELETE", "/v1/db24", strings.NewReader(""))
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("TestDeleteDB failed")
	}
}

func TestPatchDoc(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("DELETE", "/auth", strings.NewReader(""))
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)
}

func TestPostDoc(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("POST", "/v1/db/doc1/col1/", strings.NewReader(`{"json":"Object"}`))
	r.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusCreated || w.Result().Header.Get("Location") != "Posted Doc" {
		t.Errorf("TestPostDoc failed, got stat code, uri %d %s", w.Result().StatusCode, w.Result().Header.Get("Location"))
	}
}

func TestPostAuth(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("POST", "/auth", strings.NewReader(`{"username":"me"}`))
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("PostAuth failed")
	}
}

func TestDeleteAuth(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("DELETE", "/auth", strings.NewReader(""))
	r.Header.Set("Authorization", "Bearer TEST")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusNoContent {
		t.Errorf("DeleteAuthfailed")
	}
}

func TestBadLogout(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("DELETE", "/auth", strings.NewReader(""))

	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("DeleteAuthfailed")
	}
}

func TestBadLogin(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("POST", "/auth", strings.NewReader(""))
	r.Header.Set("Authorization", "Bearer TEST")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("DeleteAuthfailed")
	}
}

func TestBadDocPut(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("POST", "/auth", strings.NewReader(""))
	r.Header.Set("Authorization", "Bearer TEST")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("BadDocPutFailed")
	}
}

func TestBadURI(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("PUT", "/v1/db24/////", strings.NewReader(""))
	r.Header.Set("Authorization", "Bearer TEST")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("TestBadURI failed")
	}
}

func TestBadPost(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("POST", "/v1/db24/doc1/col1/", strings.NewReader(""))
	r.Header.Set("Authorization", "Bearer TEST")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("TestBadURI failed")
	}
}

func TestColGetIntervals(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("GET", "/v1/db24/doc1/col1/?interval=[1,2]", strings.NewReader(""))
	r.Header.Set("Authorization", "Bearer TEST")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("TestColGetIntervals failed")
	}
}

func TestColGetIntervalsDefaultRight(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("GET", "/v1/db24/doc1/col1/?interval=[1,]", strings.NewReader(""))
	r.Header.Set("Authorization", "Bearer TEST")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("TestColGetIntervals failed")
	}
}

func TestColGetIntervalsDefaultLeft(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("GET", "/v1/db24/doc1/col1/?interval=[,5]", strings.NewReader(""))
	r.Header.Set("Authorization", "Bearer TEST")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("TestColGetIntervals failed")
	}
}

func TestColGetIntervalsDefaultBoth(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("GET", "/v1/db24/doc1/col1/?interval=[,]", strings.NewReader(""))
	r.Header.Set("Authorization", "Bearer TEST")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("TestColGetIntervals failed")
	}
}

func TestColGetIntervalsBadIntervals(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("GET", "/v1/db24/doc1/col1/?interval=[", strings.NewReader(""))
	r.Header.Set("Authorization", "Bearer TEST")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("TestColGetIntervals failed")
	}
}

func TestColGetIntervalsSubscribe(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("GET", "/v1/db24/doc1/col1/?mode=subscribe", strings.NewReader(""))
	r.Header.Set("Authorization", "Bearer TEST")
	w := httptest.NewRecorder()

	go srv.ServeHTTP(w, r)
	select {
	case <-time.After(2 * time.Second):
		t.Log("PASSED")
	}
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("TestColGetIntervals failed")
	}
}

func TestColGetIntervalBadSubParam(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("GET", "/v1/db24/doc1/col1/?mode=sub", strings.NewReader(""))
	r.Header.Set("Authorization", "Bearer TEST")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("TestColGetIntervals failed")
	}

}

func TestOptions(t *testing.T) {
	srv := setup()

	r := httptest.NewRequest("OPTIONS", "/v1/", strings.NewReader(""))

	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("Testoptions failed, got status code %d", w.Result().StatusCode)
	}
	if w.Result().Header.Get("Access-Control-Allow-Methods") != "GET, POST, PUT, DELETE, OPTIONS, PATCH" {
		t.Errorf("Access-Control-Allow-Methods header is bad, got %s", w.Result().Header.Get("Access-Control-Allow-Methods"))
	}

}

func TestOptionsBadUri(t *testing.T) {
	srv := setup()

	r := httptest.NewRequest("OPTIONS", "/junk////", strings.NewReader(""))

	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)
	if w.Result().Header.Get("Access-Control-Allow-Methods") != "" {
		t.Errorf("Failed")
	}

}

func TestOptionsAuth(t *testing.T) {
	srv := setup()

	r := httptest.NewRequest("OPTIONS", "/auth", strings.NewReader(""))

	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)
	if w.Result().Header.Get("Access-Control-Allow-Methods") != "POST, DELETE, OPTIONS" {
		t.Errorf("TestOptionsAuth Failed")
	}

}

func TestDeleteDoc(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("DELETE", "/v1/db24/doc1", strings.NewReader(""))
	r.Header.Set("Authorization", "Bearer TEST")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusNoContent {
		t.Errorf("Deletedoc failed, got statcode %d", w.Result().StatusCode)
	}
}

func TestDeleteDocNoAuth(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("DELETE", "/v1/db24/doc1", strings.NewReader(""))

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("Deletedoc failed, got statcode %d", w.Result().StatusCode)
	}
}

func TestDeleteCol(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("DELETE", "/v1/db24/doc1/col1/", strings.NewReader(""))
	r.Header.Set("Authorization", "Bearer TEST")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusNoContent {
		t.Errorf("Deletedoc failed, got statcode %d", w.Result().StatusCode)
	}
}

func TestDeleteColNoAuth(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("DELETE", "/v1/db24/doc1/col1/", strings.NewReader(""))

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("Deletedoc failed, got statcode %d", w.Result().StatusCode)
	}
}

func TestDeleteColBadPath(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("DELETE", "/v1/db24/doc1/col1/doc3/", strings.NewReader(""))
	r.Header.Set("Authorization", "Bearer TEST")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("Deletedoc failed, got statcode %d", w.Result().StatusCode)
	}
}

func TestGetDocSubscribe(t *testing.T) {
	srv := setup()
	getReq := httptest.NewRequest("GET", "/v1/db24/doc1/doc2?mode=subscribe", strings.NewReader(""))
	getReq.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	go srv.ServeHTTP(w, getReq)
	select {
	case <-time.After(2 * time.Second):
		t.Log("PASSED")
	}

}

func TestGetDocSubscribeBadSubParam(t *testing.T) {
	srv := setup()
	getReq := httptest.NewRequest("GET", "/v1/db24/doc1/doc2?mode=subscbe", strings.NewReader(""))
	getReq.Header.Set("Authorization", "Bearer ADMIN")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, getReq)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("TestGetDocSubscribeBadSubParam failed, got status code %d", w.Result().StatusCode)
	}

}

func TestPutDocNoAuth(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("PUT", "/v1/db24/doc1/col1/doc2", strings.NewReader(`{"k":"v"}`))

	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("PutDoc failed")
	}
}

func TestPutDocBadPath(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("PUT", "/v1/db24/doc1/col1/doc1/", strings.NewReader(`{"k":"v"}`))

	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("PutDoc failed")
	}
}

func TestPostTop(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("POST", "/v1/db24/", strings.NewReader(`{"k":"v"}`))

	w := httptest.NewRecorder()
	r.Header.Set("Authorization", "Bearer ADMIN")
	srv.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusCreated {
		fmt.Printf("%d", w.Result().StatusCode)
		t.Errorf("PutDoc failed")
	}
}

func TestPostTopNoAuth(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("POST", "/v1/db24/", strings.NewReader(`{"k":"v"}`))

	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusUnauthorized {
		fmt.Printf("%d", w.Result().StatusCode)
		t.Errorf("PutDoc failed")
	}
}

func TestPatch(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("PATCH", "/v1/db24/doc1/col1/doc3", strings.NewReader(`{"k":"v"}`))
	r.Header.Set("Authorization", "Bearer TEST")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("TestPatch failed, got stat code %d", w.Result().StatusCode)
	}
}

func TestPatchNoAuth(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("PATCH", "/v1/db24/doc1/col1/doc3", strings.NewReader(`{"k":"v"}`))

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("TestPatchNoAuth failed, got stat code %d", w.Result().StatusCode)
	}
}

func TestPatchBadJSON(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("PATCH", "/v1/db24/doc1/col1/doc3", strings.NewReader(`{"k")`))
	r.Header.Set("Authorization", "Bearer TEST")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("TestPatch failed, got stat code %d", w.Result().StatusCode)
	}
}

// tests a bad post
func TestBadPostRequest(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("POST", "/v1/db24/doc1/", strings.NewReader(`{"k":"v"}`))
	r.Header.Set("Authorization", "Bearer TEST")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("TestBadPostRequest failed, got stat code %d", w.Result().StatusCode)
	}
}

// tests a bad post
func TestBadPostRequest2(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("POST", "/v1/db24/doc1", strings.NewReader(`{"k":"v"}`))
	r.Header.Set("Authorization", "Bearer TEST")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("TestBadPostRequest failed, got stat code %d", w.Result().StatusCode)
	}
}

// tests a bad put
func TestBadPutRequest2(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("PUT", "/v1/db24/", strings.NewReader(`{"k":"v"}`))
	r.Header.Set("Authorization", "Bearer TEST")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("TestBadPostRequest failed, got stat code %d", w.Result().StatusCode)
	}
}

// Tests a bad uri
func TestBadUri(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("GET", "/v1/db24", strings.NewReader(`{"k":"v"}`))
	r.Header.Set("Authorization", "Bearer TEST")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("TestBadPostRequest failed, got stat code %d", w.Result().StatusCode)
	}
}

func TestBadPostUri(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("POST", "/v1/db24/doc1", strings.NewReader(`{"k":"v"}`))
	r.Header.Set("Authorization", "Bearer TEST")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("TestBadPostRequest failed, got stat code %d", w.Result().StatusCode)
	}
}

func TestBadPostUriNoDoc(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("POST", "/v1/db24", strings.NewReader(`{"k":"v"}`))
	r.Header.Set("Authorization", "Bearer TEST")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("TestBadPostRequest failed, got stat code %d", w.Result().StatusCode)
	}
}

func TestBadGetDocUri(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("GET", "/v1/db24/doc1/", strings.NewReader(`{"k":"v"}`))
	r.Header.Set("Authorization", "Bearer TEST")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("TestBadPostRequest failed, got stat code %d", w.Result().StatusCode)
	}
}

func TestBadGetDBUri(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("GET", "/v1/db24", strings.NewReader(`{"k":"v"}`))
	r.Header.Set("Authorization", "Bearer TEST")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("TestBadPostRequest failed, got stat code %d", w.Result().StatusCode)
	}
}

func TestBadGetColUri(t *testing.T) {
	srv := setup()
	r := httptest.NewRequest("GET", "/v1/db24/doc1/col1", strings.NewReader(`{"k":"v"}`))
	r.Header.Set("Authorization", "Bearer TEST")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("TestBadPostRequest failed, got stat code %d", w.Result().StatusCode)
	}
}

func TestGetHandler(t *testing.T) {
	auth := New(mockAuthorizer{})
	tests := []struct {
		name           string
		resource       string
		method         string
		expectedStatus int
		expectedFunc   string // Which handler we expect to be called
	}{
		{
			name:           "Get Document",
			resource:       "dbName/collectionName/docName",
			method:         "GET",
			expectedStatus: http.StatusOK,
			expectedFunc:   "GetDoc",
		},
		{
			name:           "Get Collection",
			resource:       "dbName/collectionName/",
			method:         "GET",
			expectedStatus: http.StatusOK,
			expectedFunc:   "GetCol",
		},
		{
			name:           "Invalid Resource",
			resource:       "",
			method:         "GET",
			expectedStatus: http.StatusBadRequest,
			expectedFunc:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRG := &mockResourceGetter{}
			dbh := &DbHarness{
				auth: mockAuthorizer,
				rg:   mockRG,
			}

			req := httptest.NewRequest(tt.method, "/v1/"+tt.resource, nil)
			req.Header.Set("Authorization", "Bearer validToken")
			w := httptest.NewRecorder()

			dbh.getHandler(w, req)

			resp := w.Result()
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			// Check which method was called
			if tt.expectedFunc == "GetDoc" && !mockRG.didGetDoc {
				t.Errorf("Expected GetDoc to be called")
			}
			if tt.expectedFunc == "GetCol" && !mockRG.didGetCol {
				t.Errorf("Expected GetCol to be called")
			}
		})
	}
}
