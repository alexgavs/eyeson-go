// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package services

import (
	"context"
	"eyeson-go-server/internal/models"
	"sync"

	"github.com/reactivex/rxgo/v2"
)

// TaskQueueService manages the reactive task queue.
type TaskQueueService struct {
	taskChan chan rxgo.Item
}

var (
	once     sync.Once
	instance *TaskQueueService
)

// GetInstance returns the singleton instance of the TaskQueueService.
func GetInstance() *TaskQueueService {
	once.Do(func() {
		instance = &TaskQueueService{
			taskChan: make(chan rxgo.Item, 100),
		}
	})
	return instance
}

// Push adds a new task to the processing queue.
func (s *TaskQueueService) Push(task models.SyncTaskExtended) {
	s.taskChan <- rxgo.Of(task)
}

// GetObservable returns the observable stream of tasks.
func (s *TaskQueueService) GetObservable() rxgo.Observable {
	return rxgo.FromChannel(s.taskChan)
}

// PushContext is a helper to push a task with a context, if needed for cancellation.
func (s *TaskQueueService) PushContext(ctx context.Context, task models.SyncTaskExtended) {
	select {
	case s.taskChan <- rxgo.Of(task):
	case <-ctx.Done():
	}
}
