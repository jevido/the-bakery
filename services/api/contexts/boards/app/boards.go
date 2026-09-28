// Package app holds the boards use cases. Every one of them first checks
// that the calling member is in the board's guild.
package app

import (
	"context"
	"errors"
	"time"

	"github.com/jevido/the-bakery/services/api/contexts/boards/domain"
)

var (
	ErrNotMember        = errors.New("not a member of this guild")
	ErrBoardNotFound    = errors.New("board not found")
	ErrTaskNotFound     = errors.New("task not found")
	ErrInvalidNeighbour = errors.New("neighbour must be another task in the target column")
	ErrInvalidSibling   = errors.New("neighbour must be another subtask of the same task")
	ErrGuildArchived    = errors.New("this guild is archived; its boards are read-only")
	// ErrPositionTaken is returned by Tasks when another write took the same
	// position first; the use case recomputes and tries again.
	ErrPositionTaken = errors.New("position taken")
)

// positionAttempts bounds retries when concurrent writes pick the same key.
const positionAttempts = 3

// Memberships is guilds' answer to "is this member in this guild?" and "is
// this guild archived?".
type Memberships interface {
	IsMember(ctx context.Context, guildID, memberID uint64) (bool, error)
	IsArchived(ctx context.Context, guildID uint64) (bool, error)
}

// Boards stores boards with their columns. ByID returns the board with its
// columns in order; OfGuild returns boards without them.
type Boards interface {
	Add(ctx context.Context, b domain.Board) (domain.Board, error)
	ByID(ctx context.Context, id uint64) (domain.Board, bool, error)
	OfGuild(ctx context.Context, guildID uint64) ([]domain.Board, error)
	// ColumnByID finds a column, to reach its board.
	ColumnByID(ctx context.Context, id uint64) (domain.Column, bool, error)
	AddColumn(ctx context.Context, c domain.Column) (domain.Column, error)
	SaveColumn(ctx context.Context, c domain.Column) error
	DeleteColumn(ctx context.Context, id uint64) error
}

type Tasks interface {
	Add(ctx context.Context, t domain.Task) (domain.Task, error)
	Save(ctx context.Context, t domain.Task) error
	Delete(ctx context.Context, id uint64) error
	ByID(ctx context.Context, id uint64) (domain.Task, bool, error)
	// OfBoard returns the board's tasks ordered by column, then position.
	OfBoard(ctx context.Context, boardID uint64) ([]domain.Task, error)
	// InColumn returns one column's tasks ordered by position.
	InColumn(ctx context.Context, columnID uint64) ([]domain.Task, error)
	// AddAll stores several tasks in one transaction: all or none.
	AddAll(ctx context.Context, ts []domain.Task) ([]domain.Task, error)
	// SubtasksOf returns a task's subtasks ordered by position.
	SubtasksOf(ctx context.Context, parentID uint64) ([]domain.Task, error)
	// SubtaskCounts counts the subtasks of each of the given tasks; a task
	// without subtasks is left out.
	SubtaskCounts(ctx context.Context, parentIDs []uint64) (map[uint64]SubtaskCount, error)
}

// SubtaskCount is how many subtasks a task has, and how many are done.
type SubtaskCount struct {
	Total int
	Done  int
}

// Events takes the domain events of a change once it is stored.
type Events interface {
	Publish(ctx context.Context, events ...any)
}

type Service struct {
	memberships Memberships
	boards      Boards
	tasks       Tasks
	comments    Comments
	activity    ActivityLog
	presence    PresenceLog
	workTypes   WorkTypes
	runs        Runs
	claims      Claims
	owners      AgentOwners
	names       MemberNames
	events      Events
	now         func() time.Time
}

func NewService(memberships Memberships, boards Boards, tasks Tasks, comments Comments, activity ActivityLog, presence PresenceLog, workTypes WorkTypes, runs Runs, claims Claims, owners AgentOwners, names MemberNames, events Events) *Service {
	return &Service{
		memberships: memberships, boards: boards, tasks: tasks, comments: comments,
		activity: activity, presence: presence, workTypes: workTypes, runs: runs, claims: claims, owners: owners, names: names, events: events, now: time.Now,
	}
}

