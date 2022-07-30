package manager

import (
	"context"
	"fmt"
	"sync"

	"github.com/graphql-go/graphql"
)

// Manager manages subscriptions
type Manager struct {
	mx            sync.RWMutex
	subscriptions map[string]*Subscription
}

func NewManager() *Manager {
	return &Manager{
		subscriptions: map[string]*Subscription{},
	}
}

// Subscription interface between ws and graphql execution
type Subscription struct {
	IsSub         bool
	Channel       chan *graphql.Result
	ConnectionID  string
	OperationID   string
	OperationName string
	Context       context.Context
	CancelFunc    context.CancelFunc
}

// unsubscribe unsubscribes
func (s *Subscription) unsubscribe() {
	if s.CancelFunc != nil {
		s.CancelFunc()
	}
}

// SubscriptionCount counts all or specific connection id subscriptions
// can be used for diagnostics
func (m *Manager) SubscriptionCount(connectionIDs ...string) int {
	m.mx.RLock()
	defer m.mx.RUnlock()

	if len(connectionIDs) == 0 {
		return len(m.subscriptions)
	}

	idmap := map[string]interface{}{}
	for _, id := range connectionIDs {
		idmap[id] = nil
	}

	count := 0
	for _, sub := range m.subscriptions {
		if _, ok := idmap[sub.ConnectionID]; ok {
			count++
		}
	}

	return count
}

// HasSubscription returns true if the subscription exists
func (m *Manager) HasSubscription(operationID string) bool {
	m.mx.RLock()
	defer m.mx.RUnlock()

	_, ok := m.subscriptions[operationID]
	return ok
}

// Subscribe performs a subscribe operation
func (m *Manager) Subscribe(sub *Subscription) error {
	m.mx.Lock()
	defer m.mx.Unlock()

	if current, ok := m.subscriptions[sub.OperationID]; ok {
		if current.IsSub || (!current.IsSub && !sub.IsSub) {
			return fmt.Errorf("subscriber for %q already exists", sub.OperationID)
		}
	}

	m.subscriptions[sub.OperationID] = sub
	return nil
}

// Unsubscribe removes a single operation
func (m *Manager) Unsubscribe(operationID string) *Subscription {
	m.mx.Lock()
	defer m.mx.Unlock()

	sub, ok := m.subscriptions[operationID]
	if ok {
		sub.unsubscribe()
		delete(m.subscriptions, operationID)
	}

	return sub
}

// Unsubscribe all unsubscribes and removes all operations for a specific connection id
func (m *Manager) UnsubscribeAll() {
	m.mx.Lock()
	defer m.mx.Unlock()

	for _, sub := range m.subscriptions {
		sub.unsubscribe()
	}

	m.subscriptions = map[string]*Subscription{}
}
