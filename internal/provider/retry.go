// Copyright (c) BRIDGE IN.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"io"
	"net/http"
	"strconv"
	"time"
)

// retryTransport retries idempotent (GET/HEAD) requests that fail with a
// transient server error (5xx) or a network error, using capped exponential
// backoff and honoring Retry-After.
//
// BridgePort returns retryable 503s (with Retry-After) when its single SQLite
// writer is briefly contended; older instances surface the same transient
// condition as a 500. The provider validates the token on every Configure and
// reads on every plan/refresh, so without this a single transient blip fails the
// whole run. Retrying GET/HEAD is safe: they have no side effects and no body to
// replay. Non-idempotent methods are passed through untouched.
type retryTransport struct {
	base       http.RoundTripper
	maxRetries int
	maxWait    time.Duration
}

func newRetryTransport(base http.RoundTripper) *retryTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &retryTransport{base: base, maxRetries: 6, maxWait: 8 * time.Second}
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		return t.base.RoundTrip(req)
	}

	backoff := 250 * time.Millisecond
	var resp *http.Response
	var err error
	for attempt := 0; ; attempt++ {
		resp, err = t.base.RoundTrip(req)
		if err == nil && resp.StatusCode < http.StatusInternalServerError {
			return resp, nil
		}
		if attempt >= t.maxRetries {
			return resp, err
		}

		wait := backoff
		if resp != nil {
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if secs, e := strconv.Atoi(ra); e == nil && secs > 0 {
					wait = time.Duration(secs) * time.Second
				}
			}
			// Drain so the connection can be reused, then close.
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
			_ = resp.Body.Close()
		}
		if wait > t.maxWait {
			wait = t.maxWait
		}

		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(wait):
		}
		if backoff < t.maxWait {
			backoff *= 2
		}
	}
}
