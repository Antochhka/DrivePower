package service

import "sync"

// TransactionContext keeps runtime info for a transaction.
type TransactionContext struct {
	SessionID int64
	MeterStart int64
}

// TransactionStore stores contexts by transaction ID.
type TransactionStore struct {
	mu   sync.RWMutex
	data map[string]TransactionContext
}

// NewTransactionStore returns initialized store.
func NewTransactionStore() *TransactionStore {
	return &TransactionStore{
		data: make(map[string]TransactionContext),
	}
}

// Set stores context for transaction.
func (s *TransactionStore) Set(txID string, ctx TransactionContext) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[txID] = ctx
}

// Get returns context and bool.
func (s *TransactionStore) Get(txID string) (TransactionContext, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ctx, ok := s.data[txID]
	return ctx, ok
}

// Delete removes transaction context.
func (s *TransactionStore) Delete(txID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, txID)
}
