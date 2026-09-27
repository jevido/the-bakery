package app

import (
	"context"
	"errors"

	"github.com/jevido/the-bakery/services/api/contexts/boards/domain"
)

// WorkTypes stores each guild's work types.
type WorkTypes interface {
	// OfGuild returns the guild's work types in order.
	OfGuild(ctx context.Context, guildID uint64) ([]domain.WorkType, error)
	// AddDefaults stores the defaults unless the guild has work types
	// already; two callers at once end with one set.
	AddDefaults(ctx context.Context, wts []domain.WorkType) error
	Add(ctx context.Context, wt domain.WorkType) (domain.WorkType, error)
	Save(ctx context.Context, wt domain.WorkType) error
	Delete(ctx context.Context, guildID uint64, key string) error
	// InUse reports whether any task on the guild's boards has the key.
	InUse(ctx context.Context, guildID uint64, key string) (bool, error)
}

var ErrWorkTypeNotFound = errors.New("work type not found")

// ListWorkTypes returns the guild's work types in order. A guild that has
// none yet gets the defaults here: boards cannot react to a guild being
// founded, so the defaults appear the first time anyone asks.
func (s *Service) ListWorkTypes(ctx context.Context, guildID, memberID uint64) ([]domain.WorkType, error) {
	if err := s.requireMember(ctx, guildID, memberID); err != nil {
		return nil, err
	}
	return s.workTypesOf(ctx, guildID)
}

func (s *Service) workTypesOf(ctx context.Context, guildID uint64) ([]domain.WorkType, error) {
	wts, err := s.workTypes.OfGuild(ctx, guildID)
	if err != nil || len(wts) > 0 {
		return wts, err
	}
	if err := s.workTypes.AddDefaults(ctx, domain.DefaultsFor(guildID)); err != nil {
		return nil, err
	}
	return s.workTypes.OfGuild(ctx, guildID)
}

// checkWorkType refuses a key the guild does not have; "" (none) is fine.
func (s *Service) checkWorkType(ctx context.Context, guildID uint64, key string) error {
	if key == "" {
		return nil
	}
	wts, err := s.workTypesOf(ctx, guildID)
	if err != nil {
		return err
	}
	for _, wt := range wts {
		if wt.Key == key {
			return nil
		}
	}
	return domain.ErrUnknownWorkType
}

// AddWorkType puts a new work type at the end of the guild's list.
func (s *Service) AddWorkType(ctx context.Context, guildID, memberID uint64, key, name string) (domain.WorkType, error) {
	if err := s.requireMember(ctx, guildID, memberID); err != nil {
		return domain.WorkType{}, err
	}
	if err := s.requireWritable(ctx, guildID); err != nil {
		return domain.WorkType{}, err
	}
	for range positionAttempts {
		wts, err := s.workTypesOf(ctx, guildID)
		if err != nil {
			return domain.WorkType{}, err
		}
		for _, wt := range wts {
			if wt.Key == key {
				return domain.WorkType{}, domain.ErrDuplicateWorkType
			}
		}
		last := ""
		if len(wts) > 0 {
			last = wts[len(wts)-1].Position
		}
		wt, err := domain.NewWorkType(guildID, key, name, last)
		if err != nil {
			return domain.WorkType{}, err
		}
		wt, err = s.workTypes.Add(ctx, wt)
		if errors.Is(err, ErrPositionTaken) {
			continue
		}
		return wt, err
	}
	return domain.WorkType{}, ErrPositionTaken
}

// UpdateWorkType renames a work type and/or moves it to index in the list
// (nil leaves either as it is).
func (s *Service) UpdateWorkType(ctx context.Context, guildID, memberID uint64, key string, name *string, index *int) (domain.WorkType, error) {
	if err := s.requireMember(ctx, guildID, memberID); err != nil {
		return domain.WorkType{}, err
	}
	if err := s.requireWritable(ctx, guildID); err != nil {
		return domain.WorkType{}, err
	}
	wts, err := s.workTypesOf(ctx, guildID)
	if err != nil {
		return domain.WorkType{}, err
	}
	at := -1
	for i, wt := range wts {
		if wt.Key == key {
			at = i
		}
	}
	if at < 0 {
		return domain.WorkType{}, ErrWorkTypeNotFound
	}
	wt := wts[at]
	if name != nil {
		if err := wt.Rename(*name); err != nil {
			return domain.WorkType{}, err
		}
	}
	if index != nil {
		others := append(append([]domain.WorkType{}, wts[:at]...), wts[at+1:]...)
		i := max(0, min(*index, len(others)))
		above, below := "", ""
		if i > 0 {
			above = others[i-1].Position
		}
		if i < len(others) {
			below = others[i].Position
		}
		if err := wt.Reposition(above, below); err != nil {
			return domain.WorkType{}, err
		}
	}
	return wt, s.workTypes.Save(ctx, wt)
}

// DeleteWorkType removes a work type no task has.
func (s *Service) DeleteWorkType(ctx context.Context, guildID, memberID uint64, key string) error {
	if err := s.requireMember(ctx, guildID, memberID); err != nil {
		return err
	}
	if err := s.requireWritable(ctx, guildID); err != nil {
		return err
	}
	wts, err := s.workTypesOf(ctx, guildID)
	if err != nil {
		return err
	}
	found := false
	for _, wt := range wts {
		found = found || wt.Key == key
	}
	if !found {
		return ErrWorkTypeNotFound
	}
	inUse, err := s.workTypes.InUse(ctx, guildID, key)
	if err != nil {
		return err
	}
	if inUse {
		return domain.ErrWorkTypeInUse
	}
	return s.workTypes.Delete(ctx, guildID, key)
}
