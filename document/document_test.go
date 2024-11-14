// Description: This file contains the tests for the document package.
package document

import (
	"cmp"
	"context"
	"encoding/json"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/index_utils"

	"github.com/RICE-COMP318-FALL24/owldb-p1group24/jsondata"
	"log/slog"
	"net/http"
	"testing"
)

type dummyPayload struct {
	Name     string
	Message  string
	Likes    int
	Dislikes int
}

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

type mockSubscriptionManager struct {
	subscriptions map[int]*chan []byte
	nextID        int
}

func newMockSubscriptionManager() *mockSubscriptionManager {
	return &mockSubscriptionManager{
		subscriptions: make(map[int]*chan []byte),
		nextID:        0,
	}
}

// AddSubscriber implements the SubscriptionManager interface
func (m *mockSubscriptionManager) AddSubscriber() (ch *chan []byte, id string) {
	ch = new(chan []byte)

	id = "abc"
	return ch, id
}

// RemoveSubscriber implements the SubscriptionManager interface
func (m *mockSubscriptionManager) RemoveSubscriber(id int) {
	delete(m.subscriptions, id)
}

// Notify implements the SubscriptionManager interface
func (m *mockSubscriptionManager) Notify(evType string, payload []byte) {
	for _, ch := range m.subscriptions {
		*ch <- payload
	}
	return
}

type DummyValidator struct {
	mockError error
}

// NewDummyValidator creates a new DummyValidator.
// If mockError is nil, Validate will always succeed, otherwise it returns mockError.
func NewDummyValidator(mockError error) *DummyValidator {
	return &DummyValidator{mockError: mockError}
}

type mockSubber interface {
	AddSubscriber() (*chan []byte, string)
	RemoveSubscriber(id string)
	Notify(evType string, payload []byte)
	GenerateEvent(evType string, payload []byte) []byte
}

type mockSubberstruct struct {
	didAddSubscriber    bool
	didNotify           bool
	didRemoveSubscriber bool
	didGenerateEvent    bool
}

func (m *mockSubberstruct) AddSubscriber() (ch *chan []byte, id string) {
	m.didAddSubscriber = true
	return nil, ""
}

func (m *mockSubberstruct) RemoveSubscriber(id string) {
	m.didRemoveSubscriber = true
}

func (m *mockSubberstruct) Notify(evType string, payload []byte) {
	m.didNotify = true
}

func (m *mockSubberstruct) GenerateEvent(evType string, payload []byte) []byte {
	m.didGenerateEvent = true
	return []byte("event")
}

type mockValidator struct {
}

func (m mockValidator) Validate([]byte) error {
	return nil
}

type ImockPatcher interface {
	DoPatch(oldDoc []byte, patches []byte) (newDoc []byte, err error)
}

type mockPatcher struct {
	ImockPatcher
}

func (m mockPatcher) DoPatch(oldDoc []byte, patches []byte) (newDoc []byte, err error) {
	return []byte("newDoc"), nil
}

type mockColSubManager struct {
	addSubscriberCalled    bool
	removeSubscriberCalled bool
	notifyCalled           bool
	notifyAllCalled        bool
	generateEventCalled    bool
}

func (m *mockColSubManager) AddSubscriber(lo string, hi string) (subChan *chan []byte, id string) {

	m.addSubscriberCalled = true
	return nil, ""
}

func (m *mockColSubManager) Remove(id string) {
	slog.Debug("Removesubscriber called")
	m.removeSubscriberCalled = true
}

func (m *mockColSubManager) Notify(docname string, evType string, payload []byte) {
	m.notifyCalled = true
	slog.Debug("Notify called")
}

func (m *mockColSubManager) NotifyAll(colname string) {
	m.notifyAllCalled = true
}

func (m *mockColSubManager) GenerateEvent(evtype string, payload []byte) []byte {
	m.generateEventCalled = true

	return nil
}

type mockMessenger struct {
}

func (m mockMessenger) AddDocSubscriber(uri string) (*chan []byte, string) {
	return nil, ""
}

func (m mockMessenger) NotifyDocs(uri string, evtype string, payload []byte) {
	return
}

