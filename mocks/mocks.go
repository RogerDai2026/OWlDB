package mocks

import (
	"cmp"
	"context"
	"encoding/json"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/index_utils"
	"sync"
)

type dummyPayload struct {
	Name     string
	Message  string
	Likes    int
	Dislikes int
}

func MockPayload() []byte {
	messageStruct := dummyPayload{
		Name:     "John",
		Message:  "Hello",
		Likes:    30,
		Dislikes: -5000,
	}
	b, _ := json.Marshal(messageStruct)
	return b
}

type mockMeta struct {
}

type MockSL[K cmp.Ordered, V any] struct {
	sl map[K]V
	mu sync.Mutex
}

func NewMockSL[K cmp.Ordered, V any]() *MockSL[K, V] {
	return &MockSL[K, V]{sl: make(map[K]V)}
}

func (m *MockSL[K, V]) Find(key K) (foundValue V, found bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.sl[key]
	if !ok {
		return v, false
	}
	return v, true
}

func (m *MockSL[K, V]) Remove(key K) (foundValue V, found bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.sl[key]
	if !ok {
		return v, false
	}
	delete(m.sl, key)
	return v, true
}

func (m *MockSL[K, V]) Query(ctx context.Context, low K, hi K) (result []index_utils.Pair[K, V], err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := make([]index_utils.Pair[K, V], 0)
	for k, v := range m.sl {
		if low <= k && k <= hi {
			p := index_utils.Pair[K, V]{Key: k, Value: v}
			res = append(res, p)
		}
	}
	return res, nil
}

func (m *MockSL[K, V]) Upsert(key K, check index_utils.UpdateCheck[K, V]) (bool, error) {
	curVal, exists := m.sl[key]
	m.mu.Lock()
	defer m.mu.Unlock()
	newVal, err := check(key, curVal, exists)
	if err != nil {
		return false, err
	}
	m.sl[key] = newVal
	return true, nil

}
