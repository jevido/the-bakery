package http

import (
	"errors"

	contractshttp "github.com/goravel/framework/contracts/http"

	"github.com/jevido/the-bakery/services/api/contexts/boards/domain"
)

type columnNameRequest struct {
	Name string `json:"name"`
}

// columnFailure is failure, except that a column named in the URL that does
// not exist is 404 (in a move's body it is a field to fix, 422).
func columnFailure(ctx contractshttp.Context, err error) contractshttp.Response {
	if errors.Is(err, domain.ErrColumnNotFound) {
		return ctx.Response().Json(contractshttp.StatusNotFound, contractshttp.Json{"error": err.Error()})
	}
	return failure(ctx, err)
}

// AddColumn puts a new column at the end of the board.
func (c *Controller) AddColumn(ctx contractshttp.Context) contractshttp.Response {
	boardID, ok := routeID(ctx, "board")
	if !ok {
		return notFound(ctx)
	}
	var req columnNameRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return badRequest(ctx)
	}
	col, err := c.service.AddColumn(ctx.Context(), boardID, c.me(ctx), req.Name)
	if err != nil {
		return failure(ctx, err)
	}
	return ctx.Response().Json(contractshttp.StatusCreated, contractshttp.Json{"column": columnToJSON(col)})
}

func (c *Controller) RenameColumn(ctx contractshttp.Context) contractshttp.Response {
	columnID, ok := routeID(ctx, "column")
	if !ok {
		return notFound(ctx)
	}
	var req columnNameRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return badRequest(ctx)
	}
	col, err := c.service.RenameColumn(ctx.Context(), columnID, c.me(ctx), req.Name)
	if err != nil {
		return columnFailure(ctx, err)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"column": columnToJSON(col)})
}

type moveColumnRequest struct {
	AfterID  *uint64 `json:"after_id"`
	BeforeID *uint64 `json:"before_id"`
}

// MoveColumn puts the column right after after_id and/or right before
// before_id; with neither, at the end.
func (c *Controller) MoveColumn(ctx contractshttp.Context) contractshttp.Response {
	columnID, ok := routeID(ctx, "column")
	if !ok {
		return notFound(ctx)
	}
	var req moveColumnRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return badRequest(ctx)
	}
	col, err := c.service.MoveColumn(ctx.Context(), columnID, c.me(ctx), req.AfterID, req.BeforeID)
	if err != nil {
		return columnFailure(ctx, err)
	}
	return ctx.Response().Success().Json(contractshttp.Json{"column": columnToJSON(col)})
}

// DeleteColumn removes an empty column; a board keeps at least one.
func (c *Controller) DeleteColumn(ctx contractshttp.Context) contractshttp.Response {
	columnID, ok := routeID(ctx, "column")
	if !ok {
		return notFound(ctx)
	}
	if err := c.service.RemoveColumn(ctx.Context(), columnID, c.me(ctx)); err != nil {
		return columnFailure(ctx, err)
	}
	return ctx.Response().NoContent()
}
