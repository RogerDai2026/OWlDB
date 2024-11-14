
package db

import (
	"cmp"
	"context"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/index_utils"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/mocks"
	"log/slog"
	"net/http"
	"testing"
)

// AddChildDocument(docpath string, payload []byte, docname string, user string, overwrite bool, isPost bool, dbName string) ([]byte, int) //adds a child document
// AddChildCollection(colpath string, dbName string) ([]byte, int)

type mockSL[K cmp.Ordered, V any] struct {
	sl map[K]V
}

func (m *mockSL[K, V]) Find(key K) (foundValue V, found bool) {
	v, ok := m.sl[key]
	if !ok {
		return v, false
	}
	return v, true
}

func (m *mockSL[K, V]) Remove(key K) (foundValue V, found bool) {
	v, ok := m.sl[key]
	if !ok {
		return v, false
	}
	delete(m.sl, key)
	return v, true
}

func (m *mockSL[K, V]) Query(ctx context.Context, low K, hi K) (result []index_utils.Pair[K, V], err error) {

	res := make([]index_utils.Pair[K, V], 0)
	for k, v := range m.sl {
		if low <= k && k <= hi {
			p := index_utils.Pair[K, V]{Key: k, Value: v}
			res = append(res, p)
		}
	}
	return res, nil
}

func (m *mockSL[K, V]) Upsert(key K, check index_utils.UpdateCheck[K, V]) (bool, error) {
	curVal, exists := m.sl[key]

	newVal, err := check(key, curVal, exists)
	if err != nil {
		return false, err
	}
	m.sl[key] = newVal
	return true, nil

}

type mockDoc struct {
}

func (m mockDoc) AddChildDocument(docpath string, payload []byte, docname string, user string, overwrite bool, isPost bool, dbName string) ([]byte, int, string) {
	return nil, 201, "/v1/db/dummy/dummy/"
}

func (m mockDoc) AddChildCollection(colpath string, dbName string) ([]byte, int, string) {
	return nil, 201, "/v1/db/dummy/dummy/"
}

func (m mockDoc) GetChildDocument(docpath string, isSubscribe bool, dbName string) (payload []byte, status_code int, sub_id string, subChan *chan []byte, docEvent []byte) {
	return payload, 200, "", nil, nil
}

func (m mockDoc) GetChildCollection(colpath string, lo string, hi string, isSubscribe bool) (res []byte, stat_code int, subChan *chan []byte, subId string, docEvents [][]byte) {
	return nil, 200, nil, "", nil
}

func (m mockDoc) DeleteChildDocument(docpath string, dbName string) ([]byte, int) {
	return nil, http.StatusNoContent
}

func (m mockDoc) DeleteChildCollection(colpath string) ([]byte, int) {
	return nil, http.StatusNoContent
}

func (m mockDoc) Notify(uri string, payload []byte, evType string) {
	slog.Debug("Notify called")
}

func (m mockDoc) ApplyPatchDocument(dbName string, docPath string, patch []byte, user string) ([]byte, int) {

	slog.Debug("ApplyPatchDocument Called")
	return []byte("placeholder"), 200
}

func (m mockDoc) DoPatch(patch []byte) ([]byte, error) {

	slog.Debug("DoPatch Called")
	return []byte("placeholder"), nil
}

func (m mockDoc) UpdateDoc(newRaw []byte, user string) {

	slog.Debug("UpdateDoc Called")
}

func (m mockDoc) GetSerial() []byte {
	//TODO implement me
	return []byte("PLACEHOLDER")
}

type mockColSubber struct {
	NotifyAllInvoked     bool
	AddSubscriberInvoked bool
	NotifyInvoked        bool
	GenerateEventInvoked bool
}

func (m *mockColSubber) NotifyAll(colname string) {

	m.NotifyAllInvoked = true
	slog.Debug("NotifyAll called")
}

func (m *mockColSubber) AddSubscriber(lo string, hi string) (subChan *chan []byte, id string) {

	m.AddSubscriberInvoked = true
	slog.Debug("AddSubscriber called")

	return
}

func (m *mockColSubber) Notify(docname string, evType string, payload []byte) {

	m.NotifyInvoked = true
	slog.Debug("mockColSubber Notify invoked")
}

func (m *mockColSubber) GenerateEvent(evType string, content []byte) []byte {

	m.GenerateEventInvoked = true
	return []byte("PLACEHOLDER")
}

type mockValidator struct {
}

func (m *mockValidator) Validate(bytes []byte) error {

	return nil
}

