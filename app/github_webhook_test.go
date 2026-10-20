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

type sentDM struct{ to, message string }

type fakeMessenger struct{ sent []sentDM }

func (f *fakeMessenger) DM(discordUserID, message string) error {
	f.sent = append(f.sent, sentDM{discordUserID, message})
	return nil
}

type fakeUsers map[string]string

func (f fakeUsers) DiscordID(login string) string { return f[strings.ToLower(login)] }

func newHook(m *fakeMessenger, secret string) *GitHubWebhook {
	return &GitHubWebhook{
		Messenger: m,
		Users:     fakeUsers{"bob": "111", "carol": "222", "alice": "333"},
		Secret:    secret,
		Actions:   ParseActions(""),
	}
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

const assignBody = `{"action":"assigned","assignee":{"login":"Bob"},"sender":{"login":"alice"},
	"issue":{"number":7,"title":"Fix login","html_url":"https://github.com/acme/demo/issues/7",
	"assignees":[{"login":"Bob"},{"login":"carol"}]},"repository":{"full_name":"acme/demo"}}`

func TestRejectsBadSignature(t *testing.T) {
	m := &fakeMessenger{}
	h := newHook(m, "s3cret")

	if rec := post(h, "issues", "", assignBody); rec.Code != http.StatusUnauthorized {
		t.Fatalf("missing signature: status = %d, want 401", rec.Code)
	}
	if rec := post(h, "issues", sign("wrong", assignBody), assignBody); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong signature: status = %d, want 401", rec.Code)
	}
	if len(m.sent) != 0 {
		t.Fatalf("sent %d DMs, want 0", len(m.sent))
	}
}

func TestAssignDMsOnlyTheNewAssignee(t *testing.T) {
	m := &fakeMessenger{}
	h := newHook(m, "s3cret")

	if rec := post(h, "issues", sign("s3cret", assignBody), assignBody); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if len(m.sent) != 1 || m.sent[0].to != "111" {
		t.Fatalf("sent = %+v, want one DM to Bob (111)", m.sent)
	}
	want := "**[acme/demo] Issue #7** Fix login\nalice assign งานนี้ให้คุณ\nhttps://github.com/acme/demo/issues/7"
	if m.sent[0].message != want {
		t.Fatalf("message = %q, want %q", m.sent[0].message, want)
	}
}

func TestStatusChangeDMsAssigneesExceptSender(t *testing.T) {
	m := &fakeMessenger{}
	h := newHook(m, "")
	body := `{"action":"closed","sender":{"login":"carol"},
		"pull_request":{"number":3,"title":"Add API","html_url":"u","merged":true,
		"assignees":[{"login":"bob"},{"login":"carol"},{"login":"dave"}]},"repository":{"full_name":"acme/demo"}}`

	post(h, "pull_request", "", body)

	if len(m.sent) != 1 || m.sent[0].to != "111" {
		t.Fatalf("sent = %+v, want one DM to bob (carol is sender, dave not linked)", m.sent)
	}
	if !strings.Contains(m.sent[0].message, "carol merge แล้ว") {
		t.Fatalf("message = %q, want merged text", m.sent[0].message)
	}
}

func TestLabelMessageIncludesLabelName(t *testing.T) {
	m := &fakeMessenger{}
	h := newHook(m, "")
	body := `{"action":"labeled","label":{"name":"in-progress"},"sender":{"login":"alice"},
		"issue":{"number":1,"title":"T","html_url":"u","assignees":[{"login":"bob"}]},"repository":{"full_name":"acme/demo"}}`

	post(h, "issues", "", body)

	if len(m.sent) != 1 || !strings.Contains(m.sent[0].message, "alice เพิ่ม label `in-progress`") {
		t.Fatalf("sent = %+v", m.sent)
	}
}

func TestIgnoresActionsNotConfigured(t *testing.T) {
	m := &fakeMessenger{}
	h := newHook(m, "")
	h.Actions = ParseActions("closed")

	post(h, "issues", "", assignBody)
	post(h, "push", "", `{"ref":"refs/heads/main"}`)

	if len(m.sent) != 0 {
		t.Fatalf("sent = %+v, want none", m.sent)
	}
}

func TestPingDoesNotDM(t *testing.T) {
	m := &fakeMessenger{}
	if rec := post(newHook(m, ""), "ping", "", `{"zen":"hi"}`); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if len(m.sent) != 0 {
		t.Fatalf("sent = %+v, want none", m.sent)
	}
}
