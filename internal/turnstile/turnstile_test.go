package turnstile_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/turnstile"
)

func withSiteverifyServer(t *testing.T, handler http.HandlerFunc) string {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return server.URL
}

func TestVerifySuccess(t *testing.T) {
	endpoint := withSiteverifyServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Errorf(
				"Content-Type = %q, want %q",
				got,
				"application/x-www-form-urlencoded",
			)
		}

		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm() error = %v", err)
			return
		}

		if got := r.Form.Get("secret"); got != "secret" {
			t.Errorf("secret = %q, want %q", got, "secret")
		}
		if got := r.Form.Get("response"); got != "token" {
			t.Errorf("response = %q, want %q", got, "token")
		}
		if got := r.Form.Get("remoteip"); got != "127.0.0.1" {
			t.Errorf("remoteip = %q, want %q", got, "127.0.0.1")
		}

		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"success":true}`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	})

	if err := turnstile.VerifyWithEndpoint(
		endpoint,
		"secret",
		"token",
		"127.0.0.1",
		time.Second,
	); err != nil {
		t.Fatalf("VerifyWithEndpoint() error = %v", err)
	}
}

func TestVerifyFailure(t *testing.T) {
	endpoint := withSiteverifyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"success":false}`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	})

	if err := turnstile.VerifyWithEndpoint(endpoint, "secret", "token", "127.0.0.1", time.Second); err == nil {
		t.Fatal("VerifyWithEndpoint() expected error")
	}
}

func TestVerifyNonOKStatus(t *testing.T) {
	endpoint := withSiteverifyServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "error", http.StatusInternalServerError)
	})

	if err := turnstile.VerifyWithEndpoint(endpoint, "secret", "token", "127.0.0.1", time.Second); err == nil {
		t.Fatal("VerifyWithEndpoint() expected error")
	}
}

func TestVerifyTimeout(t *testing.T) {
	endpoint := withSiteverifyServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"success":true}`)); err != nil {
			return
		}
	})

	if err := turnstile.VerifyWithEndpoint(endpoint, "secret", "token", "127.0.0.1", 10*time.Millisecond); err == nil {
		t.Fatal("VerifyWithEndpoint() expected timeout error")
	}
}

func TestVerifyMalformedJSON(t *testing.T) {
	endpoint := withSiteverifyServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`not-json`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	})

	if err := turnstile.VerifyWithEndpoint(endpoint, "secret", "token", "127.0.0.1", time.Second); err == nil {
		t.Fatal("VerifyWithEndpoint() expected error")
	}
}
