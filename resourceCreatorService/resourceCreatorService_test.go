package resourceCreatorService

import (
	"fmt"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/mocks"
	"net/http"
	"testing"

	"github.com/RICE-COMP318-FALL24/owldb-p1group24/index_utils"
)

// Mocking Upsertdatabaser
type upserterDBMock struct {
	uploadColErr      error
	uploadDocumentErr error
}

// UploadCol mocks the behavior of uploading a collection. Returns an error status if uploadColErr is set.
func (mock *upserterDBMock) UploadCol(colpath string, dbName string) ([]byte, int, string) {
	if mock.uploadColErr != nil {
		return nil, http.StatusInternalServerError, ""
	}
	return []byte(colpath), http.StatusOK, ""
}

// UploadDocument mocks the behavior of uploading a document. Returns an error status if uploadDocumentErr is set.
func (mock *upserterDBMock) UploadDocument(docpath string, payload []byte, docname, user string, overwrite, isPost bool, dbName string) ([]byte, int, string) {
	if mock.uploadDocumentErr != nil {
		return nil, http.StatusInternalServerError, ""
	}
	return []byte(docpath), http.StatusOK, ""
}

// Mocking DatabaseIndex
type dbIndexMock struct {
	findFunc   func(key string) (Upsertdatabaser, bool)
	upsertFunc func(key string, check index_utils.UpdateCheck[string, Upsertdatabaser]) (bool, error)
}

// Upsert mocks the behavior of upserting a database entry.
func (mock *dbIndexMock) Upsert(key string, check index_utils.UpdateCheck[string, Upsertdatabaser]) (bool, error) {
	return mock.upsertFunc(key, check)
}

// Find mocks the behavior of finding a database entry by key.
func (mock *dbIndexMock) Find(key string) (Upsertdatabaser, bool) {
	return mock.findFunc(key)
}

// Mocking Validator
type validatorMock struct {
	validateErr error
}

// Validate mocks the behavior of validating JSON data. Returns an error if validateErr is set.
func (v *validatorMock) Validate(jsonData []byte) error {
	return v.validateErr
}

// Test for CreateDB
func TestCreateDB(t *testing.T) {
	mockDBIndex := &dbIndexMock{
		upsertFunc: func(key string, check index_utils.UpdateCheck[string, Upsertdatabaser]) (bool, error) {
			return true, nil
		},
	}
	service := New(mockDBIndex, func(string) Upsertdatabaser {
		return &upserterDBMock{}
	}, &validatorMock{})

	result, statusCode, _ := service.CreateDB("Neyida's DB")
	fmt.Println("CreateDB result:", string(result), "Status code:", statusCode)

}

// test PutDoc
func TestPutDoc(t *testing.T) {
	mockDB := &upserterDBMock{}

	mockDBIndex := &dbIndexMock{
		findFunc: func(key string) (Upsertdatabaser, bool) {
			return mockDB, true // Simulate that the database is found
		},
	}

	validator := &validatorMock{}
	service := New[string, Upsertdatabaser](mockDBIndex, nil, validator)

	payload := []byte(`{"name": "John Doe"}`)
	result, statusCode, _ := service.PutDoc("testDB", "testPath", "testDoc", payload, true, "user")

	//if it passes
	if statusCode != http.StatusOK {
		t.Errorf("Expected status code: %d, got: %d", http.StatusOK, statusCode)
	}
	if string(result) != "testPath" {
		t.Errorf("Expected document path to be returned, got: %s", string(result))
	}

	//If it fails
	mockDBIndex.findFunc = func(key string) (Upsertdatabaser, bool) {
		return nil, false // Simulate that the database is not found
	}
	result, statusCode, _ = service.PutDoc("missingDB", "testPath", "testDoc", payload, true, "user")
	if statusCode != http.StatusNotFound {
		t.Errorf("Expected status code: %d, got: %d", http.StatusNotFound, statusCode)
	}
	if string(result) != `"Collection does not exist"` {
		t.Errorf("Expected error message, got: %s", string(result))
	}
}

// Test for PostDoc
func TestPostDoc(t *testing.T) {
	mockDB := &upserterDBMock{}

	mockDBIndex := &dbIndexMock{
		findFunc: func(key string) (Upsertdatabaser, bool) {
			return mockDB, true // Simulate that the database is found
		},
	}
	service := New[string, Upsertdatabaser](mockDBIndex, nil, nil)

	// Test successful PutCol
	result, statusCode, _ := service.PutCol("testDB", "testCollection")

	if statusCode != http.StatusOK {
		t.Errorf("Expected status code: %d, got: %d", http.StatusOK, statusCode)
	}
	if string(result) != "testCollection" {
		t.Errorf("Expected collection path to be returned, got: %s", string(result))
	}

}

// test for put collection
func TestPutCol(t *testing.T) {
	mockDB := &upserterDBMock{}
	mockDBIndex := &dbIndexMock{
		findFunc: func(key string) (Upsertdatabaser, bool) {
			return mockDB, true // Simulate that the database is found
		},
	}
	service := New[string, Upsertdatabaser](mockDBIndex, nil, nil)

	// Test successful PutCol
	result, statusCode, _ := service.PutCol("testDB", "testCollection")

	if statusCode != http.StatusOK {
		t.Errorf("Expected status code: %d, got: %d", http.StatusOK, statusCode)
	}
	if string(result) != "testCollection" {
		t.Errorf("Expected collection path to be returned, got: %s", string(result))
	}

	// Test when the database is not found
	mockDBIndex.findFunc = func(key string) (Upsertdatabaser, bool) {
		return nil, false // Simulate that the database is not found
	}
	result, statusCode, _ = service.PutCol("missingDB", "testCollection")

	if statusCode != http.StatusNotFound {
		t.Errorf("Expected status code: %d, got: %d", http.StatusNotFound, statusCode)
	}
	if string(result) != `"Error: no such database exists"` {
		t.Errorf("Expected error message, got: %s", string(result))
	}
}

type mockValidator struct {
}

func (m mockValidator) Validate([]byte) error {
	return nil
}

func TestResourceCreatorService_PostDoc(t *testing.T) {

	mockDBIndex := mocks.NewMockSL[string, Upsertdatabaser]()
	service := New[string, Upsertdatabaser](mockDBIndex, nil, mockValidator{})

	// Test successful PutCol
	check := func(string, Upsertdatabaser, bool) (Upsertdatabaser, error) {
		return &upserterDBMock{}, nil
	}
	service.dbs.Upsert("db", check)

	service.PostDoc("db", "doc1/col1", "user", []byte("payload"))
}
