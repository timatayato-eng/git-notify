package app

import (
	"path/filepath"
	"testing"
)

func TestUserStoreLinkPersistsAndReplacesOldLogin(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "users.json")
	store, err := OpenUserStore(path)
	if err != nil {
		t.Fatalf("OpenUserStore: %v", err)
	}

	if err := store.Link("OldName", "111"); err != nil {
		t.Fatalf("Link: %v", err)
	}
	if err := store.Link("NewName", "111"); err != nil {
		t.Fatalf("Link: %v", err)
	}

	reopened, err := OpenUserStore(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if got := reopened.DiscordID("newname"); got != "111" {
		t.Fatalf("DiscordID(newname) = %q, want 111", got)
	}
	if got := reopened.DiscordID("oldname"); got != "" {
		t.Fatalf("DiscordID(oldname) = %q, want empty after relink", got)
	}
}

func TestLinkGitHubRejectsInvalidLogin(t *testing.T) {
	store, _ := OpenUserStore(filepath.Join(t.TempDir(), "users.json"))

	linkGitHub(store, "111", "not a valid name!")

	if got := store.DiscordID("not a valid name!"); got != "" {
		t.Fatalf("invalid login was stored")
	}
}

func TestNormalizeGitHubLogin(t *testing.T) {
	for _, input := range []string{"octocat", " @octocat ", "https://github.com/octocat/", "github.com/octocat"} {
		if got, ok := normalizeGitHubLogin(input); !ok || got != "octocat" {
			t.Errorf("normalizeGitHubLogin(%q) = %q, %v", input, got, ok)
		}
	}
	for _, input := range []string{"", "-bad", "has space", "a/b"} {
		if _, ok := normalizeGitHubLogin(input); ok {
			t.Errorf("normalizeGitHubLogin(%q) accepted invalid login", input)
		}
	}
}
