package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

const commentBodyMax = 10000

var (
	ErrInvalidCommentBody = errors.New("comment must be 1 to 10000 characters")
	ErrNotAuthor          = errors.New("only its author can change a comment")
)

// Comment is text a member writes on a task. It is its own aggregate: only
// its author edits or deletes it.
type Comment struct {
	ID        uint64
	TaskID    uint64
	AuthorID  uint64
	Body      string
	CreatedAt time.Time
	EditedAt  *time.Time
}

// TaskCommented is announced when a comment is written on a task.
type TaskCommented struct {
	TaskID    uint64
	BoardID   uint64
	CommentID uint64
	AuthorID  uint64
}

// WriteComment makes a comment by authorID on the task, written at now.
func WriteComment(taskID, authorID uint64, body string, now time.Time) (Comment, error) {
	body, err := cleanBody(body)
	if err != nil {
		return Comment{}, err
	}
	return Comment{TaskID: taskID, AuthorID: authorID, Body: body, CreatedAt: now}, nil
}

// Edit replaces the body; only the author may.
func (c *Comment) Edit(by uint64, body string, now time.Time) error {
	if err := c.MayChange(by); err != nil {
		return err
	}
	body, err := cleanBody(body)
	if err != nil {
		return err
	}
	c.Body, c.EditedAt = body, &now
	return nil
}

// MayChange refuses anyone but the author.
func (c Comment) MayChange(by uint64) error {
	if by != c.AuthorID {
		return ErrNotAuthor
	}
	return nil
}

// cleanBody trims surrounding blank lines and spaces but keeps the text's
// own line breaks.
func cleanBody(body string) (string, error) {
	body = strings.TrimSpace(body)
	if n := utf8.RuneCountInString(body); n < 1 || n > commentBodyMax {
		return "", ErrInvalidCommentBody
	}
	return body, nil
}
