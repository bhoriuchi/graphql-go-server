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

// Subscription interface between ws and graphql execution
type Subscription struct {
	Channel      chan *graphql.Result
	ConnectionID string
	OperationID  string
	Context      context.Context
	CancelFunc   context.CancelFunc
}

// unsubscribe unsubscribes
func (s *Subscription) unsubscribe() {
	if s.CancelFunc != nil {
		s.CancelFunc()
	}
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

	if _, ok := m.subscriptions[sub.OperationID]; ok {
		return fmt.Errorf("subscriber for %s already exists", sub.OperationID)
	}

	m.subscriptions[sub.OperationID] = sub
	return nil
}

// Unsubscribe removes a single operation
func (m *Manager) Unsubscribe(operationID string) {
	m.mx.Lock()
	defer m.mx.Unlock()

	sub, ok := m.subscriptions[operationID]
	if ok {
		sub.unsubscribe()
		delete(m.subscriptions, operationID)
	}
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
