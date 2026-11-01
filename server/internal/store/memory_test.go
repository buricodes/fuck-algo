package store

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSpendTrial(t *testing.T) {
	tests := []struct {
		name     string
		used     int
		spend    int
		limit    int
		want     int
		wantErr  error
		wantUsed int
	}{
		{"first use", 0, 1, 5, 1, nil, 1},
		{"last use", 4, 1, 5, 1, nil, 5},
		{"exhausted", 5, 1, 5, 0, ErrTrialSpent, 5},
		{"over the limit already", 7, 1, 5, 0, ErrTrialSpent, 7},
		{"batch fits", 10, 25, 300, 25, nil, 35},
		{"batch partly granted", 298, 25, 300, 2, nil, 300},
		{"zero limit", 0, 1, 0, 0, ErrTrialSpent, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMemory()
			ctx := context.Background()
			_ = m.CreateInstall(ctx, Install{ID: "i1", TokenHash: "h1", ChecksUsed: tt.used, VoiceUsed: tt.used})
			for _, c := range []Counter{CounterVoice, CounterChecks} {
				got, err := m.SpendTrial(ctx, "i1", c, tt.spend, tt.limit)
				if got != tt.want || !errors.Is(err, tt.wantErr) {
					t.Fatalf("counter %d: SpendTrial() = %d, %v; want %d, %v", c, got, err, tt.want, tt.wantErr)
				}
				in, _ := m.InstallByTokenHash(ctx, "h1")
				if in.Used(c) != tt.wantUsed {
					t.Fatalf("counter %d: used = %d, want %d", c, in.Used(c), tt.wantUsed)
				}
			}
		})
	}
}

func TestSpendTrialConcurrent(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	_ = m.CreateInstall(ctx, Install{ID: "i1", TokenHash: "h1"})
	var granted atomic.Int32
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			if n, err := m.SpendTrial(ctx, "i1", CounterVoice, 1, 5); err == nil {
				granted.Add(int32(n))
			}
		})
	}
	wg.Wait()
	if granted.Load() != 5 {
		t.Fatalf("granted %d voice uses, want exactly 5", granted.Load())
	}
}

func TestRefundTrialNeverGoesNegative(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	_ = m.CreateInstall(ctx, Install{ID: "i1", TokenHash: "h1", ChecksUsed: 2})
	_ = m.RefundTrial(ctx, "i1", CounterChecks, 5)
	in, _ := m.InstallByTokenHash(ctx, "h1")
	if in.ChecksUsed != 0 {
		t.Fatalf("checks used = %d, want 0", in.ChecksUsed)
	}
}

func TestAttachLicenceCap(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	_ = m.RecordPayment(ctx, Payment{WebhookID: "w1", ProviderPaymentID: "p1", LicenceID: "l1", KeyHash: "k", KeyHint: "ABCD", At: time.Now()})
	for i, id := range []string{"a", "b", "c"} {
		_ = m.CreateInstall(ctx, Install{ID: id, TokenHash: id})
		err := m.AttachLicence(ctx, id, "l1", 2)
		if want := i == 2; errors.Is(err, ErrLicenceFull) != want {
			t.Fatalf("install %s: err = %v, want full = %v", id, err, want)
		}
	}
	if err := m.AttachLicence(ctx, "a", "l1", 2); err != nil {
		t.Fatalf("re-attaching an attached install should succeed, got %v", err)
	}
}
