// Package http exposes boards and tasks over HTTP. Every route sits behind
// identity's RequireMember.
package http

import (
	"errors"
	"strconv"

	contractshttp "github.com/goravel/framework/contracts/http"

	"github.com/jevido/the-bakery/services/api/app/facades"
	"github.com/jevido/the-bakery/services/api/app/refusal"
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
	ColumnID      *uint64 `json:"column_id"`
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	Position      string  `json:"position"`
	Done          bool    `json:"done"`
	WorkType      *string `json:"work_type"`
	SubtasksTotal *int    `json:"subtasks_total,omitempty"`
	SubtasksDone  *int    `json:"subtasks_done,omitempty"`
	// Drafting: the one agent that may take the task, and whether agents
	// are kept off it. A board answer also says which claim holds it.
	PrioritizedAgentID *uint64    `json:"prioritized_agent_id"`
	Forbidden          bool       `json:"forbidden"`
	Claim              *claimJSON `json:"claim,omitempty"`
}

func taskToJSON(t domain.Task) taskJSON {
	out := taskJSON{ID: t.ID, BoardID: t.BoardID, ParentID: t.ParentID, Title: t.Title, Description: t.Description, Position: t.Position, Done: t.Done,
		PrioritizedAgentID: t.PrioritizedAgentID, Forbidden: t.Forbidden}
	if !t.IsSubtask() {
		out.ColumnID = &t.ColumnID
	}
	if t.WorkType != "" {
		out.WorkType = &t.WorkType
	}
	return out
}

func withCounts(out taskJSON, c app.SubtaskCount) taskJSON {
	out.SubtasksTotal, out.SubtasksDone = &c.Total, &c.Done
	return out
}

// columnJSON is a board column. Key is the old fixed-column key
// (backlog, todo, doing, done) for a column that still has its default
// name, else the name: desktop releases up to 0.1.3 read and send it.
type columnJSON struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
	Key  string `json:"column"`
}

// boardColumnJSON is a column in a board answer, with its tasks in order.
type boardColumnJSON struct {
	columnJSON
	Tasks []taskJSON `json:"tasks"`
}

var legacyKeys = map[string]string{"Backlog": "backlog", "To do": "todo", "Doing": "doing", "Done": "done"}

func columnToJSON(c domain.Column) columnJSON {
	key, ok := legacyKeys[c.Name]
	if !ok {
		key = c.Name
	}
	return columnJSON{ID: c.ID, Name: c.Name, Key: key}
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
	v, err := c.service.GetBoard(ctx.Context(), boardID, c.me(ctx))
	if err != nil {
		return failure(ctx, err)
	}
	b := v.Board
	byColumn := map[uint64][]taskJSON{}
	for _, t := range v.Tasks {
		out := withCounts(taskToJSON(t), v.Counts[t.ID])
		if cl, ok := v.Claims[t.ID]; ok {
			j := claimToJSON(cl)
			out.Claim = &j
		}
		byColumn[t.ColumnID] = append(byColumn[t.ColumnID], out)
	}
	columns := make([]boardColumnJSON, len(b.Columns))
	for i, col := range b.Columns {
		columns[i] = boardColumnJSON{columnJSON: columnToJSON(col), Tasks: byColumn[col.ID]}
		if columns[i].Tasks == nil {
			columns[i].Tasks = []taskJSON{}
		}
	}
	return ctx.Response().Success().Json(contractshttp.Json{"board": boardToJSON(b), "columns": columns})
}

type createTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	WorkType    string `json:"work_type"`
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
	t, err := c.service.CreateTask(ctx.Context(), boardID, c.me(ctx), req.Title, req.Description, req.WorkType)
	if err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().Json(contractshttp.StatusCreated, contractshttp.Json{"task": taskToJSON(t)})
}

// updateTaskRequest: a field left out stays as it is; work_type "" clears
// the task's work type.
type updateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	WorkType    *string `json:"work_type"`
	Done        *bool   `json:"done"`
	// PrioritizedAgentID 0 clears the priority.
	PrioritizedAgentID *uint64 `json:"prioritized_agent_id"`
	Forbidden          *bool   `json:"forbidden"`
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
	t, err := c.service.UpdateTask(ctx.Context(), taskID, c.me(ctx), app.TaskChanges{
		Title: req.Title, Description: req.Description, WorkType: req.WorkType, Done: req.Done,
		PrioritizedAgentID: req.PrioritizedAgentID, Forbidden: req.Forbidden,
	})
	if err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"task": taskToJSON(t)})
}

