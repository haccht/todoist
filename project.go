package todoist

type Project struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	Order        uint   `json:"order"`
	Indent       uint   `json:"indent"`
	CommentCount uint   `json:"comment_count"`
}

func (c *Client) ListProjects() (*[]Project, error) {
	resp, err := c.httpRequest("GET", restEndpoint("projects"), nil)
	if err != nil {
		return nil, err
	}

	out := &[]Project{}
	return out, decodeJSON(resp, out)
}
