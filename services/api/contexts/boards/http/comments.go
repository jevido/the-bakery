package http

import (
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"

	"github.com/jevido/the-bakery/services/api/contexts/boards/app"
)

type commentJSON struct {
	ID         uint64     `json:"id"`
	TaskID     uint64     `json:"task_id"`
	AuthorID   uint64     `json:"author_id"`
	AuthorName string     `json:"author_name"`
	Body       string     `json:"body"`
	CreatedAt  time.Time  `json:"created_at"`
	EditedAt   *time.Time `json:"edited_at"`
}

func commentToJSON(c app.AuthoredComment) commentJSON {
	return commentJSON{ID: c.ID, TaskID: c.TaskID, AuthorID: c.AuthorID, AuthorName: c.AuthorName, Body: c.Body, CreatedAt: c.CreatedAt, EditedAt: c.EditedAt}
}

type commentRequest struct {
	Body string `json:"body"`
}

// ListComments returns a task's comments, oldest first.
func (c *Controller) ListComments(ctx contractshttp.Context) contractshttp.Response {
	taskID, ok := routeID(ctx, "task")
	if !ok {
		return notFound(ctx)
	}
	cs, err := c.service.ListComments(ctx.Context(), taskID, c.me(ctx))
	if err != nil {
		return failure(ctx, err)
	}
	out := make([]commentJSON, len(cs))
	for i, cm := range cs {
		out[i] = commentToJSON(cm)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"comments": out})
}

func (c *Controller) CommentOnTask(ctx contractshttp.Context) contractshttp.Response {
	taskID, ok := routeID(ctx, "task")
	if !ok {
		return notFound(ctx)
	}
	var req commentRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return badRequest(ctx)
	}
	cm, err := c.service.CommentOnTask(ctx.Context(), taskID, c.me(ctx), req.Body)
	if err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().Json(contractshttp.StatusCreated, contractshttp.Json{"comment": commentToJSON(cm)})
}

// EditComment and DeleteComment are for the comment's author only.
func (c *Controller) EditComment(ctx contractshttp.Context) contractshttp.Response {
	commentID, ok := routeID(ctx, "comment")
	if !ok {
		return notFound(ctx)
	}
	var req commentRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return badRequest(ctx)
	}
	cm, err := c.service.EditComment(ctx.Context(), commentID, c.me(ctx), req.Body)
	if err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"comment": commentToJSON(cm)})
}

func (c *Controller) DeleteComment(ctx contractshttp.Context) contractshttp.Response {
	commentID, ok := routeID(ctx, "comment")
	if !ok {
		return notFound(ctx)
	}
	if err := c.service.DeleteComment(ctx.Context(), commentID, c.me(ctx)); err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().NoContent()
}