// moveTaskRequest: the task lands in column_id right after after_id and/or
// right before before_id; with neither it goes to the bottom. column (a
// column name, or an old fixed key) is accepted instead of column_id for
// desktop releases up to 0.1.3.
type moveTaskRequest struct {
	ColumnID uint64  `json:"column_id"`
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
	t, err := c.service.MoveTask(ctx.Context(), taskID, c.me(ctx), app.ColumnRef{ID: req.ColumnID, Name: req.Column}, req.AfterID, req.BeforeID)
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

// expandTaskRequest takes either titles, or subtasks with a description and
// a work type each.
type expandTaskRequest struct {
	Titles   []string `json:"titles"`
	Subtasks []struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		WorkType    string `json:"work_type"`
	} `json:"subtasks"`
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
	drafts := domain.Drafts(req.Titles)
	if len(req.Subtasks) > 0 {
		drafts = make([]domain.SubtaskDraft, len(req.Subtasks))
		for i, st := range req.Subtasks {
			drafts[i] = domain.SubtaskDraft{Title: st.Title, Description: st.Description, WorkType: st.WorkType}
		}
	}
	added, err := c.service.ExpandTask(ctx.Context(), taskID, c.me(ctx), drafts)
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
	// A sanction (of the member or the guild) answers for itself.
	if r, ok := refusal.As(err); ok {
		return refusal.Respond(ctx, r)
	}
	status := contractshttp.StatusInternalServerError
	field := ""
	switch {
	case errors.Is(err, app.ErrNotMember), errors.Is(err, domain.ErrNotAuthor), errors.Is(err, domain.ErrNotRunOwner),
		errors.Is(err, domain.ErrNotClaimant), errors.Is(err, app.ErrNotYourAgent):
		status = contractshttp.StatusForbidden
	case errors.Is(err, app.ErrBoardNotFound), errors.Is(err, app.ErrTaskNotFound), errors.Is(err, app.ErrCommentNotFound), errors.Is(err, app.ErrRunNotFound),
		errors.Is(err, app.ErrClaimNotFound):
		status = contractshttp.StatusNotFound
	case errors.Is(err, app.ErrPositionTaken), errors.Is(err, app.ErrGuildArchived):
		status = contractshttp.StatusConflict
	case errors.Is(err, domain.ErrInvalidBoardName):
		status, field = contractshttp.StatusUnprocessableEntity, "name"
	case errors.Is(err, domain.ErrInvalidTitle):
		status, field = contractshttp.StatusUnprocessableEntity, "title"
	case errors.Is(err, domain.ErrColumnNotFound):
		status, field = contractshttp.StatusUnprocessableEntity, "column_id"
	case errors.Is(err, domain.ErrInvalidColumnName), errors.Is(err, domain.ErrDuplicateColumn):
		status, field = contractshttp.StatusUnprocessableEntity, "name"
	case errors.Is(err, domain.ErrColumnNotEmpty), errors.Is(err, domain.ErrLastColumn):
		status = contractshttp.StatusUnprocessableEntity
	case errors.Is(err, domain.ErrInvalidColumnOrder):
		status, field = contractshttp.StatusUnprocessableEntity, "after_id"
	case errors.Is(err, app.ErrInvalidNeighbour), errors.Is(err, app.ErrInvalidSibling):
		status, field = contractshttp.StatusUnprocessableEntity, "after_id"
	case errors.Is(err, domain.ErrTooManySubtasks):
		status, field = contractshttp.StatusUnprocessableEntity, "titles"
	case errors.Is(err, domain.ErrNotSubtask):
		status, field = contractshttp.StatusUnprocessableEntity, "done"
	case errors.Is(err, domain.ErrUnknownWorkType):
		status, field = contractshttp.StatusUnprocessableEntity, "work_type"
	case errors.Is(err, domain.ErrInvalidWorkTypeKey), errors.Is(err, domain.ErrDuplicateWorkType):
		status, field = contractshttp.StatusUnprocessableEntity, "key"
	case errors.Is(err, domain.ErrInvalidWorkTypeName):
		status, field = contractshttp.StatusUnprocessableEntity, "name"
	case errors.Is(err, domain.ErrWorkTypeInUse):
		status = contractshttp.StatusConflict
	case errors.Is(err, app.ErrWorkTypeNotFound):
		status = contractshttp.StatusNotFound
	case errors.Is(err, domain.ErrInvalidCommentBody):
		status, field = contractshttp.StatusUnprocessableEntity, "body"
	case errors.Is(err, domain.ErrTaskClaimed), errors.Is(err, domain.ErrTaskForbidden), errors.Is(err, domain.ErrPrioritizedForOther):
		status = contractshttp.StatusConflict
	case errors.Is(err, domain.ErrClaimEnded):
		status = contractshttp.StatusGone
	case errors.Is(err, domain.ErrClaimSubtask), errors.Is(err, domain.ErrDraftSubtask):
		status = contractshttp.StatusUnprocessableEntity
	case errors.Is(err, domain.ErrInvalidMachineID):
		status, field = contractshttp.StatusUnprocessableEntity, "machine_id"
	case errors.Is(err, domain.ErrRunAlreadyEnded), errors.Is(err, app.ErrRunOnSubtask):
		status = contractshttp.StatusUnprocessableEntity
	case errors.Is(err, domain.ErrInvalidRunStatus):
		status, field = contractshttp.StatusUnprocessableEntity, "status"
	case errors.Is(err, domain.ErrInvalidRunStats):
		status, field = contractshttp.StatusUnprocessableEntity, "cost_usd"
	case errors.Is(err, domain.ErrInvalidRunAgent):
		status, field = contractshttp.StatusUnprocessableEntity, "agent_name"
	case errors.Is(err, domain.ErrInvalidRunKind):
		status, field = contractshttp.StatusUnprocessableEntity, "kind"
	case errors.Is(err, domain.ErrInvalidRunDetails):
		status, field = contractshttp.StatusUnprocessableEntity, "machine"
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
