package app

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeNotifier struct {
	content  string
	mentions []string
	calls    int
}

func (f *fakeNotifier) Send(content string, mentionUserIDs []string) error {
	f.calls++
	f.content = content
	f.mentions = mentionUserIDs
	return nil
}

func sign(secret, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func post(h *GitHubWebhook, event, signature, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/webhook/github", strings.NewReader(body))
	req.Header.Set("X-GitHub-Event", event)
	if signature != "" {
		req.Header.Set("X-Hub-Signature-256", signature)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestRejectsBadSignature(t *testing.T) {
	n := &fakeNotifier{}
	h := &GitHubWebhook{Notifier: n, Secret: "s3cret"}
	body := `{"ref":"refs/heads/main"}`

	if rec := post(h, "push", "", body); rec.Code != http.StatusUnauthorized {
		t.Fatalf("missing signature: status = %d, want 401", rec.Code)
	}
	if rec := post(h, "push", sign("wrong", body), body); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong signature: status = %d, want 401", rec.Code)
	}
	if n.calls != 0 {
		t.Fatalf("notifier called %d times, want 0", n.calls)
	}
}

func TestPingDoesNotNotify(t *testing.T) {
	n := &fakeNotifier{}
	h := &GitHubWebhook{Notifier: n}
	if rec := post(h, "ping", "", `{"zen":"hi"}`); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if n.calls != 0 {
		t.Fatalf("notifier called %d times, want 0", n.calls)
	}
}

func TestPushEventNotifiesChannel(t *testing.T) {
	n := &fakeNotifier{}
	h := &GitHubWebhook{Notifier: n, Secret: "s3cret"}
	body := `{"ref":"refs/heads/main","compare":"https://github.com/acme/demo/compare/a...b",
		"commits":[{},{}],"pusher":{"name":"alice"},"repository":{"full_name":"acme/demo"}}`

	if rec := post(h, "push", sign("s3cret", body), body); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	want := "**GitHub Push** `acme/demo`\nalice push 2 commit ไปที่ `main`\nhttps://github.com/acme/demo/compare/a...b"
	if n.content != want {
		t.Fatalf("content = %q, want %q", n.content, want)
	}
}

func TestAssignMentionsMappedDiscordUser(t *testing.T) {
	n := &fakeNotifier{}
	h := &GitHubWebhook{Notifier: n, DiscordUsers: ParseDiscordUserMap("Bob:111")}
	body := `{"action":"assigned","assignee":{"login":"bob"},"sender":{"login":"alice"},
		"issue":{"number":7,"title":"Fix login","html_url":"https://github.com/acme/demo/issues/7"},
		"repository":{"full_name":"acme/demo"}}`

	post(h, "issues", "", body)

	if !strings.Contains(n.content, "#7 Fix login\nalice assign ให้ <@111>") {
		t.Fatalf("content = %q, want mention of Bob", n.content)
	}
	if len(n.mentions) != 1 || n.mentions[0] != "111" {
		t.Fatalf("mentions = %v, want [111]", n.mentions)
	}
}

func TestAssignUnmappedUserUsesLogin(t *testing.T) {
	n := &fakeNotifier{}
	h := &GitHubWebhook{Notifier: n, DiscordUsers: map[string]string{}}
	body := `{"action":"assigned","assignee":{"login":"carol"},"sender":{"login":"alice"},
		"pull_request":{"number":3,"title":"Add API","html_url":"u"},"repository":{"full_name":"acme/demo"}}`

	post(h, "pull_request", "", body)

	if !strings.Contains(n.content, "alice assign ให้ carol") || len(n.mentions) != 0 {
		t.Fatalf("content = %q mentions = %v, want plain login and no mentions", n.content, n.mentions)
	}
}

func TestParseDiscordUserMap(t *testing.T) {
	got := ParseDiscordUserMap(" Alice:1 , bob:2,broken,:3,")
	if len(got) != 2 || got["alice"] != "1" || got["bob"] != "2" {
		t.Fatalf("ParseDiscordUserMap = %v", got)
	}
}
