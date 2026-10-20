package github

import "encoding/json"

type User struct {
	Login string `json:"login"`
}

type Item struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	HTMLURL string `json:"html_url"`
}

type Payload struct {
	Action  string            `json:"action"`
	Ref     string            `json:"ref"`
	Compare string            `json:"compare"`
	Commits []json.RawMessage `json:"commits"`
	Pusher  struct {
		Name string `json:"name"`
	} `json:"pusher"`
	Sender     User `json:"sender"`
	Repository struct {
		FullName string `json:"full_name"`
		HTMLURL  string `json:"html_url"`
	} `json:"repository"`
	Assignee    *User `json:"assignee"`
	PullRequest *Item `json:"pull_request"`
	Issue       *Item `json:"issue"`
}
