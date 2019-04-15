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
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// have a slow handler to make sure the request context times out
			time.Sleep(time.Second)
		}))
		defer srv.Close()

		u, err := url.Parse(srv.URL)
		assert.Ok(t, err)

		cli := rest.NewClient(&http.Client{
			// increase the old-style client timeout above context deadline
			Timeout: time.Second,
		}, u)

		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond)
		req, err := cli.NewRequest(ctx, http.MethodGet, "/foo", nil, nil)
		assert.Ok(t, err)

		_, err = cli.Do(req)
		urlErr, ok := err.(*url.Error)
		assert.Equals(t, ok, true)
		assert.Equals(t, urlErr.Err, context.DeadlineExceeded)
	})
}
