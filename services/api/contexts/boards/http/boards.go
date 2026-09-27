// Package http exposes boards and tasks over HTTP. Every route sits behind
// identity's RequireMember.
package http

import (
	"errors"
	"strconv"

	contractshttp "github.com/goravel/framework/contracts/http"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/contexts/boards/app"
	"github.com/jevido/the-bakery/services/api/contexts/boards/domain"
)

// MemberID returns the signed-in member's id; identity's, handed in by the
// context's wiring.
type MemberID func(ctx contractshttp.Context) (uint64, bool)

type Controller struct {
	service  *app.Service
	memberID MemberID
}

func NewController(service *app.Service, memberID MemberID) *Controller {
	return &Controller{service: service, memberID: memberID}
}

type boardJSON struct {
	ID      uint64 `json:"id"`
	GuildID uint64 `json:"guild_id"`
	Name    string `json:"name"`
}

func boardToJSON(b domain.Board) boardJSON {
	return boardJSON{ID: b.ID, GuildID: b.GuildID, Name: b.Name}
}

// taskJSON is a task or a subtask. A subtask has a parent_id and no
// column; a top-level task carries its subtask counts where they are known.
type taskJSON struct {
	ID            uint64  `json:"id"`
	BoardID       uint64  `json:"board_id"`
	ParentID      *uint64 `json:"parent_id"`
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	Column        string  `json:"column,omitempty"`
	Position      string  `json:"position"`
	Done          bool    `json:"done"`
	SubtasksTotal *int    `json:"subtasks_total,omitempty"`
	SubtasksDone  *int    `json:"subtasks_done,omitempty"`
}

func taskToJSON(t domain.Task) taskJSON {
	out := taskJSON{ID: t.ID, BoardID: t.BoardID, ParentID: t.ParentID, Title: t.Title, Description: t.Description, Position: t.Position, Done: t.Done}
	if !t.IsSubtask() {
		out.Column = string(t.Column)
	}
	return out
}

func withCounts(out taskJSON, c app.SubtaskCount) taskJSON {
	out.SubtasksTotal, out.SubtasksDone = &c.Total, &c.Done
	return out
}

type columnJSON struct {
	Column string     `json:"column"`
	Tasks  []taskJSON `json:"tasks"`
}

func (c *Controller) ListBoards(ctx contractshttp.Context) contractshttp.Response {
	guildID, ok := routeID(ctx, "guild")
	if !ok {
		return notFound(ctx)
	}
	boards, err := c.service.ListBoards(ctx.Context(), guildID, c.me(ctx))
	if err != nil {
		return failure(ctx, err)
	}
	out := make([]boardJSON, len(boards))
	for i, b := range boards {
		out[i] = boardToJSON(b)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"boards": out})
}

type createBoardRequest struct {
	Name string `json:"name"`
}

func (c *Controller) CreateBoard(ctx contractshttp.Context) contractshttp.Response {
	guildID, ok := routeID(ctx, "guild")
	if !ok {
		return notFound(ctx)
	}
	var req createBoardRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return badRequest(ctx)
	}
	b, err := c.service.CreateBoard(ctx.Context(), guildID, c.me(ctx), req.Name)
	if err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().Json(contractshttp.StatusCreated, contractshttp.Json{"board": boardToJSON(b)})
}

// GetBoard returns the board with every column in board order, each with
// its tasks in position order.
func (c *Controller) GetBoard(ctx contractshttp.Context) contractshttp.Response {
	boardID, ok := routeID(ctx, "board")
	if !ok {
		return notFound(ctx)
	}
	b, tasks, counts, err := c.service.GetBoard(ctx.Context(), boardID, c.me(ctx))
	if err != nil {
		return failure(ctx, err)
	}
	byColumn := map[domain.Column][]taskJSON{}
	for _, t := range tasks {
		byColumn[t.Column] = append(byColumn[t.Column], withCounts(taskToJSON(t), counts[t.ID]))
	}
	columns := make([]columnJSON, len(domain.Columns))
	for i, col := range domain.Columns {
		columns[i] = columnJSON{Column: string(col), Tasks: byColumn[col]}
		if columns[i].Tasks == nil {
			columns[i].Tasks = []taskJSON{}
		}
	}
	return ctx.Response().Success().Json(contractshttp.Json{"board": boardToJSON(b), "columns": columns})
}

type createTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (c *Controller) CreateTask(ctx contractshttp.Context) contractshttp.Response {
	boardID, ok := routeID(ctx, "board")
	if !ok {
		return notFound(ctx)
	}
	var req createTaskRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return badRequest(ctx)
	}
	t, err := c.service.CreateTask(ctx.Context(), boardID, c.me(ctx), req.Title, req.Description)
	if err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().Json(contractshttp.StatusCreated, contractshttp.Json{"task": taskToJSON(t)})
}

type updateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Done        *bool   `json:"done"`
}

func (c *Controller) UpdateTask(ctx contractshttp.Context) contractshttp.Response {
	taskID, ok := routeID(ctx, "task")
	if !ok {
		return notFound(ctx)
	}
	var req updateTaskRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return badRequest(ctx)
	}
	t, err := c.service.UpdateTask(ctx.Context(), taskID, c.me(ctx), req.Title, req.Description, req.Done)
	if err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"task": taskToJSON(t)})
}

// moveTaskRequest: the task lands right after after_id and/or right before
// before_id; with neither it goes to the bottom of column.
type moveTaskRequest struct {
	Column   string  `json:"column"`
	AfterID  *uint64 `json:"after_id"`
	BeforeID *uint64 `json:"before_id"`
}

