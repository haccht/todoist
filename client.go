package todoist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"

	"github.com/google/uuid"
)

const (
	todoistSyncAPI = "https://api.todoist.com/sync/v8"
	todoistRESTAPI = "https://api.todoist.com/rest/v1"
)

type Client struct {
	authToken string

	Logger     *log.Logger
	HTTPClient *http.Client
}

func NewClient(authToken string) *Client {
	return &Client{
		authToken:  authToken,
		Logger:     log.New(ioutil.Discard, "", log.LstdFlags),
		HTTPClient: http.DefaultClient,
	}
}

type RequestOption struct {
	Params  map[string]string
	Headers map[string]string
	Body    io.Reader
}

func NewRequestOption() *RequestOption {
	return &RequestOption{
		Params:  make(map[string]string),
		Headers: make(map[string]string),
	}
}

func restEndpoint(elm ...interface{}) *url.URL {
	u, _ := url.ParseRequestURI(todoistRESTAPI)
	for _, v := range elm {
		u.Path = path.Join(u.Path, fmt.Sprint(v))
	}

	return u
}

func syncEndpoint(elm ...interface{}) *url.URL {
	u, _ := url.ParseRequestURI(todoistSyncAPI)
	for _, v := range elm {
		u.Path = path.Join(u.Path, fmt.Sprint(v))
	}

	return u
}

type command struct {
	Type   string      `json:"type"`
	Args   interface{} `json:"args"`
	UUID   string      `json:"uuid"`
	TempID string      `json:"temp_id"`
}

func makeCommand(typeString string, args interface{}) string {
	c := command{
		Type:   typeString,
		Args:   args,
		UUID:   uuid.New().String(),
		TempID: uuid.New().String(),
	}

	commandData, err := json.Marshal([]command{c})
	if err != nil {
		return ""
	}

	return string(commandData)
}

func decodeJSON(resp *http.Response, out interface{}) error {
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)

	return dec.Decode(out)
}

func (c *Client) isPremium() (bool, error) {
	params := url.Values{}
	params.Add("sync_token", "*")
	params.Add("resource_types", "[\"user\"]")

	ro := NewRequestOption()
	ro.Body = bytes.NewBufferString(params.Encode())
	ro.Headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := c.httpRequest("POST", syncEndpoint("/sync"), ro)
	if err != nil {
		return false, err
	}

	var out interface{}
	err = decodeJSON(resp, &out)
	if err != nil {
		return false, err
	}

	isPremium, _ := out.(map[string]interface{})["user"].(map[string]interface{})["is_premium"].(bool)
	return isPremium, nil
}

func (c *Client) httpRequest(method string, u *url.URL, ro *RequestOption) (*http.Response, error) {
	if ro == nil {
		ro = NewRequestOption()
	}

	var params = make(url.Values)
	for k, v := range ro.Params {
		params.Add(k, v)
	}
	u.RawQuery = params.Encode()

	req, err := http.NewRequest(method, u.String(), ro.Body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.authToken))
	for k, v := range ro.Headers {
		req.Header.Set(k, v)
	}

	c.Logger.Printf("%s %s", method, u.String())
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || 300 <= resp.StatusCode {
		message, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s: %s", resp.Status, message)
	}

	return resp, nil
}