func (s *Service) requireMember(ctx context.Context, guildID, memberID uint64) error {
	ok, err := s.memberships.IsMember(ctx, guildID, memberID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotMember
	}
	return nil
}

// requireWritable refuses changes in an archived guild.
func (s *Service) requireWritable(ctx context.Context, guildID uint64) error {
	archived, err := s.memberships.IsArchived(ctx, guildID)
	if err != nil {
		return err
	}
	if archived {
		return ErrGuildArchived
	}
	return nil
}

// writableBoard is board for changes: the guild must not be archived.
func (s *Service) writableBoard(ctx context.Context, boardID, memberID uint64) (domain.Board, error) {
	b, err := s.board(ctx, boardID, memberID)
	if err != nil {
		return domain.Board{}, err
	}
	return b, s.requireWritable(ctx, b.GuildID)
}

func (s *Service) board(ctx context.Context, boardID, memberID uint64) (domain.Board, error) {
	b, found, err := s.boards.ByID(ctx, boardID)
	if err != nil {
		return domain.Board{}, err
	}
	if !found {
		return domain.Board{}, ErrBoardNotFound
	}
	return b, s.requireMember(ctx, b.GuildID, memberID)
}

// task is a task for changes: the member must be in its guild and the
// guild must not be archived.
func (s *Service) task(ctx context.Context, taskID, memberID uint64) (domain.Task, error) {
	t, found, err := s.tasks.ByID(ctx, taskID)
	if err != nil {
		return domain.Task{}, err
	}
	if !found {
		return domain.Task{}, ErrTaskNotFound
	}
	if _, err := s.writableBoard(ctx, t.BoardID, memberID); err != nil {
		return domain.Task{}, err
	}
	return t, nil
}

// readableTask is a task for reading; an archived guild's tasks can still
// be read.
func (s *Service) readableTask(ctx context.Context, taskID, memberID uint64) (domain.Task, error) {
	t, found, err := s.tasks.ByID(ctx, taskID)
	if err != nil {
		return domain.Task{}, err
	}
	if !found {
		return domain.Task{}, ErrTaskNotFound
	}
	if _, err := s.board(ctx, t.BoardID, memberID); err != nil {
		return domain.Task{}, err
	}
	return t, nil
}

func (s *Service) ListBoards(ctx context.Context, guildID, memberID uint64) ([]domain.Board, error) {
	if err := s.requireMember(ctx, guildID, memberID); err != nil {
		return nil, err
	}
	return s.boards.OfGuild(ctx, guildID)
}

func (s *Service) CreateBoard(ctx context.Context, guildID, memberID uint64, name string) (domain.Board, error) {
	if err := s.requireMember(ctx, guildID, memberID); err != nil {
		return domain.Board{}, err
	}
	if err := s.requireWritable(ctx, guildID); err != nil {
		return domain.Board{}, err
	}
	b, err := domain.NewBoard(guildID, name)
	if err != nil {
		return domain.Board{}, err
	}
	return s.boards.Add(ctx, b)
}

// GetBoard returns the board and its tasks, by column then position.
// GetBoard returns the board's top-level tasks by column and position, and
// how many subtasks each has.
// BoardView is a board with its tasks on it, each task's subtask count,
// and the claims holding tasks now.
type BoardView struct {
	Board  domain.Board
	Tasks  []domain.Task
	Counts map[uint64]SubtaskCount
	Claims map[uint64]domain.Claim
}

func (s *Service) GetBoard(ctx context.Context, boardID, memberID uint64) (BoardView, error) {
	b, err := s.board(ctx, boardID, memberID)
	if err != nil {
		return BoardView{}, err
	}
	tasks, err := s.tasks.OfBoard(ctx, b.ID)
	if err != nil {
		return BoardView{}, err
	}
	ids := make([]uint64, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
	}
	counts, err := s.tasks.SubtaskCounts(ctx, ids)
	if err != nil {
		return BoardView{}, err
	}
	claims, err := s.claims.ActiveOf(ctx, ids, s.now())
	if err != nil {
		return BoardView{}, err
	}
	return BoardView{Board: b, Tasks: tasks, Counts: counts, Claims: claims}, nil
}

