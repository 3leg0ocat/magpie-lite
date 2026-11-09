package update

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// A build with self-update off never asks the feed.
func TestOffNeverAsksTheFeed(t *testing.T) {
	asked := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = true
		_, _ = w.Write([]byte(`{"version":"9.9.9"}`))
	}))
	defer srv.Close()
	t.Setenv("MAGPIE_UPDATE_FEED", srv.URL)
	was := selfUpdate
	t.Cleanup(func() { selfUpdate = was })

	selfUpdate = "off"
	if _, err := Latest(context.Background()); !errors.Is(err, ErrOff) || asked {
		t.Fatalf("off: err %v, feed asked %v", err, asked)
	}
	selfUpdate = ""
	if rel, err := Latest(context.Background()); err != nil || rel.Version != "9.9.9" || !asked {
		t.Fatalf("on: %v %v, feed asked %v", rel, err, asked)
	}
}
