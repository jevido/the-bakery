package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestWriteComment(t *testing.T) {
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		body    string
		wantErr error
	}{
		{"plain", "The freezer needs a second cooler.", nil},
		{"longest", strings.Repeat("x", 10000), nil},
		{"empty", "  \n ", ErrInvalidCommentBody},
		{"too long", strings.Repeat("x", 10001), ErrInvalidCommentBody},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := WriteComment(1, 2, tt.body, now)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("WriteComment() error = %v, want %v", err, tt.wantErr)
			}
			if err == nil && (c.AuthorID != 2 || c.EditedAt != nil || !c.CreatedAt.Equal(now)) {
				t.Errorf("comment = %+v", c)
			}
		})
	}
}

func TestOnlyTheAuthorChangesAComment(t *testing.T) {
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	c, _ := WriteComment(1, 2, "First", now)
	if err := c.Edit(3, "Mine now", now); !errors.Is(err, ErrNotAuthor) {
		t.Errorf("Edit() by another error = %v, want ErrNotAuthor", err)
	}
	if err := c.MayChange(3); !errors.Is(err, ErrNotAuthor) {
		t.Errorf("MayChange() by another error = %v, want ErrNotAuthor", err)
	}
	later := now.Add(time.Minute)
	if err := c.Edit(2, "Second", later); err != nil || c.Body != "Second" || c.EditedAt == nil || !c.EditedAt.Equal(later) {
		t.Errorf("Edit() by author = %v; comment = %+v", err, c)
	}
	if err := c.Edit(2, " ", later); !errors.Is(err, ErrInvalidCommentBody) {
		t.Errorf("Edit() to empty error = %v, want ErrInvalidCommentBody", err)
	}
}
