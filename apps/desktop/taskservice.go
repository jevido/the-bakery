package main

import (
	"context"

	"github.com/jevido/the-bakery/apps/desktop/internal/api"
	"github.com/jevido/the-bakery/apps/desktop/internal/session"
)

// TaskService gives the task panel one task in full: its title,
// description and subtasks. Like BoardsService, it keeps the token in Go.
type TaskService struct {
	client  *api.Client
	session *session.Session
}

func NewTaskService(client *api.Client, s *session.Session) *TaskService {
	return &TaskService{client: client, session: s}
}

func (s *TaskService) GetTask(ctx context.Context, taskID uint64) (api.TaskDetail, error) {
	return call(s.session, func(t string) (api.TaskDetail, error) { return s.client.GetTask(ctx, t, taskID) })
}

// UpdateTask changes a task's or subtask's title and/or description; nil
// leaves one as it is.
func (s *TaskService) UpdateTask(ctx context.Context, taskID uint64, title, description *string) (api.Task, error) {
	return call(s.session, func(t string) (api.Task, error) {
		return s.client.UpdateTask(ctx, t, taskID, title, description)
	})
}

func (s *TaskService) AddSubtask(ctx context.Context, taskID uint64, title string) (api.Task, error) {
	return call(s.session, func(t string) (api.Task, error) { return s.client.AddSubtask(ctx, t, taskID, title) })
}

func (s *TaskService) SetSubtaskDone(ctx context.Context, taskID uint64, done bool) (api.Task, error) {
	return call(s.session, func(t string) (api.Task, error) { return s.client.SetSubtaskDone(ctx, t, taskID, done) })
}

func (s *TaskService) MoveSubtask(ctx context.Context, taskID uint64, afterID, beforeID *uint64) (api.Task, error) {
	return call(s.session, func(t string) (api.Task, error) {
		return s.client.MoveSubtask(ctx, t, taskID, afterID, beforeID)
	})
}

// DeleteTask deletes a task or subtask.
func (s *TaskService) DeleteTask(ctx context.Context, taskID uint64) error {
	_, err := call(s.session, func(t string) (struct{}, error) { return struct{}{}, s.client.DeleteTask(ctx, t, taskID) })
	return err
}
