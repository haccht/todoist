package todoist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
)

type Task struct {
	ID           uint   `json:"id"`
	Content      string `json:"content"`
	ProjectID    uint   `json:"project_id"`
	LabelIDs     []uint `json:"label_ids"`
	Priority     uint   `json:"priority"`
	Completed    bool   `json:"completed"`
	CommentCount uint   `json:"comment_count"`
	Order        uint   `json:"order"`
	Indent       uint   `json:"indent"`
	URL          string `json:"url"`
	Due          struct {
		Date      string `json:"date,omitempty"`
		Datetime  string `json:"datetime,omitempty"`
		Recurring bool   `json:"recurring,omitempty"`
		String    string `json:"string,omitempty"`
		Timezone  string `json:"timezone,omitempty"`
	} `json:"due"`
}

func (t *Task) DueTime() time.Time {
	switch {
	case t.Due.Datetime != "":
		time, _ := time.Parse("2006-01-02T15:04:05Z", t.Due.Datetime)
		return time
	case t.Due.Date != "":
		time, _ := time.Parse("2006-01-02", t.Due.Date)
		return time
	default:
		zero := new(time.Time)
		return *zero
	}
}

func (t *Task) DueString() string {
	switch {
	case t.Due.Datetime != "":
		return t.DueTime().Format("2006-01-02(Mon) 15:04")
	case t.Due.Date != "":
		return t.DueTime().Format("2006-01-02(Mon)")
	default:
		return ""
	}
}

func (t *Task) IsOverdue() bool {
	switch {
	case t.Due.Datetime != "":
		return t.DueTime().Before(time.Now())
	case t.Due.Date != "":
		return t.DueTime().Truncate(24 * time.Hour).Before(time.Now().Truncate(24 * time.Hour))
	default:
		return false
	}
}

func (t *Task) IsDuedate() bool {
	switch {
	case t.Due.Datetime != "", t.Due.Date != "":
		return t.DueTime().Truncate(24 * time.Hour).Equal(time.Now().Truncate(24 * time.Hour))
	default:
		return false
	}
}

func (c *Client) ListTasks(filter *map[string]interface{}) ([]*Task, error) {
	ro := new(requestOption)
	if filter != nil {
		ro.Params = make(map[string]string)
		for k, v := range *filter {
			ro.Params[k] = fmt.Sprint(v)
		}
	}

	resp, err := c.httpRequest("GET", restEndpoint("/tasks"), ro)
	if err != nil {
		return nil, err
	}

	out := []*Task{}
	return out, decodeJSON(resp, &out)
}

func (c *Client) AddTask(args *map[string]interface{}) (*Task, error) {
	data, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	ro := new(requestOption)
	ro.Body = bytes.NewBuffer(data)
	ro.Headers = make(map[string]string)
	ro.Headers["X-Request-Id"] = uuid.New().String()
	ro.Headers["Content-Type"] = "application/json"

	resp, err := c.httpRequest("POST", restEndpoint("/tasks"), ro)
	if err != nil {
		return nil, err
	}

	out := new(Task)
	return out, decodeJSON(resp, out)
}

func (c *Client) GetTask(id uint) (*Task, error) {
	resp, err := c.httpRequest("GET", restEndpoint("/tasks", id), nil)
	if err != nil {
		return nil, err
	}

	out := new(Task)
	return out, decodeJSON(resp, out)
}

func (c *Client) UpdateTask(id uint, args *map[string]interface{}) error {
	data, err := json.Marshal(args)
	if err != nil {
		return err
	}

	ro := new(requestOption)
	ro.Body = bytes.NewBuffer(data)
	ro.Headers = make(map[string]string)
	ro.Headers["X-Request-Id"] = uuid.New().String()
	ro.Headers["Content-Type"] = "application/json"

	_, err = c.httpRequest("POST", restEndpoint("/tasks", id), ro)
	return err
}

func (c *Client) DeleteTask(id uint) error {
	_, err := c.httpRequest("DELETE", restEndpoint("/tasks", id), nil)
	return err
}

func (c *Client) CloseTask(id uint) error {
	_, err := c.httpRequest("POST", restEndpoint("/tasks", id, "/close"), nil)
	return err
}

func (c *Client) ReopenTask(id uint) error {
	_, err := c.httpRequest("POST", restEndpoint("/tasks", id, "/reopen"), nil)
	return err
}

func (c *Client) MoveTask(id uint, args *map[string]interface{}) error {
	commandArgs := map[string]interface{}{"id": id}
	if args != nil {
		for k, v := range *args {
			commandArgs[k] = v
		}
	}

	params := url.Values{}
	params.Add("commands", makeCommand("item_move", commandArgs))

	ro := new(requestOption)
	ro.Body = bytes.NewBufferString(params.Encode())
	ro.Headers = make(map[string]string)
	ro.Headers["Content-Type"] = "application/x-www-form-urlencoded"

	_, err := c.httpRequest("POST", syncEndpoint("/sync"), ro)
	return err
}

func (c *Client) QuickAddTask(text string, args *map[string]interface{}) error {
	params := url.Values{"text": {text}}
	if args != nil {
		for k, v := range *args {
			params.Add(k, fmt.Sprint(v))
		}
	}

	ro := new(requestOption)
	ro.Body = bytes.NewBufferString(params.Encode())
	ro.Headers = make(map[string]string)
	ro.Headers["Content-Type"] = "application/x-www-form-urlencoded"

	_, err := c.httpRequest("POST", syncEndpoint("/quick/add"), ro)
	return err
}
