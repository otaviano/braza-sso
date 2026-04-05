package auth

import (
	"net/http/httptest"
	"testing"
)

func TestSetRefreshCookie(t *testing.T) {
	rec := httptest.NewRecorder()
	setRefreshCookie(rec, "my-refresh-token")

	cookies := rec.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == refreshCookieName {
			found = true
			if c.Value != "my-refresh-token" {
				t.Errorf("expected cookie value %q, got %q", "my-refresh-token", c.Value)
			}
			if !c.HttpOnly {
				t.Error("refresh cookie must be HttpOnly")
			}
			if !c.Secure {
				t.Error("refresh cookie must be Secure")
			}
			if c.MaxAge <= 0 {
				t.Errorf("expected positive MaxAge, got %d", c.MaxAge)
			}
		}
	}
	if !found {
		t.Fatal("refresh_token cookie not set")
	}
}

func TestClearRefreshCookie(t *testing.T) {
	rec := httptest.NewRecorder()
	clearRefreshCookie(rec)

	cookies := rec.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == refreshCookieName {
			found = true
			if c.MaxAge != -1 {
				t.Errorf("expected MaxAge=-1 to expire cookie, got %d", c.MaxAge)
			}
			if c.Value != "" {
				t.Errorf("expected empty cookie value, got %q", c.Value)
			}
			if !c.HttpOnly {
				t.Error("cleared refresh cookie must still be HttpOnly")
			}
			if !c.Secure {
				t.Error("cleared refresh cookie must still be Secure")
			}
		}
	}
	if !found {
		t.Fatal("refresh_token cookie not set when clearing")
	}
}
