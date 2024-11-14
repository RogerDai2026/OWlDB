// Package subscriptionManager provides a system for managing subscribers and sending them server-sent events (SSE).
// It supports adding, removing, and notifying subscribers, as well as generating formatted SSE messages.
package subscriptionManager

import (
	"context"
	"fmt"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/index_utils"
	"log/slog"
	"math/rand/v2"
	"strconv"
	"time"
)

// IdToSub defines an interface for managing subscriber channels.
// It includes methods for finding, removing, and upserting subscribers, as well as querying a range of subscribers.
type IdToSub[id string, ch *chan []byte] interface {
	Find(key string) (foundValue ch, found bool)                                             // Find retrieves a subscriber by their ID.
	Remove(key string) (removedChan ch, removed bool)                                        // Remove deletes a subscriber by their ID.
	Upsert(key string, check index_utils.UpdateCheck[id, ch]) (updated bool, err error)      // Upsert inserts or updates a subscriber's channel.
	Query(ctx context.Context, low id, upper id) (res []index_utils.Pair[id, ch], err error) // Query retrieves subscribers within a range of IDs.
}

// New creates a new SubscriptionManager and initializes its subscriber management system.
func New(subs IdToSub[string, *chan []byte]) *SubscriptionManager {
	return &SubscriptionManager{
		idCounter: 1,
		subs:      subs}
}

// SubscriptionManager is responsible for managing subscribers and their channels.
// It tracks subscribers, allows for adding and removing them, and notifies them of events via SSE.
type SubscriptionManager struct {
	idCounter int                           // Tracks the next subscriber ID.
	subs      IdToSub[string, *chan []byte] // Manages the channels for subscribers.
}

// AddSubscriber adds a new subscriber to the subscription manager.
// It creates a new channel for the subscriber, assigns an ID, and returns the channel and ID.
func (s *SubscriptionManager) AddSubscriber() (*chan []byte, string) {
	slog.Debug(fmt.Sprintf("Current subscriber count is %d", s.idCounter))
	newCh := make(chan []byte)
	slog.Debug("Hello from s.Addsucrbiber")
	chk := func(key string, curV *chan []byte, exists bool) (ch *chan []byte, err error) {
		return &newCh, nil
	}

	id := generateResourceName()
	s.subs.Upsert(id, chk)

	slog.Debug(fmt.Sprintf("Current subscriber count is %d", s.idCounter))

	return &newCh, id
}

// RemoveSubscriber removes a subscriber with id id
// It closes the channel associated with the subscriber
func (s *SubscriptionManager) RemoveSubscriber(id string) {
	slog.Debug(fmt.Sprintf("Removing a subscriber whose id is %s", id))
	removedChan, _ := s.subs.Remove(id)

	ch := *removedChan
	close(ch)
}

// Notify will send every subscriber an SSE of type evType, with payload byte
func (s *SubscriptionManager) Notify(evType string, payload []byte) {
	slog.Debug(fmt.Sprintf("DOCSUB MANAGER,payload is %s", string(payload)))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	slog.Debug(fmt.Sprintf("Current sub counter is %d", s.idCounter))
	subscribers, _ := s.subs.Query(ctx, string(rune(0)), string(rune(127))) //we need to notify everyone

	for _, sub := range subscribers {
		slog.Debug("Notifying subscriber")
		v := sub.Value
		*v <- s.GenerateEvent(evType, payload)

	}
}

// GenerateEvent generates a formatted sse of type evtype, with payload payload
func (s *SubscriptionManager) GenerateEvent(evtype string, payload []byte) []byte {

	res := ""
	if evtype == "update" {
		res += "event: update\n"
	} else if evtype == "delete" {
		res += "event: delete\n"
	}
	res += "data: " + string(payload) + "\n"
	res += "id: " + strconv.FormatInt(time.Now().UnixMilli(), 10)
	res += "\n\n"
	return []byte(res)
}

// generateResourceName is an internal routine to generate the name of a resource
func generateResourceName() string {
	var name string
	validChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-.~"
	leng := 12
	for i := 0; i < leng; i++ {
		ch := rand.IntN(len(validChars))
		name += string(validChars[ch])
	}
	return name

}
