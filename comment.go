package todoist

import (
	"fmt"
)

type Comment struct {
	ID        uint   `json:"id"`
	Posted    string `json:"posted"`
	CommentID uint   `json:"comment_id"`
	ProjectID uint   `json:"project_id"`
	Content   string `json:"content"`
}

func (c *Client) ListComments(args *map[string]interface{}) ([]*Comment, error) {
	ro := NewRequestOption()
	for k, v := range *args {
		ro.Params[k] = fmt.Sprint(v)
	}

	resp, err := c.httpRequest("GET", restEndpoint("comments"), ro)
	if err != nil {
		return nil, err
	}

	out := []*Comment{}
	return out, decodeJSON(resp, &out)
}