// Validate satisfies the Validator interface, returning mockError if provided.
func mockDocument() *Document {
	// Assume we have the following components created elsewhere in the code
	newColFactory := func(colName string) *Collection {
		return &Collection{Docs: &mockSL[string, *Document]{sl: make(map[string]*Document)}, Name: colName, SubscriptionManager: &mockColSubManager{}}
	}

	docColFactory := func() DocumentIndex[string, *Collection] {
		newCollections := &mockSL[string, *Collection]{sl: make(map[string]*Collection)}
		return newCollections
	}

	subscriptionManagerFactory := func() SubscriptionManager {
		return &mockSubberstruct{}
	}

	// The payload of the document (dummy data)

	messageStruct := dummyPayload{
		Name:     "John",
		Message:  "Hello",
		Likes:    30,
		Dislikes: -5000,
	}
	b, _ := json.Marshal(messageStruct)

	// Create a DummyValidator that doesn't perform actual validation
	// Pass `nil` for no errors, or provide an error for testing

	// Create the top-level document using the New signature

	topDoc := New(b, "USER", "topDoc", docColFactory, newColFactory, subscriptionManagerFactory, mockValidator{}, mockPatcher{}, mockMessenger{})

	return topDoc
}

func mockPayload() []byte {
	messageStruct := dummyPayload{
		Name:     "John",
		Message:  "Hello",
		Likes:    30,
		Dislikes: -5000,
	}
	b, _ := json.Marshal(messageStruct)
	return b
}
func TestDocument_GetChildDocument(t *testing.T) {

}

type docResponse struct {
	Path string
	Doc  string
	Meta string
}

func TestDocument_GetSerial(t *testing.T) {
	mockdoc := mockDocument()

	jv1 := mockdoc.GetSerial()

	jv2 := mockdoc.GetSerial()

	var j1 jsondata.JSONValue
	var j2 jsondata.JSONValue

	json.Unmarshal(jv1, &j1)
	json.Unmarshal(jv2, &j2)

	if !j1.Equal(j2) {
		t.Errorf("GetSerial Failed, document representation incorrect")
	}
}

func TestDocument_AddChildCollection(t *testing.T) {
	mockdoc := mockDocument()

	mockdoc.AddChildCollection("topDoc/col1", "mydb")

	_, stat2, _, _, _ := mockdoc.GetChildCollection("topDoc/col1", "a", "b", false)

	if stat2 != http.StatusOK {
		t.Errorf("AddChildCollection Failed: expected status 200, got %d", stat2)
	}

}

func TestDocument_AddDuplicateCollection(t *testing.T) {
	mockdoc := mockDocument()
	mockdoc.AddChildCollection("col1", "mydb")

	_, stat2, _ := mockdoc.AddChildCollection("col1", "mydb")

	if stat2 != http.StatusBadRequest {
		t.Errorf("AddDuplicateCollection Failed: expected status code 200,got %d", stat2)
	}
}

func TestDocument_AddChildDocumentPutOverwrite(t *testing.T) {
	mockdoc := mockDocument()
	mockdoc.AddChildCollection("topDoc/col1", "mydb")

	_, stat, _ := mockdoc.AddChildDocument("topDoc/col1/doc2", mockPayload(), "doc2", "USER", true, false, "mydb")

	if stat != http.StatusCreated {
		t.Errorf("AddChildDocumentPutOverwrite failed, expected status code 201,got %d", stat)
	}
	_, stat, uri := mockdoc.AddChildDocument("topDoc/col1/doc2", mockPayload(), "doc2", "user", true, false, "mydb")

	if stat != http.StatusOK && uri != "/v1/mydb/topDoc/col1/doc2" {
		t.Errorf("TestDocument_AddChildDocumentPutOverwrite Failed, got stat,uri %d %s", stat, uri)
	}
}

func TestDocument_AddChildDocumentPutNoOverwrite(t *testing.T) {
	mockdoc := mockDocument()
	mockdoc.AddChildCollection("topDoc/col1", "mydb")
	mockdoc.AddChildDocument("topDoc/col1/doc2", mockPayload(), "doc2", "USER", false, false, "mydb")

	_, stat, _ := mockdoc.AddChildDocument("topDoc/col1/doc2", mockPayload(), "doc2", "USER", false, false, "mydb")

	if stat != http.StatusPreconditionFailed {
		t.Errorf("Error: expected status code 412,got %d", stat)
	}
}

func TestDocument_AddChildDocumentPost(t *testing.T) {
	mockdoc := mockDocument()
	mockdoc.AddChildCollection("topDoc/col1", "mydb")
	mockdoc.AddChildDocument("topDoc/col1/doc2", mockPayload(), "doc2", "USER", false, false, "mydb")
}

