package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"git-notify/types/gitlab"
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

func stubDiscordIDs(t *testing.T, ids map[string]string) {
	t.Helper()
	original := discordIDLookup
	discordIDLookup = func(u gitlab.User) string { return ids[u.Username] }
	t.Cleanup(func() { discordIDLookup = original })
}

func post(t *testing.T, n Notifier, secret, token, event, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/webhook/gitlab", strings.NewReader(body))
	req.Header.Set("X-Gitlab-Event", event)
	if token != "" {
		req.Header.Set("X-Gitlab-Token", token)
	}
	rec := httptest.NewRecorder()
	GitLabWebhookHandler(n, secret, rec, req)
	return rec
}

func TestRejectsWrongSecret(t *testing.T) {
	n := &fakeNotifier{}
	rec := post(t, n, "s3cret", "wrong", "Push Hook", `{"object_kind":"push"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if n.calls != 0 {
		t.Fatalf("notifier called %d times, want 0", n.calls)
	}
}

func TestPushEventNotifiesChannel(t *testing.T) {
	n := &fakeNotifier{}
	body := `{"object_kind":"push","user_name":"Alice","ref":"refs/heads/main","project":{"name":"demo","web_url":"https://gitlab.com/x/demo"}}`
	rec := post(t, n, "s3cret", "s3cret", "Push Hook", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	want := "**GitLab Push** `demo`\nAlice push ไปที่ `refs/heads/main`\nhttps://gitlab.com/x/demo"
	if n.content != want {
		t.Fatalf("content = %q, want %q", n.content, want)
	}
	if len(n.mentions) != 0 {
		t.Fatalf("mentions = %v, want none", n.mentions)
	}
}

func TestAssignMentionsKnownDiscordUsers(t *testing.T) {
	stubDiscordIDs(t, map[string]string{"bob": "111"})
	n := &fakeNotifier{}
	body := `{
		"object_kind":"issue",
		"user":{"name":"Alice"},
		"project":{"name":"demo"},
		"object_attributes":{"title":"Fix login","action":"update","url":"https://gitlab.com/x/demo/-/issues/1"},
		"assignees":[{"id":2,"name":"Bob","username":"bob"},{"id":3,"name":"Carol","username":"carol"}],
		"changes":{"assignees":{"previous":[],"current":[{"id":2,"name":"Bob","username":"bob"},{"id":3,"name":"Carol","username":"carol"}]}}
	}`
	post(t, n, "", "", "Issue Hook", body)

	if !strings.Contains(n.content, "Alice assign ให้ <@111>, Carol") {
		t.Fatalf("content = %q, want mention for Bob and plain name for Carol", n.content)
	}
	if len(n.mentions) != 1 || n.mentions[0] != "111" {
		t.Fatalf("mentions = %v, want [111]", n.mentions)
	}
}

func TestNewlyAssignedIgnoresExistingAssignees(t *testing.T) {
	var p gitlab.Payload
	p.Assignees = []gitlab.User{{ID: 1, Name: "A"}, {ID: 2, Name: "B"}}
	p.Changes.AssigneeIDs = &gitlab.IntSliceChange{Previous: []int{1}, Current: []int{1, 2}}

	got := newlyAssigned(p)
	if len(got) != 1 || got[0].ID != 2 {
		t.Fatalf("newlyAssigned = %+v, want only user 2", got)
	}
}
