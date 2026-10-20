package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendOnlyPingsListedUsers(t *testing.T) {
	var got webhookMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	d := &DiscordWebhook{URL: srv.URL, Client: srv.Client()}
	if err := d.Send("hi <@111> @everyone", []string{"111"}); err != nil {
		t.Fatalf("Send: %v", err)
	}

	if got.Content != "hi <@111> @everyone" {
		t.Fatalf("content = %q", got.Content)
	}
	if len(got.AllowedMentions.Parse) != 0 {
		t.Fatalf("parse = %v, want empty so @everyone is not pinged", got.AllowedMentions.Parse)
	}
	if len(got.AllowedMentions.Users) != 1 || got.AllowedMentions.Users[0] != "111" {
		t.Fatalf("users = %v, want [111]", got.AllowedMentions.Users)
	}
}

func TestSendReturnsErrorOnFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	d := &DiscordWebhook{URL: srv.URL, Client: srv.Client()}
	if err := d.Send("hi", nil); err == nil {
		t.Fatal("Send returned nil error for 400 response")
	}
}

func TestTruncateLongContent(t *testing.T) {
	got := truncate("ภาษาไทยยาว", 5)
	if len([]rune(got)) != 5 {
		t.Fatalf("truncate length = %d, want 5", len([]rune(got)))
	}
}