func TestDocument_AddMultipleChildCollections(t *testing.T) {
	mockdoc := mockDocument()

	mockdoc.AddChildCollection("topDoc/col1", "mydb")
	mockdoc.AddChildCollection("topDoc/col2", "mydb")
	mockdoc.AddChildCollection("topDoc/col3", "mydb")

	_, stat, _, _, _ := mockdoc.GetChildCollection("topDoc/col1", "a", "b", false)

	if stat != http.StatusOK {
		t.Errorf("AddMultipleChildCollections failed, expected 200 and got %d", stat)
	}
	_, stat, _, _, _ = mockdoc.GetChildCollection("topDoc/col2", "a", "b", false)

	if stat != http.StatusOK {
		t.Errorf("AddMultipleChildCollections failed, expected 200 and got %d", stat)
	}

	_, stat, _, _, _ = mockdoc.GetChildCollection("topDoc/col3", "a", "b", false)
	if stat != http.StatusOK {
		t.Errorf("AddMultipleChildCollections failed, expected 200 and got %d", stat)
	}

}
func TestDocument_PutDocNoCollection(t *testing.T) {
	mockdoc := mockDocument()

	_, stat, _ := mockdoc.AddChildDocument("topDoc/col1/doc2", mockPayload(), "doc2", "USER", false, false, "mydb")

	if stat != http.StatusNotFound {
		t.Errorf("Error: expected status code 404,got %d", stat)
	}
}

func TestDocument_GetChildCollectionSingle(t *testing.T) {
	mockDoc := mockDocument()

	mockDoc.AddChildCollection("topDoc/col1", "mydb")

	mockDoc.AddChildDocument("topDoc/col1/doc2", mockPayload(), "doc2", "test", true, true, "mydb")

	_, stat, _, _, _ := mockDoc.GetChildCollection("topDoc/col1", "a", "z", false)

	if stat != http.StatusOK {
		t.Errorf("GetChildCollectionSingle failed, expected status code 200,got %d", stat)
	}

}

func TestDocument_GetChildCollectionMultipleDocs(t *testing.T) {
	mockDoc := mockDocument()

	mockDoc.AddChildCollection("topDoc/col1", "mydb")

	mockDoc.AddChildDocument("topDoc/col1/doc2", mockPayload(), "doc2", "test", true, true, "mydb")
	mockDoc.AddChildDocument("topDoc/col1/doc3", mockPayload(), "doc3", "test", true, true, "mydb")
	mockDoc.AddChildDocument("topDoc/col1/doc4", mockPayload(), "doc4", "test", true, true, "mydb")
	mockDoc.AddChildDocument("topDoc/col1/doc5", mockPayload(), "doc5", "test", true, true, "mydb")
	mockDoc.AddChildDocument("topDoc/col1/doc6", mockPayload(), "doc6", "test", true, true, "mydb")

	_, stat, _, _, _ := mockDoc.GetChildCollection("topDoc/col1", "a", "z", false)

	if stat != http.StatusOK {
		t.Errorf("GetChildCollectionMultipleDocsFailed: expected status code 200,got %d", stat)
	}

}

func TestDocument_GetChildCollectionLimitedRange(t *testing.T) {
	mockDoc := mockDocument()

	mockDoc.AddChildCollection("topDoc/col1", "mydb")

	mockDoc.AddChildDocument("topDoc/col1/a", mockPayload(), "a", "test", true, true, "mydb")
	mockDoc.AddChildDocument("topDoc/col1/b", mockPayload(), "b", "test", true, true, "mydb")
	mockDoc.AddChildDocument("topDoc/col1/c", mockPayload(), "c", "test", true, true, "mydb")
	mockDoc.AddChildDocument("topDoc/col1/d", mockPayload(), "d", "test", true, true, "mydb")
	mockDoc.AddChildDocument("topDoc/col1/e", mockPayload(), "e", "test", true, true, "mydb")

	_, stat, _, _, _ := mockDoc.GetChildCollection("topDoc/col1", "c", "e", false)

	if stat != http.StatusOK {
		t.Errorf("GetChildCollection Failed, expected 200, got %d", stat)
	}

}

func TestDocument_AddChildDocument(t *testing.T) {

	topDoc := mockDocument()

	topDoc.AddChildCollection("topDoc/col1", "db")

	topDoc.AddChildDocument("topDoc/col1/child", mockPayload(), "child", "USER", true, false, "db")

	_, stat, _, _, _ := topDoc.GetChildDocument("topDoc/col1/child", false, "")

	if stat != http.StatusOK {
		t.Errorf("Error in AddChildDocument, expected status code 200 but got %d", stat)
	}
}

func TestDocument_DeleteChildCollection(t *testing.T) {
	topDoc := mockDocument()

	topDoc.AddChildCollection("topDoc/col1", "db")

	_, stat := topDoc.DeleteChildCollection("topDoc/col1")

	if stat != http.StatusNoContent {
		t.Errorf("Delete Child Collection failed")
	}
}

func TestDocument_DeleteChildCollectionCollectionDoesntExist(t *testing.T) {
	topDoc := mockDocument()

	_, stat := topDoc.DeleteChildCollection("topDoc/col1/")

	if stat != http.StatusNotFound {
		t.Errorf("TestDocument_DeleteChildCollectionCollectionDoesntExist failed, expected 404 got %d", stat)
	}
}

