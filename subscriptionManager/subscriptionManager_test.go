package subscriptionManager

import (
	"fmt"

	"github.com/RICE-COMP318-FALL24/owldb-p1group24/mocks"
	"regexp"
	"testing"
	"time"
)

func TestSubscriptionManager_AddSubscriber(t *testing.T) {

	sm := New(mocks.NewMockSL[string, *chan []byte]())

	subChan, _ := sm.AddSubscriber()
	if subChan == nil {
		t.Errorf("AddSubscriber failed, no channel was give")
	}
}

func TestSubscriptionManager_Notify(t *testing.T) {
	sm := New(mocks.NewMockSL[string, *chan []byte]())
	subChan, _ := sm.AddSubscriber()
	if subChan == nil {
		t.Errorf("AddSubscriber failed, no channel was give")
	}
	go sm.Notify("update", []byte("payload"))

	select {
	case val := <-*subChan:
		fmt.Printf(string(val))
		return
	}
}

func TestSubscriptionManager_RemoveSubscriber(t *testing.T) {
	sm := New(mocks.NewMockSL[string, *chan []byte]())
	subChan, id := sm.AddSubscriber()
	if subChan == nil {
		t.Errorf("AddSubscriber failed, no channel was give")
	}
	sm.RemoveSubscriber(id)

	if _, ok := <-*subChan; ok {
		t.Errorf("Remove Subscriber failed; channel was not closed")
	}

}

func TestSubscriptionManager_GenerateEvent(t *testing.T) {
	sm := New(mocks.NewMockSL[string, *chan []byte]())

	eventUpdate := sm.GenerateEvent("update", []byte("payload"))

	regex := regexp.MustCompile(`^(event: .+\ndata: .+\nid: )\d+`)
	matches := regex.MatchString(string(eventUpdate))
	if !matches {
		t.Errorf("GenerateEvent Failed: malformed SSE")
	}

	eventDelete := sm.GenerateEvent("delete", []byte("payload"))
	matches = regex.MatchString(string(eventDelete))
	if !matches {
		t.Errorf("GenerateEvent failed: malformed SSE")
	}
}

func TestColSubscriptionManager_AddSubscriber(t *testing.T) {
	subs := mocks.NewMockSL[string, Colsubscriber]()
	sm := NewColSubManager(subs)
	ch, _ := sm.AddSubscriber("a", "d")
	if ch == nil {
		t.Errorf("TestColSubscriptionManager failed")
	}

}

func TestColSubscriptionManager_Notify(t *testing.T) {
	subs := mocks.NewMockSL[string, Colsubscriber]()
	sm := NewColSubManager(subs)

	chanAD, _ := sm.AddSubscriber("a", "d")

	chanBG, _ := sm.AddSubscriber("b", "g")

	go sm.Notify("a", "update", []byte("payload"))
	for {
		select {

		case val := <-*chanBG:
			fmt.Println(string(val))
			t.Errorf("ColSubscription failed, listener not in range ")

		case val := <-*chanAD:
			fmt.Println(string(val))
			continue

		case <-time.After(1 * time.Second):
			return
		}
	}

}

func TestMessager_AddDocSubscriber(t *testing.T) {

	sl := mocks.NewMockSL[string, *SubscriptionManager]()
	var idtosubfactory IdToSubFactory = func() IdToSub[string, *chan []byte] {
		return mocks.NewMockSL[string, *chan []byte]()
	}
	messager := NewMessager(idtosubfactory, sl)
	ch, _ := messager.AddDocSubscriber("db/doc1")
	if ch == nil {
		t.Errorf("AddDocSubscriber failed, channel returned was nil")
	}
}

func TestMessager_AddMultipleDocSubscribers(t *testing.T) {
	sl := mocks.NewMockSL[string, *SubscriptionManager]()
	var idtosubfactory IdToSubFactory = func() IdToSub[string, *chan []byte] {
		return mocks.NewMockSL[string, *chan []byte]()
	}
	messager := NewMessager(idtosubfactory, sl)
	ch, id1 := messager.AddDocSubscriber("db/doc1")
	if ch == nil {
		t.Errorf("AddDocSubscriber failed, channel returned was nil")
	}

	ch2, id2 := messager.AddDocSubscriber("db/doc1")
	if ch == ch2 {
		t.Errorf("AddMultipleSubscriber failed, got the same channel twice")
	}
	if id1 == id2 {
		t.Errorf("AddMultipleSubscriber failed, got duplicate ids")
	}
}

func TestMessager_NotifyDocs(t *testing.T) {
	sl := mocks.NewMockSL[string, *SubscriptionManager]()
	var idtosubfactory IdToSubFactory = func() IdToSub[string, *chan []byte] {
		return mocks.NewMockSL[string, *chan []byte]()
	}
	messager := NewMessager(idtosubfactory, sl)
	ch, _ := messager.AddDocSubscriber("db/doc1")
	if ch == nil {
		t.Errorf("AddDocSubscriber failed, channel returned was nil")
	}

	go messager.NotifyDocs("db/doc1", "update", []byte("payload"))

	select {
	case val := <-*ch:
		regex := regexp.MustCompile(`^(event: .+\ndata: .+\nid: )\d+`)
		matches := regex.MatchString(string(val))
		if !matches {
			t.Errorf("GenerateEvent Failed: malformed SSE")
		}
		return
	}
}