// WatchBoard checks that the member may watch the board's events: the same
// rule as reading it.
func (s *Service) WatchBoard(ctx context.Context, boardID, memberID uint64) error {
	_, err := s.board(ctx, boardID, memberID)
	return err
}

// GetTask returns a task and, for a top-level task, its subtasks in order.
func (s *Service) GetTask(ctx context.Context, taskID, memberID uint64) (domain.Task, []domain.Task, error) {
	t, err := s.readableTask(ctx, taskID, memberID)
	if err != nil {
		return domain.Task{}, nil, err
	}
	if t.IsSubtask() {
		return t, []domain.Task{}, nil
	}
	subtasks, err := s.tasks.SubtasksOf(ctx, t.ID)
	return t, subtasks, err
}

// AddSubtask adds one subtask at the end of the task's subtasks.
func (s *Service) AddSubtask(ctx context.Context, taskID, memberID uint64, title string) (domain.Task, error) {
	added, err := s.ExpandTask(ctx, taskID, memberID, domain.Drafts([]string{title}))
	if err != nil {
		return domain.Task{}, err
	}
	return added[0], nil
}

// ExpandTask adds 1 to 50 subtasks at the end of the task's subtasks, in one
// transaction.
func (s *Service) ExpandTask(ctx context.Context, taskID, memberID uint64, drafts []domain.SubtaskDraft) ([]domain.Task, error) {
	parent, err := s.task(ctx, taskID, memberID)
	if err != nil {
		return nil, err
	}
	for _, d := range drafts {
		if d.WorkType == "" {
			continue
		}
		b, _, err := s.boards.ByID(ctx, parent.BoardID)
		if err != nil {
			return nil, err
		}
		if err := s.checkWorkType(ctx, b.GuildID, d.WorkType); err != nil {
			return nil, err
		}
	}
	for range positionAttempts {
		siblings, err := s.tasks.SubtasksOf(ctx, parent.ID)
		if err != nil {
			return nil, err
		}
		last := ""
		if len(siblings) > 0 {
			last = siblings[len(siblings)-1].Position
		}
		subtasks, err := domain.Expand(parent, drafts, last)
		if err != nil {
			return nil, err
		}
		added, err := s.tasks.AddAll(ctx, subtasks)
		if errors.Is(err, ErrPositionTaken) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, t := range added {
			ev := t.Added()
			ev.ActorID = memberID
			s.events.Publish(ctx, ev)
		}
		return added, nil
	}
	return nil, ErrPositionTaken
}

// CreateTask adds a task at the bottom of the board's first column, with a
// work type key of the guild ("" for none).
func (s *Service) CreateTask(ctx context.Context, boardID, memberID uint64, title, description, workType string) (domain.Task, error) {
	b, err := s.writableBoard(ctx, boardID, memberID)
	if err != nil {
		return domain.Task{}, err
	}
	if err := s.checkWorkType(ctx, b.GuildID, workType); err != nil {
		return domain.Task{}, err
	}
	first := b.First()
	for range positionAttempts {
		inColumn, err := s.tasks.InColumn(ctx, first.ID)
		if err != nil {
			return domain.Task{}, err
		}
		last := ""
		if len(inColumn) > 0 {
			last = inColumn[len(inColumn)-1].Position
		}
		t, ev, err := domain.NewTask(b.ID, first.ID, title, description, last)
		if err != nil {
			return domain.Task{}, err
		}
		t.WorkType = workType
		t, err = s.tasks.Add(ctx, t)
		if errors.Is(err, ErrPositionTaken) {
			continue
		}
		if err != nil {
			return domain.Task{}, err
		}
		ev.TaskID, ev.ActorID = t.ID, memberID
		s.events.Publish(ctx, ev)
		return t, nil
	}
	return domain.Task{}, ErrPositionTaken
}

