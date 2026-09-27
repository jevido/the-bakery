package app

import (
	"context"
	"errors"

	"github.com/jevido/the-bakery/services/api/contexts/boards/domain"
)

// columnBoard is the board a column belongs to, for changes.
func (s *Service) columnBoard(ctx context.Context, columnID, memberID uint64) (domain.Board, error) {
	c, found, err := s.boards.ColumnByID(ctx, columnID)
	if err != nil {
		return domain.Board{}, err
	}
	if !found {
		return domain.Board{}, domain.ErrColumnNotFound
	}
	return s.writableBoard(ctx, c.BoardID, memberID)
}

// ResolveColumn finds the board's column a ref names, for a member of its
// guild.
func (s *Service) ResolveColumn(ctx context.Context, boardID, memberID uint64, ref ColumnRef) (domain.Column, error) {
	b, err := s.board(ctx, boardID, memberID)
	if err != nil {
		return domain.Column{}, err
	}
	return ref.on(b)
}

// AddColumn puts a new column at the end of the board.
func (s *Service) AddColumn(ctx context.Context, boardID, memberID uint64, name string) (domain.Column, error) {
	for range positionAttempts {
		b, err := s.writableBoard(ctx, boardID, memberID)
		if err != nil {
			return domain.Column{}, err
		}
		c, err := b.AddColumn(name)
		if err != nil {
			return domain.Column{}, err
		}
		c, err = s.boards.AddColumn(ctx, c)
		if errors.Is(err, ErrPositionTaken) {
			continue
		}
		if err != nil {
			return domain.Column{}, err
		}
		s.events.Publish(ctx, domain.ColumnCreated{ColumnID: c.ID, BoardID: c.BoardID, ActorID: memberID, Name: c.Name, Position: c.Position})
		return c, nil
	}
	return domain.Column{}, ErrPositionTaken
}

func (s *Service) RenameColumn(ctx context.Context, columnID, memberID uint64, name string) (domain.Column, error) {
	b, err := s.columnBoard(ctx, columnID, memberID)
	if err != nil {
		return domain.Column{}, err
	}
	c, err := b.RenameColumn(columnID, name)
	if err != nil {
		return domain.Column{}, err
	}
	if err := s.boards.SaveColumn(ctx, c); err != nil {
		return domain.Column{}, err
	}
	s.events.Publish(ctx, domain.ColumnRenamed{ColumnID: c.ID, BoardID: c.BoardID, ActorID: memberID, Name: c.Name})
	return c, nil
}

// MoveColumn puts a column right after afterID and/or right before
// beforeID; with neither, at the end.
func (s *Service) MoveColumn(ctx context.Context, columnID, memberID uint64, afterID, beforeID *uint64) (domain.Column, error) {
	for range positionAttempts {
		b, err := s.columnBoard(ctx, columnID, memberID)
		if err != nil {
			return domain.Column{}, err
		}
		c, err := b.MoveColumn(columnID, afterID, beforeID)
		if err != nil {
			return domain.Column{}, err
		}
		err = s.boards.SaveColumn(ctx, c)
		if errors.Is(err, ErrPositionTaken) {
			continue
		}
		if err != nil {
			return domain.Column{}, err
		}
		s.events.Publish(ctx, domain.ColumnMoved{ColumnID: c.ID, BoardID: c.BoardID, ActorID: memberID, Position: c.Position})
		return c, nil
	}
	return domain.Column{}, ErrPositionTaken
}

// RemoveColumn deletes an empty column; a board keeps at least one.
func (s *Service) RemoveColumn(ctx context.Context, columnID, memberID uint64) error {
	b, err := s.columnBoard(ctx, columnID, memberID)
	if err != nil {
		return err
	}
	inColumn, err := s.tasks.InColumn(ctx, columnID)
	if err != nil {
		return err
	}
	if err := b.RemoveColumn(columnID, len(inColumn) > 0); err != nil {
		return err
	}
	if err := s.boards.DeleteColumn(ctx, columnID); err != nil {
		return err
	}
	s.events.Publish(ctx, domain.ColumnDeleted{ColumnID: columnID, BoardID: b.ID, ActorID: memberID})
	return nil
}
