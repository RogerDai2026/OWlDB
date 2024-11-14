// Package subscriptionManager provides functionality for managing collection-level subscribers
// and sending server-sent events (SSE) to subscribers based on changes in the collection.
package subscriptionManager

import (
	"context"
	"fmt"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/index_utils"
	"log/slog"
	"strconv"
	"time"
)

// internal data structure for each subscriber
type Colsubscriber struct {
	ch *chan []byte //the channel on which to send events
	lo string       //the lower bound
	hi string       //the upper bound
}

// IdToCSub defines an interface for managing collection-level subscribers.
// It provides methods to remove, upsert, and query subscribers.
type IdToCSub[id string, sub Colsubscriber] interface {
	Remove(key string) (removedChan sub, removed bool)                                        // Removes a subscriber by ID.
	Upsert(key string, check index_utils.UpdateCheck[id, sub]) (updated bool, err error)      // Upserts (inserts or updates) a subscriber.
	Query(ctx context.Context, low id, upper id) (res []index_utils.Pair[id, sub], err error) // Queries subscribers within an ID range.
}

// ColSubscriptionManager manages the subscribers to a collection and tracks their channels.
type ColSubscriptionManager struct {
	subs IdToCSub[string, Colsubscriber] //active subscribers to a collection
	//an internal id mapper
}

// NewColSubManager creates a new ColSubscriptionManager with the provided subscriber management system.
func NewColSubManager(subs IdToCSub[string, Colsubscriber]) *ColSubscriptionManager {
	return &ColSubscriptionManager{
		subs: subs,
	}
}

// AddSubscriber adds a subscriber to a collection. Returns the channel on which it will send
// all future events
func (c *ColSubscriptionManager) AddSubscriber(lo string, hi string) (subChan *chan []byte, id string) {
	ch := make(chan []byte)
	cs := Colsubscriber{ch: &ch, lo: lo, hi: hi}

	check := func(key string, curV Colsubscriber, exists bool) (newV Colsubscriber, err error) {
		return cs, nil
	}
	subId := generateResourceName()
	c.subs.Upsert(subId, check)

	return &ch, subId
}

// Notify will take a document name and event (as a series of bytes), notify every collection subscriber
// listening on a range that contains a document.
func (c *ColSubscriptionManager) Notify(docname string, evType string, payload []byte) {
	slog.Debug(fmt.Sprintf("Notifying subscribers using about an update to %s", docname))
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()
	subs, _ := c.subs.Query(ctx, string(rune(0)), string(rune(127)))
	for _, v := range subs {
		lower, upper := v.Value.lo, v.Value.hi
		if lower <= docname && docname <= upper { //notify based on the ranges they are listening to
			*v.Value.ch <- c.GenerateEvent(evType, payload)
		}

	}
}

// NotifyAll sends a "delete" event to all subscribers of a collection.
// It notifies every active subscriber regardless of their specific range.
func (c *ColSubscriptionManager) NotifyAll(colname string) {
	slog.Debug(fmt.Sprintf("Notifying subscribers using about an update to %s", colname))
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()
	subs, _ := c.subs.Query(ctx, string(rune(0)), string(rune(127)))
	for _, v := range subs {

		//notify based on the ranges they are listening to
		*v.Value.ch <- c.GenerateEvent("delete", []byte(colname))

	}
}

// Remove removes a subscriber from the collection
// It closes the channel associated with the subscriber
func (c *ColSubscriptionManager) Remove(id string) {

	slog.Debug(fmt.Sprintf("Removing subscriber with id %s \n]]", id))
	_, removed := c.subs.Remove(id)
	if removed == false {
		slog.Warn(fmt.Sprintf("Warning: a removal of a subscriber was unsuccessful"))
		return
	}

}

// GenerateEvent creates a formatted server-sent event (SSE) message.
// The event type can be "update" or "delete", and the event data is included in the payload.
func (c *ColSubscriptionManager) GenerateEvent(evType string, payload []byte) []byte {
	res := ""
	if evType == "update" {
		res += "event: update\n"
	} else if evType == "delete" {
		res += "event: delete\n"
	}
	res += "data: " + string(payload) + "\n"
	res += "id: " + strconv.FormatInt(time.Now().UnixMilli(), 10)
	res += "\n\n"
	return []byte(res)
}
