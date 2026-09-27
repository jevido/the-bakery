package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/register":
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte(`{"error":"an account with this email already exists","field":"email"}`))
		case "/api/login":
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"email or password is incorrect"}`))
		case "/api/me":
			if r.Header.Get("Authorization") != "Bearer good" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"not signed in"}`))
				return
			}
			w.Write([]byte(`{"member":{"id":1,"email":"ada@bakery.test","display_name":"Ada"}}`))
		}
	}))
	defer srv.Close()
	c := New(srv.URL)
	ctx := t.Context()

	_, _, err := c.Register(ctx, "ada@bakery.test", "Ada", "firstlanding")
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.Field != "email" || apiErr.Status != 422 {
		t.Errorf("Register error = %#v, want 422 on email", err)
	}

	_, _, err = c.Login(ctx, "ada@bakery.test", "wrong")
	if !errors.Is(err, ErrUnauthorized) || err.Error() != "email or password is incorrect" {
		t.Errorf("Login error = %v, want ErrUnauthorized with the API's message", err)
	}

	if _, err := c.Me(ctx, "bad"); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("Me(bad) error = %v, want ErrUnauthorized", err)
	}
	m, err := c.Me(ctx, "good")
	if err != nil || m.DisplayName != "Ada" {
		t.Errorf("Me(good) = %+v, %v", m, err)
	}

	var unreachable *Unreachable
	if _, err := New("http://127.0.0.1:1").Me(ctx, "x"); !errors.As(err, &unreachable) {
		t.Errorf("unreachable error = %v", err)
	}
}