// TaskChanges are the fields UpdateTask changes; nil leaves one as it is.
// An empty WorkType clears it.
type TaskChanges struct {
	Title       *string
	Description *string
	WorkType    *string
	Done        *bool
	// PrioritizedAgentID prioritizes the task for one agent (0 clears it);
	// Forbidden keeps agents off it.
	PrioritizedAgentID *uint64
	Forbidden          *bool
}

// UpdateTask changes a task's title, description and work type, and ticks a
// subtask off or on.
func (s *Service) UpdateTask(ctx context.Context, taskID, memberID uint64, c TaskChanges) (domain.Task, error) {
	title, description, done := c.Title, c.Description, c.Done
	t, err := s.task(ctx, taskID, memberID)
	if err != nil {
		return domain.Task{}, err
	}
	if c.WorkType != nil {
		b, _, err := s.boards.ByID(ctx, t.BoardID)
		if err != nil {
			return domain.Task{}, err
		}
		if err := s.checkWorkType(ctx, b.GuildID, *c.WorkType); err != nil {
			return domain.Task{}, err
		}
	}
	edited, err := t.Edit(title, description, c.WorkType)
	if err != nil {
		return domain.Task{}, err
	}
	drafted, err := t.Draft(c.PrioritizedAgentID, c.Forbidden)
	if err != nil {
		return domain.Task{}, err
	}
	if drafted {
		if edited == nil {
			edited = &domain.TaskEdited{TaskID: t.ID, BoardID: t.BoardID, ParentID: t.ParentID}
		}
		edited.Drafting = true
	}
	var completed *domain.SubtaskCompleted
	var reopened *domain.SubtaskReopened
	if done != nil {
		if *done {
			completed, err = t.Complete()
		} else {
			reopened, err = t.Reopen()
		}
		if err != nil {
			return domain.Task{}, err
		}
	}
	if err := s.tasks.Save(ctx, t); err != nil {
		return domain.Task{}, err
	}
	if edited != nil {
		edited.ActorID = memberID
		s.events.Publish(ctx, *edited)
	}
	if completed != nil {
		completed.ActorID = memberID
		s.events.Publish(ctx, *completed)
	}
	if reopened != nil {
		reopened.ActorID = memberID
		s.events.Publish(ctx, *reopened)
	}
	return t, nil
}

// MoveTask puts a task in column right after the task afterID and/or right
// before the task beforeID. With neither, it goes to the bottom of the
// column. A subtask has no column: it moves among its siblings the same way,
// and column is ignored.
func (s *Service) MoveTask(ctx context.Context, taskID, memberID uint64, column ColumnRef, afterID, beforeID *uint64) (domain.Task, error) {
	t, err := s.task(ctx, taskID, memberID)
	if err != nil {
		return domain.Task{}, err
	}
	if t.IsSubtask() {
		return s.moveSubtask(ctx, t, memberID, afterID, beforeID)
	}
	b, found, err := s.boards.ByID(ctx, t.BoardID)
	if err != nil {
		return domain.Task{}, err
	}
	if !found {
		return domain.Task{}, ErrBoardNotFound
	}
	target, err := column.on(b)
	if err != nil {
		return domain.Task{}, err
	}
	col := target.ID
	for range positionAttempts {
		inColumn, err := s.tasks.InColumn(ctx, col)
		if err != nil {
			return domain.Task{}, err
		}
		above, below, err := neighbours(inColumn, t.ID, afterID, beforeID)
		if err != nil {
			return domain.Task{}, err
		}
		moved := t
		ev, err := moved.Move(col, above, below)
		if errors.Is(err, domain.ErrInvalidPosition) {
			// after/before given in the wrong order.
			return domain.Task{}, ErrInvalidNeighbour
		}
		if err != nil {
			return domain.Task{}, err
		}
		err = s.tasks.Save(ctx, moved)
		if errors.Is(err, ErrPositionTaken) {
			continue
		}
		if err != nil {
			return domain.Task{}, err
		}
		ev.ActorID = memberID
		s.events.Publish(ctx, ev)
		return moved, nil
	}
	return domain.Task{}, ErrPositionTaken
}

