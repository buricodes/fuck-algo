package store

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

func memoryUser(t *testing.T, voice, checks int) (*Memory, int64) {
	t.Helper()
	m := NewMemory()
	u, err := m.CreateUser(context.Background(), User{ClerkID: "user_1", VoiceUsed: voice, ChecksUsed: checks})
	if err != nil {
		t.Fatal(err)
	}
	return m, u.ID
}

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
			m, id := memoryUser(t, tt.used, tt.used)
			ctx := context.Background()
			for _, c := range []Counter{CounterVoice, CounterChecks} {
				got, err := m.SpendTrial(ctx, id, c, tt.spend, tt.limit)
				if got != tt.want || !errors.Is(err, tt.wantErr) {
					t.Fatalf("counter %d: SpendTrial() = %d, %v; want %d, %v", c, got, err, tt.want, tt.wantErr)
				}
				u, _ := m.UserByClerkID(ctx, "user_1")
				if u.Used(c) != tt.wantUsed {
					t.Fatalf("counter %d: used = %d, want %d", c, u.Used(c), tt.wantUsed)
				}
			}
		})
	}
}

func TestSpendTrialConcurrent(t *testing.T) {
	m, id := memoryUser(t, 0, 0)
	ctx := context.Background()
	var granted atomic.Int32
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			if n, err := m.SpendTrial(ctx, id, CounterVoice, 1, 5); err == nil {
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
	m, id := memoryUser(t, 0, 2)
	ctx := context.Background()
	_ = m.RefundTrial(ctx, id, CounterChecks, 5)
	u, _ := m.UserByClerkID(ctx, "user_1")
	if u.ChecksUsed != 0 {
		t.Fatalf("checks used = %d, want 0", u.ChecksUsed)
	}
}

func TestSpendTrialUnknownUser(t *testing.T) {
	if _, err := NewMemory().SpendTrial(context.Background(), 42, CounterVoice, 1, 5); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
