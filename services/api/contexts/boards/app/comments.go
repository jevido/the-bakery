package app

import (
	"context"
	"errors"

	"github.com/jevido/the-bakery/services/api/contexts/boards/domain"
)

var ErrCommentNotFound = errors.New("comment not found")

type Comments interface {
	Add(ctx context.Context, c domain.Comment) (domain.Comment, error)
	Save(ctx context.Context, c domain.Comment) error
	Delete(ctx context.Context, id uint64) error
	ByID(ctx context.Context, id uint64) (domain.Comment, bool, error)
	// OfTask returns a task's comments, oldest first.
	OfTask(ctx context.Context, taskID uint64) ([]domain.Comment, error)
}

// MemberNames is identity's answer to "what are these members called?".
// Boards keeps only member ids.
type MemberNames interface {
	DisplayNames(ctx context.Context, ids []uint64) (map[uint64]string, error)
}

// AuthoredComment is a comment with its author's display name, for reading.
type AuthoredComment struct {
	domain.Comment
	AuthorName string
}

func (s *Service) ListComments(ctx context.Context, taskID, memberID uint64) ([]AuthoredComment, error) {
	t, err := s.readableTask(ctx, taskID, memberID)
	if err != nil {
		return nil, err
	}
	cs, err := s.comments.OfTask(ctx, t.ID)
	if err != nil {
		return nil, err
	}
	return s.authored(ctx, cs...)
}

func (s *Service) CommentOnTask(ctx context.Context, taskID, memberID uint64, body string) (AuthoredComment, error) {
	t, err := s.task(ctx, taskID, memberID)
	if err != nil {
		return AuthoredComment{}, err
	}
	c, err := domain.WriteComment(t.ID, memberID, body, s.now())
	if err != nil {
		return AuthoredComment{}, err
	}
	if c, err = s.comments.Add(ctx, c); err != nil {
		return AuthoredComment{}, err
	}
	s.events.Publish(ctx, domain.TaskCommented{TaskID: t.ID, BoardID: t.BoardID, ActorID: memberID, CommentID: c.ID})
	out, err := s.authored(ctx, c)
	if err != nil {
		return AuthoredComment{}, err
	}
	return out[0], nil
}

func (s *Service) EditComment(ctx context.Context, commentID, memberID uint64, body string) (AuthoredComment, error) {
	c, err := s.comment(ctx, commentID, memberID)
	if err != nil {
		return AuthoredComment{}, err
	}
	if err := c.Edit(memberID, body, s.now()); err != nil {
		return AuthoredComment{}, err
	}
	if err := s.comments.Save(ctx, c); err != nil {
		return AuthoredComment{}, err
	}
	out, err := s.authored(ctx, c)
	if err != nil {
		return AuthoredComment{}, err
	}
	return out[0], nil
}

func (s *Service) DeleteComment(ctx context.Context, commentID, memberID uint64) error {
	c, err := s.comment(ctx, commentID, memberID)
	if err != nil {
		return err
	}
	if err := c.MayChange(memberID); err != nil {
		return err
	}
	return s.comments.Delete(ctx, c.ID)
}

// comment is a comment for changes: the member must be in the task's guild
// and the guild must not be archived. Whether they wrote it is the
// aggregate's call.
func (s *Service) comment(ctx context.Context, commentID, memberID uint64) (domain.Comment, error) {
	c, found, err := s.comments.ByID(ctx, commentID)
	if err != nil {
		return domain.Comment{}, err
	}
	if !found {
		return domain.Comment{}, ErrCommentNotFound
	}
	if _, err := s.task(ctx, c.TaskID, memberID); err != nil {
		return domain.Comment{}, err
	}
	return c, nil
}

func (s *Service) authored(ctx context.Context, cs ...domain.Comment) ([]AuthoredComment, error) {
	ids := make([]uint64, 0, len(cs))
	for _, c := range cs {
		ids = append(ids, c.AuthorID)
	}
	names, err := s.names.DisplayNames(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]AuthoredComment, len(cs))
	for i, c := range cs {
		out[i] = AuthoredComment{Comment: c, AuthorName: names[c.AuthorID]}
	}
	return out, nil
}
