package store

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func stores(t *testing.T) map[string]func(t *testing.T) Store {
	out := map[string]func(t *testing.T) Store{
		"memory": func(*testing.T) Store { return NewMemory() },
	}
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		return out
	}
	out["postgres"] = func(t *testing.T) Store {
		ctx := context.Background()
		pg, err := OpenPostgres(ctx, url, 20)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pg.pool.Exec(ctx, `TRUNCATE users, checkout_sessions, payments, webhook_events RESTART IDENTITY CASCADE`); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(pg.Close)
		return pg
	}
	return out
}

func eachStore(t *testing.T, run func(t *testing.T, s Store)) {
	for name, open := range stores(t) {
		t.Run(name, func(t *testing.T) { run(t, open(t)) })
	}
}

func addUser(t *testing.T, s Store, clerkID string) User {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	u, err := s.CreateUser(context.Background(), User{ClerkID: clerkID, CreatedAt: now, LastSeenAt: now})
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestContractUsers(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store) {
		ctx := context.Background()
		if _, err := s.UserByClerkID(ctx, "user_a"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("missing user: err = %v, want ErrNotFound", err)
		}
		a := addUser(t, s, "user_a")
		again := addUser(t, s, "user_a")
		b := addUser(t, s, "user_b")
		if a.ID == 0 || again.ID != a.ID || b.ID == a.ID || a.Email != "" || a.Active() {
			t.Fatalf("a = %+v, again = %+v, b = %+v", a, again, b)
		}
		if err := s.SetUserEmail(ctx, a.ID, "a@example.com"); err != nil {
			t.Fatal(err)
		}
		if err := s.TouchUser(ctx, a.ID, time.Now()); err != nil {
			t.Fatal(err)
		}
		got, err := s.UserByClerkID(ctx, "user_a")
		if err != nil || got.ID != a.ID || got.Email != "a@example.com" {
			t.Fatalf("got %+v, %v", got, err)
		}
	})
}

func TestContractSpendTrialIsAtomic(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store) {
		ctx := context.Background()
		u := addUser(t, s, "user_a")
		var granted atomic.Int32
		var wg sync.WaitGroup
		for range 20 {
			wg.Go(func() {
				if n, err := s.SpendTrial(ctx, u.ID, CounterChecks, 3, 25); err == nil {
					granted.Add(int32(n))
				}
			})
		}
		wg.Wait()
		got, _ := s.UserByClerkID(ctx, "user_a")
		if granted.Load() != 25 || got.ChecksUsed != 25 {
			t.Fatalf("granted %d, used %d; want both 25", granted.Load(), got.ChecksUsed)
		}
		if _, err := s.SpendTrial(ctx, u.ID, CounterChecks, 1, 25); !errors.Is(err, ErrTrialSpent) {
			t.Fatalf("err = %v, want ErrTrialSpent", err)
		}
		_ = s.RefundTrial(ctx, u.ID, CounterChecks, 2)
		if n, err := s.SpendTrial(ctx, u.ID, CounterChecks, 5, 25); n != 2 || err != nil {
			t.Fatalf("after refund got %d, %v; want 2", n, err)
		}
	})
}

func TestContractPaymentActivatesUser(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store) {
		ctx := context.Background()
		a := addUser(t, s, "user_a")
		addUser(t, s, "user_b")
		now := time.Now().UTC().Truncate(time.Second)
		if err := s.CreateCheckoutSession(ctx, CheckoutSession{ID: "cks_1", UserID: a.ID, Status: CheckoutPending, CreatedAt: now}); err != nil {
			t.Fatal(err)
		}
		if err := s.SetCheckoutProviderID(ctx, "cks_1", "cks_dodo_1"); err != nil {
			t.Fatal(err)
		}
		pay := Payment{Provider: "dodo", WebhookID: "msg_1", EventType: "payment.succeeded", ProviderPaymentID: "pay_1",
			CheckoutSessionID: "cks_1", ClerkID: "user_b", Amount: 500, Currency: "USD", At: now}
		if matched, err := s.RecordPayment(ctx, pay); err != nil || !matched {
			t.Fatalf("first delivery: %v, %v", matched, err)
		}
		if _, err := s.RecordPayment(ctx, pay); !errors.Is(err, ErrDuplicate) {
			t.Fatalf("second delivery: err = %v, want ErrDuplicate", err)
		}
		pay.WebhookID = "msg_2"
		if _, err := s.RecordPayment(ctx, pay); err != nil {
			t.Fatalf("same payment, new webhook: %v", err)
		}

		session, err := s.CheckoutSession(ctx, "cks_1")
		if err != nil || session.Status != CheckoutPaid || session.PaidAt == nil || session.UserID != a.ID ||
			session.ProviderSessionID == nil || *session.ProviderSessionID != "cks_dodo_1" {
			t.Fatalf("session = %+v, %v", session, err)
		}
		if got, _ := s.UserByClerkID(ctx, "user_a"); !got.Active() {
			t.Fatalf("payer not activated: %+v", got)
		}
		if got, _ := s.UserByClerkID(ctx, "user_b"); got.Active() {
			t.Fatal("the session owner must win over the metadata user")
		}
	})
}

func TestContractPaymentWithoutSession(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store) {
		ctx := context.Background()
		addUser(t, s, "user_a")
		now := time.Now().UTC()
		pay := Payment{Provider: "dodo", WebhookID: "msg_1", EventType: "payment.succeeded", ProviderPaymentID: "pay_1", ClerkID: "user_a", At: now}
		if matched, err := s.RecordPayment(ctx, pay); err != nil || !matched {
			t.Fatalf("by clerk id: %v, %v", matched, err)
		}
		if got, _ := s.UserByClerkID(ctx, "user_a"); !got.Active() {
			t.Fatal("user not activated from metadata")
		}
		orphan := Payment{Provider: "dodo", WebhookID: "msg_2", EventType: "payment.succeeded", ProviderPaymentID: "pay_2", ClerkID: "user_gone", At: now}
		if matched, err := s.RecordPayment(ctx, orphan); err != nil || matched {
			t.Fatalf("unknown payer: %v, %v; want recorded but unmatched", matched, err)
		}
	})
}
