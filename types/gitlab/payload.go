package gitlab

type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
}

type AssigneeChange struct {
	Previous []User `json:"previous"`
	Current  []User `json:"current"`
}

type IntSliceChange struct {
	Previous []int `json:"previous"`
	Current  []int `json:"current"`
}

type IntChange struct {
	Previous *int `json:"previous"`
	Current  *int `json:"current"`
}

type Payload struct {
	ObjectKind string `json:"object_kind"`
	UserName   string `json:"user_name"`
	User       User   `json:"user"`
	Ref        string `json:"ref"`
	Project    struct {
		Name   string `json:"name"`
		WebURL string `json:"web_url"`
	} `json:"project"`
	ObjectAttributes struct {
		Title  string `json:"title"`
		Action string `json:"action"`
		URL    string `json:"url"`
	} `json:"object_attributes"`
	Assignees []User `json:"assignees"`
	Changes   struct {
		Assignees   *AssigneeChange `json:"assignees"`
		AssigneeIDs *IntSliceChange `json:"assignee_ids"`
		AssigneeID  *IntChange      `json:"assignee_id"`
	} `json:"changes"`
}
