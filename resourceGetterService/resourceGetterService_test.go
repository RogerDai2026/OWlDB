package resourceGetterService

import (
	"fmt"
	"testing"

	"github.com/RICE-COMP318-FALL24/owldb-p1group24/mocks"
	"net/http"
)

// goodmockDB is a mock implementation of a database that simulates successful responses.
type goodmockDB struct{}

// GetColSerial simulates a successful column retrieval, returning http.StatusOK.
func (m *goodmockDB) GetColSerial(string, string, string, bool) ([]byte, int, *chan []byte, string, [][]byte) {
	return nil, http.StatusOK, nil, "", nil
}

// GetDocumentSerial simulates a successful document retrieval, returning http.StatusOK.
func (m *goodmockDB) GetDocumentSerial(string, bool) ([]byte, *chan []byte, string, int, []byte) {
	return nil, nil, "", http.StatusOK, nil
}

// badmockDB is a mock implementation of a database, similar to goodmockDB.
type badmockDB struct{}

// GetColSerial simulates a column retrieval, returning http.StatusOK.
func (m *badmockDB) GetColSerial(string, string, string, bool) ([]byte, int, *chan []byte, string, [][]byte) {
	return nil, http.StatusOK, nil, "", nil
}

// GetDocumentSerial simulates a document retrieval, returning http.StatusOK.
func (m *badmockDB) GetDocumentSerial(string, bool) ([]byte, *chan []byte, string, int, []byte) {
	return nil, nil, "", http.StatusOK, nil
}

// TestGetColDB tests the scenario where the database is not found when attempting to get a column using goodmockDB.
func TestGetColDB(t *testing.T) {
	dbs := mocks.NewMockSL[string, *goodmockDB]()
	rcs := New[string, 
	*goodmockDB](dbs)

	_, stat, _, _, _ := rcs.GetCol("fakeDB", "doc1/col1", "a", "z", false)

	if stat != http.StatusNotFound {
		t.Errorf("TestGetColNoDB failed")
	}
}

// TestGetColNoDB tests the scenario where the database is not found when attempting to get a column using badmockDB.
func TestGetColNoDB(t *testing.T) {
	dbs := mocks.NewMockSL[string, *badmockDB]()
	rcs := New[string, *badmockDB](dbs)
	_, stat, _, _, _ := rcs.GetCol("fakeDB", "doc1/col1", "a", "z", false)

	if stat != http.StatusNotFound {
		t.Errorf("TestGetColDB failed")
	}
}

// TestGetDocNoDB tests the scenario where a document is requested from a non-existent database using badmockDB.
func TestGetDocNoDB(t *testing.T) {
	dbs := mocks.NewMockSL[string, *badmockDB]()
	rcs := New[string, *badmockDB](dbs)
	_, stat, _, _, _ := rcs.GetDoc("fakeDB", "doc1/col1", false)

	if stat != http.StatusNotFound {
		fmt.Printf("%d", stat)
		t.Errorf("TestGetColDB failed")
	}
}

// TestGetDocDB tests the retrieval of a document from an existing database using goodmockDB.
func TestGetDocDB(t *testing.T) {
	dbs := mocks.NewMockSL[string, *goodmockDB]()
	dbs.Upsert("goodDB", func(string, *goodmockDB, bool) (*goodmockDB, error) {
		return &goodmockDB{}, nil
	})
	rcs := New[string, *goodmockDB](dbs)
	_, stat, _, _, _ := rcs.GetDoc("goodDB", "doc1/col1", false)

	if stat != http.StatusOK {
		fmt.Printf("%d", stat)
		t.Errorf("TestGetColDB failed")
	}
}
