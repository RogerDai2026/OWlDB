package document

import (
	"context"
	"encoding/json"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/index_utils"
	"net/http"
)

// CollectionIndex encapsulates the behaviors needed for the Collection indices to function properly
// It is a generic interface that can be used to index any type of collection
type CollectionIndex[K string, V *Document] interface {
	Find(key K) (foundValue V, found bool)
	Query(ctx context.Context, low K, hi K) ([]index_utils.Pair[K, V], error)
	Upsert(key K, check index_utils.UpdateCheck[K, V]) (updated bool, err error)
	Remove(key K) (removedVal V, removed bool)
}

// ColSubscriptionManager is responsible for managing the subscriptions of a given collection. We inject it's
// implementation in main, so for now our Collection delegates to this interface
type ColSubscriptionManager interface {
	AddSubscriber(lo string, hi string) (subChan *chan []byte, id string) // Adds a subscriber
	Remove(id string)                                                     // Removes a subscriber
	Notify(docname string, evType string, payload []byte)                 // Notifies a subscriber of a change
	NotifyAll(colname string)
	GenerateEvent(evtype string, payload []byte) []byte // Notifies all subscribers of a change
}

// Collection encapsulates the functionalities associated with Collections
// We consider collections to be an internal property of documents
type Collection struct {
	Name                string                             //the name of the collection
	Docs                CollectionIndex[string, *Document] //the indices to other documents in the collection
	SubscriptionManager ColSubscriptionManager             //manages the subscriptions for a collection
}

// CSerialize serializes the documents of a collection whose name lies in the range [lo,hi].
// Returns a serialized representation of the collection, and a status code
func (c *Collection) CSerialize(ctx context.Context, lo string, hi string) ([]byte, int) {
	res, err := c.Docs.Query(ctx, lo, hi)
	if err != nil {
		errmsg, _ := json.Marshal(err.Error())
		return errmsg, http.StatusBadRequest
	}
	resArr := []json.RawMessage{}
	for _, pair := range res {
		bytes := pair.Value.GetSerial()
		var rj json.RawMessage = bytes
		resArr = append(resArr, rj)
	}
	resSerial, _ := json.Marshal(resArr)
	return resSerial, http.StatusOK
}

// Creates a new collection
type Factory func(colName string) *Collection
