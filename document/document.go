// Package document defines the structure and behaviors of documents within the database.
// It includes document management, collection indexing, validation, patching, and subscription handling.
package document

import (
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/index_utils"
	"time"
)

// DocumentIndex defines the interface for managing collections within a document.
// It provides methods to find, upsert, and remove collections.
type DocumentIndex[Idx string, Col *Collection] interface {
	Find(key Idx) (foundVal Col, found bool)
	Upsert(key Idx, check index_utils.UpdateCheck[Idx, Col]) (updated bool, err error)
	Remove(key Idx) (removedVal Col, removed bool)
}

// Messager defines the interface for managing subscriptions within a document.
// It provides methods to notify documents and collections about updates
type Messager interface {
	AddDocSubscriber(uri string) (*chan []byte, string) //Adds a subscriber to the resource at doc

	NotifyDocs(uri string, evtype string, payload []byte)
}

// Validator defines an interface for validating documents.
type Validator interface {
	Validate(b []byte) error
}

// DocumentIndexFactory DocumentIndex holds collections
type DocumentIndexFactory[T DocumentIndex[string, *Collection]] func() T

// CollectionFactory is a function type that creates a new collection
type CollectionFactory func(string) *Collection

// SubscriptionManager manages subscriptions for changes to documents.
// It provides methods to add and remove subscribers and notify them of changes.
type SubscriptionManager interface {
	AddSubscriber() (ch *chan []byte, id string)        // Adds a new subscriber and returns a channel and subscriber ID.
	RemoveSubscriber(id string)                         // Removes a subscriber by its ID.
	Notify(evType string, payload []byte)               // Notifies subscribers of a document event (e.g., update or delete).
	GenerateEvent(evType string, payload []byte) []byte // Generates an event message for a given event type and payload.
}

// Patcher defines an interface for applying patches to documents.
type Patcher interface {
	DoPatch(oldDoc []byte, patches []byte) (newDoc []byte, err error) // DoPatch applies a patch to the old document and returns the new document.
}

// Document defines the structure of documents within the database.
type Document struct {
	Info docInfo `json:"info"` //the document's info; exposed as a JSON via API calls

	collectionFactory CollectionFactory //a factory function enabling the document to create new collections

	docCollectionFactory DocumentIndexFactory[DocumentIndex[string, *Collection]] //a factory enabling the user to inject their own data structures for holding other collection objects

	collections DocumentIndex[string, *Collection] //the collections belonging to a document

	sm SubscriptionManager //sm contains an interface SubscriptionManager, enabling the delegation of subscription management to any concrete implementation abiding by the method signatures

	smFactory SubscriptionManagerFactory //smFactory is a factory function that creates a new SubscriptionManager

	validator Validator // Validator for checking document contents.

	patcher Patcher // Patcher for applying modifications to the document.

	messager Messager //Messager for handling subscriptions
}

// metadata is a struct that holds information about the document
type metadata struct {
	CreatedBy      string `json:"createdBy"`      //the user who created the document
	CreatedAt      int64  `json:"createdAt"`      //the time the document was created
	LastModifiedBy string `json:"lastModifiedBy"` //the user who last modified the document
	LastModifiedAt int64  `json:"lastModifiedAt"` //the time the document was last modified
}

// docInfo contains the document's path, metadata, and the actual contents of the document.
type docInfo struct {
	Path string   `json:"path"` // The document's path.
	Meta metadata `json:"meta"` // Metadata associated with the document.
	Doc  []byte   `json:"doc"`  // The document's actual contents in byte form.
}

// newMeta creates a new metadata object with the current timestamp and the provided user.
// It sets both the creation and last modification timestamps to the current time.
func newMeta(user string) metadata {
	return metadata{
		CreatedBy:      user,
		CreatedAt:      time.Now().UnixMilli(),
		LastModifiedBy: user,
		LastModifiedAt: time.Now().UnixMilli(),
	}
}

// New creates a new document object
// It initializes the document with the provided payload, user, path, and dependencies.
func New(payload []byte, user string, path string, dcf DocumentIndexFactory[DocumentIndex[string, *Collection]], ccf CollectionFactory, smfactory SubscriptionManagerFactory, v Validator, patcher Patcher, messager Messager) *Document {

	return &Document{

		Info: docInfo{Doc: payload, Meta: newMeta(user), Path: "/" + path},

		docCollectionFactory: dcf,

		collections: dcf(),

		sm:                smfactory(),
		collectionFactory: ccf,
		smFactory:         smfactory,

		patcher:   patcher,
		validator: v,
		messager:  messager,
	}
}
