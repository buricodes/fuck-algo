package store

import (
	"context"
	"sync"
	"time"
)

type Memory struct {
	mu        sync.Mutex
	installs  map[string]*Install
	byToken   map[string]string
	licences  map[string]*Licence
	byKey     map[string]string
	payments  map[string]bool
	checkouts map[string]*CheckoutSession
	webhooks  map[string]bool
}

var _ Store = (*Memory)(nil)

func NewMemory() *Memory {
	return &Memory{
		installs:  map[string]*Install{},
		byToken:   map[string]string{},
		licences:  map[string]*Licence{},
		byKey:     map[string]string{},
		payments:  map[string]bool{},
		checkouts: map[string]*CheckoutSession{},
		webhooks:  map[string]bool{},
	}
}

func (m *Memory) Ping(context.Context) error { return nil }
func (m *Memory) Close()                     {}

func (m *Memory) CreateInstall(_ context.Context, in Install) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.installs[in.ID] = &in
	m.byToken[in.TokenHash] = in.ID
	return nil
}

func (m *Memory) InstallByTokenHash(_ context.Context, tokenHash string) (Install, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byToken[tokenHash]
	if !ok {
		return Install{}, ErrNotFound
	}
	return m.joined(m.installs[id]), nil
}

func (m *Memory) InstallExists(_ context.Context, id string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.installs[id]
	return ok, nil
}

func (m *Memory) TouchInstall(_ context.Context, id string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if in, ok := m.installs[id]; ok {
		in.LastSeenAt = at
	}
	return nil
}

func (m *Memory) SpendTrial(_ context.Context, installID string, c Counter, n, limit int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	in, ok := m.installs[installID]
	if !ok {
		return 0, ErrNotFound
	}
	counter := counterOf(in, c)
	if *counter >= limit {
		return 0, ErrTrialSpent
	}
	granted := min(n, limit-*counter)
	*counter += granted
	return granted, nil
}

func (m *Memory) RefundTrial(_ context.Context, installID string, c Counter, n int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if in, ok := m.installs[installID]; ok {
		counter := counterOf(in, c)
		*counter = max(0, *counter-n)
	}
	return nil
}

func (m *Memory) LicenceByKeyHash(_ context.Context, keyHash string) (Licence, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byKey[keyHash]
	if !ok {
		return Licence{}, ErrNotFound
	}
	return *m.licences[id], nil
}

func (m *Memory) AttachLicence(_ context.Context, installID, licenceID string, maxInstalls int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.attach(installID, licenceID, maxInstalls)
}

func (m *Memory) CreateCheckoutSession(_ context.Context, s CheckoutSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
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

func (m *Memory) ClearRevealKeys(_ context.Context, createdBefore time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.checkouts {
		if s.CreatedAt.Before(createdBefore) {
			s.RevealKey = nil
		}
	}
	return nil
}

func (m *Memory) RecordPayment(_ context.Context, p Payment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.webhooks[p.WebhookID] {
		return ErrDuplicate
	}
	m.webhooks[p.WebhookID] = true
	if m.payments[p.ProviderPaymentID] {
		return nil
	}
	m.payments[p.ProviderPaymentID] = true
	m.licences[p.LicenceID] = &Licence{ID: p.LicenceID, KeyHash: p.KeyHash, KeyHint: p.KeyHint, Status: LicenceStatusActive, CreatedAt: p.At}
	m.byKey[p.KeyHash] = p.LicenceID
	if s, ok := m.checkouts[p.CheckoutSessionID]; ok {
		at, licenceID, key := p.At, p.LicenceID, p.RevealKey
		s.Status, s.LicenceID, s.RevealKey, s.PaidAt = CheckoutPaid, &licenceID, &key, &at
	}
	if _, ok := m.installs[p.InstallID]; ok {
		return m.attach(p.InstallID, p.LicenceID, p.MaxInstalls)
	}
	return nil
}

func (m *Memory) attach(installID, licenceID string, maxInstalls int) error {
	in, ok := m.installs[installID]
	if !ok {
		return ErrNotFound
	}
	if in.LicenceID != nil && *in.LicenceID == licenceID {
		return nil
	}
	taken := 0
	for _, other := range m.installs {
		if other.LicenceID != nil && *other.LicenceID == licenceID {
			taken++
		}
	}
	if taken >= maxInstalls {
		return ErrLicenceFull
	}
	in.LicenceID = &licenceID
	return nil
}

func (m *Memory) joined(in *Install) Install {
	out := *in
	if in.LicenceID == nil {
		return out
	}
	if l, ok := m.licences[*in.LicenceID]; ok {
		hint := l.KeyHint
		out.KeyHint = &hint
		out.LicenceActive = l.Status == LicenceStatusActive
	}
	return out
}

func counterOf(in *Install, c Counter) *int {
	if c == CounterVoice {
		return &in.VoiceUsed
	}
	return &in.ChecksUsed
}