func TestDatabase_UploadDocument(t *testing.T) {

	var mockDcf DocFactory[mockDoc] = func([]byte, string, string) mockDoc {
		return mockDoc{}
	}
	docIndex := mocks.NewMockSL[string, mockDoc]()
	db := New[string, mockDoc]("db", mockDcf, docIndex, &mockColSubber{}, &mockValidator{})

	_, stat, _ := db.UploadDocument("doc1", mocks.MockPayload(), "doc1", "USER", true, false, "db")
	if stat != http.StatusCreated {
		t.Errorf("Error TestDatabase_UploadDocument, expected stat code 201,got %d", stat)
	}
}

func TestDatabase_DeleteDoc(t *testing.T) {
	var mockDcf DocFactory[mockDoc] = func([]byte, string, string) mockDoc {
		return mockDoc{}
	}
	docIndex := mocks.NewMockSL[string, mockDoc]()
	db := New[string, mockDoc]("db", mockDcf, docIndex, &mockColSubber{}, &mockValidator{})
	db.UploadDocument("doc1", mocks.MockPayload(), "doc1", "USER", true, false, "db")
	_, stat := db.DeleteDoc("doc1")

	if stat != http.StatusNoContent {
		t.Errorf("TestDatabase_DeleteDoc failed, got stat code %d", stat)
	}
}

func TestDatabase_UploadDocumentNooverwrite(t *testing.T) {
	var mockDcf DocFactory[mockDoc] = func([]byte, string, string) mockDoc {
		return mockDoc{}
	}
	docIndex := mocks.NewMockSL[string, mockDoc]()
	mockSubber := &mockColSubber{
		NotifyAllInvoked:     false,
		AddSubscriberInvoked: false,
		NotifyInvoked:        false,
		GenerateEventInvoked: false,
	}
	db := New[string, mockDoc]("db", mockDcf, docIndex, mockSubber, &mockValidator{})

	db.UploadDocument("doc1", mocks.MockPayload(), "doc1", "USER", false, false, "db")
	_, stat, _ := db.UploadDocument("doc1", mocks.MockPayload(), "doc1", "USER", false, false, "db")

	if stat != http.StatusPreconditionFailed {
		t.Errorf("TestDatabase_UploadDocumentNooverwrite failed, got stat code %d", stat)
	}
}

func TestDatabase_UploadDocumentOverwrite(t *testing.T) {
	var mockDcf DocFactory[mockDoc] = func([]byte, string, string) mockDoc {
		return mockDoc{}
	}
	docIndex := mocks.NewMockSL[string, mockDoc]()
	mockSubber := &mockColSubber{
		NotifyAllInvoked:     false,
		AddSubscriberInvoked: false,
		NotifyInvoked:        false,
		GenerateEventInvoked: false,
	}
	db := New[string, mockDoc]("db", mockDcf, docIndex, mockSubber, &mockValidator{})

	db.UploadDocument("doc1", mocks.MockPayload(), "doc1", "USER", false, false, "db")
	_, stat, _ := db.UploadDocument("doc1", mocks.MockPayload(), "doc1", "USER", true, false, "db")

	if stat != http.StatusOK {
		t.Errorf("TestDatabase_UploadDocumentNooverwrite failed, got stat code %d", stat)
	}

}

func TestDatabase_Notify(t *testing.T) {
	var mockDcf DocFactory[mockDoc] = func([]byte, string, string) mockDoc {
		return mockDoc{}
	}
	docIndex := mocks.NewMockSL[string, mockDoc]()
	mockSubber := &mockColSubber{
		NotifyAllInvoked:     false,
		AddSubscriberInvoked: false,
		NotifyInvoked:        false,
		GenerateEventInvoked: false,
	}
	db := New[string, mockDoc]("db", mockDcf, docIndex, mockSubber, &mockValidator{})
	db.UploadDocument("doc1", mocks.MockPayload(), "doc1", "USER", false, false, "db")

	if !mockSubber.NotifyInvoked {
		t.Errorf("TestDatabase_Notify: not invoked")
	}
}

func TestDatabase_PatchNotFound(t *testing.T) {

}

func TestDatabase_UploadColNotFound(t *testing.T) {
	var mockDcf DocFactory[mockDoc] = func([]byte, string, string) mockDoc {
		return mockDoc{}
	}
	docIndex := mocks.NewMockSL[string, mockDoc]()
	mockSubber := &mockColSubber{
		NotifyAllInvoked:     false,
		AddSubscriberInvoked: false,
		NotifyInvoked:        false,
		GenerateEventInvoked: false,
	}
	db := New[string, mockDoc]("db", mockDcf, docIndex, mockSubber, &mockValidator{})
	db.UploadCol("doc1/col1/doc2/col2", "db")

}

