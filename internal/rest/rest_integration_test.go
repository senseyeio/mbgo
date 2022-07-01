// Copyright (c) 2018 Senseye Ltd. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in the LICENSE file.

//go:build integration
// +build integration

package rest_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/senseyeio/mbgo/internal/assert"
	"github.com/senseyeio/mbgo/internal/rest"
)

func TestClient_Do_Integration(t *testing.T) {
	t.Run("should error when the request context timeout deadline is exceeded", func(t *testing.T) {
		timeout := time.Millisecond * 10

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// have a slow handler to make sure the request context times out
			time.Sleep(timeout * 2)
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()

		u, err := url.Parse(srv.URL)
		assert.MustOk(t, err)

		cli := rest.NewClient(&http.Client{
			// increase the old-style client timeout above context deadline
			Timeout: timeout * 2,
		}, u)

		ctx, _ := context.WithTimeout(context.Background(), timeout)
		req, err := cli.NewRequest(ctx, http.MethodGet, "/foo", nil, nil)
		assert.MustOk(t, err)

		_, err = cli.Do(req)
		urlErr, ok := err.(*url.Error)
		assert.Equals(t, ok, true)
		assert.Equals(t, context.DeadlineExceeded, urlErr.Err)
	})
}
