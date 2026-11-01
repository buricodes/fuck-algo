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

const (
	installA = "6f1c2b8e-3d4a-4f5b-9c6d-7e8f9a0b1c2a"
	installB = "6f1c2b8e-3d4a-4f5b-9c6d-7e8f9a0b1c2b"
	installC = "6f1c2b8e-3d4a-4f5b-9c6d-7e8f9a0b1c2c"
	licenceA = "7a1c2b8e-3d4a-4f5b-9c6d-7e8f9a0b1c2a"
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
		if _, err := pg.pool.Exec(ctx, `TRUNCATE installs, licences, checkout_sessions, webhook_events CASCADE`); err != nil {
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

func addInstall(t *testing.T, s Store, id string) {
	t.Helper()
	if err := s.CreateInstall(context.Background(), Install{ID: id, TokenHash: "hash-" + id, Version: "1.0.0", Browser: "chrome", CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
}

func TestContractSpendTrialIsAtomic(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store) {
		ctx := context.Background()
		addInstall(t, s, installA)
		var granted atomic.Int32
		var wg sync.WaitGroup
		for range 20 {
			wg.Go(func() {
				if n, err := s.SpendTrial(ctx, installA, CounterChecks, 3, 25); err == nil {
					granted.Add(int32(n))
				}
			})
		}
		wg.Wait()
		in, _ := s.InstallByTokenHash(ctx, "hash-"+installA)
		if granted.Load() != 25 || in.ChecksUsed != 25 {
			t.Fatalf("granted %d, used %d; want both 25", granted.Load(), in.ChecksUsed)
		}
		if _, err := s.SpendTrial(ctx, installA, CounterChecks, 1, 25); !errors.Is(err, ErrTrialSpent) {
			t.Fatalf("err = %v, want ErrTrialSpent", err)
		}
		_ = s.RefundTrial(ctx, installA, CounterChecks, 2)
		if n, err := s.SpendTrial(ctx, installA, CounterChecks, 5, 25); n != 2 || err != nil {
			t.Fatalf("after refund got %d, %v; want 2", n, err)
		}
	})
}

func TestContractPaymentAndActivation(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store) {
		ctx := context.Background()
		for _, id := range []string{installA, installB, installC} {
			addInstall(t, s, id)
		}
		now := time.Now().UTC().Truncate(time.Second)
		ia := installA
		if err := s.CreateCheckoutSession(ctx, CheckoutSession{ID: "cks_1", InstallID: &ia, Status: CheckoutPending, CreatedAt: now}); err != nil {
			t.Fatal(err)
		}
		pay := Payment{Provider: "dodo", WebhookID: "msg_1", EventType: "payment.succeeded", ProviderPaymentID: "pay_1",
			CheckoutSessionID: "cks_1", InstallID: installA, LicenceID: licenceA, KeyHash: "keyhash", KeyHint: "N234",
			RevealKey: "FA-ABCD-EFGH-JKLM-N234", Amount: 500, Currency: "USD", MaxInstalls: 2, At: now}
		if err := s.RecordPayment(ctx, pay); err != nil {
			t.Fatal(err)
		}
		if err := s.RecordPayment(ctx, pay); !errors.Is(err, ErrDuplicate) {
			t.Fatalf("second delivery: err = %v, want ErrDuplicate", err)
		}
		pay.WebhookID, pay.LicenceID, pay.KeyHash = "msg_2", "7a1c2b8e-3d4a-4f5b-9c6d-7e8f9a0b1c2f", "other"
		if err := s.RecordPayment(ctx, pay); err != nil {
			t.Fatalf("same payment, new webhook: %v", err)
		}
		if _, err := s.LicenceByKeyHash(ctx, "other"); !errors.Is(err, ErrNotFound) {
			t.Fatal("a second licence was created for the same payment")
		}

		session, err := s.CheckoutSession(ctx, "cks_1")
		if err != nil || session.Status != CheckoutPaid || session.RevealKey == nil || session.PaidAt == nil {
			t.Fatalf("session = %+v, %v", session, err)
		}
		in, _ := s.InstallByTokenHash(ctx, "hash-"+installA)
		if !in.LicenceActive || in.KeyHint == nil || *in.KeyHint != "N234" {
			t.Fatalf("install not activated: %+v", in)
		}

		l, err := s.LicenceByKeyHash(ctx, "keyhash")
		if err != nil {
			t.Fatal(err)
		}
		if err := s.AttachLicence(ctx, installB, l.ID, 2); err != nil {
			t.Fatal(err)
		}
		if err := s.AttachLicence(ctx, installC, l.ID, 2); !errors.Is(err, ErrLicenceFull) {
			t.Fatalf("third install: err = %v, want ErrLicenceFull", err)
		}
		if err := s.AttachLicence(ctx, installB, l.ID, 2); err != nil {
			t.Fatalf("re-attach: %v", err)
		}

		if err := s.ClearRevealKeys(ctx, now.Add(time.Second)); err != nil {
			t.Fatal(err)
		}
		if session, _ = s.CheckoutSession(ctx, "cks_1"); session.RevealKey != nil {
			t.Fatal("reveal key was not cleared")
		}
	})
}