func TestDatabase_DeleteColNotFound(t *testing.T) {
	var mockDcf DocFactory[mockDoc] = func([]byte, string, string) mockDoc {
		return mockDoc{}
	}
	docIndex := mocks.NewMockSL[string, mockDoc]()
	db := New[string, mockDoc]("db", mockDcf, docIndex, &mockColSubber{}, &mockValidator{})

	db.UploadDocument("doc1", mocks.MockPayload(), "doc1", "USER", true, false, "db")
	db.DeleteCol("doc2/col1/")
}

func TestDatabase_DeleteCol(t *testing.T) {
	var mockDcf DocFactory[mockDoc] = func([]byte, string, string) mockDoc {
		return mockDoc{}
	}
	docIndex := mocks.NewMockSL[string, mockDoc]()
	db := New[string, mockDoc]("db", mockDcf, docIndex, &mockColSubber{}, &mockValidator{})

	db.UploadDocument("doc1", mocks.MockPayload(), "doc1", "USER", true, false, "db")
	db.DeleteCol("doc1/col1/")
}

func TestDatabase_GetDocumentSerial(t *testing.T) {
	var mockDcf DocFactory[mockDoc] = func([]byte, string, string) mockDoc {
		return mockDoc{}
	}
	docIndex := mocks.NewMockSL[string, mockDoc]()
	db := New[string, mockDoc]("db", mockDcf, docIndex, &mockColSubber{}, &mockValidator{})

	db.UploadDocument("doc1", mocks.MockPayload(), "doc1", "USER", true, false, "db")
	db.GetDocumentSerial("doc1", true)
}

func TestDatabase_GetDocumentSerialNotFound(t *testing.T) {

}

func TestDatabase_UploadCol(t *testing.T) {
	var mockDcf DocFactory[mockDoc] = func([]byte, string, string) mockDoc {
		return mockDoc{}
	}
	docIndex := mocks.NewMockSL[string, mockDoc]()
	db := New[string, mockDoc]("db", mockDcf, docIndex, &mockColSubber{}, &mockValidator{})

	db.UploadDocument("doc1", mocks.MockPayload(), "doc1", "USER", true, false, "db")
	//var mockDcf DocFactory[mockDoc] = func([]byte, string, string) mockDoc {
	//	return mockDoc{}
	//}
	//docIndex := mocks.NewMockSL[string, mockDoc]()
	//db := New[string, mockDoc]("db", mockDcf, docIndex, &mockColSubber{}, mockValidator{})

	db.UploadCol("doc1/col1/", "db")
}

func TestDatabase_GetColSerialTop(t *testing.T) {
	var mockDcf DocFactory[mockDoc] = func([]byte, string, string) mockDoc {
		return mockDoc{}
	}
	docIndex := mocks.NewMockSL[string, mockDoc]()
	db := New[string, mockDoc]("db", mockDcf, docIndex, &mockColSubber{}, &mockValidator{})
	db.UploadDocument("doc1", mocks.MockPayload(), "doc1", "USER", true, false, "db")
	db.GetColSerial("", "", "z", false)
}

func TestDatabase_GetColSerialSubscribe(t *testing.T) {
	var mockDcf DocFactory[mockDoc] = func([]byte, string, string) mockDoc {
		return mockDoc{}
	}
	docIndex := mocks.NewMockSL[string, mockDoc]()
	db := New[string, mockDoc]("db", mockDcf, docIndex, &mockColSubber{}, &mockValidator{})
	db.UploadDocument("doc1", mocks.MockPayload(), "doc1", "USER", true, false, "db")
	db.GetColSerial("", "", "z", true)
}

func TestDatabase_GetColSerial(t *testing.T) {
	var mockDcf DocFactory[mockDoc] = func([]byte, string, string) mockDoc {
		return mockDoc{}
	}
	docIndex := mocks.NewMockSL[string, mockDoc]()
	db := New[string, mockDoc]("db", mockDcf, docIndex, &mockColSubber{}, &mockValidator{})
	db.UploadDocument("doc1", mocks.MockPayload(), "doc1", "USER", true, false, "db")
	db.GetColSerial("doc1/col1/", "", "z", false)
}

func TestDatabase_GetColSerialTopSubscribe(t *testing.T) {
	var mockDcf DocFactory[mockDoc] = func([]byte, string, string) mockDoc {
		return mockDoc{}
	}
	docIndex := mocks.NewMockSL[string, mockDoc]()
	db := New[string, mockDoc]("db", mockDcf, docIndex, &mockColSubber{}, &mockValidator{})
	db.UploadDocument("doc1", mocks.MockPayload(), "doc1", "USER", true, false, "db")
	db.GetColSerial("doc1/col1", "", "z", true)
}

