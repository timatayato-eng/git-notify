package github

type User struct {
	Login string `json:"login"`
}

type Item struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	HTMLURL   string `json:"html_url"`
	Merged    bool   `json:"merged"`
	Assignees []User `json:"assignees"`
}

type Payload struct {
	Action     string `json:"action"`
	Sender     User   `json:"sender"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	Assignee    *User `json:"assignee"`
	PullRequest *Item `json:"pull_request"`
	Issue       *Item `json:"issue"`
	Label       *struct {
		Name string `json:"name"`
	} `json:"label"`
}
