package store

import (
	"context"
	"sync"
	"time"
)

type Memory struct {
	mu        sync.Mutex
	nextID    int64
	users     map[int64]*User
	byClerkID map[string]int64
	payments  map[string]bool
	checkouts map[string]*CheckoutSession
	webhooks  map[string]bool
}

var _ Store = (*Memory)(nil)

func NewMemory() *Memory {
	return &Memory{
		users:     map[int64]*User{},
		byClerkID: map[string]int64{},
		payments:  map[string]bool{},
		checkouts: map[string]*CheckoutSession{},
		webhooks:  map[string]bool{},
	}
}

func (m *Memory) Ping(context.Context) error { return nil }
func (m *Memory) Close()                     {}

func (m *Memory) UserByClerkID(_ context.Context, clerkID string) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byClerkID[clerkID]
	if !ok {
		return User{}, ErrNotFound
	}
	return *m.users[id], nil
}

func (m *Memory) CreateUser(_ context.Context, u User) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id, ok := m.byClerkID[u.ClerkID]; ok {
		return *m.users[id], nil
	}
	m.nextID++
	u.ID = m.nextID
	m.users[u.ID] = &u
	m.byClerkID[u.ClerkID] = u.ID
	return u, nil
}

func (m *Memory) SetUserEmail(_ context.Context, userID int64, email string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[userID]
	if !ok {
		return ErrNotFound
	}
	u.Email = email
	return nil
}

func (m *Memory) TouchUser(_ context.Context, userID int64, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.users[userID]; ok {
		u.LastSeenAt = at
	}
	return nil
}

func (m *Memory) SpendTrial(_ context.Context, userID int64, c Counter, n, limit int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[userID]
	if !ok {
		return 0, ErrNotFound
	}
	counter := counterOf(u, c)
	if *counter >= limit {
		return 0, ErrTrialSpent
	}
	granted := min(n, limit-*counter)
	*counter += granted
	return granted, nil
}

func (m *Memory) RefundTrial(_ context.Context, userID int64, c Counter, n int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.users[userID]; ok {
		counter := counterOf(u, c)
		*counter = max(0, *counter-n)
	}
	return nil
}

func (m *Memory) CreateCheckoutSession(_ context.Context, s CheckoutSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.users[s.UserID]; !ok {
		return ErrNotFound
	}
	m.checkouts[s.ID] = &s
	return nil
}

func (m *Memory) SetCheckoutProviderID(_ context.Context, id, providerSessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.checkouts[id]
	if !ok {
		return ErrNotFound
	}
	s.ProviderSessionID = &providerSessionID
	return nil
}

func (m *Memory) CheckoutSession(_ context.Context, id string) (CheckoutSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.checkouts[id]
	if !ok {
		return CheckoutSession{}, ErrNotFound
	}
	return *s, nil
}

func (m *Memory) RecordPayment(_ context.Context, p Payment) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.webhooks[p.WebhookID] {
		return false, ErrDuplicate
	}
	m.webhooks[p.WebhookID] = true
	if m.payments[p.ProviderPaymentID] {
		return true, nil
	}
	m.payments[p.ProviderPaymentID] = true
	u := m.payer(p)
	if s, ok := m.checkouts[p.CheckoutSessionID]; ok {
		at := p.At
		s.Status, s.PaidAt = CheckoutPaid, &at
	}
	if u == nil {
		return false, nil
	}
	if u.LicensedAt == nil {
		at := p.At
		u.LicensedAt = &at
	}
	return true, nil
}

func (m *Memory) payer(p Payment) *User {
	if s, ok := m.checkouts[p.CheckoutSessionID]; ok {
		return m.users[s.UserID]
	}
	if id, ok := m.byClerkID[p.ClerkID]; ok && p.ClerkID != "" {
		return m.users[id]
	}
	return nil
}

func counterOf(u *User, c Counter) *int {
	if c == CounterVoice {
		return &u.VoiceUsed
	}
	return &u.ChecksUsed
}