func TestDatabase_DeleteDocNested(t *testing.T) {
	var mockDcf DocFactory[mockDoc] = func([]byte, string, string) mockDoc {
		return mockDoc{}
	}
	docIndex := mocks.NewMockSL[string, mockDoc]()
	db := New[string, mockDoc]("db", mockDcf, docIndex, &mockColSubber{}, &mockValidator{})
	db.UploadDocument("doc1", mocks.MockPayload(), "doc1", "USER", true, false, "db")
	db.DeleteDoc("doc1/col1/doc2")
}

func TestDatabase_GetColSerialNotFound(t *testing.T) {
	var mockDcf DocFactory[mockDoc] = func([]byte, string, string) mockDoc {
		return mockDoc{}
	}
	docIndex := mocks.NewMockSL[string, mockDoc]()
	db := New[string, mockDoc]("db", mockDcf, docIndex, &mockColSubber{}, &mockValidator{})
	db.UploadDocument("doc1", mocks.MockPayload(), "doc1", "USER", true, false, "db")
	db.GetColSerial("doc2/col1/doc2/col2", "", "z", true)
}

func TestDatabase_DeleteDocNotFound(t *testing.T) {
	var mockDcf DocFactory[mockDoc] = func([]byte, string, string) mockDoc {
		return mockDoc{}
	}
	docIndex := mocks.NewMockSL[string, mockDoc]()
	db := New[string, mockDoc]("db", mockDcf, docIndex, &mockColSubber{}, &mockValidator{})
	db.UploadDocument("doc1", mocks.MockPayload(), "doc1", "USER", true, false, "db")
	db.DeleteDoc("doc2/col1/doc4")
}

func TestDatabase_PatchTop(t *testing.T) {
	var mockDcf DocFactory[mockDoc] = func([]byte, string, string) mockDoc {
		return mockDoc{}
	}
	docIndex := mocks.NewMockSL[string, mockDoc]()
	db := New[string, mockDoc]("db", mockDcf, docIndex, &mockColSubber{}, &mockValidator{})
	db.UploadDocument("doc1", mocks.MockPayload(), "doc1", "USER", true, false, "db")

	db.Patch("doc1", []byte("patch"), "user")
}

func TestDatabase_PatchNested(t *testing.T) {
	var mockDcf DocFactory[mockDoc] = func([]byte, string, string) mockDoc {
		return mockDoc{}
	}
	docIndex := mocks.NewMockSL[string, mockDoc]()
	db := New[string, mockDoc]("db", mockDcf, docIndex, &mockColSubber{}, &mockValidator{})
	db.UploadDocument("doc1", mocks.MockPayload(), "doc1", "USER", true, false, "db")

	db.Patch("doc1/col1/doc2", []byte("patch"), "user")
}

func TestDatabase_NotifyAll(t *testing.T) {
	var mockDcf DocFactory[mockDoc] = func([]byte, string, string) mockDoc {
		return mockDoc{}
	}
	docIndex := mocks.NewMockSL[string, mockDoc]()
	db := New[string, mockDoc]("db", mockDcf, docIndex, &mockColSubber{}, &mockValidator{})
	db.NotifyAll("/")
}

func TestDatabase_UploadDocNested(t *testing.T) {
	var mockDcf DocFactory[mockDoc] = func([]byte, string, string) mockDoc {
		return mockDoc{}
	}
	docIndex := mocks.NewMockSL[string, mockDoc]()
	db := New[string, mockDoc]("db", mockDcf, docIndex, &mockColSubber{}, &mockValidator{})
	db.UploadDocument("doc1", mocks.MockPayload(), "doc1", "USER", true, false, "db")

	db.UploadDocument("doc1/col1/doc2", mocks.MockPayload(), "doc2", "USER", false, false, "db")
}

func TestDatabase_RemoveTopNotFound(t *testing.T) {
	var mockDcf DocFactory[mockDoc] = func([]byte, string, string) mockDoc {
		return mockDoc{}
	}
	docIndex := mocks.NewMockSL[string, mockDoc]()
	db := New[string, mockDoc]("db", mockDcf, docIndex, &mockColSubber{}, &mockValidator{})
	db.UploadDocument("doc1", mocks.MockPayload(), "doc1", "USER", true, false, "db")
	_, stat := db.DeleteDoc("doc2")

	if stat != http.StatusNotFound {
		t.Errorf("TestDatabase_DeleteDoc failed, got stat code %d", stat)
	}
}
