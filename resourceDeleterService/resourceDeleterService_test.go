package resourceDeleterService_test

import (
	"encoding/json"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/mocks"
	"net/http"
	"reflect"
	"testing"

	"github.com/RICE-COMP318-FALL24/owldb-p1group24/resourceDeleterService"
)

// Mock Deletedatabaser to simulate the deletion behavior
type MockDeletedatabaser struct {
	deleteDocCalled bool
	deleteColCalled bool
	notifyAllCalled bool
}

// Mock Deletedatabaser to simulate the deletion behavior
func (m *MockDeletedatabaser) DeleteDoc(docpath string) ([]byte, int) {
	m.deleteDocCalled = true
	return []byte(`{"success":"Document deleted"}`), http.StatusOK
}

func (m *MockDeletedatabaser) DeleteCol(colpath string) ([]byte, int) {
	m.deleteColCalled = true
	return []byte(`{"success":"Collection deleted"}`), http.StatusOK
}

func (m *MockDeletedatabaser) NotifyAll(colname string) {
	m.notifyAllCalled = true
}

// Mock DatabaseIndex to simulate finding and removing databases
type MockDatabaseIndex struct {
	dbs map[string]*MockDeletedatabaser
}

func (m *MockDatabaseIndex) Find(key string) (*MockDeletedatabaser, bool) {
	db, found := m.dbs[key]
	return db, found
}

func (m *MockDatabaseIndex) Remove(key string) (*MockDeletedatabaser, bool) {
	db, found := m.dbs[key]
	if found {
		delete(m.dbs, key)
		return db, true
	}
	return nil, false
}

// setupService initializes the service with a mock database index
func setupService() *resourceDeleterService.ResourceDeleterService[string, *MockDeletedatabaser] {
	dbs := mocks.NewMockSL[string, *MockDeletedatabaser]()
	chk := func(string, *MockDeletedatabaser, bool) (new *MockDeletedatabaser, err error) {
		return &MockDeletedatabaser{
			deleteDocCalled: false,
			deleteColCalled: false,
			notifyAllCalled: false,
		}, nil
	}
	dbs.Upsert("db1", chk)
	// Correct way to instantiate ResourceDeleterService
	return resourceDeleterService.New[string, *MockDeletedatabaser](dbs)
}

func TestDeleteDoc_Found(t *testing.T) {
	service := setupService()

	resp, status := service.DeleteDoc("db1", "/path/to/doc")

	expectedResp := []byte(`{"success":"Document deleted"}`)
	expectedStatus := http.StatusOK

	if !reflect.DeepEqual(resp, expectedResp) || status != expectedStatus {
		t.Errorf("Expected response: %s, status: %d, got response: %s, status: %d", expectedResp, expectedStatus, resp, status)
	}
}

func TestDeleteDoc_NotFound(t *testing.T) {
	service := setupService()

	resp, status := service.DeleteDoc("db2", "/path/to/doc")

	expectedResp, _ := json.Marshal("Error: database does not exist")
	expectedStatus := http.StatusNotFound

	if !reflect.DeepEqual(resp, expectedResp) || status != expectedStatus {
		t.Errorf("Expected response: %s, status: %d, got response: %s, status: %d", expectedResp, expectedStatus, resp, status)
	}
}

func TestDeleteDB_NotFound(t *testing.T) {
	service := setupService()

	resp, status := service.DeleteDB("db2")

	expectedResp, _ := json.Marshal("Error: database does not exist")
	expectedStatus := http.StatusNotFound

	if !reflect.DeepEqual(resp, expectedResp) || status != expectedStatus {
		t.Errorf("Expected response: %s, status: %d, got response: %s, status: %d", expectedResp, expectedStatus, resp, status)
	}
}

func TestDeleteDB_Found(t *testing.T) {
	service := setupService()

	resp, status := service.DeleteDB("db1")

	expectedResp := []byte(`"Deleted."`)
	expectedStatus := http.StatusNoContent

	if !reflect.DeepEqual(resp, expectedResp) || status != expectedStatus {
		t.Errorf("Expected response: %s, status: %d, got response: %s, status: %d", expectedResp, expectedStatus, resp, status)
	}
}

func TestDeleteCol_Found(t *testing.T) {
	service := setupService()

	resp, status := service.DeleteCol("db1", "/path/to/col")

	expectedResp := []byte(`{"success":"Collection deleted"}`)
	expectedStatus := http.StatusOK

	if !reflect.DeepEqual(resp, expectedResp) || status != expectedStatus {
		t.Errorf("Expected response: %s, status: %d, got response: %s, status: %d", expectedResp, expectedStatus, resp, status)
	}
}

func TestDeleteCol_NotFound(t *testing.T) {
	service := setupService()

	resp, status := service.DeleteCol("db2", "/path/to/col")

	expectedResp, _ := json.Marshal("Error: no such database exists")
	expectedStatus := http.StatusNotFound

	if !reflect.DeepEqual(resp, expectedResp) || status != expectedStatus {
		t.Errorf("Expected response: %s, status: %d, got response: %s, status: %d", expectedResp, expectedStatus, resp, status)
	}
}

//Need to test
//Delete Database not found
//Delete Database found
//Delete Document not found
//Delete Document found
//Delete Collection not found
//Delete Collection found