func (s *Service) moveSubtask(ctx context.Context, t domain.Task, memberID uint64, afterID, beforeID *uint64) (domain.Task, error) {
	for range positionAttempts {
		siblings, err := s.tasks.SubtasksOf(ctx, *t.ParentID)
		if err != nil {
			return domain.Task{}, err
		}
		above, below, err := neighbours(siblings, t.ID, afterID, beforeID)
		if errors.Is(err, ErrInvalidNeighbour) {
			return domain.Task{}, ErrInvalidSibling
		}
		if err != nil {
			return domain.Task{}, err
		}
		moved := t
		ev, err := moved.Reposition(above, below)
		if errors.Is(err, domain.ErrInvalidPosition) {
			return domain.Task{}, ErrInvalidSibling
		}
		if err != nil {
			return domain.Task{}, err
		}
		err = s.tasks.Save(ctx, moved)
		if errors.Is(err, ErrPositionTaken) {
			continue
		}
		if err != nil {
			return domain.Task{}, err
		}
		ev.ActorID = memberID
		s.events.Publish(ctx, ev)
		return moved, nil
	}
	return domain.Task{}, ErrPositionTaken
}

// ColumnRef names a column of a board by id or, when ID is 0, by name.
type ColumnRef struct {
	ID   uint64
	Name string
}

// legacyColumns are the keys of the four fixed columns boards had before
// they owned their columns. Desktop releases up to 0.1.3 still send them.
var legacyColumns = map[string]string{"backlog": "Backlog", "todo": "To do", "doing": "Doing", "done": "Done"}

func (r ColumnRef) on(b domain.Board) (domain.Column, error) {
	if r.ID != 0 {
		if c, ok := b.Column(r.ID); ok {
			return c, nil
		}
		return domain.Column{}, domain.ErrColumnNotFound
	}
	if c, ok := b.ColumnNamed(r.Name); ok {
		return c, nil
	}
	if name, ok := legacyColumns[r.Name]; ok {
		if c, ok := b.ColumnNamed(name); ok {
			return c, nil
		}
	}
	return domain.Column{}, domain.ErrColumnNotFound
}

// neighbours finds the positions just above and below where the moved task
// should land, among the column's other tasks.
func neighbours(inColumn []domain.Task, movedID uint64, afterID, beforeID *uint64) (above, below string, err error) {
	others := make([]domain.Task, 0, len(inColumn))
	for _, t := range inColumn {
		if t.ID != movedID {
			others = append(others, t)
		}
	}
	index := func(id uint64) int {
		for i, t := range others {
			if t.ID == id {
				return i
			}
		}
		return -1
	}
	switch {
	case afterID != nil:
		i := index(*afterID)
		if i < 0 {
			return "", "", ErrInvalidNeighbour
		}
		above = others[i].Position
		if beforeID != nil {
			j := index(*beforeID)
			if j < 0 {
				return "", "", ErrInvalidNeighbour
			}
			below = others[j].Position
		} else if i+1 < len(others) {
			below = others[i+1].Position
		}
	case beforeID != nil:
		j := index(*beforeID)
		if j < 0 {
			return "", "", ErrInvalidNeighbour
		}
		below = others[j].Position
		if j > 0 {
			above = others[j-1].Position
		}
	default:
		if len(others) > 0 {
			above = others[len(others)-1].Position
		}
	}
	return above, below, nil
}

func (s *Service) DeleteTask(ctx context.Context, taskID, memberID uint64) error {
	t, err := s.task(ctx, taskID, memberID)
	if err != nil {
		return err
	}
	if err := s.tasks.Delete(ctx, t.ID); err != nil {
		return err
	}
	ev := t.Deleted()
	ev.ActorID = memberID
	s.events.Publish(ctx, ev)
	return nil
}
