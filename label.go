package todoist

type Label struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Order uint   `json:"order"`
}

func (c *Client) ListLabels() (*[]Label, error) {
	resp, err := c.httpRequest("GET", restEndpoint("labels"), nil)
	if err != nil {
		return nil, err
	}

	out := &[]Label{}
	return out, decodeJSON(resp, out)
}
