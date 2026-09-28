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
	// Body is the whole answer, for refusals that carry more (a 409 with the
	// agent as it is now).
	Body []byte
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
	ID           uint64 `json:"id"`
	Email        string `json:"email"`
	DisplayName  string `json:"display_name"`
	PortraitSeed string `json:"portrait_seed"`
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

// RerollPortrait gives the member a new portrait seed.
func (c *Client) RerollPortrait(ctx context.Context, token string) (Member, error) {
	var res struct {
		Member Member `json:"member"`
	}
	if err := c.do(ctx, http.MethodPost, "/api/me/portrait", token, nil, &res); err != nil {
		return Member{}, err
	}
	return res.Member, nil
}

// do sends a JSON request and decodes a JSON response into out (if not nil).
func (c *Client) do(ctx context.Context, method, path, token string, in, out any) error {
	return c.doWith(ctx, method, path, token, nil, in, out)
}

// doWith is do with extra request headers.
func (c *Client) doWith(ctx context.Context, method, path, token string, headers map[string]string, in, out any) error {
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
	for k, v := range headers {
		req.Header.Set(k, v)
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
		body, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))
		_ = json.Unmarshal(body, &e)
		if res.StatusCode == http.StatusUnauthorized && token != "" {
			return ErrUnauthorized
		}
		if e.Error == "" {
			e.Error = http.StatusText(res.StatusCode)
		}
		return &Error{Status: res.StatusCode, Message: e.Error, Field: e.Field, Body: body}
	}
	if out == nil || res.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding %s %s: %w", method, path, err)
	}
	return nil
}

// Handoff returns a one-time code that signs the token's member in on the
// website.
func (c *Client) Handoff(ctx context.Context, token string) (string, error) {
	var res struct {
		Code string `json:"code"`
	}
	if err := c.do(ctx, http.MethodPost, "/api/web/handoff", token, nil, &res); err != nil {
		return "", err
	}
	return res.Code, nil
}

// PersonalToken is a personal token as the API lists it; never its secret.
type PersonalToken struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

// CreatePersonalToken makes a personal token for the member. The secret
// comes back this once.
func (c *Client) CreatePersonalToken(ctx context.Context, token, name string) (PersonalToken, string, error) {
	var res struct {
		Token         string        `json:"token"`
		PersonalToken PersonalToken `json:"personal_token"`
	}
	err := c.do(ctx, http.MethodPost, "/api/tokens", token, map[string]string{"name": name}, &res)
	return res.PersonalToken, res.Token, err
}

// RevokePersonalToken stops one of the member's personal tokens.
func (c *Client) RevokePersonalToken(ctx context.Context, token string, id uint64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/tokens/%d", id), token, nil, nil)
}

// Report tells the operators about a member or a guild ("member" or
// "guild").
func (c *Client) Report(ctx context.Context, token, targetKind string, targetID uint64, reason string) error {
	in := map[string]any{"target_kind": targetKind, "target_id": targetID, "reason": reason}
	return c.do(ctx, http.MethodPost, "/api/reports", token, in, nil)
}
