package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"sync/atomic"
	"time"

	"golang.org/x/net/websocket"
)

const (
	EventTypeMessage = "message"
)

var (
	reMsg = regexp.MustCompile(`(?:<@(.+)>)?(?::?\s+)?(.*)`)
)

// Client represents a slack client.
type Client struct {
	ID       string
	Name     string
	Users    map[string]string
	Channels map[string]string
	Ims      map[string]string
	socket   *websocket.Conn
	counter  uint64
	latest   atomic.Value // latest pong reply time
	token    string
}

type connectResponse struct {
	OK       bool                        `json:"ok"`
	Error    string                      `json:"error"`
	URL      string                      `json:"url"`
	Self     struct{ ID, Name string }   `json:"self"`
	Users    []struct{ ID, Name string } `json:"users"`
	Channels []struct {
		ID, Name string
		IsMember bool `json:"is_member"`
	} `json:"channels"`
	Ims []struct {
		ID     string
		UserID string `json:"user"`
	} `json:"ims"`
}

// New creates a slack bot from API token.
// https://[YOURTEAM].slack.com/services/new/bot
func New(token string) (*Client, error) {
	bot := Client{
		Users:    map[string]string{},
		Channels: map[string]string{},
		Ims:      map[string]string{},
		token:    token,
	}

	// access slack api
	resp, err := bot.rtmStart(token)
	if err != nil {
		return nil, fmt.Errorf("api connection error, %v", err)
	}
	if !resp.OK {
		return nil, fmt.Errorf("connection error, %v", resp.Error)
	}

	// get realtime connection
	if e := bot.dial(resp.URL); e != nil {
		return nil, e
	}

	// save properties
	bot.ID = resp.Self.ID
	bot.Name = resp.Self.Name
	for _, u := range resp.Users {
		bot.Users[u.ID] = u.Name
	}
	for _, c := range resp.Channels {
		if c.IsMember {
			bot.Channels[c.ID] = c.Name
		}
	}
	for _, im := range resp.Ims {
		bot.Ims[im.ID] = im.UserID
	}
	return &bot, nil
}

func (c Client) rtmStart(token string) (*connectResponse, error) {
	q := url.Values{}
	q.Set("token", token)
	u := &url.URL{
		Scheme:   "https",
		Host:     "slack.com",
		Path:     "/api/rtm.start",
		RawQuery: q.Encode(),
	}
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with code %d", resp.StatusCode)
	}
	var body connectResponse
	dec := json.NewDecoder(resp.Body)
	if e := dec.Decode(&body); e != nil {
		return nil, fmt.Errorf("response decode error, %v", err)
	}
	return &body, nil
}

func (c *Client) dial(url string) error {
	ws, err := websocket.Dial(url, "", "https://api.slack.com/")
	if err != nil {
		return fmt.Errorf("dial error, %v", err)
	}
	c.socket = ws
	return nil
}

// UserName returns a slack username from the user id.
func (c Client) UserName(uid string) string {
	name, _ := c.Users[uid]
	return name
}

// GetMessage receives a message from the slack channel.
func (c *Client) GetMessage() (Message, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	ch := make(chan error, 1)

	var msg Message
	go func() {
		ch <- websocket.JSON.Receive(c.socket, &msg)
	}()

	select {
	case err := <-ch:
		return msg, err
	case <-ctx.Done():
		return msg, fmt.Errorf("timeout")
	}
	return msg, nil
}

// PostMessage sends a message to the slack channel.
func (c *Client) PostMessage(m Message) error {
	m.ID = atomic.AddUint64(&c.counter, 1)
	return websocket.JSON.Send(c.socket, m)
}

// UploadImage uploads a image by files.upload API.
func (c *Client) UploadImage(channels, title, filename, filetype, comment string, img io.Reader) error {
	if c.token == "" {
		return fmt.Errorf("slack token is empty")

	}
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return fmt.Errorf("multipart create from file error, %v, %v", title, err)
	}
	if _, err := io.Copy(part, img); err != nil {
		return fmt.Errorf("file copy error, %v, %v", title, err)
	}
	// for slack settings
	settings := map[string]string{
		"token":           c.token,
		"channels":        channels,
		"filetype":        filetype,
		"title":           title,
		"initial_comment": comment,
	}
	for k, v := range settings {
		if err := mw.WriteField(k, v); err != nil {
			return fmt.Errorf("write field error, %v:%v, %v", k, v, err)
		}
	}
	if err := mw.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://slack.com/api/files.upload", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	cl := &http.Client{Timeout: time.Duration(10) * time.Second}
	resp, err := cl.Do(req)
	if err != nil {
		return fmt.Errorf("slack files.upload error, %v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response error, %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack files.upload, %v, %v", resp.Status, body)
	}
	return nil
}

// Close implements the io.Closer interface.
func (c *Client) Close() error {
	return c.socket.Close()
}