func (c *Controller) MoveTask(ctx contractshttp.Context) contractshttp.Response {
	taskID, ok := routeID(ctx, "task")
	if !ok {
		return notFound(ctx)
	}
	var req moveTaskRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return badRequest(ctx)
	}
	t, err := c.service.MoveTask(ctx.Context(), taskID, c.me(ctx), req.Column, req.AfterID, req.BeforeID)
	if err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"task": taskToJSON(t)})
}

// GetTask returns a task with its subtasks in order.
func (c *Controller) GetTask(ctx contractshttp.Context) contractshttp.Response {
	taskID, ok := routeID(ctx, "task")
	if !ok {
		return notFound(ctx)
	}
	t, subtasks, err := c.service.GetTask(ctx.Context(), taskID, c.me(ctx))
	if err != nil {
		return failure(ctx, err)
	}
	out := taskToJSON(t)
	if !t.IsSubtask() {
		done := 0
		for _, st := range subtasks {
			if st.Done {
				done++
			}
		}
		out = withCounts(out, app.SubtaskCount{Total: len(subtasks), Done: done})
	}
	return ctx.Response().Success().Json(contractshttp.Json{"task": out, "subtasks": tasksToJSON(subtasks)})
}

type addSubtaskRequest struct {
	Title string `json:"title"`
}

func (c *Controller) AddSubtask(ctx contractshttp.Context) contractshttp.Response {
	taskID, ok := routeID(ctx, "task")
	if !ok {
		return notFound(ctx)
	}
	var req addSubtaskRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return badRequest(ctx)
	}
	t, err := c.service.AddSubtask(ctx.Context(), taskID, c.me(ctx), req.Title)
	if err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().Json(contractshttp.StatusCreated, contractshttp.Json{"subtask": taskToJSON(t)})
}

type expandTaskRequest struct {
	Titles []string `json:"titles"`
}

// ExpandTask adds several subtasks at once, all or none.
func (c *Controller) ExpandTask(ctx contractshttp.Context) contractshttp.Response {
	taskID, ok := routeID(ctx, "task")
	if !ok {
		return notFound(ctx)
	}
	var req expandTaskRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return badRequest(ctx)
	}
	added, err := c.service.ExpandTask(ctx.Context(), taskID, c.me(ctx), req.Titles)
	if err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().Json(contractshttp.StatusCreated, contractshttp.Json{"subtasks": tasksToJSON(added)})
}

func tasksToJSON(ts []domain.Task) []taskJSON {
	out := make([]taskJSON, len(ts))
	for i, t := range ts {
		out[i] = taskToJSON(t)
	}
	return out
}

func (c *Controller) DeleteTask(ctx contractshttp.Context) contractshttp.Response {
	taskID, ok := routeID(ctx, "task")
	if !ok {
		return notFound(ctx)
	}
	if err := c.service.DeleteTask(ctx.Context(), taskID, c.me(ctx)); err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().NoContent()
}

func (c *Controller) me(ctx contractshttp.Context) uint64 {
	id, _ := c.memberID(ctx)
	return id
}

func routeID(ctx contractshttp.Context, key string) (uint64, bool) {
	id, err := strconv.ParseUint(ctx.Request().Route(key), 10, 64)
	return id, err == nil
}

func failure(ctx contractshttp.Context, err error) contractshttp.Response {
	status := contractshttp.StatusInternalServerError
	field := ""
	switch {
	case errors.Is(err, app.ErrNotMember):
		status = contractshttp.StatusForbidden
	case errors.Is(err, app.ErrBoardNotFound), errors.Is(err, app.ErrTaskNotFound):
		status = contractshttp.StatusNotFound
	case errors.Is(err, app.ErrPositionTaken), errors.Is(err, app.ErrGuildArchived):
		status = contractshttp.StatusConflict
	case errors.Is(err, domain.ErrInvalidBoardName):
		status, field = contractshttp.StatusUnprocessableEntity, "name"
	case errors.Is(err, domain.ErrInvalidTitle):
		status, field = contractshttp.StatusUnprocessableEntity, "title"
	case errors.Is(err, domain.ErrInvalidColumn):
		status, field = contractshttp.StatusUnprocessableEntity, "column"
	case errors.Is(err, app.ErrInvalidNeighbour), errors.Is(err, app.ErrInvalidSibling):
		status, field = contractshttp.StatusUnprocessableEntity, "after_id"
	case errors.Is(err, domain.ErrTooManySubtasks):
		status, field = contractshttp.StatusUnprocessableEntity, "titles"
	case errors.Is(err, domain.ErrNotSubtask):
		status, field = contractshttp.StatusUnprocessableEntity, "done"
	case errors.Is(err, domain.ErrNestedSubtask):
		status = contractshttp.StatusUnprocessableEntity
	}
	if status == contractshttp.StatusInternalServerError {
		facades.Log().WithContext(ctx.Context()).Error(err)
		return ctx.Response().Json(status, contractshttp.Json{"error": "something went wrong"})
	}
	body := contractshttp.Json{"error": err.Error()}
	if field != "" {
		body["field"] = field
	}
	return ctx.Response().Json(status, body)
}

func notFound(ctx contractshttp.Context) contractshttp.Response {
	return ctx.Response().Json(contractshttp.StatusNotFound, contractshttp.Json{"error": "not found"})
}

func badRequest(ctx contractshttp.Context) contractshttp.Response {
	return ctx.Response().Json(contractshttp.StatusBadRequest, contractshttp.Json{"error": "request body must be JSON"})
}