func TestDocument_DeleteChildNestedCollectionCollectionDoesntExist(t *testing.T) {
	topDoc := mockDocument()

	_, stat := topDoc.DeleteChildCollection("topDoc/col1/doc2/col3/")

	if stat != http.StatusNotFound {
		t.Errorf("TestDocument_DeleteChildCollectionCollectionDoesntExist failed, expected 404 got %d", stat)
	}
}

func TestDocument_DeeplyNestedNotFoundDoc(t *testing.T) {
	topDoc := mockDocument()

	_, stat := topDoc.DeleteChildDocument("topDoc/col1/doc2/col3/doc4", "")

	if stat != http.StatusNotFound {
		t.Errorf("TestDocument_DeleteChildCollectionCollectionDoesntExist failed, expected 404 got %d", stat)
	}
}

func TestDocument_ApplyPatchDocumentNotFound(t *testing.T) {
	topDoc := mockDocument()

	_, stat := topDoc.ApplyPatchDocument("db", "doc1/col2/doc3", []byte("patch"), "user")

	if stat != http.StatusNotFound {
		t.Errorf("TestDocument_ApplyPatchDocument Failed,got status code %d", stat)
	}

}

func TestDocument_ApplyPatch(t *testing.T) {
	topDoc := mockDocument()
	topDoc.AddChildCollection("topDoc/col1", "db")
	topDoc.AddChildDocument("topDoc/col1/child", mockPayload(), "child", "USER", true, false, "db")
	_, stat := topDoc.ApplyPatchDocument("db", "topDoc/col1/child", []byte("patch"), "user")
	if stat != http.StatusOK {
		t.Errorf("ApplyPatch failed,got: %d", stat)
	}
}

func TestDocument_ApplyPatchNoDoc(t *testing.T) {
	topDoc := mockDocument()
	topDoc.AddChildCollection("topDoc/col1", "db")
	//topDoc.AddChildDocument("topDoc/col1/child", mockPayload(), "child", "USER", true, false, "db")
	_, stat := topDoc.ApplyPatchDocument("db", "topDoc/col1/child", []byte("patch"), "user")
	if stat != http.StatusNotFound {
		t.Errorf("ApplyPatch failed,got: %d", stat)
	}

}

func TestDocument_ApplyPatchNoCol(t *testing.T) {
	topDoc := mockDocument()
	//topDoc.AddChildCollection("topDoc/col1", "db")
	topDoc.AddChildDocument("topDoc/col1/child", mockPayload(), "child", "USER", true, false, "db")
	_, stat := topDoc.ApplyPatchDocument("db", "topDoc/col1/child", []byte("patch"), "user")
	if stat != http.StatusNotFound {
		t.Errorf("ApplyPatch failed,got: %d", stat)
	}
}

func TestDocument_DeleteChildDocument(t *testing.T) {
	topDoc := mockDocument()
	topDoc.AddChildCollection("topDoc/col1", "db")
	topDoc.AddChildDocument("topDoc/col1/child", mockPayload(), "child", "USER", true, false, "db")
	_, stat := topDoc.DeleteChildDocument("topDoc/col1/child", "mydb")
	if stat != http.StatusNoContent {
		t.Errorf("%d", stat)
	}
}

func TestDocument_DeleteChildNoDoc(t *testing.T) {
	topDoc := mockDocument()
	topDoc.AddChildCollection("topDoc/col1", "db")
	_, stat := topDoc.DeleteChildDocument("topDoc/col1/child", "mydb")
	if stat != http.StatusNotFound {
		t.Errorf("%d", stat)
	}
}

func TestDocument_DeleteChildNoCol(t *testing.T) {
	topDoc := mockDocument()

	_, stat := topDoc.DeleteChildDocument("topDoc/col1/child", "mydb")
	if stat != http.StatusNotFound {
		t.Errorf("%d", stat)
	}
}

// Tests that the code covering subscription requests is covered correctly
func TestDocument_AddChildDocumentSubscriberNotif(t *testing.T) {
	topDoc := mockDocument()
	topDoc.AddChildCollection("topDoc/col1", "db")

	topDoc.AddChildDocument("topDoc/col1/child", mockPayload(), "child", "USER", true, false, "db")
	topDoc.GetChildDocument("topDoc/col1/child", true, "")

}

// tests that the collection subscription request is covered correctly
func TestCollectionSubRequest(t *testing.T) {
	topDoc := mockDocument()
	topDoc.AddChildCollection("topDoc/col1", "db")
	topDoc.GetChildCollection("topDoc/col1", "", "", true)
}
