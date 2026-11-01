package slots

import (
	"context"
	"time"

	"github.com/buricodes/fuck-algo/server/pkg/apperr"
)

const waitTimeout = 5 * time.Second

func Acquire(ctx context.Context, sem chan struct{}) (release func(), err error) {
	waitCtx, cancel := context.WithTimeout(ctx, waitTimeout)
	defer cancel()
	select {
	case sem <- struct{}{}:
		return func() { <-sem }, nil
	case <-waitCtx.Done():
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, apperr.New(apperr.RateLimited, "We are very busy right now. Try again in a moment.")
	}
}
