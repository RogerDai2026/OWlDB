package subscriptionManager

import (
	"fmt"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/index_utils"
	"log/slog"
)

type UriToDocs[uri string, docSubs *SubscriptionManager] interface {
	Upsert(key string, check index_utils.UpdateCheck[uri, docSubs]) (updated bool, err error) // Upserts (inserts or updates) a subscriber.
	Find(key string) (foundDsm *SubscriptionManager, found bool)
}

// Messager contains all active subscriptions
type Messager struct {
	idtosubfactory IdToSubFactory

	docSubs UriToDocs[string, *SubscriptionManager]
}

// IdToSubFactory is a factory function for
type IdToSubFactory func() IdToSub[string, *chan []byte]

func NewMessager(idtosubfactory IdToSubFactory, docsubs UriToDocs[string, *SubscriptionManager]) *Messager {
	return &Messager{
		idtosubfactory: idtosubfactory,
		docSubs:        docsubs,
	}
}

// AddDocSubscriber adds a subscriber to a resource located at uri. It returns the initial sse sent to the client,
// alongside a channel to listen on
func (m *Messager) AddDocSubscriber(uri string) (*chan []byte, string) {
	var resChan *chan []byte
	var id string
	slog.Debug(fmt.Sprintf("Adding a subscriber to the doc at uri %s", uri))
	check := func(uri string, curDsm *SubscriptionManager, exists bool) (newDsm *SubscriptionManager, err error) {
		if exists {

			resChan, id = curDsm.AddSubscriber()
			return curDsm, nil
		} else {

			idtosub := m.idtosubfactory()
			newDsm = New(idtosub)
			resChan, id = newDsm.AddSubscriber()
		}
		return newDsm, nil
	}
	m.docSubs.Upsert(uri, check)
	return resChan, id
}

// NotifyDocs notifies all subscribers to a document about an update
func (m *Messager) NotifyDocs(uri string, evtype string, payload []byte) {
	dsm, found := m.docSubs.Find(uri)

	if found {
		slog.Debug("Found, going to notify now!")
		dsm.Notify(evtype, payload)
	}
}
