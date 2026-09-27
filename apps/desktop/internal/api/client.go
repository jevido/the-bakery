// Package api is a small typed client for The Bakery API. The desktop app
// talks to the API only through it, from Go, so the token never reaches the
// webview.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultURL is where `task dev` runs the API.
const DefaultURL = "http://127.0.0.1:4810"

// ErrUnauthorized means the API refused the token or the credentials.
var ErrUnauthorized = errors.New("not signed in")

// Error is a refusal the API explained: a message and, for invalid input,
// the field it is about.
type Error struct {
	Status  int
	Message string
	Field   string
}

func (e *Error) Error() string { return e.Message }

// Is makes a 401 match ErrUnauthorized while keeping the API's own message.
func (e *Error) Is(target error) bool {
	return target == ErrUnauthorized && e.Status == http.StatusUnauthorized
}

// Unreachable wraps a failure to talk to the API at all.
type Unreachable struct {
	URL string
	Err error
}

func (e *Unreachable) Error() string {
	return fmt.Sprintf("the API at %s is not reachable", e.URL)
}

func (e *Unreachable) Unwrap() error { return e.Err }

type Member struct {
	ID          uint64 `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) BaseURL() string { return c.baseURL }

type authResponse struct {
	Token  string `json:"token"`
	Member Member `json:"member"`
}

func (c *Client) Register(ctx context.Context, email, displayName, password string) (string, Member, error) {
	var res authResponse
	body := map[string]string{"email": email, "display_name": displayName, "password": password}
	if err := c.do(ctx, http.MethodPost, "/api/register", "", body, &res); err != nil {
		return "", Member{}, err
	}
	return res.Token, res.Member, nil
}

func (c *Client) Login(ctx context.Context, email, password string) (string, Member, error) {
	var res authResponse
	body := map[string]string{"email": email, "password": password}
	if err := c.do(ctx, http.MethodPost, "/api/login", "", body, &res); err != nil {
		return "", Member{}, err
	}
	return res.Token, res.Member, nil
}

func (c *Client) Me(ctx context.Context, token string) (Member, error) {
	var res struct {
		Member Member `json:"member"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/me", token, nil, &res); err != nil {
		return Member{}, err
	}
	return res.Member, nil
}

// do sends a JSON request and decodes a JSON response into out (if not nil).
func (c *Client) do(ctx context.Context, method, path, token string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("encoding request: %w", err)
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return &Unreachable{URL: c.baseURL, Err: err}
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		var e struct {
			Error string `json:"error"`
			Field string `json:"field"`
		}
		_ = json.NewDecoder(res.Body).Decode(&e)
		if res.StatusCode == http.StatusUnauthorized && token != "" {
			return ErrUnauthorized
		}
		if e.Error == "" {
			e.Error = http.StatusText(res.StatusCode)
		}
		return &Error{Status: res.StatusCode, Message: e.Error, Field: e.Field}
	}
	if out == nil || res.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding %s %s: %w", method, path, err)
	}
	return nil
}
